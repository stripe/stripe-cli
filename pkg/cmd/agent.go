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
	"github.com/stripe/stripe-cli/pkg/validators"
)

// providerOrder is the canonical display order for known clients. Providers not
// listed here are appended afterward in alphabetical order.
var providerOrder = []string{
	agentsetup.ClientClaudeCode,
	agentsetup.ClientCursor,
	agentsetup.ClientCodex,
}

type agentCmd struct {
	cmd *cobra.Command
}

type agentSetupCmd struct {
	cmd *cobra.Command

	statusOnly  bool
	yes         bool
	force       bool
	jsonOutput  bool
	client      string
	skills      bool
	skillsScope string

	providers map[string]agentsetup.Provider

	// Skills installation is injectable so tests can avoid the network and
	// point installs at temp directories.
	skillsInstall   func(ctx context.Context, destDir string) ([]string, error)
	skillsLocalDir  func() (string, error)
	skillsGlobalDir func() (string, error)
}

type agentSetupJSON struct {
	Status  string              `json:"status"`
	Clients []agentsetup.Status `json:"clients"`
	Actions []agentsetup.Plan   `json:"actions,omitempty"`
	Errors  []string            `json:"errors,omitempty"`
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
		skillsLocalDir:  func() (string, error) { return skillsDirUnder(os.Getwd) },
		skillsGlobalDir: func() (string, error) { return skillsDirUnder(os.UserHomeDir) },
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
	asc.cmd.Flags().BoolVar(&asc.force, "force", false, "Reinstall even when agent tooling is already installed")
	asc.cmd.Flags().StringVar(&asc.client, "client", "", "Limit setup to a single client (default: all detected clients)")
	asc.cmd.Flags().BoolVar(&asc.jsonOutput, "json", false, "Write machine-readable status output")
	asc.cmd.Flags().BoolVar(&asc.skills, "skills", false, "Install Stripe skills directly, without the interactive prompt")
	asc.cmd.Flags().StringVar(&asc.skillsScope, "skills-scope", skillsScopeLocal, "Where to install skills: local (current directory) or global (home directory)")

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

	if asc.skillsScope != skillsScopeLocal && asc.skillsScope != skillsScopeGlobal {
		return fmt.Errorf("invalid --skills-scope %q; use %q or %q", asc.skillsScope, skillsScopeLocal, skillsScopeGlobal)
	}

	ctx := commandContextOrBackground(cmd)
	out := cmd.OutOrStdout()

	statuses := detectAll(providers)

	if asc.jsonOutput {
		return asc.writeJSON(out, providers, statuses)
	}

	detected := detectedStatuses(statuses)

	if asc.statusOnly {
		if len(detected) == 0 {
			printNothingDetected(out)
			return nil
		}
		printStatusTable(out, statuses)
		return nil
	}

	sel, scope, err := asc.resolveSelection(cmd, out, detected)
	if err != nil {
		return err
	}
	if sel == nil {
		return nil // nothing to do; a message was already printed
	}

	return asc.install(ctx, out, providers, *sel, scope)
}

// resolveSelection decides which agents and/or skills to install. It uses the
// interactive TUI when attached to a terminal (and neither --yes nor --skills
// forced a non-interactive run), otherwise it derives the selection from flags.
// A nil Selection means there is nothing to do and a message has been printed.
func (asc *agentSetupCmd) resolveSelection(cmd *cobra.Command, out io.Writer, detected []agentsetup.Status) (*Selection, string, error) {
	scope := asc.skillsScope

	useTUI := isInteractiveTerminal() && !asc.yes && !asc.skills
	if !useTUI {
		// --skills on its own means "install skills only" (the alternative to
		// configuring agents). Agents are installed when --yes is given, or via
		// the interactive picker.
		agents := detected
		if asc.skills && !asc.yes {
			agents = nil
		}
		if len(agents) == 0 && !asc.skills {
			printNothingDetected(out)
			return nil, scope, nil
		}
		return &Selection{Agents: agents, InstallSkills: asc.skills}, scope, nil
	}

	sel, err := RunSelectionTUI(detected)
	if err != nil {
		return nil, scope, err
	}
	if sel == nil {
		fmt.Fprintln(out, "Canceled. No changes made.")
		return nil, scope, nil
	}
	if len(sel.Agents) == 0 && !sel.InstallSkills {
		fmt.Fprintln(out, "Nothing selected. No changes made.")
		return nil, scope, nil
	}

	// Mirror `npx skills add` by asking where to install skills, unless the
	// scope was already pinned with --skills-scope.
	if sel.InstallSkills && !cmd.Flags().Changed("skills-scope") {
		chosen, ok, err := RunSkillsScopeTUI()
		if err != nil {
			return nil, scope, err
		}
		if !ok {
			fmt.Fprintln(out, "Canceled. No changes made.")
			return nil, scope, nil
		}
		scope = chosen
	}

	return sel, scope, nil
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

	var successCount, skipCount, errCount int
	for _, status := range sel.Agents {
		provider := providers[status.Client]
		plan := provider.Plan(status, asc.force)

		fmt.Fprintf(out, "\n  %s\n", status.DisplayName)
		switch plan.Action {
		case agentsetup.ActionNone:
			fmt.Fprintln(out, "  already set up")
			skipCount++
			continue
		case agentsetup.ActionManual:
			// Setup can't be automated (e.g. Cursor). Surface the instruction and
			// treat it as skipped, not a failure.
			fmt.Fprintf(out, "  %s manual step: %s\n", warn, plan.Manual)
			skipCount++
			continue
		}

		err := spinner.New().
			WithLabel("Installing...").
			WithOutput(os.Stderr).
			Run(func() error { return provider.Apply(ctx, out, plan) })
		if err != nil {
			fmt.Fprintf(out, "  %s error: %v\n", cross, err)
			errCount++
			continue
		}
		fmt.Fprintf(out, "  %s done\n", check)
		successCount++
	}

	if sel.InstallSkills {
		if asc.installSkills(ctx, out, scope, check, cross) {
			successCount++
		} else {
			errCount++
		}
	}

	fmt.Fprintf(out, "\n%d installed, %d skipped, %d errors\n", successCount, skipCount, errCount)
	if errCount > 0 {
		return fmt.Errorf("%d item(s) failed to set up", errCount)
	}
	return nil
}

