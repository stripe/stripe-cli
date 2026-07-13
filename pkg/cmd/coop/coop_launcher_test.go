package coopcmd

import (
	"errors"
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
