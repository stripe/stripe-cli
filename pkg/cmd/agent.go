package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/spf13/cobra"
	"golang.org/x/term"

	"github.com/stripe/stripe-cli/pkg/agentsetup"
	"github.com/stripe/stripe-cli/pkg/agentskills"
	"github.com/stripe/stripe-cli/pkg/ansi"
	"github.com/stripe/stripe-cli/pkg/docs/spinner"
	"github.com/stripe/stripe-cli/pkg/stripe"
	"github.com/stripe/stripe-cli/pkg/useragent"
	"github.com/stripe/stripe-cli/pkg/validators"
)

// agentClientID maps the AI agent identifiers returned by
// useragent.DetectAIAgent to this command's provider ids. Agents not backed by
// a provider (e.g. gemini_cli) map to "".
var agentClientID = map[string]string{
	"claude_code": agentsetup.ClientClaudeCode,
	"codex_cli":   agentsetup.ClientCodex,
	"cursor":      agentsetup.ClientCursor,
}

// providerOrder is the canonical display order for known clients. Providers not
// listed here are appended afterward in alphabetical order.
var providerOrder = []string{
	agentsetup.ClientClaudeCode,
	agentsetup.ClientCodex,
	agentsetup.ClientCursor,
}

type agentCmd struct {
	cmd *cobra.Command
}

type agentSetupCmd struct {
	cmd *cobra.Command

	statusOnly bool
	yes        bool
	force      bool
	jsonOutput bool
	client     string

	providers map[string]agentsetup.Provider

	// Skills installation is injectable so tests can avoid the network and
	// point installs at temp directories.
	skillsInstall   func(ctx context.Context, destDir string) ([]string, error)
	skillsCheck     func(ctx context.Context, destDir string) (*agentskills.DirStatus, error)
	skillsLocalDir  func() (string, error)
	skillsGlobalDir func() (string, error)
	skillsDirsExist func(localDir, globalDir string) bool

	// callingAgent returns the AI agent invoking this command (e.g.
	// "claude_code"), or "" for a human shell. Injectable for tests.
	callingAgent func() string
	// isInteractive reports whether an interactive picker can be shown.
	// Injectable for tests.
	isInteractive func() bool
}

type agentSetupJSON struct {
	Clients []agentsetup.Status `json:"clients"`
	Skills  *skillsScopesJSON   `json:"skills,omitempty"`
	Actions []agentsetup.Plan   `json:"actions,omitempty"`
	Errors  []string            `json:"errors,omitempty"`
}

type skillsScopesJSON struct {
	Local  *agentskills.DirStatus `json:"local,omitempty"`
	Global *agentskills.DirStatus `json:"global,omitempty"`
}

// skillsScopes holds the check result for local and global skill directories.
type skillsScopes struct {
	Local  agentskills.DirStatus
	Global agentskills.DirStatus
}

func newAgentCmd() *agentCmd {
	ac := &agentCmd{}
	ac.cmd = &cobra.Command{
		Use:   "agent",
		Short: "Set up AI agent tooling",
		Long:  "Set up AI agent tooling for Stripe development.",
		Args:  validators.NoArgs,
	}
	ac.cmd.AddCommand(newAgentSetupCmd().cmd)
	return ac
}

func newAgentSetupCmd() *agentSetupCmd {
	asc := &agentSetupCmd{
		providers: agentsetup.DefaultProviders(),
		skillsInstall: func(ctx context.Context, destDir string) ([]string, error) {
			return agentskills.Install(ctx, nil, destDir)
		},
		skillsCheck: func(ctx context.Context, destDir string) (*agentskills.DirStatus, error) {
			return agentskills.Check(ctx, nil, destDir)
		},
		skillsLocalDir:  func() (string, error) { return skillsDirUnder(os.Getwd) },
		skillsGlobalDir: func() (string, error) { return skillsDirUnder(os.UserHomeDir) },
		skillsDirsExist: skillsDirsExist,
		callingAgent:    func() string { return useragent.DetectAIAgent(os.Getenv) },
		isInteractive:   isInteractiveTerminal,
	}

	asc.cmd = &cobra.Command{
		Use:           "setup",
		Short:         "Install or check Stripe agent tooling",
		Long:          "Detect installed AI coding clients and install Stripe agent tooling for them.",
		Args:          validators.NoArgs,
		RunE:          asc.runSetup,
		SilenceUsage:  true,
		SilenceErrors: true,
	}
	asc.cmd.Flags().BoolVar(&asc.statusOnly, "status", false, "Check installed agent tooling without making changes")
	asc.cmd.Flags().BoolVarP(&asc.yes, "yes", "y", false, "Set up every detected client without prompting")
	asc.cmd.Flags().BoolVar(&asc.force, "force", false, "Reinstall even when agent tooling or skills are already up to date")
	asc.cmd.Flags().StringVar(&asc.client, "client", "", fmt.Sprintf("Limit setup to a single client; supported values: %s", agentsetup.SupportedProviderIDs(asc.providers)))
	asc.cmd.Flags().BoolVar(&asc.jsonOutput, "json", false, "Write machine-readable status output")

	return asc
}

