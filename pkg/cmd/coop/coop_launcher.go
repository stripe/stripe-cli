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
			strconv.Quote(promptPath), strconv.Quote(promptPath), strconv.Quote(launcherPath), strconv.Quote(agent.path), flags)

	case "codex":
		flags := ""
		if autoApprove {
			flags = " --dangerously-bypass-approvals-and-sandbox"
		}
		script = fmt.Sprintf("#!/bin/bash\nprompt=$(cat %s)\nrm -f %s %s\nexec %s%s \"$prompt\"\n",
			strconv.Quote(promptPath), strconv.Quote(promptPath), strconv.Quote(launcherPath), strconv.Quote(agent.path), flags)

	default:
		script = fmt.Sprintf("#!/bin/bash\nprompt=$(cat %s)\nrm -f %s %s\nexec %s \"$prompt\"\n",
			strconv.Quote(promptPath), strconv.Quote(promptPath), strconv.Quote(launcherPath), strconv.Quote(agent.path))
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
		return agentCmd, func() {
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
		cmd := fmt.Sprintf("%s coop debug-agent --session %s", strconv.Quote(stripeBin), strconv.Quote(sessionID))
		return shellCommandWithCoopEnv(cmd), nil, nil
	}
}

func shellCommandWithCoopEnv(cmd string) string {
	if xdgConfigHome := os.Getenv("XDG_CONFIG_HOME"); xdgConfigHome != "" && !strings.HasPrefix(cmd, "XDG_CONFIG_HOME=") {
		return fmt.Sprintf("XDG_CONFIG_HOME=%s %s", strconv.Quote(xdgConfigHome), cmd)
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

	paneCmd, cleanup, err := buildPaneCmd(session)
	if err != nil {
		rc.abortStartedSession(session, "agent pane command failed")
		return err
	}
	paneCmd = shellCommandWithCoopEnv(paneCmd)

	if err := runTmux("split-window", "-h", "-p", "60", "bash", "-c", paneCmd); err != nil {
		if cleanup != nil {
			cleanup()
		}
		rc.abortStartedSession(session, "tmux split failed")
		return fmt.Errorf("tmux split failed: %w", err)
	}

	store, err := coop.NewStore(coopConfigFolder())
	if err != nil {
		return err
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

	tuiCmd := fmt.Sprintf("%s coop join", strconv.Quote(stripeBin))
	if blueprintID == "" {
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
	return tui.RunWaiting(store, existingIDs, tui.WithSandboxClaimURL(coopSandboxClaimURL()))
}
