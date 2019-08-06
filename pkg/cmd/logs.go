package cmd

import (
	"github.com/spf13/cobra"

	logs "github.com/stripe/stripe-cli/pkg/cmd/logs"
	"github.com/stripe/stripe-cli/pkg/config"
	"github.com/stripe/stripe-cli/pkg/validators"
)

// LogsCmd is a wrapper for the base logs command
type LogsCmd struct {
	Cmd *cobra.Command
	cfg *config.Config
}

func newLogsCmd(config *config.Config) *LogsCmd {
	logsCmd := &LogsCmd{
		cfg: config,
	}

	logsCmd.Cmd = &cobra.Command{
		Use:   "logs",
		Args:  validators.NoArgs,
		Short: "Top-level package for logs commands with Stripe.",
		Long: `
The logs package contains the sub-command 'tail', which allows you to tail your API request logs
in real-time from Stripe.

Invokable via:
    $ stripe logs tail
`,
	}

	logsCmd.Cmd.AddCommand(logs.NewLogsTailCmd(logsCmd.cfg).Cmd)

	return logsCmd
}
