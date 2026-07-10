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
	"github.com/spf13/viper"
	"github.com/stretchr/testify/require"

	"github.com/stripe/stripe-cli/pkg/cmd/resource"
	"github.com/stripe/stripe-cli/pkg/config"
)

func executeCommand(root *cobra.Command, args ...string) (output string, err error) {
	_, output, err = executeCommandC(root, args...)
	return output, err
}

func executeCommandC(root *cobra.Command, args ...string) (c *cobra.Command, output string, err error) {
	buf := new(bytes.Buffer)
	root.SetOut(buf)
	root.SetErr(buf)
	root.SetArgs(args)

	c, err = root.ExecuteC()

	// Resets args for the next test run to avoid arguments for flags being carried over
	root.SetArgs([]string{})

	return c, buf.String(), err
}

func TestGetPathNoXDG(t *testing.T) {
	actual := Config.GetConfigFolder("")
	expected, err := homedir.Dir()
	expected += filepath.Join("/", ".config", "stripe")

	require.NoError(t, err)
	require.Equal(t, expected, actual)
}

func TestGetPathXDG(t *testing.T) {
	actual := Config.GetConfigFolder("/some/xdg/path")
	expected := filepath.Join("/", "some", "xdg", "path", "stripe")

	require.Equal(t, expected, actual)
}

func TestHelpFlag(t *testing.T) {
	Execute(context.Background())

	output, err := executeCommand(rootCmd, "--help")

	require.Contains(t, output, "Stripe commands:")
	require.NoError(t, err)
}

func TestSandboxVisibleInHelp(t *testing.T) {
	Execute(context.Background())

	output, err := executeCommand(rootCmd, "--help")
	require.NoError(t, err)
	require.Contains(t, output, "sandbox")
}

func TestExampleCommands(t *testing.T) {
	{
		_, err := executeCommand(rootCmd, "foo")
		require.Equal(t, "unknown command \"foo\" for \"stripe\"\n\nDid you mean this?\n\tcoop\n", err.Error())
	}
	{
		_, err := executeCommand(rootCmd, "listen", "foo")
		require.Equal(t, "`stripe listen` does not take any positional arguments. See `stripe listen --help` for supported flags and usage", err.Error())
	}
	{
		_, err := executeCommand(rootCmd, "post")
		require.Equal(t, "`stripe post` requires exactly 1 positional argument. See `stripe post --help` for supported flags and usage", err.Error())
	}
	{
		_, err := executeCommand(rootCmd, "samples", "create", "foo", "foo", "foo")
		require.Equal(t, "`stripe samples create` accepts at maximum 2 positional arguments. See `stripe samples create --help` for supported flags and usage", err.Error())
	}
}

func TestReadProjectDefault(t *testing.T) {
	executeCommand(rootCmd, "version")
	require.Equal(t, "default", Config.Profile.ProfileName)
}

func TestReadProjectFromEnv(t *testing.T) {
	// Run this test in a subprocess since side effects from other tests interfere with this
	if os.Getenv("BE_TestReadProjectFromEnv") == "1" {
		t.Setenv("STRIPE_PROJECT_NAME", "from-env")

		executeCommand(rootCmd, "version")

		require.Equal(t, "from-env", Config.Profile.ProfileName)
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

	require.Equal(t, "from-flag", Config.Profile.ProfileName)
}

func TestReadProjectFlagHasPrecedence(t *testing.T) {
	t.Setenv("STRIPE_PROJECT_NAME", "from-env")

	executeCommand(rootCmd, "version", "--project-name", "from-flag")

	require.Equal(t, "from-flag", Config.Profile.ProfileName)
}

func TestReBindKeysSkipsChangedFlags(t *testing.T) {
	// Reproduce the CI failure: viper.Reset() (called from GetMachineUUID → WriteConfigField)
	// clears the pflag binding. Without it, viper falls back to the env var value and
	// ReBindKeys overwrites an explicitly-passed --project-name flag.
	t.Setenv("STRIPE_PROJECT_NAME", "from-env")

	// Simulate viper.Reset() having been called — clears the pflag binding so
	// viper.GetString("project-name") returns the env var value, not the flag value.
	viper.Reset()
	viper.BindEnv("project-name", "STRIPE_PROJECT_NAME")
	t.Cleanup(func() {
		// Restore bindings so subsequent tests still work.
		viper.BindPFlag("project-name", rootCmd.PersistentFlags().Lookup("project-name"))
		viper.BindEnv("project-name", "STRIPE_PROJECT_NAME")
	})

	flag := rootCmd.PersistentFlags().Lookup("project-name")
	origValue := flag.Value.String()
	origChanged := flag.Changed
	defer func() {
		flag.Value.Set(origValue)
		flag.Changed = origChanged
	}()

	flag.Value.Set("from-flag")
	flag.Changed = true

	ReBindKeys()

	require.Equal(t, "from-flag", flag.Value.String())
}

func TestV2BillingOverrides(t *testing.T) {
	Execute(context.Background())

	output, err := executeCommand(rootCmd, "v2", "billing")

	require.Contains(t, output, "meter_event_stream")
	require.NoError(t, err)
}

func TestDatabasesHiddenButDirectlyAddressable(t *testing.T) {
	root := &cobra.Command{
		Use:         "stripe",
		Annotations: map[string]string{"get": "http"},
	}
	root.SetUsageTemplate(getUsageTemplate())
	root.AddCommand(&cobra.Command{Use: "version", Short: "Get the version of the Stripe CLI", RunE: noop})
	err := resource.AddDatabasesCmd(root, &config.Config{})
	require.NoError(t, err)

	helpOutput, err := executeCommand(root, "--help")
	require.NoError(t, err)
	require.NotContains(t, helpOutput, "databases")

	output, err := executeCommand(root, "databases", "--help")
	require.NoError(t, err)
	require.Contains(t, output, "Manage StripeDB")
	require.Contains(t, output, "unstable preview APIs")
	require.Contains(t, output, "users")
}
