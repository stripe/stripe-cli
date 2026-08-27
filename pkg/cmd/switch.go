package cmd

import (
	"github.com/spf13/cobra"

	"github.com/stripe/stripe-cli/pkg/login"
	"github.com/stripe/stripe-cli/pkg/validators"
)

type switchCmd struct {
	cmd *cobra.Command
}

type switchContextCmd struct {
	cmd           *cobra.Command
	livemode      bool
	accessBaseURL string
}

func newSwitchCmd() *switchCmd {
	sc := &switchCmd{}
	sc.cmd = &cobra.Command{
		Use:   "switch",
		Short: "Switch active context or profile",
	}

	ctxCmd := &switchContextCmd{}
	ctxCmd.cmd = &cobra.Command{
		Use:   "context [account_id]",
		Args:  validators.MaximumNArgs(1),
		Short: "Switch to a different authorized account context",
		Long: `Switch to a different authorized account context.

Without an argument, shows an interactive list of your authorized accounts and
modes. Navigate with ↑↓, confirm with enter, or cancel with esc.

With an account ID, switches directly to that account. Add --live to switch to live mode.`,
		Example: `stripe switch context
  stripe switch context acct_1234
  stripe switch context acct_1234 --live`,
		RunE: ctxCmd.run,
	}
	ctxCmd.cmd.Flags().BoolVar(&ctxCmd.livemode, "live", false, "Select live mode for the given account")
	ctxCmd.cmd.Flags().StringVar(&ctxCmd.accessBaseURL, "access-base", login.DefaultAccessBaseURL, "Sets the access base URL")
	ctxCmd.cmd.Flags().MarkHidden("access-base") //nolint:errcheck

	sc.cmd.AddCommand(ctxCmd.cmd)
	return sc
}

func (sc *switchContextCmd) run(cmd *cobra.Command, args []string) error {
	if err := login.ValidateAccessBaseURL(sc.accessBaseURL); err != nil {
		return err
	}
	accountID := ""
	if len(args) > 0 {
		accountID = args[0]
	}
	return login.SwitchContext(cmd.Context(), sc.accessBaseURL, &Config, accountID, sc.livemode)
}
