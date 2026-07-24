package coopcmd

import (
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"charm.land/huh/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/stripe/stripe-cli/pkg/coop"
)

func TestNormalizeCoopTmuxSessionDimensionsUsesTerminalSize(t *testing.T) {
	width, height := normalizeCoopTmuxSessionDimensions(260, 60, nil)

	assert.Equal(t, 260, width)
	assert.Equal(t, 60, height)
}

func TestNormalizeCoopTmuxSessionDimensionsFallsBack(t *testing.T) {
	tests := []struct {
		name   string
		width  int
		height int
		err    error
	}{
		{name: "size error", width: 260, height: 60, err: errors.New("not a terminal")},
		{name: "zero width", width: 0, height: 60},
		{name: "zero height", width: 260, height: 0},
		{name: "negative width", width: -1, height: 60},
		{name: "negative height", width: 260, height: -1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			width, height := normalizeCoopTmuxSessionDimensions(tt.width, tt.height, tt.err)

			assert.Equal(t, defaultCoopTmuxSessionWidth, width)
			assert.Equal(t, defaultCoopTmuxSessionHeight, height)
		})
	}
}

func TestExplicitBlueprintPromptIncludesSessionProtocol(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())

	rc := &coopRunCmd{language: "node"}
	session, err := rc.startSessionQuietly("one-time-payment")
	require.NoError(t, err)

	prompt, err := rc.buildAgentPromptForSession(session)
	require.NoError(t, err)

	assert.Contains(t, prompt, session.ID)
	assert.Contains(t, prompt, `"agent_instructions"`)
	assert.Contains(t, prompt, `"nodes"`)
	assert.Contains(t, prompt, `"next": "stripe coop agent start-work --session=`+session.ID+` --step=1`)
	assert.Contains(t, prompt, "Understand the project")
	assert.Contains(t, prompt, "Start by running the \"next\" command exactly as written")
}

func TestPromptAutoApproveReturnsPromptErrors(t *testing.T) {
	promptErr := errors.New("permission prompt canceled")
	originalSelectString := selectString
	selectString = func(title string, options []huh.Option[string], value *string) error {
		return promptErr
	}
	t.Cleanup(func() {
		selectString = originalSelectString
	})

	autoApprove, err := (&coopRunCmd{}).promptAutoApprove(&agentInfo{name: "claude"})

	require.ErrorIs(t, err, promptErr)
	assert.False(t, autoApprove)
}

func TestFallbackPaneBuildFailureAbortsStartedSession(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())

	rc := &coopRunCmd{language: "node"}
	buildErr := errors.New("pane build failed")

	err := rc.runFallbackWithCommand("/stripe", "one-time-payment", func(session *coop.Session) (coopPaneCommand, func(), error) {
		require.NotNil(t, session)
		return coopPaneCommand{}, nil, buildErr
	})
	require.ErrorIs(t, err, buildErr)

	store, err := coop.NewStore(coopConfigFolder())
	require.NoError(t, err)
	session, err := store.LatestSession()
	require.NoError(t, err)
	assert.Equal(t, coop.SessionAborted, session.Status)
	node, err := session.NodeByNumber(1)
	require.NoError(t, err)
	assert.Contains(t, node.Activity, "agent pane command failed")

	_, err = store.LatestActiveSession()
	assert.Error(t, err)
}

func TestFallbackJoinInstructionsIncludeCoopEnv(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())

	rc := &coopRunCmd{language: "node"}
	output := captureStdout(t, func() {
		err := rc.runFallbackWithCommand("/stripe", "one-time-payment", func(session *coop.Session) (coopPaneCommand, func(), error) {
			require.NotNil(t, session)
			return coopPaneCommand{cmd: "true"}, nil, nil
		})
		require.NoError(t, err)
	})

	assert.Contains(t, output, "Open another terminal and run: XDG_CONFIG_HOME=")
	assert.Contains(t, output, " stripe coop join coop_")
}

