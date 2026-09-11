package cmd

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/stripe/stripe-cli/pkg/errorcategory"
)

// errCommandRemoved is returned by the shims for commands removed in v1.60.0.
// root.go recognizes this sentinel to suppress duplicate error output and error
// reporting while still exiting non-zero, so callers that scripted a removed
// command see a failure rather than a silent no-op.
var errCommandRemoved = errorcategory.New(errorcategory.UserInput, "command removed")

func deprecatedCommandMessage(command string) string {
	return fmt.Sprintf("The `%s` command is no longer available in Stripe CLI v1.60.0 and later. To use it, install a version earlier than v1.60.0.", command)
}

func newDeprecatedCommand(use, command string) *cobra.Command {
	cmd := &cobra.Command{
		Use:                use,
		Hidden:             true,
		DisableFlagParsing: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			fmt.Fprintln(cmd.ErrOrStderr(), deprecatedCommandMessage(command))
			// Flag parsing is disabled so the removed commands' old flags don't
			// produce an unknown-flag error, which also routes -h and --help
			// here instead of to cobra's help handling. Asking for help is not
			// a failed invocation, so exit zero, matching `stripe help <cmd>`.
			if isHelpRequest(args) {
				return nil
			}
			return errCommandRemoved
		},
	}
	cmd.SetHelpFunc(func(cmd *cobra.Command, args []string) {
		fmt.Fprintln(cmd.ErrOrStderr(), deprecatedCommandMessage(command))
	})
	return cmd
}

func newSamplesCmd() *cobra.Command {
	return newDeprecatedCommand("samples", "stripe samples")
}

func newServeCmd() *cobra.Command {
	cmd := newDeprecatedCommand("serve", "stripe serve")
	cmd.Aliases = []string{"srv"}
	return cmd
}

func newTerminalQuickstartCmd() *cobra.Command {
	return newDeprecatedCommand("quickstart", "stripe terminal quickstart")
}
