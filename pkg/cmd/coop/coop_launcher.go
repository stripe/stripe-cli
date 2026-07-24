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
	name string // "claude" or "codex"
	path string
}

// shellQuote wraps s in POSIX single quotes so it is safe to interpolate into a
// `bash -c` command line. Unlike strconv.Quote (which produces a Go string
// literal), single quoting neutralizes shell metacharacters including command
// substitution ($(...), backticks), which would otherwise execute when the
// generated launcher script runs. Embedded single quotes are escaped as '\”.
func shellQuote(s string) string {
	return "'" + strings.ReplaceAll(s, "'", `'\''`) + "'"
}

// paneHint is one "here's what you can do next" line shown in the agent pane
// after the agent process exits.
type paneHint struct {
	label   string
	command string
}

// coopPaneCommand describes what runs in the agent pane, plus the hints shown
// once that command exits.
type coopPaneCommand struct {
	cmd   string
	hints []paneHint
}

type coopPaneCommandBuilder func(session *coop.Session) (coopPaneCommand, func(), error)

var (
	selectString = helpers.Select[string]
	runTmux      = func(args ...string) error {
		return exec.Command("tmux", args...).Run()
	}
)

const (
	defaultCoopTmuxSessionWidth  = 200
	defaultCoopTmuxSessionHeight = 50
)

func (rc *coopRunCmd) detectAgent() (*agentInfo, error) {
	if rc.agent != "" {
		path, err := exec.LookPath(rc.agent)
		if err != nil {
			return nil, fmt.Errorf("agent %q not found in PATH", rc.agent)
		}
		name := rc.agent
		if strings.Contains(path, "claude") {
			name = "claude"
		} else if strings.Contains(path, "codex") {
			name = "codex"
		}
		return &agentInfo{name: name, path: path}, nil
	}

	claudePath, claudeErr := exec.LookPath("claude")
	codexPath, codexErr := exec.LookPath("codex")

	hasClaude := claudeErr == nil
	hasCodex := codexErr == nil

	switch {
	case hasClaude && hasCodex:
		var choice string
		err := selectString("Multiple agents detected. Which would you like to use?",
			[]huh.Option[string]{
				huh.NewOption("Claude Code", "claude"),
				huh.NewOption("Codex", "codex"),
			},
			&choice,
		)
		if err != nil {
			return nil, err
		}
		if choice == "codex" {
			return &agentInfo{name: "codex", path: codexPath}, nil
		}
		return &agentInfo{name: "claude", path: claudePath}, nil

	case hasClaude:
		return &agentInfo{name: "claude", path: claudePath}, nil
	case hasCodex:
		return &agentInfo{name: "codex", path: codexPath}, nil
	default:
		return nil, fmt.Errorf("no AI agent found in PATH.\n  Install Claude Code: https://docs.anthropic.com/en/docs/claude-code\n  Or specify a custom agent: --agent=<command>")
	}
}

func (rc *coopRunCmd) promptAutoApprove(agent *agentInfo) (bool, error) {
	var choice string

	var title string
	switch agent.name {
	case "claude":
		title = "Permission mode for Claude Code:"
	case "codex":
		title = "Permission mode for Codex:"
	default:
		return false, nil
	}

	err := selectString(title,
		[]huh.Option[string]{
			huh.NewOption("Normal — agent asks before running commands", "normal"),
			huh.NewOption("Auto-approve — skip all permission prompts (faster, less safe)", "auto"),
		},
		&choice,
	)
	if err != nil {
		return false, err
	}
	return choice == "auto", nil
}

