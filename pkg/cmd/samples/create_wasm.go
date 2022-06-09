//go:build wasm
// +build wasm

package samples

import (
	"github.com/spf13/cobra"

	"github.com/stripe/stripe-cli/pkg/config"
	"github.com/stripe/stripe-cli/pkg/validators"
)

// CreateCmd wraps the `create` command for samples which generates a new
// project
type CreateCmd struct {
	cfg *config.Config
	Cmd *cobra.Command

	forceRefresh bool
}

// NewCreateCmd creates and returns a create command for samples
func NewCreateCmd(config *config.Config) *CreateCmd {
	createCmd := &CreateCmd{
		cfg:          config,
		forceRefresh: false,
	}
	createCmd.Cmd = &cobra.Command{
		Use:   "create <sample> [destination]",
		Args:  validators.MaximumNArgs(2),
		Short: "Setup and bootstrap a Stripe Sample",
		Long: `The create command will locally clone a sample, let you select which integration,
client, and server you want to run. It then automatically bootstraps the
local configuration to let you get started faster.`,
		Example: `stripe samples create accept-a-payment
  stripe samples create react-elements-card-payment my-payments-form`,
	}

	createCmd.Cmd.Flags().BoolVar(&createCmd.forceRefresh, "force-refresh", false, "Forcefully refresh the local samples cache")

	return createCmd
}