// skillsDirUnder returns the .agents/skills directory beneath the base returned
// by baseDir (the current working directory for local, the home directory for
// global), matching the layout `npx skills add` produces.
func skillsDirUnder(baseDir func() (string, error)) (string, error) {
	base, err := baseDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(base, ".agents", "skills"), nil
}

func (asc *agentSetupCmd) runSetup(cmd *cobra.Command, _ []string) error {
	providers, err := asc.selectedProviders()
	if err != nil {
		return err
	}

	ctx := commandContextOrBackground(cmd)
	out := cmd.OutOrStdout()

	if asc.jsonOutput || asc.statusOnly {
		var statuses []agentsetup.Status
		detectFn := func() error {
			statuses = detectAll(providers)
			for _, s := range statuses {
				if s.Detected {
					sendAgentEvent(ctx, "Agent Setup: Client Detected", s.Client)
				}
			}
			return nil
		}
		if asc.statusOnly && !asc.jsonOutput {
			if err := spinner.New().
				WithLabel("Checking agent setup...").
				WithOutput(os.Stderr).
				Run(detectFn); err != nil {
				return err
			}
		} else if err := detectFn(); err != nil {
			return err
		}

		detected := detectedStatuses(statuses)
		view, err := asc.skillsStatusView(ctx, detected)
		if err != nil {
			return err
		}
		if asc.jsonOutput {
			return asc.writeJSON(out, providers, statuses, view)
		}
		if len(detected) > 0 {
			printStatusTable(out, detected)
			fmt.Fprintln(out)
		} else {
			printNothingDetected(out)
			fmt.Fprintln(out)
		}
		if view.show {
			printSkillsStatusTable(out, view.scopes, view.allowInstall)
		}
		return nil
	}

	statuses := detectAll(providers)

	for _, s := range statuses {
		if s.Detected {
			sendAgentEvent(ctx, "Agent Setup: Client Detected", s.Client)
		}
	}

	detected := detectedStatuses(statuses)

	var skills skillsScopes
	if asc.needsInteractiveSkills() {
		var err error
		skills, err = asc.loadSkillsScopes(ctx)
		if err != nil {
			return err
		}
	}

	sel, scope, err := asc.resolveSelection(cmd, out, detected, skills)
	if err != nil {
		return err
	}
	if sel == nil {
		return nil // nothing to do; a message was already printed
	}

	return asc.install(ctx, out, providers, *sel, scope)
}

func (asc *agentSetupCmd) loadSkillsScopes(ctx context.Context) (skillsScopes, error) {
	var skills skillsScopes
	runErr := spinner.New().
		WithLabel("Checking skills...").
		WithOutput(os.Stderr).
		Run(func() error {
			var e error
			skills, e = asc.checkSkillsScopes(ctx)
			return e
		})
	return skills, runErr
}

type skillsStatusView struct {
	scopes       skillsScopes
	show         bool
	allowInstall bool
}

