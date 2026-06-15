package cmd

import (
	"os"

	"github.com/spf13/cobra"
	"golang.org/x/term"

	"github.com/stripe/stripe-cli/pkg/i18n"
	"github.com/stripe/stripe-cli/pkg/login"
	"github.com/stripe/stripe-cli/pkg/stripe"
	"github.com/stripe/stripe-cli/pkg/validators"
)

type loginCmd struct {
	cmd              *cobra.Command
	interactive      bool
	dashboardBaseURL string
	nonInteractive   bool
	completeURL      string
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
		Use:     "login",
		Args:    validators.NoArgs,
		Short:   i18n.T("login.short"),
		Long:    i18n.T("login.long"),
		Example: i18n.T("login.example"),
		Annotations: map[string]string{
			AIAgentHelpAnnotationKey: "  Prefer setting STRIPE_API_KEY or using `--api-key` over `stripe login` for non-interactive use.\n" +
				"  If authentication is required, run `stripe login` — in agent contexts it automatically outputs\n" +
				"  a browser URL and a `next_step` command to complete login with user action.",
		},
		RunE: lc.runLoginCmd,
	}
	lc.cmd.Flags().BoolVarP(&lc.interactive, "interactive", "i", false, i18n.T("login.flags.interactive"))
	lc.cmd.Flags().BoolVar(&lc.nonInteractive, "non-interactive", false, i18n.T("login.flags.non_interactive"))
	lc.cmd.Flags().StringVar(&lc.completeURL, "complete", "", i18n.T("login.flags.complete"))

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
		Short:   i18n.T("login.list.short"),
		Example: i18n.T("login.list.example"),
		RunE:    listCmd.listLoggedInAccountsCmd,
	}

	lc.cmd.AddCommand(listCmd.cmd)

	switchCmd := &loginSwitchCmd{}
	switchCmd.cmd = &cobra.Command{
		Use:     "switch",
		Args:    validators.ExactArgs(1),
		Short:   i18n.T("login.switch.short"),
		Example: i18n.T("login.switch.example"),
		RunE:    switchCmd.switchLoggedInAccountCmd,
	}

	lc.cmd.AddCommand(switchCmd.cmd)
	return lc
}

func (lc *loginCmd) runLoginCmd(cmd *cobra.Command, args []string) error {
	if err := stripe.ValidateDashboardBaseURL(lc.dashboardBaseURL); err != nil {
		return err
	}

	if lc.completeURL != "" {
		return login.PollForLogin(cmd.Context(), lc.completeURL, &Config)
	}

	if lc.nonInteractive || !shouldAutoLogin(os.Getenv, term.IsTerminal(int(os.Stdin.Fd()))) {
		return login.InitiateLogin(cmd.Context(), lc.dashboardBaseURL, &Config)
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
