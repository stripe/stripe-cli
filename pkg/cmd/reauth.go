package cmd

import (
	"strings"

	"github.com/spf13/cobra"

	"github.com/stripe/stripe-cli/pkg/errorcategory"
	"github.com/stripe/stripe-cli/pkg/login"
	"github.com/stripe/stripe-cli/pkg/validators"
)

type reauthCmd struct {
	cmd           *cobra.Command
	accessBaseURL string
}

func newReauthCmd() *reauthCmd {
	rc := &reauthCmd{}
	rc.cmd = &cobra.Command{
		Use:   "reauth",
		Args:  validators.NoArgs,
		Short: "Re-authorize the CLI for the current OAuth session",
		Long: `Re-authorize the CLI to change permissions or authorize access to
additional accounts or sandboxes.

Opens the Stripe Dashboard so you can re-consent for the current OAuth session.`,
		Example: `stripe reauth`,
		RunE:    rc.runReauthCmd,
	}
	rc.cmd.Flags().StringVar(&rc.accessBaseURL, "access-base", login.DefaultAccessBaseURL, "Sets the access base URL")
	rc.cmd.Flags().MarkHidden("access-base") //nolint:errcheck
	return rc
}

func (rc *reauthCmd) runReauthCmd(cmd *cobra.Command, args []string) error {
	uat, _ := Config.Profile.GetUAT()
	if !strings.HasPrefix(uat, "oak_") {
		return errorcategory.Errorf(errorcategory.UserInput, "reauth requires an OAuth session; run 'stripe login' to authenticate")
	}
	return login.Reauth(cmd.Context(), rc.accessBaseURL, uat)
}
