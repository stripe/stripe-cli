//go:build wasm
// +build wasm

package resource

import (
	"github.com/spf13/cobra"

	"github.com/stripe/stripe-cli/pkg/config"
	"github.com/stripe/stripe-cli/pkg/validators"
)

// QuickstartCmd starts a prompt flow for connecting a Terminal reader to their Stripe account
type QuickstartCmd struct {
	cfg *config.Config
	cmd *cobra.Command
}

// NewQuickstartCmd returns a new terminal quickstart command
func NewQuickstartCmd(parentCmd *cobra.Command, config *config.Config) {
	quickstartCmd := &QuickstartCmd{
		cfg: config,
	}

	quickstartCmd.cmd = &cobra.Command{
		Use:     "quickstart",
		Args:    validators.MaximumNArgs(0),
		Short:   "Set up a Terminal reader and take a test payment",
		Example: `stripe terminal quickstart --api-key sk_123`,
	}

	parentCmd.AddCommand(quickstartCmd.cmd)
}
