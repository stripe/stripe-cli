package resource

import (
	"github.com/spf13/cobra"

	"github.com/stripe/stripe-cli/pkg/checkout"
	"github.com/stripe/stripe-cli/pkg/config"
	"github.com/stripe/stripe-cli/pkg/validators"
)

// CheckoutRunCmd creates a run command for checkout
type CheckoutRunCmd struct {
	Cmd *cobra.Command
	Cfg *config.Config

	port string
}

// NewCheckoutRunCmd returns a new CheckoutRunCmd.
func NewCheckoutRunCmd(parentCmd *cobra.Command, cfg *config.Config) *cobra.Command {
	cc := CheckoutRunCmd{
		Cfg: cfg,
	}

	cc.Cmd = &cobra.Command{
		Use:   "run",
		Args:  validators.NoArgs,
		Short: "Run checkout session",
		Long:  "Run checkout session",
		RunE: func(cmd *cobra.Command, args []string) error {
			server := checkout.Server{
				Cfg:  cc.Cfg,
				Port: cc.port,
			}

			return server.Run()
		},
	}

	cc.Cmd.Flags().StringVar(&cc.port, "port", "4242", "Provide a custom port to serve content from.")

	parentCmd.AddCommand(cc.Cmd)

	return cc.Cmd
}
