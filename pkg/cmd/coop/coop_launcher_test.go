package coopcmd

import (
	"errors"
	"os"
	"path/filepath"
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
	assert.Contains(t, prompt, `"steps"`)
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
	err := rc.runInNewTmuxWithCommand("/stripe", "one-time-payment", func(session *coop.Session) (string, func(), error) {
		require.NotNil(t, session)
		return "agent", func() { cleanupCalled = true }, nil
	})
	require.ErrorIs(t, err, splitErr)
	assert.True(t, cleanupCalled)
	assert.True(t, hasTmuxCall(tmuxCalls, "kill-session", "-t", "stripe-coop"))

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