// installSkills fetches and writes Stripe skills to the local or global
// .agents/skills directory. It returns true on success.
func (asc *agentSetupCmd) installSkills(ctx context.Context, out io.Writer, scope, check, cross string) bool {
	fmt.Fprintf(out, "\n  Stripe skills (%s)\n", scope)

	dirFn := asc.skillsLocalDir
	if scope == skillsScopeGlobal {
		dirFn = asc.skillsGlobalDir
	}
	dir, err := dirFn()
	if err != nil {
		fmt.Fprintf(out, "  %s error: resolving skills directory: %v\n", cross, err)
		return false
	}

	var installed []string
	runErr := spinner.New().
		WithLabel("Installing skills...").
		WithOutput(os.Stderr).
		Run(func() error {
			var e error
			installed, e = asc.skillsInstall(ctx, dir)
			return e
		})
	if runErr != nil {
		fmt.Fprintf(out, "  %s error: %v\n", cross, runErr)
		return false
	}

	fmt.Fprintf(out, "  %s installed %d skill(s) to %s: %s\n", check, len(installed), dir, strings.Join(installed, ", "))
	return true
}

func (asc *agentSetupCmd) writeJSON(w io.Writer, providers map[string]agentsetup.Provider, statuses []agentsetup.Status) error {
	result := agentSetupJSON{
		Status:  aggregateStatus(statuses),
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

	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	if err := enc.Encode(result); err != nil {
		return err
	}
	if len(result.Errors) > 0 {
		return fmt.Errorf("%s", result.Errors[0])
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

func detectedStatuses(statuses []agentsetup.Status) []agentsetup.Status {
	var detected []agentsetup.Status
	for _, s := range statuses {
		if s.Detected {
			detected = append(detected, s)
		}
	}
	return detected
}

// aggregateStatus collapses per-client statuses into a single headline status,
// surfacing the most actionable state first.
func aggregateStatus(statuses []agentsetup.Status) string {
	var missing, installed bool
	for _, s := range statuses {
		switch s.Status {
		case agentsetup.StatusError:
			return agentsetup.StatusError
		case agentsetup.StatusMissing:
			missing = true
		case agentsetup.StatusInstalled:
			installed = true
		}
	}
	switch {
	case missing:
		return agentsetup.StatusMissing
	case installed:
		return agentsetup.StatusInstalled
	default:
		return agentsetup.StatusNotDetected
	}
}

// printStatusTable renders a compact, aligned, color-coded view of each client
// and its Stripe plugin state. Colors are disabled automatically when the writer
// is not a TTY (via ansi.Color).
func printStatusTable(w io.Writer, statuses []agentsetup.Status) {
	color := ansi.Color(w)

	fmt.Fprintln(w, color.Bold("AI coding clients").String())
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
			state = "Stripe plugin installed"
			if detail := pluginDetail(s.Plugin); detail != "" {
				state += "  " + color.Faint(detail).String()
			}
		default:
			icon = color.Yellow("•").String()
			state = "Stripe plugin not installed"
			needsSetup = true
		}
		fmt.Fprintf(w, "  %s  %-*s  %s\n", icon, nameWidth, s.DisplayName, state)
	}

	if needsSetup {
		fmt.Fprintf(w, "\nRun %s to install the Stripe plugin where it's missing.\n", color.Bold("stripe agent setup").String())
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

func printNothingDetected(w io.Writer) {
	fmt.Fprint(w, `No AI coding clients detected on this machine.

Supported clients for automatic setup:
  • Claude Code   https://claude.ai/code
  • Cursor        https://cursor.com
  • Codex CLI     https://github.com/openai/codex

Once a client is installed, re-run: stripe agent setup

Or install Stripe skills directly (no agent required):
  stripe agent setup --skills               # into ./.agents/skills
  stripe agent setup --skills --skills-scope global   # into ~/.agents/skills
`)
}

func isInteractiveTerminal() bool {
	return term.IsTerminal(int(os.Stdin.Fd())) && term.IsTerminal(int(os.Stdout.Fd()))
}
