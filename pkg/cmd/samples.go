package cmd

import (
	"github.com/spf13/cobra"

	"github.com/stripe/stripe-cli/pkg/cmd/samples"
)

type samplesCmd struct {
	Cmd *cobra.Command
}

func newSamplesCmd() *cobra.Command {
	samplesCmd := &samplesCmd{
		Cmd: &cobra.Command{
			// TODO: fixtures subcommand
			Use:   "samples",
			Short: `Sample integrations built by Stripe`,
			Long:  ``,
		},
	}

	samplesCmd.Cmd.AddCommand(samples.NewCreateCmd(&Config))
	samplesCmd.Cmd.AddCommand(samples.NewListCmd())

	return samplesCmd.Cmd
}
