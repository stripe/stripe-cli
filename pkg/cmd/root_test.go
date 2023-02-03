package cmd

import (
	"bytes"
	"context"
	"os"
	"os/exec"
	"path/filepath"
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
	expected += filepath.Join("/", ".config", "stripe")

	require.NoError(t, err)
	require.Equal(t, actual, expected)
}

func TestGetPathXDG(t *testing.T) {
	actual := Config.GetConfigFolder("/some/xdg/path")
	expected := filepath.Join("/", "some", "xdg", "path", "stripe")

	require.Equal(t, actual, expected)
}

func TestHelpFlag(t *testing.T) {
	Execute(context.Background())

	output, err := executeCommand(rootCmd, "--help")

	require.Contains(t, output, "Stripe commands:")
	require.NoError(t, err)
}

func TestExampleCommands(t *testing.T) {
	{
		_, err := executeCommand(rootCmd, "foo")
		require.Equal(t, err.Error(), "unknown command \"foo\" for \"stripe\"")
	}
	{
		_, err := executeCommand(rootCmd, "listen", "foo")
		require.Equal(t, err.Error(), "`stripe listen` does not take any positional arguments. See `stripe listen --help` for supported flags and usage")
	}
	{
		_, err := executeCommand(rootCmd, "post")
		require.Equal(t, err.Error(), "`stripe post` requires exactly 1 positional argument. See `stripe post --help` for supported flags and usage")
	}
	{
		_, err := executeCommand(rootCmd, "samples", "create", "foo", "foo", "foo")
		require.Equal(t, err.Error(), "`stripe samples create` accepts at maximum 2 positional arguments. See `stripe samples create --help` for supported flags and usage")
	}
}

func TestReadProjectDefault(t *testing.T) {
	executeCommand(rootCmd, "version")
	require.Equal(t, Config.Profile.ProfileName, "default")
}

func TestReadProjectFromEnv(t *testing.T) {
	// Run this test in a subprocess since side effects from other tests interfere with this
	if os.Getenv("BE_TestReadProjectFromEnv") == "1" {
		os.Setenv("STRIPE_PROJECT_NAME", "from-env")
		defer os.Unsetenv("STRIPE_PROJECT_NAME")

		executeCommand(rootCmd, "version")

		require.Equal(t, Config.Profile.ProfileName, "from-env")
		return
	}
	cmd := exec.Command(os.Args[0], "-test.run=TestReadProjectFromEnv")
	cmd.Env = append(os.Environ(), "BE_TestReadProjectFromEnv=1")
	err := cmd.Run()
	if err != nil {
		t.Fatalf("process ran with err %v, want success", err)
	}
}

func TestReadProjectFromFlag(t *testing.T) {
	executeCommand(rootCmd, "version", "--project-name", "from-flag")

	require.Equal(t, Config.Profile.ProfileName, "from-flag")
}

func TestReadProjectFlagHasPrecedence(t *testing.T) {
	os.Setenv("STRIPE_PROJECT_NAME", "from-env")
	defer os.Unsetenv("STRIPE_PROJECT_NAME")

	executeCommand(rootCmd, "version", "--project-name", "from-flag")

	require.Equal(t, Config.Profile.ProfileName, "from-flag")
}