func (asc *agentSetupCmd) skillsStatusView(ctx context.Context, detected []agentsetup.Status) (skillsStatusView, error) {
	if len(detected) == 0 {
		scopes, err := asc.loadSkillsScopes(ctx)
		return skillsStatusView{scopes: scopes, show: true, allowInstall: true}, err
	}
	localDir, err := asc.skillsLocalDir()
	if err != nil {
		return skillsStatusView{}, fmt.Errorf("resolving local skills directory: %w", err)
	}
	globalDir, err := asc.skillsGlobalDir()
	if err != nil {
		return skillsStatusView{}, fmt.Errorf("resolving global skills directory: %w", err)
	}
	if !asc.skillsDirsExist(localDir, globalDir) {
		return skillsStatusView{}, nil
	}
	scopes, err := asc.loadSkillsScopes(ctx)
	if err != nil {
		return skillsStatusView{}, err
	}
	if !skillsScopesHasInstalled(scopes) {
		return skillsStatusView{}, nil
	}
	return skillsStatusView{scopes: scopes, show: true}, nil
}

func (asc *agentSetupCmd) needsInteractiveSkills() bool {
	if asc.yes {
		return false
	}
	if asc.callingAgent() != "" {
		return false
	}
	return asc.isInteractive()
}

// resolveSelection decides which agents and/or skills to install. It uses the
// interactive TUI when attached to a terminal (and --yes was not passed),
// otherwise it derives the selection from flags.
// A nil Selection means there is nothing to do and a message has been printed.
func (asc *agentSetupCmd) resolveSelection(cmd *cobra.Command, out io.Writer, detected []agentsetup.Status, skills skillsScopes) (*Selection, string, error) {
	// Detect the calling agent up front.
	agent := asc.callingAgent()

	// Inside a coding agent (and no explicit --client), only ever set up that
	// agent — never other clients, even with --yes, since the agent can't act on
	// another client's plugin. --client is the explicit escape hatch.
	if agent != "" && asc.client == "" {
		if id := agentClientID[agent]; id != "" {
			if scoped := statusesForClient(detected, id); len(scoped) > 0 {
				fmt.Fprintf(out, "Detected %s — setting up its Stripe plugin.\n", scoped[0].DisplayName)
				return &Selection{Agents: scoped}, "", nil
			}
			// The agent's binary isn't among the detected clients; fall through.
		} else {
			// Known agent with no Stripe plugin (e.g. Gemini CLI): install
			// client-agnostic skills instead of another client's plugin.
			fmt.Fprintf(out, "Detected %s, which has no Stripe plugin — installing Stripe skills instead.\n", agent)
			return &Selection{InstallSkills: true}, skillsScopeLocal, nil
		}
	}

	// Nothing detected: show guidance, and if interactive, offer to install skills.
	if len(detected) == 0 {
		if asc.isInteractive() && agent == "" {
			printNothingDetectedInteractive(out)
			fmt.Fprintln(out)
			chosen, ok, err := RunSkillsScopeTUI(skills)
			if err != nil {
				return nil, "", err
			}
			if !ok {
				fmt.Fprintln(out, "Canceled. No changes made.")
				return nil, "", nil
			}
			return &Selection{InstallSkills: true}, chosen, nil
		}
		printNothingDetected(out)
		return nil, "", nil
	}

	// A human at a terminal picks interactively. Agent runtimes can allocate a
	// PTY, so we also require no calling agent (checked above via the early
	// returns) before trusting the TTY.
	ctx := cmd.Context()
	useTUI := asc.isInteractive() && !asc.yes && agent == ""
	if !useTUI {
		// --yes installs every detected client (no skills).
		return &Selection{Agents: detected}, "", nil
	}

	sel, err := RunSelectionTUI(detected, skills)
	if err != nil {
		return nil, "", err
	}
	if sel == nil {
		sendAgentEvent(ctx, "Agent Setup: TUI", "canceled")
		fmt.Fprintln(out, "Canceled. No changes made.")
		return nil, "", nil
	}
	if len(sel.Agents) == 0 && !sel.InstallSkills {
		sendAgentEvent(ctx, "Agent Setup: TUI", "nothing_selected")
		fmt.Fprintln(out, "Nothing selected. No changes made.")
		return nil, "", nil
	}

	sendAgentEvent(ctx, "Agent Setup: TUI", "confirmed")

	// If skills were selected in the TUI, ask where to install them.
	if sel.InstallSkills {
		fmt.Fprintln(out)
		chosen, ok, err := RunSkillsScopeTUI(skills)
		if err != nil {
			return nil, "", err
		}
		if !ok {
			fmt.Fprintln(out, "Canceled. No changes made.")
			return nil, "", nil
		}
		return sel, chosen, nil
	}

	return sel, "", nil
}

