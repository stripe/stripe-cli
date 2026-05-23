package cmd_test

import (
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
