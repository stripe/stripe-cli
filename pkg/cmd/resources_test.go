package cmd

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestResources(t *testing.T) {
	Execute()

	output, err := executeCommand(rootCmd, "resources")

	require.Contains(t, output, "Available Namespaces:")
	require.Contains(t, output, "Available Resources:")
	require.NoError(t, err)
}