func (rc *coopRunCmd) buildAgentCmd(agent *agentInfo, promptPath string, autoApprove bool) (string, error) {
	launcherPath := promptPath + ".sh"
	var script string

	switch agent.name {
	case "claude":
		flags := ""
		if autoApprove {
			flags = " --dangerously-skip-permissions"
		}
		script = fmt.Sprintf("#!/bin/bash\nprompt=$(cat %s)\nrm -f %s %s\nexec %s%s \"$prompt\"\n",
			shellQuote(promptPath), shellQuote(promptPath), shellQuote(launcherPath), shellQuote(agent.path), flags)

	case "codex":
		flags := ""
		if autoApprove {
			flags = " --dangerously-bypass-approvals-and-sandbox"
		}
		script = fmt.Sprintf("#!/bin/bash\nprompt=$(cat %s)\nrm -f %s %s\nexec %s%s \"$prompt\"\n",
			shellQuote(promptPath), shellQuote(promptPath), shellQuote(launcherPath), shellQuote(agent.path), flags)

	default:
		script = fmt.Sprintf("#!/bin/bash\nprompt=$(cat %s)\nrm -f %s %s\nexec %s \"$prompt\"\n",
			shellQuote(promptPath), shellQuote(promptPath), shellQuote(launcherPath), shellQuote(agent.path))
	}

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
	return func(session *coop.Session) (coopPaneCommand, func(), error) {
		prompt := discoveryPrompt
		if session != nil {
			var err error
			prompt, err = rc.buildAgentPromptForSession(session)
			if err != nil {
				return coopPaneCommand{}, nil, err
			}
		}
		promptPath, err := writePromptFile(prompt)
		if err != nil {
			return coopPaneCommand{}, nil, err
		}

		agentCmd, err := rc.buildAgentCmd(agent, promptPath, autoApprove)
		if err != nil {
			os.Remove(promptPath)
			return coopPaneCommand{}, nil, err
		}
		// The returned command is run via `bash -c`, so the launcher path itself
		// must be shell-quoted — otherwise a temp dir (TMPDIR) containing a space
		// or shell syntax would be parsed by bash before the launcher runs. The
		// cleanup closure keeps the raw path for os.Remove.
		pane := coopPaneCommand{
			cmd:   shellQuote(agentCmd),
			hints: agentRestartHints(agent, autoApprove),
		}
		return pane, func() {
			os.Remove(promptPath)
			os.Remove(agentCmd)
		}, nil
	}
}

// agentRestartHints lists the commands that bring the agent back after it
// exits. The generated launcher deletes itself on start, so restarting means
// invoking the agent directly — with the same permission mode the developer
// picked when the session was launched.
func agentRestartHints(agent *agentInfo, autoApprove bool) []paneHint {
	invocation := agent.name
	if autoApprove {
		switch agent.name {
		case "claude":
			invocation += " --dangerously-skip-permissions"
		case "codex":
			invocation += " --dangerously-bypass-approvals-and-sandbox"
		}
	}

	if agent.name == "claude" {
		return []paneHint{
			{label: "Resume where the agent left off", command: invocation + " --continue"},
			{label: "Start the agent fresh", command: invocation},
		}
	}
	return []paneHint{{label: "Start the agent again", command: invocation}}
}

func (rc *coopRunCmd) debugAgentPaneCommandBuilder(stripeBin string) coopPaneCommandBuilder {
	return func(session *coop.Session) (coopPaneCommand, func(), error) {
		sessionID := ""
		if session != nil {
			sessionID = session.ID
		}
		cmd := fmt.Sprintf("%s coop debug-agent --session %s", shellQuote(stripeBin), shellQuote(sessionID))
		pane := coopPaneCommand{
			cmd:   shellCommandWithCoopEnv(cmd),
			hints: []paneHint{{label: "Run the debug agent again", command: cmd}},
		}
		return pane, nil, nil
	}
}

