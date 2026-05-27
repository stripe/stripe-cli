package cmd_test

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stripe/stripe-cli-docs-plugin/cmd"
)

func TestNew(t *testing.T) {
	root := cmd.New().Root()

	assert.Equal(t, "docs", root.Use)
	assert.NotEmpty(t, root.Short)
	assert.NotEmpty(t, root.Long)
}

func TestVersionCommand(t *testing.T) {
	root := cmd.New().WithOptions(cmd.WithVersion("0.0.1")).Root()

	out := new(bytes.Buffer)
	root.SetOut(out)
	root.SetArgs([]string{"version"})

	err := root.Execute()

	assert.NoError(t, err)
	assert.Equal(t, "stripe docs version 0.0.1\n", out.String())
}
