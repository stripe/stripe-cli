package cmd

import (
	"bytes"
	"testing"

	"github.com/mitchellh/go-homedir"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
)

func executeCommand(root *cobra.Command, args ...string) (output string, err error) {
	_, output, err = executeCommandC(root, args...)
	return output, err
}

func executeCommandC(root *cobra.Command, args ...string) (c *cobra.Command, output string, err error) {
	buf := new(bytes.Buffer)
	root.SetOutput(buf)
	root.SetArgs(args)

	c, err = root.ExecuteC()

	return c, buf.String(), err
}

func TestGetPathNoXDG(t *testing.T) {
	actual := Config.GetProfilesFolder("")
	expected, err := homedir.Dir()
	expected += "/.config/stripe"

	assert.Nil(t, err)
	assert.Equal(t, actual, expected)
}

func TestGetPathXDG(t *testing.T) {
	actual := Config.GetProfilesFolder("/some/xdg/path")
	expected := "/some/xdg/path/stripe"

	assert.Equal(t, actual, expected)
}

func TestHelpFlag(t *testing.T) {
	Execute()
	output, err := executeCommand(rootCmd, "--help")

	assert.Contains(t, output, "Stripe commands:")
	assert.NoError(t, err)
}
