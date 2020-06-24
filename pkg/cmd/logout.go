package cmd

import (
	"github.com/spf13/cobra"

	"github.com/stripe/stripe-cli/pkg/logout"
	"github.com/stripe/stripe-cli/pkg/validators"
)

type logoutCmd struct {
	cmd *cobra.Command
	all bool
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

	lc.cmd.Flags().BoolVarP(&lc.all, "all", "a", false, "Clear credentials for all projects you are currently logged into.")

	return lc
}

func (lc *logoutCmd) runLogoutCmd(cmd *cobra.Command, args []string) error {
	if lc.all {
		return logout.All(&Config)
	}

	return logout.Logout(&Config)
}
