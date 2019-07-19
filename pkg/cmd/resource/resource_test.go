package resource

import (
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
)

func TestNewResourceCmd(t *testing.T) {
	parentCmd := &cobra.Command{Annotations: make(map[string]string)}

	rc := NewResourceCmd(parentCmd, "foo")

	assert.Equal(t, "foo", rc.Name)
	assert.True(t, parentCmd.HasSubCommands())
	val, ok := parentCmd.Annotations["foo"]
	assert.True(t, ok)
	assert.Equal(t, "resource", val)
	assert.Contains(t, rc.Cmd.UsageTemplate(), "Available Operations")
}
