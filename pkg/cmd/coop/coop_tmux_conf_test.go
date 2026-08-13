package coopcmd

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTmuxVersionAtLeast(t *testing.T) {
	tests := []struct {
		output string
		want   bool
	}{
		{output: "tmux 3.7b", want: true},
		{output: "tmux 3.2a", want: true},
		{output: "tmux 3.2", want: true},
		{output: "tmux 3.1c", want: false},
		{output: "tmux 3.0a", want: false},
		{output: "tmux 10.0", want: true},
		{output: "tmux next-3.8", want: false}, // unparseable prefix: assume old
		{output: "tmux", want: false},
		{output: "", want: false},
	}

	for _, tt := range tests {
		t.Run(tt.output, func(t *testing.T) {
			assert.Equal(t, tt.want, tmuxVersionAtLeast(tt.output, 3, 2))
		})
	}
}

// The status line is read back through the attaching client's locale; without
// a UTF-8 LANG/LC_ALL, non-ASCII characters silently degrade to underscores.
// Keep the shipped config pure ASCII.
func TestCoopTmuxConfIsASCIIOnly(t *testing.T) {
	for i, b := range []byte(coopTmuxBaseConf + coopTmuxModernConf) {
		require.Less(t, b, byte(128), "non-ASCII byte at offset %d", i)
	}
}

func TestCoopTmuxConfGatesModernOptionsOnVersion(t *testing.T) {
	original := tmuxVersionOutput
	t.Cleanup(func() { tmuxVersionOutput = original })

	tmuxVersionOutput = func() (string, error) { return "tmux 3.0a", nil }
	conf := coopTmuxConf()
	assert.NotContains(t, conf, "allow-passthrough", "tmux 3.0 rejects unknown options")
	assert.Contains(t, conf, "set -g mouse on")
	assert.Contains(t, conf, "set -g set-titles on")

	tmuxVersionOutput = func() (string, error) { return "tmux 3.7b", nil }
	conf = coopTmuxConf()
	assert.Contains(t, conf, "set -g allow-passthrough on")
	assert.Contains(t, conf, "set -g focus-events on")
	assert.Contains(t, conf, ",*:RGB")

	tmuxVersionOutput = func() (string, error) { return "", errors.New("no tmux") }
	assert.NotContains(t, coopTmuxConf(), "allow-passthrough")
}

func TestWriteCoopTmuxConf(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())

	path, err := writeCoopTmuxConf()
	require.NoError(t, err)
	assert.Equal(t, "coop.tmux.conf", filepath.Base(path))

	content, err := os.ReadFile(path)
	require.NoError(t, err)
	assert.Contains(t, string(content), "set -g mouse on")
	assert.Contains(t, string(content), "set -g bell-action any")
}

func TestTmuxReviewAlerterRenamesAndRestores(t *testing.T) {
	originalRunTmux := runTmux
	var calls [][]string
	runTmux = func(args ...string) error {
		calls = append(calls, append([]string(nil), args...))
		return nil
	}
	t.Cleanup(func() { runTmux = originalRunTmux })

	originalOutput := runTmuxOutput
	runTmuxOutput = func(args ...string) (string, error) {
		if len(args) > 0 && args[len(args)-1] == "#{window_name}" {
			return "my-project\n", nil
		}
		return "off\n", nil // user keeps automatic-rename off
	}
	t.Cleanup(func() { runTmuxOutput = originalOutput })

	a := &tmuxReviewAlerter{tuiPane: "%1"}
	a.Alert(true, true)
	a.Alert(false, true)

	require.Len(t, calls, 2)
	assert.Equal(t, []string{"rename-window", "-t", "%1", "coop: REVIEW"}, calls[0])
	assert.Equal(t, []string{"rename-window", "-t", "%1", "my-project"}, calls[1],
		"the user's own window name must come back")
	assert.NotContains(t, calls[1], "automatic-rename",
		"automatic-rename was off before; co-op must not switch it on")
}

