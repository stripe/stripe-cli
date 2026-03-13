package cmd

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestWhoamiCmdExists(t *testing.T) {
	// Test that the whoami command is properly registered
	cmd := newWhoamiCmd()
	require.NotNil(t, cmd)
	require.NotNil(t, cmd.cmd)
	require.Equal(t, "whoami", cmd.cmd.Use)
	require.Equal(t, "Show the current Stripe account details", cmd.cmd.Short)
}

func TestWhoamiCmdFlags(t *testing.T) {
	cmd := newWhoamiCmd()

	// Check that the format flag exists
	formatFlag := cmd.cmd.Flags().Lookup("format")
	require.NotNil(t, formatFlag)
	require.Equal(t, "default", formatFlag.DefValue)
}

func TestWhoamiCmdHelp(t *testing.T) {
	output, err := executeCommand(rootCmd, "whoami", "--help")

	require.NoError(t, err)
	require.Contains(t, output, "whoami")
	require.Contains(t, output, "current Stripe account")
}
