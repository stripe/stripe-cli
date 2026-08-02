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

// paneCommand describes the agent launch in each of the two launch modes:
// shell is interpolated into a tmux pane's command line; argv runs directly in
// this terminal with no shell, so the fallback works where bash doesn't exist
// (Windows, minimal containers, NixOS).
type paneCommand struct {
	shell string
	argv  []string
}

type coopPaneCommandBuilder func(session *coop.Session) (paneCommand, func(), error)

var (
	selectString = helpers.Select[string]
	runTmux      = func(args ...string) error {
		return exec.Command("tmux", args...).Run()
	}
)

const (
	defaultCoopTmuxSessionWidth  = 200
	defaultCoopTmuxSessionHeight = 50

	// The TUI pane gets a fixed column width and the agent pane gets everything
	// else. Fixed (rather than a percentage) because the TUI's usefulness stops
	// growing past its card width while the agent benefits from every extra
	// column — and it makes the review card render identically on every
	// terminal. Both widths sit below tui.SplitWorkspaceMinWidth so the
	// companion pane always renders the single-column layout.
	coopTUIPaneWidth       = 48
	coopTUIPaneWidthNarrow = 40

	// Minimum agent-pane widths paired with each TUI tier. Below the narrow
	// tier a side-by-side split squeezes both panes past usefulness, so the
	// launcher runs the agent full-width with manual join instructions instead.
	coopAgentPaneMinWidth       = 80
	coopAgentPaneMinWidthNarrow = 72

	// tmux draws a one-column divider between horizontally split panes.
	coopPaneDividerWidth = 1
	claudeCoopAgents     = `{"coop-cost-effective-worker":{"description":"Use proactively for well-bounded, self-contained exploration, documentation research, test execution, log analysis, and verification tasks when delegation saves main-session context and cost.","prompt":"Work as a focused cost-effective Co-op subagent. Complete only the bounded task you receive, verify your result, and return concise evidence to the main agent. Find and honor applicable repository guidance, including AGENTS.md and CLAUDE.md, before acting. Do not choose or change the Stripe integration or run Co-op lifecycle commands.","model":"haiku"}}`
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
	var bypassLabel string
	switch agent.name {
	case "claude":
		title = "Permission mode for Claude Code:"
		bypassLabel = "Bypass permissions — skip safety checks (isolated environments only)"
	case "codex":
		title = "Permission mode for Codex:"
		bypassLabel = "Bypass approvals and sandbox — skip safety checks (isolated environments only)"
	default:
		return false, nil
	}

	err := selectString(title,
		[]huh.Option[string]{
			huh.NewOption("Normal — agent asks before running commands", "normal"),
			huh.NewOption(bypassLabel, "bypass"),
		},
		&choice,
	)
	if err != nil {
		return false, err
	}
	return choice == "bypass", nil
}

// agentFlags returns the per-agent CLI flags as an argv fragment. The tmux
// launcher script and the direct-exec fallback both build from this so the two
// launch modes cannot drift.
func agentFlags(agent *agentInfo, autoApprove bool) []string {
	switch agent.name {
	case "claude":
		flags := []string{"--agents", claudeCoopAgents}
		if autoApprove {
			flags = append(flags, "--dangerously-skip-permissions")
		}
		return flags
	case "codex":
		if autoApprove {
			return []string{"--dangerously-bypass-approvals-and-sandbox"}
		}
		return nil
	default:
		return nil
	}
}

func (rc *coopRunCmd) buildAgentCmd(agent *agentInfo, promptPath string, autoApprove bool) (string, error) {
	launcherPath := promptPath + ".sh"

	flags := ""
	for _, flag := range agentFlags(agent, autoApprove) {
		flags += " " + shellQuote(flag)
	}
	// `#!/usr/bin/env bash`, not `#!/bin/bash`: the script runs inside tmux
	// panes whose shell resolves bash from PATH, and /bin/bash does not exist
	// on NixOS.
	script := fmt.Sprintf("#!/usr/bin/env bash\nprompt=$(cat %s)\nrm -f %s %s\nexec %s%s \"$prompt\"\n",
		shellQuote(promptPath), shellQuote(promptPath), shellQuote(launcherPath), shellQuote(agent.path), flags)

	if err := os.WriteFile(launcherPath, []byte(script), 0700); err != nil {
		return "", fmt.Errorf("creating agent launcher: %w", err)
	}
	return launcherPath, nil
}

func (rc *coopRunCmd) hasTmux() bool {
	_, err := exec.LookPath("tmux")
	return err == nil
}

// coopTUIPaneWidthFor returns the fixed TUI pane width for a terminal (or
// tmux pane) of totalWidth columns, or 0 when totalWidth is too narrow for a
// side-by-side split to leave both panes usable.
func coopTUIPaneWidthFor(totalWidth int) int {
	switch {
	case totalWidth >= coopTUIPaneWidth+coopPaneDividerWidth+coopAgentPaneMinWidth:
		return coopTUIPaneWidth
	case totalWidth >= coopTUIPaneWidthNarrow+coopPaneDividerWidth+coopAgentPaneMinWidthNarrow:
		return coopTUIPaneWidthNarrow
	default:
		return 0
	}
}

