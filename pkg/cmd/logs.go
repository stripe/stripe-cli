package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/stripe/stripe-cli/pkg/validators"
)

// LogsCmd is a wrapper for the base logs command
type LogsCmd struct {
	Cmd *cobra.Command
}

func newLogsCmd() *LogsCmd {
	logsCmd := &LogsCmd{}

	logsCmd.Cmd = &cobra.Command{
		Use:   "logs",
		Args:  validators.NoArgs,
		Short: "Top-level package for logs commands with Stripe.",
		Long: fmt.Sprintf(`
The logs package contains the sub-command 'tail', which allows you to tail your API request logs
in real-time from Stripe.

Invokable via:
    $ stripe logs tail
`),
	}

	logsCmd.Cmd.AddCommand(NewLogsTailCmd().Cmd)

	return logsCmd
}