// selectedProviders returns the providers to operate on, honoring the --client
// filter when set.
func (asc *agentSetupCmd) selectedProviders() (map[string]agentsetup.Provider, error) {
	if asc.client == "" {
		return asc.providers, nil
	}
	provider, ok := asc.providers[asc.client]
	if !ok {
		return nil, fmt.Errorf("unsupported agent client %q; supported clients: %s", asc.client, agentsetup.SupportedProviderIDs(asc.providers))
	}
	return map[string]agentsetup.Provider{asc.client: provider}, nil
}

// install applies the selection (agent plugins and/or Stripe skills), showing a
// spinner per item and printing a summary. It returns an error only when at
// least one install fails.
func (asc *agentSetupCmd) install(ctx context.Context, out io.Writer, providers map[string]agentsetup.Provider, sel Selection, scope string) error {
	fmt.Fprintln(out, "\nSetting up Stripe agent tooling:")

	color := ansi.Color(out)
	check := color.Green("✔").String()
	cross := color.Red("✗").String()
	warn := color.Yellow("⚠").String()

	var installedCount, updatedCount, skipCount, errCount int
	for _, status := range sel.Agents {
		provider := providers[status.Client]
		plan := provider.Plan(status, asc.force)

		fmt.Fprintf(out, "\n  %s\n", status.DisplayName)
		switch plan.Action {
		case agentsetup.ActionNone:
			fmt.Fprintln(out, "  already set up")
			sendAgentEvent(ctx, "Agent Setup: Plugin Install", status.Client+":skip")
			skipCount++
			continue
		case agentsetup.ActionManual:
			// Setup can't be automated (e.g. Cursor). Surface the instruction and
			// treat it as skipped, not a failure.
			fmt.Fprintf(out, "  %s manual step: %s\n", warn, plan.Manual)
			sendAgentEvent(ctx, "Agent Setup: Plugin Install", status.Client+":manual")
			skipCount++
			continue
		}

		err := spinner.New().
			WithLabel("Installing...").
			WithOutput(os.Stderr).
			Run(func() error { return provider.Apply(ctx, out, plan) })
		if err != nil {
			fmt.Fprintf(out, "  %s error: %v\n", cross, err)
			sendAgentEvent(ctx, "Agent Setup: Plugin Install", status.Client+":error")
			errCount++
			continue
		}
		fmt.Fprintf(out, "  %s done\n", check)
		sendAgentEvent(ctx, "Agent Setup: Plugin Install", status.Client+":success")
		installedCount++
	}

	if sel.InstallSkills {
		ok, installed, updated, skipped := asc.installSkills(ctx, out, scope, check, cross)
		if !ok {
			errCount++
		} else if !skipped {
			installedCount += installed
			updatedCount += updated
		}
		// Skills already up to date are a successful no-op; they don't count as skipped.
	}

	fmt.Fprintf(out, "\n%d installed, %d updated, %d skipped, %d errors\n", installedCount, updatedCount, skipCount, errCount)
	if errCount > 0 {
		return fmt.Errorf("%d item(s) failed to set up", errCount)
	}
	return nil
}

