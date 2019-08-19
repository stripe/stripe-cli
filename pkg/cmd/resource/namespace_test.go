package resource

import (
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/require"
)

func TestNewNamespaceCmd_NonEmptyName(t *testing.T) {
	rootCmd := &cobra.Command{Annotations: make(map[string]string)}

	nsc := NewNamespaceCmd(rootCmd, "foo")

	require.Equal(t, "foo", nsc.Name)
	require.True(t, rootCmd.HasSubCommands())
	val, ok := rootCmd.Annotations["foo"]
	require.True(t, ok)
	require.Equal(t, "namespace", val)
	require.Contains(t, nsc.Cmd.UsageTemplate(), "Available Resources")
}

func TestNewNamespaceCmd_EmptyName(t *testing.T) {
	rootCmd := &cobra.Command{Annotations: make(map[string]string)}

	nsc := NewNamespaceCmd(rootCmd, "")

	require.Equal(t, "", nsc.Name)
	require.False(t, rootCmd.HasSubCommands())
	_, ok := rootCmd.Annotations[""]
	require.False(t, ok)
	require.Contains(t, nsc.Cmd.UsageTemplate(), "Available Resources")
}
