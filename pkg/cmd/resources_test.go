package cmd

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestResources(t *testing.T) {
	Execute()

	output, err := executeCommand(rootCmd, "resources")

	require.Contains(t, output, "Available commands:")
	require.NoError(t, err)
}
