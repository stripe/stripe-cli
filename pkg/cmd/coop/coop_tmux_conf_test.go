package coopcmd

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

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

	a := &tmuxReviewAlerter{tuiPane: "%1"}
	a.Alert(true, true)
	a.Alert(false, true)

	require.Len(t, calls, 2)
	assert.Equal(t, []string{"rename-window", "-t", "%1", "coop: REVIEW"}, calls[0])
	assert.Equal(t, []string{"set-window-option", "-t", "%1", "automatic-rename", "on"}, calls[1])
}
