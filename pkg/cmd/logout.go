package cmd

import (
	"github.com/spf13/cobra"

	"github.com/stripe/stripe-cli/pkg/login"
	"github.com/stripe/stripe-cli/pkg/logout"
	"github.com/stripe/stripe-cli/pkg/validators"
)

type logoutCmd struct {
	cmd           *cobra.Command
	all           bool
	accessBaseURL string
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
	lc.cmd.Flags().StringVar(&lc.accessBaseURL, "access-base", login.DefaultAccessBaseURL, "Sets the access base URL")
	lc.cmd.Flags().MarkHidden("access-base") //nolint:errcheck

	return lc
}

func (lc *logoutCmd) runLogoutCmd(cmd *cobra.Command, args []string) error {
	if lc.all {
		return logout.All(cmd.Context(), lc.accessBaseURL, &Config)
	}
	return logout.Logout(cmd.Context(), lc.accessBaseURL, &Config)
}