// installSkills fetches and writes Stripe skills to the local or global
// .agents/skills directory. It returns (ok, installed, updated, skipped):
// skipped is true when skills were already current and --force was not set.
func (asc *agentSetupCmd) installSkills(ctx context.Context, out io.Writer, scope, check, cross string) (bool, int, int, bool) {
	fmt.Fprintf(out, "\n  Stripe skills (%s)\n", scope)

	dirFn := asc.skillsLocalDir
	if scope == skillsScopeGlobal {
		dirFn = asc.skillsGlobalDir
	}
	dir, err := dirFn()
	if err != nil {
		fmt.Fprintf(out, "  %s error: resolving skills directory: %v\n", cross, err)
		sendAgentEvent(ctx, "Agent Setup: Skills Install", scope+":error")
		return false, 0, 0, false
	}

	var priorStatus *agentskills.DirStatus
	checkErr := spinner.New().
		WithLabel("Checking skills...").
		WithOutput(os.Stderr).
		Run(func() error {
			var e error
			priorStatus, e = asc.skillsCheck(ctx, dir)
			return e
		})
	if checkErr != nil {
		fmt.Fprintf(out, "  %s error: %v\n", cross, checkErr)
		sendAgentEvent(ctx, "Agent Setup: Skills Install", scope+":error")
		return false, 0, 0, false
	}

	if priorStatus.Status == agentskills.StatusCurrent && !asc.force {
		fmt.Fprintf(out, "  %s already up to date\n", check)
		sendAgentEvent(ctx, "Agent Setup: Skills Install", scope+":skip")
		return true, 0, 0, true
	}

	var skillNames []string
	runErr := spinner.New().
		WithLabel("Installing skills...").
		WithOutput(os.Stderr).
		Run(func() error {
			var e error
			skillNames, e = asc.skillsInstall(ctx, dir)
			return e
		})
	if runErr != nil {
		fmt.Fprintf(out, "  %s error: %v\n", cross, runErr)
		sendAgentEvent(ctx, "Agent Setup: Skills Install", scope+":error")
		return false, 0, 0, false
	}

	priorByName := make(map[string]agentskills.SkillCheck, len(priorStatus.Skills))
	for _, skill := range priorStatus.Skills {
		priorByName[skill.Name] = skill
	}

	var installed, updated int
	for _, name := range skillNames {
		prior, ok := priorByName[name]
		if !ok || prior.Status == agentskills.StatusNotInstalled {
			installed++
			continue
		}
		updated++
	}

	switch {
	case installed > 0 && updated > 0:
		fmt.Fprintf(out, "  %s installed %d and updated %d skill(s) to %s: %s\n", check, installed, updated, dir, strings.Join(skillNames, ", "))
	case updated > 0:
		fmt.Fprintf(out, "  %s updated %d skill(s) to %s: %s\n", check, updated, dir, strings.Join(skillNames, ", "))
	default:
		fmt.Fprintf(out, "  %s installed %d skill(s) to %s: %s\n", check, installed, dir, strings.Join(skillNames, ", "))
	}

	sendAgentEvent(ctx, "Agent Setup: Skills Install", scope+":success")
	return true, installed, updated, false
}

func (asc *agentSetupCmd) checkSkillsScopes(ctx context.Context) (skillsScopes, error) {
	localDir, err := asc.skillsLocalDir()
	if err != nil {
		return skillsScopes{}, fmt.Errorf("resolving local skills directory: %w", err)
	}
	globalDir, err := asc.skillsGlobalDir()
	if err != nil {
		return skillsScopes{}, fmt.Errorf("resolving global skills directory: %w", err)
	}

	local, err := asc.skillsCheck(ctx, localDir)
	if local == nil {
		return skillsScopes{}, err
	}
	global, err := asc.skillsCheck(ctx, globalDir)
	if global == nil {
		return skillsScopes{}, err
	}

	return skillsScopes{Local: *local, Global: *global}, nil
}

func skillsScopeCheckFailed(d agentskills.DirStatus) bool {
	return d.Status == agentskills.StatusError
}

func skillsScopeHasInstalled(d agentskills.DirStatus) bool {
	switch d.Status {
	case agentskills.StatusCurrent, agentskills.StatusOutOfDate:
		return true
	case agentskills.StatusError:
		return d.InstalledCount > 0
	default:
		return false
	}
}

func skillsScopesHasInstalled(scopes skillsScopes) bool {
	return skillsScopeHasInstalled(scopes.Local) || skillsScopeHasInstalled(scopes.Global)
}

func skillsDirsExist(localDir, globalDir string) bool {
	for _, dir := range []string{localDir, globalDir} {
		entries, err := os.ReadDir(dir)
		if err != nil {
			continue
		}
		for _, e := range entries {
			if e.IsDir() {
				return true
			}
		}
	}
	return false
}

func skillsScopeNeedsUpdate(d agentskills.DirStatus) bool {
	return d.Status == agentskills.StatusOutOfDate
}

func skillsScopeNeedsInstall(d agentskills.DirStatus) bool {
	return d.Status == agentskills.StatusNotInstalled
}

// skillsScopeVisible reports whether a scope should be surfaced to the user
// given the other scope's state. A not-installed scope is hidden when the other
// scope already has skills, so we don't nag about the scope the user isn't
// using. Shared by the text (--status) and JSON output paths so they stay
// consistent.
func skillsScopeVisible(d agentskills.DirStatus, scopes skillsScopes) bool {
	return !skillsScopeNeedsInstall(d) || !skillsScopesHasInstalled(scopes)
}

