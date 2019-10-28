package cmd

import (
	"github.com/spf13/cobra"

	"github.com/stripe/stripe-cli/pkg/cmd/samples"
)

type samplesCmd struct {
	cmd *cobra.Command
}

func newSamplesCmd() *samplesCmd {
	samplesCmd := &samplesCmd{
		cmd: &cobra.Command{
			Use:   "samples",
			Short: `Sample integrations built by Stripe`,
			Long:  ``,
		},
	}

	samplesCmd.cmd.AddCommand(samples.NewCreateCmd(&Config).Cmd)
	samplesCmd.cmd.AddCommand(samples.NewListCmd().Cmd)

	return samplesCmd
}