func TestFallbackWaitInstructionsIncludeCoopEnv(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())

	rc := &coopRunCmd{language: "node"}
	output := captureStdout(t, func() {
		err := rc.runFallbackWithCommand("/stripe", "", func(session *coop.Session) (coopPaneCommand, func(), error) {
			require.Nil(t, session)
			return coopPaneCommand{cmd: "true"}, nil, nil
		})
		require.NoError(t, err)
	})

	assert.Contains(t, output, "Open another terminal and run: XDG_CONFIG_HOME=")
	assert.Contains(t, output, " stripe coop join --wait")
}

func TestNewTmuxSplitFailureKillsTmuxSessionAndAbortsStartedSession(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())

	splitErr := errors.New("split failed")
	var tmuxCalls [][]string
	originalRunTmux := runTmux
	runTmux = func(args ...string) error {
		tmuxCalls = append(tmuxCalls, append([]string(nil), args...))
		switch args[0] {
		case "has-session":
			return errors.New("session not found")
		case "split-window":
			return splitErr
		default:
			return nil
		}
	}
	t.Cleanup(func() {
		runTmux = originalRunTmux
	})

	cleanupCalled := false
	rc := &coopRunCmd{language: "node"}
	err := rc.runInNewTmuxWithCommand("/stripe", "one-time-payment", func(session *coop.Session) (coopPaneCommand, func(), error) {
		require.NotNil(t, session)
		return coopPaneCommand{cmd: "agent"}, func() { cleanupCalled = true }, nil
	})
	require.ErrorIs(t, err, splitErr)
	assert.True(t, cleanupCalled)
	assert.True(t, hasTmuxCall(tmuxCalls, "kill-session", "-t", "stripe-coop"))
	newSessionCall := findTmuxCall(tmuxCalls, "new-session")
	require.NotNil(t, newSessionCall)
	assert.Contains(t, newSessionCall[len(newSessionCall)-1], "XDG_CONFIG_HOME=")
	assert.Contains(t, newSessionCall[len(newSessionCall)-1], " coop join ")
	splitCall := findTmuxCall(tmuxCalls, "split-window")
	require.NotNil(t, splitCall)
	assert.Contains(t, splitCall[len(splitCall)-1], "XDG_CONFIG_HOME=")

	store, err := coop.NewStore(coopConfigFolder())
	require.NoError(t, err)
	session, err := store.LatestSession()
	require.NoError(t, err)
	assert.Equal(t, coop.SessionAborted, session.Status)
	node, err := session.NodeByNumber(1)
	require.NoError(t, err)
	assert.Contains(t, node.Activity, "tmux split-window failed")
}

func hasTmuxCall(calls [][]string, want ...string) bool {
	for _, call := range calls {
		if len(call) != len(want) {
			continue
		}
		matches := true
		for i := range call {
			if call[i] != want[i] {
				matches = false
				break
			}
		}
		if matches {
			return true
		}
	}
	return false
}

func findTmuxCall(calls [][]string, command string) []string {
	for _, call := range calls {
		if len(call) > 0 && call[0] == command {
			return call
		}
	}
	return nil
}

func TestShellQuoteNeutralizesShellMetacharacters(t *testing.T) {
	cases := []struct {
		name string
		in   string
		want string
	}{
		{"plain", "/usr/local/bin/claude", `'/usr/local/bin/claude'`},
		{"command substitution", "/tmp/a$(touch pwned)b", `'/tmp/a$(touch pwned)b'`},
		{"backticks", "/tmp/`whoami`", "'/tmp/`whoami`'"},
		{"embedded single quote", "it's", `'it'\''s'`},
		{"empty", "", `''`},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := shellQuote(tc.in)
			assert.Equal(t, tc.want, got)
			// The quoted form must contain no bare $(, backtick, or unescaped quote
			// that could break out of the single-quoted context.
			assert.True(t, len(got) >= 2 && got[0] == '\'' && got[len(got)-1] == '\'')
		})
	}
}

