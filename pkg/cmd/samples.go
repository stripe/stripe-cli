package cmd

import (
	"github.com/spf13/cobra"
	"github.com/stripe/stripe-cli/pkg/cmd/samples"
)

type SamplesCmd struct {
	Cmd *cobra.Command
}

func newSamplesCmd() *SamplesCmd {
	samplesCmd := &SamplesCmd{
		Cmd: &cobra.Command{
			// TODO: fixtures subcommand
			Use:   "samples",
			Short: `Sample integrations built by Stripe`,
			Long:  ``,
		},
	}

	samplesCmd.Cmd.AddCommand(samples.NewCreateCmd(&Config).Cmd)
	samplesCmd.Cmd.AddCommand(samples.NewListCmd().Cmd)

	return samplesCmd
}