func (asc *agentSetupCmd) writeJSON(w io.Writer, providers map[string]agentsetup.Provider, statuses []agentsetup.Status, view skillsStatusView) error {
	result := agentSetupJSON{
		Clients: statuses,
	}
	for _, status := range statuses {
		if plan := providers[status.Client].Plan(status, asc.force); plan.Action != agentsetup.ActionNone {
			result.Actions = append(result.Actions, plan)
		}
		if status.Status == agentsetup.StatusError {
			result.Errors = append(result.Errors, status.Error)
		}
	}
	if view.show {
		result.Skills = &skillsScopesJSON{}
		if skillsScopeVisible(view.scopes.Local, view.scopes) {
			result.Skills.Local = &view.scopes.Local
		}
		if skillsScopeVisible(view.scopes.Global, view.scopes) {
			result.Skills.Global = &view.scopes.Global
		}
		if skillsScopeNeedsUpdate(view.scopes.Local) || skillsScopeNeedsUpdate(view.scopes.Global) {
			result.Actions = append(result.Actions, agentsetup.Plan{
				Action:  "update_skills",
				Command: []string{"stripe", "agent", "setup"},
			})
		} else if view.allowInstall && !skillsScopesHasInstalled(view.scopes) &&
			(skillsScopeNeedsInstall(view.scopes.Local) || skillsScopeNeedsInstall(view.scopes.Global)) {
			result.Actions = append(result.Actions, agentsetup.Plan{
				Action:  "install_skills",
				Command: []string{"stripe", "agent", "setup"},
			})
		}
	}

	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	if err := enc.Encode(result); err != nil {
		return err
	}
	if len(result.Errors) > 0 {
		return fmt.Errorf("%s", strings.Join(result.Errors, "; "))
	}
	return nil
}

// detectAll runs Detect for every provider and returns statuses in canonical
// display order.
func detectAll(providers map[string]agentsetup.Provider) []agentsetup.Status {
	ids := orderedProviderIDs(providers)
	statuses := make([]agentsetup.Status, 0, len(ids))
	for _, id := range ids {
		statuses = append(statuses, providers[id].Detect())
	}
	return statuses
}

func orderedProviderIDs(providers map[string]agentsetup.Provider) []string {
	seen := make(map[string]bool, len(providers))
	ids := make([]string, 0, len(providers))
	for _, id := range providerOrder {
		if _, ok := providers[id]; ok {
			ids = append(ids, id)
			seen[id] = true
		}
	}
	var rest []string
	for id := range providers {
		if !seen[id] {
			rest = append(rest, id)
		}
	}
	sort.Strings(rest)
	return append(ids, rest...)
}

func statusesForClient(statuses []agentsetup.Status, client string) []agentsetup.Status {
	var out []agentsetup.Status
	for _, s := range statuses {
		if s.Client == client {
			out = append(out, s)
		}
	}
	return out
}

func detectedStatuses(statuses []agentsetup.Status) []agentsetup.Status {
	var detected []agentsetup.Status
	for _, s := range statuses {
		if s.Detected {
			detected = append(detected, s)
		}
	}
	return detected
}

// printStatusTable renders a compact, aligned, color-coded view of each client
// and its Stripe plugin state. Colors are disabled automatically when the writer
// is not a TTY (via ansi.Color).
func printStatusTable(w io.Writer, statuses []agentsetup.Status) {
	color := ansi.Color(w)

	fmt.Fprintln(w, color.Bold("Detected agents with supported Stripe plugins:").String())
	fmt.Fprintln(w)

	nameWidth := 0
	for _, s := range statuses {
		if n := len(s.DisplayName); n > nameWidth {
			nameWidth = n
		}
	}

	needsSetup := false
	for _, s := range statuses {
		var icon, state string
		switch {
		case !s.Detected:
			icon = color.Faint("–").String()
			state = color.Faint("not detected").String()
		case s.Status == agentsetup.StatusError:
			icon = color.Red("✗").String()
			state = color.Red("error: " + s.Error).String()
		case s.Status == agentsetup.StatusInstalled:
			icon = color.Green("✔").String()
			state = "plugin installed"
			if detail := pluginDetail(s.Plugin); detail != "" {
				state += "  " + color.Faint(detail).String()
			}
		case s.Error != "":
			icon = color.Yellow("•").String()
			state = s.Error
		default:
			icon = color.Yellow("•").String()
			state = "plugin not installed"
			needsSetup = true
		}
		fmt.Fprintf(w, "  %s  %-*s  %s\n", icon, nameWidth, s.DisplayName, state)
	}

	if needsSetup {
		fmt.Fprintf(w, "\nRun %s to install the Stripe plugin where it's missing.\n", color.Bold("stripe agent setup").String())
	}
}