// TestAgentPaneCommandShellQuotesLauncherPath guards the launcher path itself
// (not just the values inside the generated script) against a TMPDIR containing
// a space or shell syntax, since the pane command is executed via `bash -c`.
func TestAgentPaneCommandShellQuotesLauncherPath(t *testing.T) {
	tmp := filepath.Join(t.TempDir(), "dir with $(spaces)")
	require.NoError(t, os.MkdirAll(tmp, 0o755))
	// Redirect os.CreateTemp across platforms: Unix reads TMPDIR, Windows TMP/TEMP.
	t.Setenv("TMPDIR", tmp)
	t.Setenv("TMP", tmp)
	t.Setenv("TEMP", tmp)

	rc := &coopRunCmd{}
	build := rc.agentPaneCommandBuilder(&agentInfo{name: "claude", path: "/usr/local/bin/claude"}, "discovery prompt", false)
	pane, cleanup, err := build(nil)
	require.NoError(t, err)
	require.NotNil(t, cleanup)
	defer cleanup()

	// The generated launcher is the only .sh under the temp dir.
	matches, err := filepath.Glob(filepath.Join(tmp, "*.sh"))
	require.NoError(t, err)
	require.Len(t, matches, 1)

	// The pane command must be exactly the single-quoted launcher path, so
	// `bash -c` executes the launcher instead of parsing the path.
	assert.Equal(t, shellQuote(matches[0]), pane.cmd)
}

// TestHoldOpenPaneScriptSurvivesAgentExit runs the generated pane script
// through a real bash so the printf quoting is exercised the way tmux would
// exercise it: the agent command runs, the notice explains the exit, and the
// script ends by handing the pane to a shell instead of letting tmux tear it
// down.
func TestHoldOpenPaneScriptSurvivesAgentExit(t *testing.T) {
	if _, err := exec.LookPath("bash"); err != nil {
		t.Skip("bash not available")
	}
	t.Setenv("XDG_CONFIG_HOME", "/tmp/xdg config")

	script := holdOpenPaneScript(coopPaneCommand{
		cmd: `printf 'agent ran with %s\n' "$XDG_CONFIG_HOME"; exit 3`,
		hints: []paneHint{
			{label: "Resume where the agent left off", command: "claude --continue"},
		},
	})

	cmd := exec.Command("bash", "-c", script)
	// The trailing `exec $SHELL` needs a closed stdin to return; in a tmux pane
	// it is an interactive shell that stays put.
	cmd.Stdin = strings.NewReader("")
	out, err := cmd.CombinedOutput()
	require.NoError(t, err, string(out))

	output := string(out)
	assert.Contains(t, output, "agent ran with /tmp/xdg config")
	assert.Contains(t, output, "The agent exited")
	assert.Contains(t, output, "(status 3)")
	assert.Contains(t, output, "still")
	assert.Contains(t, output, "Resume where the agent left off")
	assert.Contains(t, output, "claude --continue")
	assert.Contains(t, output, "ctrl-b then left arrow")
	assert.Contains(t, output, "exit")
}

// TestHoldOpenPaneScriptQuotesHints guards against a hint containing shell
// syntax being evaluated when the notice prints.
func TestHoldOpenPaneScriptQuotesHints(t *testing.T) {
	if _, err := exec.LookPath("bash"); err != nil {
		t.Skip("bash not available")
	}

	script := holdOpenPaneScript(coopPaneCommand{
		cmd:   "true",
		hints: []paneHint{{label: "Restart", command: "'/tmp/my agent' $(touch pwned) `whoami`"}},
	})

	cmd := exec.Command("bash", "-c", script)
	cmd.Dir = t.TempDir()
	cmd.Stdin = strings.NewReader("")
	out, err := cmd.CombinedOutput()
	require.NoError(t, err, string(out))

	assert.Contains(t, string(out), "'/tmp/my agent' $(touch pwned) `whoami`")
	_, statErr := os.Stat(filepath.Join(cmd.Dir, "pwned"))
	assert.True(t, os.IsNotExist(statErr), "command substitution in a hint must not execute")
}

