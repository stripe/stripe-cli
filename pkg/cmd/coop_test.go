package cmd

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestCoopCommandDoesNotRegisterLegacyAgentCommands(t *testing.T) {
	cmd := newCoopCmd().cmd

	_, _, err := cmd.Find([]string{"step"})
	require.Error(t, err)

	_, _, err = cmd.Find([]string{"next-steps"})
	require.Error(t, err)
}
