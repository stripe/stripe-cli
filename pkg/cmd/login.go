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
	list             bool
	dashboardBaseURL string
	switchProfile    string
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
	lc.cmd.Flags().BoolVarP(&lc.list, "list", "l", false, "List all available profiles")
	lc.cmd.Flags().StringVar(&lc.switchProfile, "switch", "", "Switch to a different profile")

	// Hidden configuration flags, useful for dev/debugging
	lc.cmd.Flags().StringVar(&lc.dashboardBaseURL, "dashboard-base", stripe.DefaultDashboardBaseURL, "Sets the dashboard base URL")
	lc.cmd.Flags().MarkHidden("dashboard-base") // #nosec G104

	return lc
}

func (lc *loginCmd) runLoginCmd(cmd *cobra.Command, args []string) error {
	if err := stripe.ValidateDashboardBaseURL(lc.dashboardBaseURL); err != nil {
		return err
	}
	if lc.list {
		return Config.ListProfiles()
	}

	if lc.interactive {
		return login.InteractiveLogin(cmd.Context(), &Config)
	}

	if lc.switchProfile != "" {
		return Config.SwitchProfile(lc.switchProfile)
	}

	return login.Login(cmd.Context(), lc.dashboardBaseURL, &Config)
}
