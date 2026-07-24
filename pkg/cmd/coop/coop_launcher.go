package coopcmd

import (
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"

	"charm.land/huh/v2"
	"github.com/google/uuid"
	"golang.org/x/term"

	"github.com/stripe/stripe-cli/pkg/coop"
	"github.com/stripe/stripe-cli/pkg/coop/helpers"
	"github.com/stripe/stripe-cli/pkg/coop/tui"
)

type agentInfo struct {
	harness harness
	path    string
}

// shellQuote wraps s in POSIX single quotes so it is safe to interpolate into a
// `bash -c` command line. Unlike strconv.Quote (which produces a Go string
// literal), single quoting neutralizes shell metacharacters including command
// substitution ($(...), backticks), which would otherwise execute when the
// generated launcher script runs. Embedded single quotes are escaped as '\”.
func shellQuote(s string) string {
	return "'" + strings.ReplaceAll(s, "'", `'\''`) + "'"
}

type coopPaneCommandBuilder func(session *coop.Session) (string, func(), error)

var (
	selectString = helpers.Select[string]
	runTmux      = func(args ...string) error {
		return exec.Command("tmux", args...).Run()
	}
)

const (
	defaultCoopTmuxSessionWidth  = 200
	defaultCoopTmuxSessionHeight = 50
	claudeCoopAgents             = `{"coop-cost-effective-worker":{"description":"Use proactively for well-bounded, self-contained exploration, documentation research, test execution, log analysis, and verification tasks when delegation saves main-session context and cost.","prompt":"Work as a focused cost-effective Co-op subagent. Complete only the bounded task you receive, verify your result, and return concise evidence to the main agent. Find and honor applicable repository guidance, including AGENTS.md and CLAUDE.md, before acting. Do not choose or change the Stripe integration or run Co-op lifecycle commands.","model":"haiku"}}`
)

// lookPath is indirected so tests can control which harnesses appear installed.
var lookPath = exec.LookPath

func (rc *coopRunCmd) detectAgent() (*agentInfo, error) {
	if rc.agent != "" {
		path, err := lookPath(rc.agent)
		if err != nil {
			return nil, fmt.Errorf("agent %q not found in PATH", rc.agent)
		}
		h, ok := harnessFor(rc.agent, path)
		if !ok {
			h = customHarness(rc.agent)
		}
		return &agentInfo{harness: h, path: path}, nil
	}

	installed := installedHarnesses()

	switch len(installed) {
	case 0:
		return nil, noAgentFoundError()
	case 1:
		return &installed[0], nil
	}

	options := make([]huh.Option[string], 0, len(installed))
	for _, agent := range installed {
		options = append(options, huh.NewOption(agent.harness.displayName, agent.harness.id))
	}

	var choice string
	if err := selectString("Multiple agents detected. Which would you like to use?", options, &choice); err != nil {
		return nil, err
	}
	for _, agent := range installed {
		if agent.harness.id == choice {
			return &agent, nil
		}
	}
	// The picker only offers ids drawn from installed, so this is unreachable
	// unless the prompt is stubbed out; fall back to the first detected agent
	// rather than returning a nil agent to the caller.
	return &installed[0], nil
}

// installedHarnesses returns the registry entries whose binary is on PATH, in
// registry order.
func installedHarnesses() []agentInfo {
	var found []agentInfo
	for _, h := range supportedHarnesses {
		path, err := lookPath(h.binary)
		if err != nil {
			continue
		}
		found = append(found, agentInfo{harness: h, path: path})
	}
	return found
}

func (rc *coopRunCmd) promptAutoApprove(agent *agentInfo) (bool, error) {
	if !agent.harness.offersAutoApprove() {
		return false, nil
	}

	var choice string
	title := fmt.Sprintf("Permission mode for %s:", agent.harness.displayName)

	err := selectString(title,
		[]huh.Option[string]{
			huh.NewOption("Normal — agent asks before running commands", "normal"),
			huh.NewOption(agent.harness.bypassOption(), "bypass"),
		},
		&choice,
	)
	if err != nil {
		return false, err
	}
	return choice == "bypass", nil
}

func (rc *coopRunCmd) buildAgentCmd(agent *agentInfo, promptPath string, autoApprove bool) (string, error) {
	launcherPath := promptPath + ".sh"

	// The agent path is user-controlled and must be quoted. Everything else is a
	// compile-time constant from the registry, so it is interpolated as-is.
	argv := []string{shellQuote(agent.path)}
	argv = append(argv, agent.harness.args...)
	if autoApprove && agent.harness.autoApproveFlag != "" {
		argv = append(argv, agent.harness.autoApproveFlag)
	}
	if agent.harness.promptFlag != "" {
		argv = append(argv, agent.harness.promptFlag)
	}
	argv = append(argv, `"$prompt"`)

	env := ""
	if autoApprove && agent.harness.autoApproveEnv != "" {
		env = fmt.Sprintf("export %s\n", agent.harness.autoApproveEnv)
	}

	script := fmt.Sprintf("#!/bin/bash\nprompt=$(cat %s)\nrm -f %s %s\n%sexec %s\n",
		shellQuote(promptPath), shellQuote(promptPath), shellQuote(launcherPath), env, strings.Join(argv, " "))

	if err := os.WriteFile(launcherPath, []byte(script), 0700); err != nil {
		return "", fmt.Errorf("creating agent launcher: %w", err)
	}
	return launcherPath, nil
}

