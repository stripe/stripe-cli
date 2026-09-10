package cmd

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"github.com/stripe/stripe-cli/pkg/errorcategory"
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
	format        string
}

// switchOutput is the stable JSON schema for `stripe switch --format json`.
type switchOutput struct {
	AccountID   string `json:"account_id"`
	DisplayName string `json:"display_name"`
	Mode        string `json:"mode"`
}

func newSwitchCmd() *switchCmd {
	sc := &switchCmd{}
	ctxCmd := &switchContextCmd{}

	sc.cmd = &cobra.Command{
		Use:   "switch [account_id]",
		Args:  validators.MaximumNArgs(1),
		Short: "Switch to a different authorized account context",
		Long: `Switch to a different authorized account context.

Without an argument, shows an interactive list of your authorized accounts and
modes. Navigate with ↑↓, confirm with enter, or cancel with esc.

With an account ID, switches directly to that account. Add --live to switch to live mode.`,
		Example: `stripe switch
  stripe switch acct_1234
  stripe switch acct_1234 --live
  stripe switch acct_1234 --format json`,
		RunE: ctxCmd.run,
	}
	ctxCmd.cmd = sc.cmd
	addSwitchContextFlags(sc.cmd, ctxCmd)

	// Kept as a hidden alias for backwards compatibility; 'stripe switch' is
	// now the preferred way to switch context.
	legacyCtxCmd := &cobra.Command{
		Use:    "context [account_id]",
		Args:   validators.MaximumNArgs(1),
		Short:  "Switch to a different authorized account context",
		Hidden: true,
		RunE:   ctxCmd.run,
	}
	addSwitchContextFlags(legacyCtxCmd, ctxCmd)

	sc.cmd.AddCommand(legacyCtxCmd)
	return sc
}

func addSwitchContextFlags(cmd *cobra.Command, ctxCmd *switchContextCmd) {
	cmd.Flags().BoolVar(&ctxCmd.livemode, "live", false, "Select live mode for the given account")
	cmd.Flags().StringVar(&ctxCmd.format, "format", "", "Output format: 'json' for a stable JSON schema (suitable for scripting)")
	cmd.Flags().StringVar(&ctxCmd.accessBaseURL, "access-base", login.DefaultAccessBaseURL, "Sets the access base URL")
	cmd.Flags().MarkHidden("access-base") //nolint:errcheck
}

func (sc *switchContextCmd) run(cmd *cobra.Command, args []string) error {
	if err := login.ValidateAccessBaseURL(sc.accessBaseURL); err != nil {
		return err
	}
	accountID := ""
	if len(args) > 0 {
		accountID = args[0]
	}
	if accountID == "" && strings.EqualFold(sc.format, "json") {
		return errorcategory.UserInputErrorf("--format json requires an account_id argument; the interactive selector isn't scriptable")
	}
	result, err := login.SwitchContext(cmd.Context(), sc.accessBaseURL, &Config, accountID, sc.livemode)
	if err != nil {
		return err
	}
	if result == nil {
		return nil
	}

	if strings.EqualFold(sc.format, "json") {
		out := switchOutput{
			AccountID:   result.Account.ID,
			DisplayName: result.Account.Name,
			Mode:        result.Mode,
		}
		b, err := json.MarshalIndent(out, "", "  ")
		if err != nil {
			return err
		}
		fmt.Fprintln(cmd.OutOrStdout(), string(b))
		return nil
	}

	fmt.Printf("Active context: %s · %s (%s)\n", result.Account.Name, result.DisplayMode(), result.Account.ID)
	return nil
}
