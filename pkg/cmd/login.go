package cmd

import (
	"github.com/spf13/cobra"

	"github.com/stripe/stripe-cli/pkg/login"
	"github.com/stripe/stripe-cli/pkg/stripe"
	"github.com/stripe/stripe-cli/pkg/validators"
)

type loginCmd struct {
	cmd              *cobra.Command
	interactive      bool
	dashboardBaseURL string
}

type loginListCmd struct {
	cmd *cobra.Command
}

type loginSwitchCmd struct {
	cmd *cobra.Command
}

func newLoginCmd() *loginCmd {
	lc := &loginCmd{}

	lc.cmd = &cobra.Command{
		Use:   "login",
		Args:  validators.NoArgs,
		Short: "Login to your Stripe account",
		Long:  `Login to your Stripe account to setup the CLI`,
		RunE:  lc.runLoginCmd,
	}
	lc.cmd.Flags().BoolVarP(&lc.interactive, "interactive", "i", false, "Run interactive configuration mode if you cannot open a browser")

	// TODO: a flag to replace existing account?
	// TODO: what happens to if already logged into that account? - profile name should be the account id
	// TODO: what happens when we log out; do we pick a new account, or give the user a choice, or just live
	// in a logged out state but with credentials saved?

	// Hidden configuration flags, useful for dev/debugging
	lc.cmd.Flags().StringVar(&lc.dashboardBaseURL, "dashboard-base", stripe.DefaultDashboardBaseURL, "Sets the dashboard base URL")
	lc.cmd.Flags().MarkHidden("dashboard-base") // #nosec G104

	listCmd := &loginListCmd{}
	listCmd.cmd = &cobra.Command{
		Use:     "list",
		Args:    validators.MaximumNArgs(0),
		Short:   "Lists all available logged-in accounts",
		Example: `stripe login list`,
		RunE:    listCmd.listLoggedInAccountsCmd,
	}

	lc.cmd.AddCommand(listCmd.cmd)

	switchCmd := &loginSwitchCmd{}
	switchCmd.cmd = &cobra.Command{
		Use:     "switch",
		Args:    validators.ExactArgs(1),
		Short:   "Switch to a different logged-in account",
		Example: `stripe login switch <account_name>`,
		RunE:    switchCmd.switchLoggedInAccountCmd,
	}

	lc.cmd.AddCommand(switchCmd.cmd)
	return lc
}

func (lc *loginCmd) runLoginCmd(cmd *cobra.Command, args []string) error {
	if err := stripe.ValidateDashboardBaseURL(lc.dashboardBaseURL); err != nil {
		return err
	}
	if lc.interactive {
		return login.InteractiveLogin(cmd.Context(), &Config)
	}

	return login.Login(cmd.Context(), lc.dashboardBaseURL, &Config)
}

// TODO: we should support bash completion for account names
func (lc *loginListCmd) listLoggedInAccountsCmd(cmd *cobra.Command, args []string) error {
	return Config.ListProfiles()
}

func (lc *loginSwitchCmd) switchLoggedInAccountCmd(cmd *cobra.Command, args []string) error {
	return Config.SwitchProfile(args[0])
}
