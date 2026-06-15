package cmd

import (
	"github.com/spf13/cobra"

	"github.com/stripe/stripe-cli/pkg/i18n"
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
		Short: i18n.T("logout.short"),
		Long:  i18n.T("logout.long"),
		RunE:  lc.runLogoutCmd,
	}

	lc.cmd.Flags().BoolVarP(&lc.all, "all", "a", false, i18n.T("logout.flags.all"))

	return lc
}

func (lc *logoutCmd) runLogoutCmd(cmd *cobra.Command, args []string) error {
	if lc.all {
		return logout.All(&Config)
	}

	return logout.Logout(&Config)
}