func TestTmuxReviewAlerterRestoresAutomaticRenameWhenItWasOn(t *testing.T) {
	originalRunTmux := runTmux
	originalOutput := runTmuxOutput
	var calls [][]string
	runTmux = func(args ...string) error {
		calls = append(calls, append([]string(nil), args...))
		return nil
	}
	runTmuxOutput = func(args ...string) (string, error) {
		if len(args) > 0 && args[len(args)-1] == "#{window_name}" {
			return "zsh\n", nil
		}
		return "on\n", nil
	}
	t.Cleanup(func() {
		runTmux = originalRunTmux
		runTmuxOutput = originalOutput
	})

	a := &tmuxReviewAlerter{tuiPane: "%1"}
	a.Alert(true, true)
	a.Restore()

	require.Len(t, calls, 3)
	assert.Equal(t, []string{"set-window-option", "-t", "%1", "automatic-rename", "on"}, calls[2])
}

func TestTmuxReviewAlerterRestoreIsSafeWithoutRename(t *testing.T) {
	originalRunTmux := runTmux
	var calls [][]string
	runTmux = func(args ...string) error {
		calls = append(calls, append([]string(nil), args...))
		return nil
	}
	t.Cleanup(func() { runTmux = originalRunTmux })

	// Quitting a session that never showed a review must touch nothing.
	(&tmuxReviewAlerter{tuiPane: "%1"}).Restore()
	assert.Empty(t, calls)
}

func TestCoopPaneWidthPinReassertsWidthAfterResize(t *testing.T) {
	t.Setenv(coopTUIWidthEnv, "48")
	originalRunTmux := runTmux
	originalOutput := runTmuxOutput
	var calls [][]string
	runTmux = func(args ...string) error {
		calls = append(calls, append([]string(nil), args...))
		return nil
	}
	runTmuxOutput = func(args ...string) (string, error) { return "200\n", nil }
	t.Cleanup(func() {
		runTmux = originalRunTmux
		runTmuxOutput = originalOutput
	})

	pin := coopPaneWidthPin("%1")
	require.NotNil(t, pin)

	// tmux redistributed the pane down to 29 columns after a resize.
	pin(29, 40)
	require.Len(t, calls, 1)
	assert.Equal(t, []string{"resize-pane", "-t", "%1", "-x", "48"}, calls[0])

	// Already correct: nothing to do.
	calls = nil
	pin(48, 40)
	assert.Empty(t, calls)
}

func TestCoopPaneWidthPinLeavesNarrowWindowsAlone(t *testing.T) {
	t.Setenv(coopTUIWidthEnv, "48")
	originalRunTmux := runTmux
	originalOutput := runTmuxOutput
	var calls [][]string
	runTmux = func(args ...string) error {
		calls = append(calls, append([]string(nil), args...))
		return nil
	}
	// Window shrank below the split threshold entirely.
	runTmuxOutput = func(args ...string) (string, error) { return "90\n", nil }
	t.Cleanup(func() {
		runTmux = originalRunTmux
		runTmuxOutput = originalOutput
	})

	coopPaneWidthPin("%1")(20, 40)
	assert.Empty(t, calls, "pinning 48 columns in a 90-column window would starve the agent")
}

func TestCoopPaneWidthPinInactiveOutsideLauncher(t *testing.T) {
	t.Setenv(coopTUIWidthEnv, "")
	assert.Nil(t, coopPaneWidthPin("%1"), "a hand-run coop join must never resize the user's pane")
}

func TestTmuxAgentResumerRefusesToTypeAtABusyAgent(t *testing.T) {
	originalRunTmux := runTmux
	originalOutput := runTmuxOutput
	var sends [][]string
	runTmux = func(args ...string) error {
		sends = append(sends, append([]string(nil), args...))
		return nil
	}
	runTmuxOutput = func(args ...string) (string, error) { return "%2\t1\t0\n", nil }
	t.Cleanup(func() {
		runTmux = originalRunTmux
		runTmuxOutput = originalOutput
	})

	resumer := newTmuxAgentResumer("%1")
	resumer.keyDelay = 0
	// No heartbeat: not parked in await-review. But the session still has an
	// active node, so the agent is mid-work and may have a dialog open.
	resumer.heartbeatAge = func(string) (time.Duration, error) { return 0, errors.New("no heartbeat") }
	resumer.agentIsWorking = func(string) bool { return true }

	err := resumer.Notify("session_busy")

	require.ErrorIs(t, err, errAgentBusy)
	assert.Empty(t, sends, "typing into a working agent can answer an open permission dialog")
}
