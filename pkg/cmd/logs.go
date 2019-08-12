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
		Short: "Interact with Stripe request logs",
		Long:  `The logs command allows you to interact with your API request logs from Stripe. The first supported feature is log tailing, which allows you to view your API request logs in real-time.`,
	}

	logsCmd.Cmd.AddCommand(logs.NewTailCmd(logsCmd.cfg).Cmd)

	return logsCmd
}