// printSkillsStatusTable renders the local and global skills install state.
func printSkillsStatusTable(w io.Writer, skills skillsScopes, allowInstall bool) {
	color := ansi.Color(w)

	fmt.Fprintln(w, color.Bold("Stripe skills:").String())
	fmt.Fprintln(w)

	scopeWidth := len("global")
	needsInstall := false
	needsUpdate := false
	for _, entry := range []struct {
		name string
		dir  agentskills.DirStatus
	}{
		{"local", skills.Local},
		{"global", skills.Global},
	} {
		if skillsScopeNeedsInstall(entry.dir) {
			needsInstall = true
			if allowInstall && skillsScopeVisible(entry.dir, skills) {
				icon := color.Yellow("•").String()
				fmt.Fprintf(w, "  %s  %-*s  %s  %s\n", icon, scopeWidth, entry.name, "not installed", color.Faint(entry.dir.Dir).String())
			}
			continue
		}
		if skillsScopeNeedsUpdate(entry.dir) {
			needsUpdate = true
		}
		icon, state := skillsScopeStatusLine(w, entry.dir)
		if state == "" {
			continue
		}
		fmt.Fprintf(w, "  %s  %-*s  %s  %s\n", icon, scopeWidth, entry.name, state, color.Faint(entry.dir.Dir).String())
	}

	switch {
	case needsUpdate:
		fmt.Fprintf(w, "\nRun %s to update your Stripe skills.\n", color.Bold("stripe agent setup").String())
	case needsInstall && allowInstall && !skillsScopesHasInstalled(skills):
		fmt.Fprintf(w, "\nRun %s to install your Stripe skills.\n", color.Bold("stripe agent setup").String())
	}
}

func skillsScopeStatusLine(w io.Writer, d agentskills.DirStatus) (icon, state string) {
	color := ansi.Color(w)
	switch d.Status {
	case agentskills.StatusError:
		return color.Red("✗").String(), color.Red("error: " + d.Error).String()
	case agentskills.StatusCurrent:
		return color.Green("✔").String(), "installed"
	case agentskills.StatusOutOfDate:
		return color.Yellow("⚠").String(), "outdated"
	default:
		return "", ""
	}
}

// pluginDetail summarizes an installed plugin, e.g. "stripe@cursor-public 0.1.0, user".
func pluginDetail(p agentsetup.PluginStatus) string {
	detail := p.ID
	if p.Version != "" {
		if detail != "" {
			detail += " "
		}
		detail += p.Version
	}
	if p.Scope != "" {
		detail += ", " + p.Scope
	}
	return detail
}

func printNothingDetectedInteractive(w io.Writer) {
	fmt.Fprint(w, `No supported AI coding clients detected on this machine.

Supported clients for automatic setup:
  • Claude Code   https://claude.ai/code
  • Cursor        https://cursor.com
  • Codex CLI     https://openai.com/codex/

You can still install Stripe skills.
`)
}

func printNothingDetected(w io.Writer) {
	fmt.Fprint(w, `No supported AI coding clients detected on this machine.

Supported clients for automatic setup:
  • Claude Code   https://claude.ai/code
  • Cursor        https://cursor.com
  • Codex CLI     https://openai.com/codex/

Once a client is installed, re-run: stripe agent setup
`)
}

func isInteractiveTerminal() bool {
	return term.IsTerminal(int(os.Stdin.Fd())) && term.IsTerminal(int(os.Stdout.Fd()))
}

func sendAgentEvent(ctx context.Context, eventName, eventValue string) {
	if tc := stripe.GetTelemetryClient(ctx); tc != nil {
		go tc.SendEvent(ctx, eventName, eventValue)
	}
}