// holdOpenPaneScript wraps the agent pane command so the tmux pane outlives it.
// tmux destroys a pane the moment its command exits, so quitting the agent used
// to make the right half of the split vanish — which reads as a crash rather
// than "your agent stopped". Instead the pane explains what happened and hands
// itself to an interactive shell the developer can restart the agent from.
//
// The co-op env is exported (rather than prefixed onto the command) so that the
// shell left behind, and anything started from it, still points at the same
// co-op session state.
func holdOpenPaneScript(pane coopPaneCommand) string {
	var script strings.Builder
	if xdgConfigHome := os.Getenv("XDG_CONFIG_HOME"); xdgConfigHome != "" {
		fmt.Fprintf(&script, "export XDG_CONFIG_HOME=%s\n", shellQuote(xdgConfigHome))
	}
	// ctrl-c reaches the whole foreground process group, so without this the
	// wrapper dies alongside the agent and tmux closes the pane anyway. A no-op
	// handler (rather than ignoring the signal) is reset to the default in the
	// agent process, so the agent's own ctrl-c handling is untouched. SIGHUP is
	// deliberately left alone so `tmux kill-pane` still works.
	script.WriteString("trap ':' INT TERM\n")
	// The agent command runs in a subshell so that an `exit` inside it ends the
	// subshell rather than this script — the notice below must always run.
	script.WriteString("(\n" + pane.cmd + "\n)\n")
	script.WriteString(agentExitNoticeScript(pane.hints))
	// A login shell would re-run profile scripts that may cd elsewhere; a plain
	// interactive shell keeps the pane in the project directory.
	script.WriteString("exec \"${SHELL:-/bin/bash}\"\n")
	return script.String()
}

// agentExitNoticeScript renders the message shown in the agent pane once the
// agent exits. Every line goes through printf so the notice survives whatever
// the agent left on screen.
func agentExitNoticeScript(hints []paneHint) string {
	const rule = `printf '\033[90m──────────────────────────────────────────────────────\033[0m\n'`

	lines := []string{
		`__coop_agent_status=$?`,
		`printf '\n'`,
		rule,
		`printf '\033[1mThe agent exited\033[0m (status %s).\n' "$__coop_agent_status"`,
		`printf 'This pane stayed open on purpose — your co-op session is still\n'`,
		`printf 'running in the left pane.\n'`,
		`printf '\n'`,
	}
	for _, hint := range hints {
		lines = append(lines, fmt.Sprintf(`printf '  %%s\n    \033[1m%%s\033[0m\n' %s %s`,
			shellQuote(hint.label), shellQuote(hint.command)))
	}
	lines = append(lines,
		`printf '  Jump back to the co-op TUI\n    \033[1mctrl-b then left arrow\033[0m\n'`,
		`printf '  Close this pane\n    \033[1mexit\033[0m\n'`,
		rule,
		`printf '\n'`,
	)
	return strings.Join(lines, "\n") + "\n"
}

func shellCommandWithCoopEnv(cmd string) string {
	if xdgConfigHome := os.Getenv("XDG_CONFIG_HOME"); xdgConfigHome != "" && !strings.HasPrefix(cmd, "XDG_CONFIG_HOME=") {
		return fmt.Sprintf("XDG_CONFIG_HOME=%s %s", shellQuote(xdgConfigHome), cmd)
	}
	return cmd
}

func (rc *coopRunCmd) runInTmuxSplit(stripeBin string, agent *agentInfo, agentPrompt string, autoApprove bool, blueprintID string) error {
	return rc.runInTmuxSplitWithCommand(stripeBin, blueprintID, rc.agentPaneCommandBuilder(agent, agentPrompt, autoApprove))
}