func TestAgentRestartHints(t *testing.T) {
	claude := agentRestartHints(&agentInfo{name: "claude", path: "/usr/local/bin/claude"}, false)
	require.Len(t, claude, 2)
	assert.Equal(t, "claude --continue", claude[0].command)
	assert.Equal(t, "claude", claude[1].command)

	claudeAuto := agentRestartHints(&agentInfo{name: "claude"}, true)
	assert.Equal(t, "claude --dangerously-skip-permissions --continue", claudeAuto[0].command)

	codex := agentRestartHints(&agentInfo{name: "codex"}, true)
	require.Len(t, codex, 1)
	assert.Equal(t, "codex --dangerously-bypass-approvals-and-sandbox", codex[0].command)
}

// TestTmuxSplitPaneCommandHoldsPaneOpen pins the wiring: the command handed to
// tmux must be the hold-open script, not the bare agent command.
func TestTmuxSplitPaneCommandHoldsPaneOpen(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())

	var splitCmd string
	originalRunTmux := runTmux
	runTmux = func(args ...string) error {
		if args[0] == "split-window" {
			splitCmd = args[len(args)-1]
			return errors.New("stop before the TUI runs")
		}
		return nil
	}
	t.Cleanup(func() { runTmux = originalRunTmux })

	rc := &coopRunCmd{language: "node"}
	err := rc.runInTmuxSplitWithCommand("/stripe", "one-time-payment", func(session *coop.Session) (coopPaneCommand, func(), error) {
		return coopPaneCommand{cmd: "agent", hints: []paneHint{{label: "Start the agent again", command: "claude"}}}, nil, nil
	})
	require.Error(t, err)

	assert.Contains(t, splitCmd, "export XDG_CONFIG_HOME=")
	assert.Contains(t, splitCmd, "(\nagent\n)")
	assert.Contains(t, splitCmd, "The agent exited")
	assert.Contains(t, splitCmd, `exec "${SHELL:-/bin/bash}"`)
}

// TestFallbackDoesNotHoldTerminalOpen keeps the non-tmux path unchanged: the
// developer is in their own shell already, so the agent exiting should return
// them to it rather than nesting another shell.
func TestFallbackDoesNotHoldTerminalOpen(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())

	rc := &coopRunCmd{language: "node"}
	output := captureStdout(t, func() {
		err := rc.runFallbackWithCommand("/stripe", "", func(session *coop.Session) (coopPaneCommand, func(), error) {
			return coopPaneCommand{cmd: "printf fallback-ran", hints: []paneHint{{label: "Restart", command: "claude"}}}, nil, nil
		})
		require.NoError(t, err)
	})

	assert.Contains(t, output, "fallback-ran")
	assert.NotContains(t, output, "The agent exited")
}

// TestHoldOpenPaneScriptSurvivesInterrupt covers the most common way a
// developer stops the agent: ctrl-c, which the terminal delivers to every
// process in the pane's foreground group — the wrapper included. Without the
// trap the wrapper dies too and tmux closes the pane before the notice prints.
func TestHoldOpenPaneScriptSurvivesInterrupt(t *testing.T) {
	if _, err := exec.LookPath("bash"); err != nil {
		t.Skip("bash not available")
	}

	// $$ is the wrapper's PID even inside the subshell, so this interrupts the
	// wrapper the same way ctrl-c would.
	script := holdOpenPaneScript(coopPaneCommand{cmd: "kill -INT $$"})

	cmd := exec.Command("bash", "-c", script)
	cmd.Stdin = strings.NewReader("")
	out, err := cmd.CombinedOutput()
	require.NoError(t, err, string(out))
	assert.Contains(t, string(out), "The agent exited")
}
