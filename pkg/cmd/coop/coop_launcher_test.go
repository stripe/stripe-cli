package coopcmd

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"sync"
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
	assert.Contains(t, prompt, "intentional 10-minute Co-op timeout")
	assert.Contains(t, prompt, "stripe coop agent resume")
	assert.Contains(t, prompt, "Do not spin, poll forever")
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

	err := rc.runFallbackWithCommand("/stripe", "one-time-payment", func(session *coop.Session) (string, func(), error) {
		require.NotNil(t, session)
		return "", nil, buildErr
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
		err := rc.runFallbackWithCommand("/stripe", "one-time-payment", func(session *coop.Session) (string, func(), error) {
			require.NotNil(t, session)
			return "true", nil, nil
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
		err := rc.runFallbackWithCommand("/stripe", "", func(session *coop.Session) (string, func(), error) {
			require.Nil(t, session)
			return "true", nil, nil
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
	originalRunTmuxOutput := runTmuxOutput
	runTmux = func(args ...string) error {
		tmuxCalls = append(tmuxCalls, append([]string(nil), args...))
		switch args[0] {
		case "has-session":
			return errors.New("session not found")
		default:
			return nil
		}
	}
	runTmuxOutput = func(args ...string) (string, error) {
		tmuxCalls = append(tmuxCalls, append([]string(nil), args...))
		if args[0] == "split-window" {
			return "", splitErr
		}
		return "", nil
	}
	t.Cleanup(func() {
		runTmux = originalRunTmux
		runTmuxOutput = originalRunTmuxOutput
	})

	cleanupCalled := false
	rc := &coopRunCmd{language: "node"}
	err := rc.runInNewTmuxWithCommand("/stripe", "one-time-payment", func(session *coop.Session) (string, func(), error) {
		require.NotNil(t, session)
		return "agent", func() { cleanupCalled = true }, nil
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

func TestSplitCoopAgentPaneTagsReturnedPane(t *testing.T) {
	originalRunTmux := runTmux
	originalRunTmuxOutput := runTmuxOutput
	var calls [][]string
	runTmuxOutput = func(args ...string) (string, error) {
		calls = append(calls, append([]string(nil), args...))
		return "%7\n", nil
	}
	runTmux = func(args ...string) error {
		calls = append(calls, append([]string(nil), args...))
		return nil
	}
	t.Cleanup(func() {
		runTmux = originalRunTmux
		runTmuxOutput = originalRunTmuxOutput
	})

	pane, err := splitCoopAgentPane("-h", "bash", "-c", "agent")

	require.NoError(t, err)
	assert.Equal(t, "%7", pane)
	require.Len(t, calls, 2)
	assert.Equal(t, []string{"split-window", "-P", "-F", "#{pane_id}", "-h", "bash", "-c", "agent"}, calls[0])
	assert.Equal(t, []string{"set-option", "-p", "-t", "%7", coopAgentPaneOption, "1"}, calls[1])
}

func TestTmuxAgentResumerSerializesConcurrentWakeUps(t *testing.T) {
	originalRunTmux := runTmux
	originalRunTmuxOutput := runTmuxOutput
	var callsMu sync.Mutex
	var calls [][]string
	runTmuxOutput = func(args ...string) (string, error) {
		return "%1\t\n%2\t1\n", nil
	}
	runTmux = func(args ...string) error {
		callsMu.Lock()
		defer callsMu.Unlock()
		calls = append(calls, append([]string(nil), args...))
		return nil
	}
	t.Cleanup(func() {
		runTmux = originalRunTmux
		runTmuxOutput = originalRunTmuxOutput
	})

	resumer := newTmuxAgentResumer("%1")
	resumer.keyDelay = 0
	var wg sync.WaitGroup
	errs := make(chan error, 3)
	for _, sessionID := range []string{"session_one", "session_two", "session_three"} {
		wg.Add(1)
		go func() {
			defer wg.Done()
			errs <- resumer.Notify(sessionID)
		}()
	}
	wg.Wait()
	close(errs)
	for err := range errs {
		require.NoError(t, err)
	}

	callsMu.Lock()
	defer callsMu.Unlock()
	require.Len(t, calls, 6)
	for i := 0; i < len(calls); i += 2 {
		require.Len(t, calls[i], 5)
		assert.Equal(t, []string{"send-keys", "-t", "%2", "-l"}, calls[i][:4])
		assert.True(t, strings.Contains(calls[i][4], "stripe coop agent resume --session=session_"))
		assert.Equal(t, []string{"send-keys", "-t", "%2", "Enter"}, calls[i+1])
	}
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
	paneCmd, cleanup, err := build(nil)
	require.NoError(t, err)
	require.NotNil(t, cleanup)
	defer cleanup()

	// The generated launcher is the only .sh under the temp dir.
	matches, err := filepath.Glob(filepath.Join(tmp, "*.sh"))
	require.NoError(t, err)
	require.Len(t, matches, 1)

	// The pane command must be exactly the single-quoted launcher path, so
	// `bash -c` executes the launcher instead of parsing the path.
	assert.Equal(t, shellQuote(matches[0]), paneCmd)
}
