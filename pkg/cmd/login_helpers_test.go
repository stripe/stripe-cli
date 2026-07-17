package cmd

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestShouldAutoLogin(t *testing.T) {
	t.Run("no agent, interactive terminal", func(t *testing.T) {
		t.Setenv("AI_AGENT", "")
		t.Setenv("CLAUDECODE", "")
		require.True(t, shouldAutoLogin(true))
	})
	t.Run("no agent, non-TTY stdin", func(t *testing.T) {
		t.Setenv("AI_AGENT", "")
		t.Setenv("CLAUDECODE", "")
		require.False(t, shouldAutoLogin(false))
	})
	t.Run("agent with real TTY", func(t *testing.T) {
		t.Setenv("AI_AGENT", "")
		t.Setenv("CLAUDECODE", "1")
		require.False(t, shouldAutoLogin(true))
	})
	t.Run("agent, no TTY", func(t *testing.T) {
		t.Setenv("AI_AGENT", "")
		t.Setenv("CLAUDECODE", "1")
		require.False(t, shouldAutoLogin(false))
	})
}
