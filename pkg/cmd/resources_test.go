package cmd

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestResources(t *testing.T) {
	Execute()
	output, err := executeCommand(rootCmd, "resources")

	assert.Contains(t, output, "Available Namespaces:")
	assert.Contains(t, output, "Available Resources:")
	assert.NoError(t, err)
}