func (rc *coopRunCmd) runInTmuxSplitWithCommand(stripeBin string, blueprintID string, buildPaneCmd coopPaneCommandBuilder) error {
	var session *coop.Session
	if blueprintID != "" {
		var err error
		session, err = rc.startSessionQuietly(blueprintID)
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

	pane, cleanup, err := buildPaneCmd(session)
	if err != nil {
		rc.abortStartedSession(session, "agent pane command failed")
		return err
	}
	paneCmd := holdOpenPaneScript(pane)

	if err := runTmux("split-window", "-h", "-p", "60", "bash", "-c", paneCmd); err != nil {
		if cleanup != nil {
			cleanup()
		}
		rc.abortStartedSession(session, "tmux split failed")
		return fmt.Errorf("tmux split failed: %w", err)
	}

	if blueprintID != "" {
		return tui.Run(store, session.ID, tui.WithSandboxClaimURL(coopSandboxClaimURL()))
	}

	return runCoopTUIWait(store)
}

func (rc *coopRunCmd) runInNewTmux(stripeBin string, agent *agentInfo, agentPrompt string, autoApprove bool, blueprintID string) error {
	return rc.runInNewTmuxWithCommand(stripeBin, blueprintID, rc.agentPaneCommandBuilder(agent, agentPrompt, autoApprove))
}

func (rc *coopRunCmd) runInNewTmuxWithCommand(stripeBin string, blueprintID string, buildPaneCmd coopPaneCommandBuilder) error {
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
	if blueprintID != "" {
		var err error
		session, err = rc.startSessionQuietly(blueprintID)
		if err != nil {
			return err
		}
	}

	tuiCmd := fmt.Sprintf("%s coop join", shellQuote(stripeBin))
	if blueprintID == "" {
		tuiCmd += " --wait"
	} else {
		tuiCmd += " " + session.ID
	}
	tuiCmd = shellCommandWithCoopEnv(tuiCmd)

	pane, cleanup, err := buildPaneCmd(session)
	if err != nil {
		rc.abortStartedSession(session, "agent pane command failed")
		return err
	}
	paneCmd := holdOpenPaneScript(pane)

	width, height := coopTmuxSessionDimensions()
	if err := runTmux("new-session", "-d", "-s", sessionName, "-x", strconv.Itoa(width), "-y", strconv.Itoa(height),
		"bash", "-c", tuiCmd); err != nil {
		if cleanup != nil {
			cleanup()
		}
		rc.abortStartedSession(session, "tmux new-session failed")
		return fmt.Errorf("tmux new-session failed: %w", err)
	}

	if err := runTmux("split-window", "-h", "-t", sessionName, "-p", "60",
		"bash", "-c", paneCmd); err != nil {
		if cleanup != nil {
			cleanup()
		}
		killTmuxSession(sessionName)
		rc.abortStartedSession(session, "tmux split-window failed")
		return fmt.Errorf("tmux split-window failed: %w", err)
	}

	runTmux("select-pane", "-t", sessionName+":0.1")

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

func (rc *coopRunCmd) runFallback(stripeBin string, agent *agentInfo, agentPrompt string, autoApprove bool, blueprintID string) error {
	return rc.runFallbackWithCommand(stripeBin, blueprintID, rc.agentPaneCommandBuilder(agent, agentPrompt, autoApprove))
}

func (rc *coopRunCmd) runFallbackWithCommand(stripeBin string, blueprintID string, buildPaneCmd coopPaneCommandBuilder) error {
	fmt.Println("tmux not found — running agent in this terminal.")

	var session *coop.Session
	if blueprintID != "" {
		var err error
		session, err = rc.startSessionQuietly(blueprintID)
		if err != nil {
			return err
		}
		fmt.Printf("Session started: %s\n", session.ID)
		fmt.Printf("Open another terminal and run: %s\n", shellCommandWithCoopEnv("stripe coop join "+session.ID))
	} else {
		fmt.Printf("Open another terminal and run: %s\n", shellCommandWithCoopEnv("stripe coop join --wait"))
	}
	fmt.Println()

	pane, cleanup, err := buildPaneCmd(session)
	if err != nil {
		rc.abortStartedSession(session, "agent pane command failed")
		return err
	}
	if cleanup != nil {
		defer cleanup()
	}
	// No hold-open wrapper here: this runs in the developer's own terminal, so
	// exiting the agent returns them to their existing shell.
	agentExec := exec.Command("bash", "-c", pane.cmd)
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
	return tui.RunWaiting(store, existingIDs, tui.WithSandboxClaimURL(coopSandboxClaimURL()))
}
