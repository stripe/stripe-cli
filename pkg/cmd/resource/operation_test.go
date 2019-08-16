package resource

import (
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"

	"github.com/stripe/stripe-cli/pkg/config"
)

func TestNewOperationCmd(t *testing.T) {
	parentCmd := &cobra.Command{Annotations: make(map[string]string)}

	oc := NewOperationCmd(parentCmd, "foo", "/v1/bars/{id}", "get", &config.Config{})

	assert.Equal(t, "foo", oc.Name)
	assert.Equal(t, "/v1/bars/{id}", oc.Path)
	assert.Equal(t, "GET", oc.HTTPVerb)
	assert.Equal(t, []string{"{id}"}, oc.URLParams)
	assert.True(t, parentCmd.HasSubCommands())
	val, ok := parentCmd.Annotations["foo"]
	assert.True(t, ok)
	assert.Equal(t, "operation", val)
	assert.Contains(t, oc.Cmd.UsageTemplate(), "<id>")
}