func (rc *coopRunCmd) hasTmux() bool {
	_, err := exec.LookPath("tmux")
	return err == nil
}

func (rc *coopRunCmd) agentPaneCommandBuilder(agent *agentInfo, discoveryPrompt string, autoApprove bool) coopPaneCommandBuilder {
	return func(session *coop.Session) (string, func(), error) {
		prompt := discoveryPrompt
		if session != nil {
			var err error
			prompt, err = rc.buildAgentPromptForSession(session)
			if err != nil {
				return "", nil, err
			}
		}
		promptPath, err := writePromptFile(prompt)
		if err != nil {
			return "", nil, err
		}

		agentCmd, err := rc.buildAgentCmd(agent, promptPath, autoApprove)
		if err != nil {
			os.Remove(promptPath)
			return "", nil, err
		}
		// The returned command is run via `bash -c`, so the launcher path itself
		// must be shell-quoted — otherwise a temp dir (TMPDIR) containing a space
		// or shell syntax would be parsed by bash before the launcher runs. The
		// cleanup closure keeps the raw path for os.Remove.
		return shellQuote(agentCmd), func() {
			os.Remove(promptPath)
			os.Remove(agentCmd)
		}, nil
	}
}

func (rc *coopRunCmd) debugAgentPaneCommandBuilder(stripeBin string) coopPaneCommandBuilder {
	return func(session *coop.Session) (string, func(), error) {
		sessionID := ""
		if session != nil {
			sessionID = session.ID
		}
		cmd := fmt.Sprintf("%s coop debug-agent --session %s", shellQuote(stripeBin), shellQuote(sessionID))
		return shellCommandWithCoopEnv(cmd), nil, nil
	}
}

func shellCommandWithCoopEnv(cmd string) string {
	if xdgConfigHome := os.Getenv("XDG_CONFIG_HOME"); xdgConfigHome != "" && !strings.HasPrefix(cmd, "XDG_CONFIG_HOME=") {
		return fmt.Sprintf("XDG_CONFIG_HOME=%s %s", shellQuote(xdgConfigHome), cmd)
	}
	return cmd
}

func (rc *coopRunCmd) runInTmuxSplit(stripeBin string, agent *agentInfo, agentPrompt string, autoApprove bool, blueprint *coop.Blueprint) error {
	return rc.runInTmuxSplitWithCommand(stripeBin, blueprint, rc.agentPaneCommandBuilder(agent, agentPrompt, autoApprove))
}

func (rc *coopRunCmd) runInTmuxSplitWithCommand(stripeBin string, blueprint *coop.Blueprint, buildPaneCmd coopPaneCommandBuilder) error {
	var session *coop.Session
	if blueprint != nil {
		var err error
		session, err = rc.startSessionQuietly(blueprint)
		if err != nil {
			return err
		}
	}

	// Create the store before launching the agent pane, so a store failure
	// doesn't leave an orphaned agent pane and a dangling "active" session with
	// no TUI driving it.
	store, err := coop.NewStore(coopConfigFolder())
	if err != nil {
		rc.abortStartedSession(session, "store creation failed")
		return err
	}

	paneCmd, cleanup, err := buildPaneCmd(session)
	if err != nil {
		rc.abortStartedSession(session, "agent pane command failed")
		return err
	}
	paneCmd = shellCommandWithCoopEnv(paneCmd)

	if _, err := splitCoopAgentPane("-h", "-p", "60", "bash", "-c", paneCmd); err != nil {
		if cleanup != nil {
			cleanup()
		}
		rc.abortStartedSession(session, "tmux split failed")
		return fmt.Errorf("tmux split failed: %w", err)
	}

	if blueprint != nil {
		return tui.Run(store, session.ID, coopTUIOptions()...)
	}

	return runCoopTUIWait(store)
}

func (rc *coopRunCmd) runInNewTmux(stripeBin string, agent *agentInfo, agentPrompt string, autoApprove bool, blueprint *coop.Blueprint) error {
	return rc.runInNewTmuxWithCommand(stripeBin, blueprint, rc.agentPaneCommandBuilder(agent, agentPrompt, autoApprove))
}

