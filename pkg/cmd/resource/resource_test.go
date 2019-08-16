package resource

import (
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/require"
)

func TestNewResourceCmd(t *testing.T) {
	parentCmd := &cobra.Command{Annotations: make(map[string]string)}

	rc := NewResourceCmd(parentCmd, "foo")

	require.Equal(t, "foo", rc.Name)
	require.True(t, parentCmd.HasSubCommands())
	val, ok := parentCmd.Annotations["foo"]
	require.True(t, ok)
	require.Equal(t, "resource", val)
	require.Contains(t, rc.Cmd.UsageTemplate(), "Available Operations")
}