// currentTmuxPaneWidth reports the width of the tmux pane this process is
// running in. On any failure it assumes the default session width, which
// yields the standard TUI width; tmux clamps an oversized -l request rather
// than failing the split.
func currentTmuxPaneWidth() int {
	out, err := runTmuxOutput("display-message", "-p", "#{pane_width}")
	if err != nil {
		return defaultCoopTmuxSessionWidth
	}
	width, err := strconv.Atoi(strings.TrimSpace(out))
	if err != nil || width <= 0 {
		return defaultCoopTmuxSessionWidth
	}
	return width
}

func (rc *coopRunCmd) agentPaneCommandBuilder(agent *agentInfo, discoveryPrompt string, autoApprove bool) coopPaneCommandBuilder {
	return func(session *coop.Session) (paneCommand, func(), error) {
		prompt := discoveryPrompt
		if session != nil {
			var err error
			prompt, err = rc.buildAgentPromptForSession(session)
			if err != nil {
				return paneCommand{}, nil, err
			}
		}
		promptPath, err := writePromptFile(prompt)
		if err != nil {
			return paneCommand{}, nil, err
		}

		agentCmd, err := rc.buildAgentCmd(agent, promptPath, autoApprove)
		if err != nil {
			os.Remove(promptPath)
			return paneCommand{}, nil, err
		}
		// The shell command is run via `bash -c`, so the launcher path itself
		// must be shell-quoted — otherwise a temp dir (TMPDIR) containing a
		// space or shell syntax would be parsed by bash before the launcher
		// runs. The cleanup closure keeps the raw path for os.Remove.
		return paneCommand{
				shell: shellQuote(agentCmd),
				argv:  append(append([]string{agent.path}, agentFlags(agent, autoApprove)...), prompt),
			}, func() {
				os.Remove(promptPath)
				os.Remove(agentCmd)
			}, nil
	}
}

func (rc *coopRunCmd) debugAgentPaneCommandBuilder(stripeBin string) coopPaneCommandBuilder {
	return func(session *coop.Session) (paneCommand, func(), error) {
		sessionID := ""
		if session != nil {
			sessionID = session.ID
		}
		cmd := fmt.Sprintf("%s coop debug-agent --session %s", shellQuote(stripeBin), shellQuote(sessionID))
		return paneCommand{
			shell: shellCommandWithCoopEnv(cmd),
			argv:  []string{stripeBin, "coop", "debug-agent", "--session", sessionID},
		}, nil, nil
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
	paneWidth := currentTmuxPaneWidth()
	tuiWidth := coopTUIPaneWidthFor(paneWidth)
	if tuiWidth == 0 {
		return rc.runFallbackWithReason(stripeBin, blueprint, buildPaneCmd,
			fmt.Sprintf("This tmux pane is %d columns — too narrow to split. Running the agent here.", paneWidth))
	}

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
	shellCmd := shellCommandWithCoopEnv(paneCmd.shell)

	agentWidth := paneWidth - tuiWidth - coopPaneDividerWidth
	if _, err := splitCoopAgentPane("-h", "-l", strconv.Itoa(agentWidth), "bash", "-c", shellCmd); err != nil {
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

	width, height := coopTmuxSessionDimensions()
	tuiWidth := coopTUIPaneWidthFor(width)
	if tuiWidth == 0 {
		return rc.runFallbackWithReason(stripeBin, blueprint, buildPaneCmd,
			fmt.Sprintf("This terminal is %d columns — too narrow for a split view. Running the agent here.", width))
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
	shellCmd := shellCommandWithCoopEnv(paneCmd.shell)

	if err := runTmux("new-session", "-d", "-s", sessionName, "-x", strconv.Itoa(width), "-y", strconv.Itoa(height),
		"bash", "-c", tuiCmd); err != nil {
		if cleanup != nil {
			cleanup()
		}
		rc.abortStartedSession(session, "tmux new-session failed")
		return fmt.Errorf("tmux new-session failed: %w", err)
	}

	agentWidth := width - tuiWidth - coopPaneDividerWidth
	agentPane, err := splitCoopAgentPane("-h", "-t", sessionName, "-l", strconv.Itoa(agentWidth), "bash", "-c", shellCmd)
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
	return rc.runFallbackWithReason(stripeBin, blueprint, buildPaneCmd, "tmux not found — running agent in this terminal.")
}

func (rc *coopRunCmd) runFallbackWithReason(stripeBin string, blueprint *coop.Blueprint, buildPaneCmd coopPaneCommandBuilder, reason string) error {
	fmt.Println(reason)

	// Print the absolute binary path: the second terminal may be a login shell
	// whose PATH doesn't carry a version-manager shim or ~/.local/bin yet.
	joinBin := "stripe"
	if stripeBin != "" {
		joinBin = shellQuote(stripeBin)
	}

	var session *coop.Session
	if blueprint != nil {
		var err error
		session, err = rc.startSessionQuietly(blueprint)
		if err != nil {
			return err
		}
		fmt.Printf("Session started: %s\n", session.ID)
		fmt.Printf("Open another terminal and run: %s\n", shellCommandWithCoopEnv(joinBin+" coop join "+session.ID))
	} else {
		fmt.Printf("Open another terminal and run: %s\n", shellCommandWithCoopEnv(joinBin+" coop join --wait"))
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
	if len(paneCmd.argv) == 0 {
		rc.abortStartedSession(session, "agent command empty")
		return fmt.Errorf("agent command is empty")
	}
	// Direct exec, no shell: this is the only launch path that must work
	// everywhere, including Windows and containers without bash.
	agentExec := exec.Command(paneCmd.argv[0], paneCmd.argv[1:]...)
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
