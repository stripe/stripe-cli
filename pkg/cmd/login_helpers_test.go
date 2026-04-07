package cmd

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestShouldAutoLogin(t *testing.T) {
	noAgent := func(string) string { return "" }
	claudeAgent := func(k string) string {
		if k == "CLAUDECODE" {
			return "1"
		}
		return ""
	}

	require.True(t, shouldAutoLogin(noAgent, true))       // interactive terminal, no agent → auto-login
	require.False(t, shouldAutoLogin(noAgent, false))     // non-TTY stdin (CI, /dev/null) → fast-fail
	require.False(t, shouldAutoLogin(claudeAgent, true))  // agent with real TTY → fast-fail
	require.False(t, shouldAutoLogin(claudeAgent, false)) // agent, no TTY → fast-fail
}
