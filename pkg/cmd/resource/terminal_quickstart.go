package resource

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/stripe/stripe-cli/pkg/config"
	"github.com/stripe/stripe-cli/pkg/terminal"
	"github.com/stripe/stripe-cli/pkg/validators"
	"github.com/stripe/stripe-cli/pkg/version"
)

// QuickstartCmd starts a prompt flow for connecting a Terminal reader to their Stripe account
type QuickstartCmd struct {
	cfg *config.Config
	cmd *cobra.Command
}

// NewQuickstartCmd returns a new terminal quickstart command
func NewQuickstartCmd(parentCmd *cobra.Command, config *config.Config) {
	quickstartCmd := &QuickstartCmd{
		cfg: config,
	}

	quickstartCmd.cmd = &cobra.Command{
		Use:     "quickstart",
		Args:    validators.MaximumNArgs(0),
		Short:   "Set up a Terminal reader and take a test payment",
		Example: `stripe terminal quickstart --api-key sk_123`,
		RunE:    quickstartCmd.runQuickstartCmd,
	}

	parentCmd.AddCommand(quickstartCmd.cmd)
}

func (cc *QuickstartCmd) runQuickstartCmd(cmd *cobra.Command, args []string) error {
	version.CheckLatestVersion()

	key, err := cc.cfg.Profile.GetAPIKey(false)

	if err != nil {
		return fmt.Errorf(err.Error())
	}

	err = validators.APIKeyNotRestricted(key)

	if err != nil {
		return fmt.Errorf(err.Error())
	}

	readers := terminal.ReaderNames()
	reader, err := terminal.ReaderTypeSelectPrompt(readers)

	if err != nil {
		return fmt.Errorf(err.Error())
	}

	if reader == terminal.ReaderList["verifone-p400"].Name {
		err = terminal.QuickstartP400(cmd.Context(), cc.cfg)
		if err != nil {
			return fmt.Errorf(err.Error())
		}
	}

	return nil
}