func (rc *coopRunCmd) runInNewTmuxWithCommand(stripeBin string, blueprint *coop.Blueprint, buildPaneCmd coopPaneCommandBuilder) error {
	sessionName := "stripe-coop"

	// Check for existing session
	if err := runTmux("has-session", "-t", sessionName); err == nil {
		var choice string
		if err := selectString("A co-op tmux session already exists. What would you like to do?",
			[]huh.Option[string]{
				huh.NewOption("Reattach to existing session", "attach"),
				huh.NewOption("Start fresh (kills existing session)", "fresh"),
			},
			&choice,
		); err != nil {
			return err
		}

		if choice == "attach" {
			attach := exec.Command("tmux", "attach-session", "-t", sessionName)
			attach.Stdin = os.Stdin
			attach.Stdout = os.Stdout
			attach.Stderr = os.Stderr
			return attach.Run()
		}
		killTmuxSession(sessionName)
	}

	var session *coop.Session
	if blueprint != nil {
		var err error
		session, err = rc.startSessionQuietly(blueprint)
		if err != nil {
			return err
		}
	}

	tuiCmd := fmt.Sprintf("%s coop join", shellQuote(stripeBin))
	if blueprint == nil {
		tuiCmd += " --wait"
	} else {
		tuiCmd += " " + session.ID
	}
	tuiCmd = shellCommandWithCoopEnv(tuiCmd)

	paneCmd, cleanup, err := buildPaneCmd(session)
	if err != nil {
		rc.abortStartedSession(session, "agent pane command failed")
		return err
	}
	paneCmd = shellCommandWithCoopEnv(paneCmd)

	width, height := coopTmuxSessionDimensions()
	if err := runTmux("new-session", "-d", "-s", sessionName, "-x", strconv.Itoa(width), "-y", strconv.Itoa(height),
		"bash", "-c", tuiCmd); err != nil {
		if cleanup != nil {
			cleanup()
		}
		rc.abortStartedSession(session, "tmux new-session failed")
		return fmt.Errorf("tmux new-session failed: %w", err)
	}

	agentPane, err := splitCoopAgentPane("-h", "-t", sessionName, "-p", "60", "bash", "-c", paneCmd)
	if err != nil {
		if cleanup != nil {
			cleanup()
		}
		killTmuxSession(sessionName)
		rc.abortStartedSession(session, "tmux split-window failed")
		return fmt.Errorf("tmux split-window failed: %w", err)
	}

	runTmux("select-pane", "-t", agentPane)

	attach := exec.Command("tmux", "attach-session", "-t", sessionName)
	attach.Stdin = os.Stdin
	attach.Stdout = os.Stdout
	attach.Stderr = os.Stderr
	return attach.Run()
}

func killTmuxSession(sessionName string) {
	_ = runTmux("kill-session", "-t", sessionName)
}

func coopTmuxSessionDimensions() (int, int) {
	width, height, err := term.GetSize(int(os.Stdout.Fd()))
	return normalizeCoopTmuxSessionDimensions(width, height, err)
}

func normalizeCoopTmuxSessionDimensions(width, height int, err error) (int, int) {
	if err != nil || width <= 0 || height <= 0 {
		return defaultCoopTmuxSessionWidth, defaultCoopTmuxSessionHeight
	}
	return width, height
}

func (rc *coopRunCmd) runFallback(stripeBin string, agent *agentInfo, agentPrompt string, autoApprove bool, blueprint *coop.Blueprint) error {
	return rc.runFallbackWithCommand(stripeBin, blueprint, rc.agentPaneCommandBuilder(agent, agentPrompt, autoApprove))
}

func (rc *coopRunCmd) runFallbackWithCommand(stripeBin string, blueprint *coop.Blueprint, buildPaneCmd coopPaneCommandBuilder) error {
	fmt.Println("tmux not found — running agent in this terminal.")

	var session *coop.Session
	if blueprint != nil {
		var err error
		session, err = rc.startSessionQuietly(blueprint)
		if err != nil {
			return err
		}
		fmt.Printf("Session started: %s\n", session.ID)
		fmt.Printf("Open another terminal and run: %s\n", shellCommandWithCoopEnv("stripe coop join "+session.ID))
	} else {
		fmt.Printf("Open another terminal and run: %s\n", shellCommandWithCoopEnv("stripe coop join --wait"))
	}
	fmt.Println()

	paneCmd, cleanup, err := buildPaneCmd(session)
	if err != nil {
		rc.abortStartedSession(session, "agent pane command failed")
		return err
	}
	if cleanup != nil {
		defer cleanup()
	}
	agentExec := exec.Command("bash", "-c", paneCmd)
	agentExec.Stdin = os.Stdin
	agentExec.Stdout = os.Stdout
	agentExec.Stderr = os.Stderr
	return agentExec.Run()
}

func generateShortID() string {
	return uuid.New().String()[:8]
}

func writePromptFile(prompt string) (string, error) {
	f, err := os.CreateTemp("", "stripe-coop-prompt-*.txt")
	if err != nil {
		return "", fmt.Errorf("creating prompt file: %w", err)
	}
	if _, err := f.WriteString(prompt); err != nil {
		f.Close()
		return "", fmt.Errorf("writing prompt file: %w", err)
	}
	if err := f.Close(); err != nil {
		return "", fmt.Errorf("closing prompt file: %w", err)
	}
	return f.Name(), nil
}

func runCoopTUIWait(store *coop.Store) error {
	existingIDs := make(map[string]bool)
	if ids, err := store.List(); err == nil {
		for _, id := range ids {
			existingIDs[id] = true
		}
	}
	return tui.RunWaiting(store, existingIDs, coopTUIOptions()...)
}
