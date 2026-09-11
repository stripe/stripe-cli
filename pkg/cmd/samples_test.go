package cmd

import (
	"strings"
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/require"

	"github.com/stripe/stripe-cli/pkg/cmdutil"
)

func newRootWithDeprecatedCmds() *cobra.Command {
	// Mirror the real root's silencing so the shims' output is what a user
	// actually sees; without it cobra appends its own error and usage block.
	root := &cobra.Command{Use: "stripe", SilenceUsage: true, SilenceErrors: true}
	terminal := &cobra.Command{Use: "terminal"}
	terminal.AddCommand(newTerminalQuickstartCmd())
	root.AddCommand(newSamplesCmd(), newServeCmd(), terminal)
	return root
}

func TestDeprecatedCommands(t *testing.T) {
	tests := []struct {
		name     string
		args     []string
		expected string
	}{
		{
			name:     "samples",
			args:     []string{"samples", "create", "checkout", "--force", "--integration", "react"},
			expected: "The `stripe samples` command is no longer available in Stripe CLI v1.60.0 and later. To use it, install a version earlier than v1.60.0.\n",
		},
		{
			name:     "serve",
			args:     []string{"serve", ".", "--port", "8080"},
			expected: "The `stripe serve` command is no longer available in Stripe CLI v1.60.0 and later. To use it, install a version earlier than v1.60.0.\n",
		},
		{
			name:     "serve alias",
			args:     []string{"srv", "."},
			expected: "The `stripe serve` command is no longer available in Stripe CLI v1.60.0 and later. To use it, install a version earlier than v1.60.0.\n",
		},
		{
			name:     "terminal quickstart",
			args:     []string{"terminal", "quickstart", "--api-key", "example"},
			expected: "The `stripe terminal quickstart` command is no longer available in Stripe CLI v1.60.0 and later. To use it, install a version earlier than v1.60.0.\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			output, err := executeCommand(newRootWithDeprecatedCmds(), tt.args...)

			// Invoking a removed command fails, so anything that scripted it
			// sees a non-zero exit rather than a silent no-op.
			require.ErrorIs(t, err, errCommandRemoved)
			require.Equal(t, tt.expected, output)
		})
	}
}

func TestDeprecatedCommandHelp(t *testing.T) {
	tests := []struct {
		name     string
		args     []string
		expected string
	}{
		{
			name:     "samples",
			args:     []string{"help", "samples"},
			expected: "The `stripe samples` command is no longer available in Stripe CLI v1.60.0 and later. To use it, install a version earlier than v1.60.0.\n",
		},
		{
			name:     "samples help flag",
			args:     []string{"samples", "--help"},
			expected: "The `stripe samples` command is no longer available in Stripe CLI v1.60.0 and later. To use it, install a version earlier than v1.60.0.\n",
		},
		{
			name:     "samples help shorthand",
			args:     []string{"samples", "-h"},
			expected: "The `stripe samples` command is no longer available in Stripe CLI v1.60.0 and later. To use it, install a version earlier than v1.60.0.\n",
		},
		{
			name:     "serve",
			args:     []string{"help", "serve"},
			expected: "The `stripe serve` command is no longer available in Stripe CLI v1.60.0 and later. To use it, install a version earlier than v1.60.0.\n",
		},
		{
			name:     "serve help flag",
			args:     []string{"serve", "--help"},
			expected: "The `stripe serve` command is no longer available in Stripe CLI v1.60.0 and later. To use it, install a version earlier than v1.60.0.\n",
		},
		{
			name:     "terminal quickstart",
			args:     []string{"help", "terminal", "quickstart"},
			expected: "The `stripe terminal quickstart` command is no longer available in Stripe CLI v1.60.0 and later. To use it, install a version earlier than v1.60.0.\n",
		},
		{
			name:     "terminal quickstart help flag",
			args:     []string{"terminal", "quickstart", "--help"},
			expected: "The `stripe terminal quickstart` command is no longer available in Stripe CLI v1.60.0 and later. To use it, install a version earlier than v1.60.0.\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			output, err := executeCommand(newRootWithDeprecatedCmds(), tt.args...)

			// Asking for help is not a failed invocation.
			require.NoError(t, err)
			require.Equal(t, tt.expected, output)
		})
	}
}

func TestDeprecatedCommandsHidden(t *testing.T) {
	root := newRootWithDeprecatedCmds()

	rootHelp, err := executeCommand(root, "--help")
	require.NoError(t, err)
	require.NotContains(t, rootHelp, "samples")
	require.NotContains(t, rootHelp, "serve")

	terminalHelp, err := executeCommand(root, "terminal", "--help")
	require.NoError(t, err)
	require.NotContains(t, terminalHelp, "quickstart")
}

// The tests above build their own root, so they can't catch the shims falling
// out of the real command tree. In particular quickstart is attached by looking
// up `terminal` in root.go, which silently attaches nothing if that lookup ever
// stops resolving.
func TestDeprecatedCommandsRegisteredOnRoot(t *testing.T) {
	paths := [][]string{
		{"samples"},
		{"serve"},
		{"srv"},
		{"terminal", "quickstart"},
	}

	for _, path := range paths {
		t.Run(strings.Join(path, " "), func(t *testing.T) {
			cmd, ok := cmdutil.FindSubCmd(rootCmd, path...)

			require.True(t, ok, "expected `stripe %s` to be registered", strings.Join(path, " "))
			require.True(t, cmd.Hidden, "expected `stripe %s` to stay out of help", strings.Join(path, " "))
			require.NotNil(t, cmd.RunE)
		})
	}
}
