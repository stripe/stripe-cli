package cmd

import (
	"os"

	"github.com/spf13/cobra"
	"golang.org/x/term"

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
		Use:   "login",
		Args:  validators.NoArgs,
		Short: "Login to your Stripe account",
		Long: `Login to your Stripe account to set up the CLI.

By default (when stdin is a terminal), this opens a browser-based OAuth flow: it
prints a pairing code, launches your browser to the Stripe Dashboard, and waits for
you to approve the request before saving your credentials.

Use --interactive when a browser is unavailable:

  --interactive
      Prompts you to paste an API key directly. Useful for SSH sessions or
      CI environments with a human operator.

For agents and scripts, use the two-step non-interactive flow:

  --non-interactive
      Prints a JSON object containing a browser_url, a verification_code to
      confirm the pairing, and a next_step command, then exits immediately.
      Activates automatically when stdin is not a terminal.
      Immediately run the next_step command from the JSON output to poll
      while the user approves in the browser; it blocks until authentication
      completes.

  --complete <poll-url>
      Polls the given URL (from the next_step of a prior --non-interactive run)
      until the user approves in the browser, then saves credentials.`,
		Example: `# Standard browser login (default for TTY users)
  stripe login

  # Paste an API key instead of using a browser
  stripe login --interactive

  # Non-interactive: get links and exit (useful for agents/scripts)
  stripe login --non-interactive

  # Two-step agent-driven flow:
  #   Step 1 – get the browser URL, verification code, and poll URL
  stripe login --non-interactive
  #   Step 2 – after the user approves in the browser, complete login
  stripe login --complete 'https://dashboard.stripe.com/stripecli/auth/...'`,
		Annotations: map[string]string{
			AIAgentHelpAnnotationKey: "  Prefer setting STRIPE_API_KEY or using `--api-key` over `stripe login` for non-interactive use.\n" +
				"  If authentication is required, run `stripe login` — in agent contexts it automatically outputs\n" +
				"  a browser URL and a `next_step` command to complete login with user action.",
		},
		RunE: lc.runLoginCmd,
	}
	lc.cmd.Flags().BoolVarP(&lc.interactive, "interactive", "i", false, "Run interactive configuration mode if you cannot open a browser")
	lc.cmd.Flags().BoolVar(&lc.nonInteractive, "non-interactive", false, "Print login URL and verification code as JSON and exit; immediately run the next_step command from the output to poll while the user approves in the browser")
	lc.cmd.Flags().StringVar(&lc.completeURL, "complete", "", "Complete a browser login by polling the given URL (from 'stripe login --non-interactive')")

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
