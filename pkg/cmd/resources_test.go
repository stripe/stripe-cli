package cmd

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestResources(t *testing.T) {
	output, err := executeCommand(rootCmd, "resources")

	require.Contains(t, output, "Available commands:")
	require.NoError(t, err)
}

func TestConflictWithPluginCommand(t *testing.T) {
	pluginCommands, err := getAllPluginCommands()
	require.NoError(t, err)

	for _, cmd := range rootCmd.Commands() {
		for _, pluginCommand := range pluginCommands {
			// TO-DO: this is a patch.
			// this check and this patch PR https://github.com/stripe/stripe-cli/pull/887
			// should be removed once openapi spec is updated to not use `apps`
			if cmd.Use == "apps" {
				continue
			}
			require.False(t, cmd.Use == pluginCommand)
		}
	}
}
