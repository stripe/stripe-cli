package cmd

import (
	"os"

	"github.com/spf13/cobra"

	"github.com/stripe/stripe-cli/pkg/logout"
	"github.com/stripe/stripe-cli/pkg/validators"
)

type logoutCmd struct {
	cmd *cobra.Command
}

func newLogoutCmd() *logoutCmd {
	lc := &logoutCmd{}

	lc.cmd = &cobra.Command{
		Use:   "logout",
		Args:  validators.NoArgs,
		Short: "Logout of your Stripe account",
		Long:  `Logout of your Stripe account from the CLI`,
		RunE:  lc.runLogoutCmd,
	}

	return lc
}

func (lc *logoutCmd) runLogoutCmd(cmd *cobra.Command, args []string) error {
	return logout.Logout(&Config, os.Stdin)
}
