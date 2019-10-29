package cmd

import (
	"bytes"
	"testing"

	"github.com/mitchellh/go-homedir"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/require"
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
	actual := Config.GetConfigFolder("")
	expected, err := homedir.Dir()
	expected += "/.config/stripe"

	require.NoError(t, err)
	require.Equal(t, actual, expected)
}

func TestGetPathXDG(t *testing.T) {
	actual := Config.GetConfigFolder("/some/xdg/path")
	expected := "/some/xdg/path/stripe"

	require.Equal(t, actual, expected)
}

func TestHelpFlag(t *testing.T) {
	Execute()

	output, err := executeCommand(rootCmd, "--help")

	require.Contains(t, output, "Stripe commands:")
	require.NoError(t, err)
}
