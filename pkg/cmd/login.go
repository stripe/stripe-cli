package cmd

import (
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"golang.org/x/term"

	"github.com/stripe/stripe-cli/pkg/config"
	"github.com/stripe/stripe-cli/pkg/errorcategory"
	"github.com/stripe/stripe-cli/pkg/login"
	"github.com/stripe/stripe-cli/pkg/stripe"
	"github.com/stripe/stripe-cli/pkg/useragent"
	"github.com/stripe/stripe-cli/pkg/validators"
)

// revokeToken and initiateLogin are package variables so tests can stub out
// the network calls made by runLoginCmd.
var (
	revokeToken   = login.RevokeToken
	initiateLogin = login.InitiateLogin
)

type loginCmd struct {
	cmd              *cobra.Command
	interactive      bool
	dashboardBaseURL string
	accessBaseURL    string
	nonInteractive   bool
	completeURL      string
	completeDevice   bool
	newSession       bool
}

type loginListCmd struct {
	cmd           *cobra.Command
	accessBaseURL string
}

type loginSwitchCmd struct {
	cmd           *cobra.Command
	livemode      bool
	accessBaseURL string
}

func newLoginCmd() *loginCmd {
	lc := &loginCmd{}

	lc.cmd = &cobra.Command{
		Use:   "login",
		Args:  validators.NoArgs,
		Short: "Log in to your Stripe account",
		Long: `Log in to your Stripe account to set up the CLI.

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
			AIAgentHelpAnnotationKey: "  If you do not have an account, run `stripe sandbox create` instead (provisions a claimable sandbox without a browser).\n" +
				"  Prefer setting STRIPE_API_KEY or using `--api-key` over `stripe login` for non-interactive use.\n" +
				"  If authentication is required, run `stripe login` — in agent contexts it automatically outputs\n" +
				"  a browser URL and a `next_step` command to complete login with user action.",
		},
		RunE: lc.runLoginCmd,
	}
	lc.cmd.Flags().BoolVarP(&lc.interactive, "interactive", "i", false, "Run interactive configuration mode if you cannot open a browser")
	lc.cmd.Flags().BoolVar(&lc.nonInteractive, "non-interactive", false, "Print login URL and verification code as JSON and exit; immediately run the next_step command from the output to poll while the user approves in the browser")
	lc.cmd.Flags().StringVar(&lc.completeURL, "complete", "", "Complete a browser login by polling the given URL (from 'stripe login --non-interactive')")
	lc.cmd.Flags().BoolVar(&lc.completeDevice, "complete-device", false, "Complete an OAuth device authorization started by 'stripe login --non-interactive'")
	lc.cmd.Flags().MarkHidden("complete-device") // #nosec G104

	// TODO: a flag to replace existing account?
	// TODO: what happens to if already logged into that account? - profile name should be the account id
	// TODO: what happens when we log out; do we pick a new account, or give the user a choice, or just live
	// in a logged out state but with credentials saved?

	// Hidden configuration flags, useful for dev/debugging
	lc.cmd.Flags().StringVar(&lc.dashboardBaseURL, "dashboard-base", stripe.DefaultDashboardBaseURL, "Sets the dashboard base URL")
	lc.cmd.Flags().MarkHidden("dashboard-base") // #nosec G104
	lc.cmd.Flags().StringVar(&lc.accessBaseURL, "access-base", login.DefaultAccessBaseURL, "Sets the access base URL")
	lc.cmd.Flags().MarkHidden("access-base") // #nosec G104
	lc.cmd.Flags().BoolVar(&lc.newSession, "new-session", false, "Force a new login even if already authenticated")

	listCmd := &loginListCmd{}
	listCmd.cmd = &cobra.Command{
		Use:     "list",
		Args:    validators.MaximumNArgs(0),
		Short:   "Lists all available logged-in accounts",
		Example: `stripe login list`,
		RunE:    listCmd.listLoggedInAccountsCmd,
	}
	listCmd.cmd.Flags().StringVar(&listCmd.accessBaseURL, "access-base", login.DefaultAccessBaseURL, "Sets the access base URL")
	listCmd.cmd.Flags().MarkHidden("access-base") // #nosec G104

	lc.cmd.AddCommand(listCmd.cmd)

	switchCmd := &loginSwitchCmd{}
	switchCmd.cmd = &cobra.Command{
		Use:     "switch [account_id]",
		Args:    validators.MaximumNArgs(1),
		Short:   "Alias for 'stripe switch context'",
		Example: `stripe login switch\n  stripe login switch acct_1234\n  stripe login switch acct_1234 --live`,
		RunE:    switchCmd.switchLoggedInAccountCmd,
	}
	switchCmd.cmd.Flags().BoolVar(&switchCmd.livemode, "live", false, "Select live mode for the given account")
	switchCmd.cmd.Flags().StringVar(&switchCmd.accessBaseURL, "access-base", login.DefaultAccessBaseURL, "Sets the access base URL")
	switchCmd.cmd.Flags().MarkHidden("access-base") // #nosec G104

	lc.cmd.AddCommand(switchCmd.cmd)
	return lc
}

func (lc *loginCmd) runLoginCmd(cmd *cobra.Command, args []string) error {
	if err := stripe.ValidateDashboardBaseURL(lc.dashboardBaseURL); err != nil {
		return err
	}
	if err := login.ValidateAccessBaseURL(lc.accessBaseURL); err != nil {
		return err
	}

	if lc.completeDevice {
		return login.PollPendingDeviceAuth(cmd.Context(), &Config)
	}

	if lc.completeURL != "" {
		return login.PollForLogin(cmd.Context(), lc.completeURL, &Config)
	}

	uat, _ := Config.Profile.GetUAT()
	if !lc.newSession {
		if strings.HasPrefix(uat, "oak_") {
			identity := Config.Profile.GetDisplayName()
			if identity == "" {
				if ac, _ := config.GetActiveContext(); ac != nil {
					identity = ac.AccountID
				}
			}
			if identity != "" {
				fmt.Fprintf(cmd.OutOrStdout(), "You're already logged in as %s.\n", identity)
			} else {
				fmt.Fprintln(cmd.OutOrStdout(), "You're already logged in.")
			}
			fmt.Fprintln(cmd.OutOrStdout(), "Run 'stripe reauth' to change permissions or authorize access to additional accounts or sandboxes.")
			fmt.Fprintln(cmd.OutOrStdout(), "To log in as a different user, run: stripe login --new-session")
			return nil
		}
	} else if strings.HasPrefix(uat, "oak_") {
		// Revoke the previous OAuth session before starting a new one, same as `stripe logout`.
		if err := revokeToken(cmd.Context(), lc.accessBaseURL); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: token revocation failed: %s\n", err)
		}
	}

	if lc.nonInteractive || !shouldAutoLogin(os.Getenv, term.IsTerminal(int(os.Stdin.Fd()))) {
		if useragent.DetectAIAgent(os.Getenv) != "" {
			fmt.Fprintln(os.Stderr, "If you do not have an account, run `stripe sandbox create` instead (provisions a claimable sandbox without a browser).")
		}
		return initiateLogin(cmd.Context(), lc.dashboardBaseURL, lc.accessBaseURL, &Config)
	}

	if lc.interactive {
		return login.InteractiveLogin(cmd.Context(), &Config)
	}

	return login.Login(cmd.Context(), lc.dashboardBaseURL, lc.accessBaseURL, &Config)
}

// TODO: we should support bash completion for account names
func (lc *loginListCmd) listLoggedInAccountsCmd(cmd *cobra.Command, args []string) error {
	uat, _ := Config.Profile.GetUAT()
	if strings.HasPrefix(uat, "oak_") {
		if err := login.ValidateAccessBaseURL(lc.accessBaseURL); err != nil {
			return err
		}
		return login.PrintAuthorizedContexts(cmd.Context(), lc.accessBaseURL, uat)
	}
	return Config.ListProfiles()
}

func (lc *loginSwitchCmd) switchLoggedInAccountCmd(cmd *cobra.Command, args []string) error {
	uat, _ := Config.Profile.GetUAT()
	if strings.HasPrefix(uat, "oak_") {
		if err := login.ValidateAccessBaseURL(lc.accessBaseURL); err != nil {
			return err
		}
		accountID := ""
		if len(args) > 0 {
			accountID = args[0]
		}
		return login.SwitchContext(cmd.Context(), lc.accessBaseURL, &Config, accountID, lc.livemode)
	}
	if len(args) == 0 {
		return errorcategory.Errorf(errorcategory.UserInput, "account name required")
	}
	return Config.SwitchProfile(args[0])
}
