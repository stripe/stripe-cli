package cmd

import (
	"fmt"

	"github.com/spf13/afero"
	"github.com/spf13/cobra"

	"github.com/stripe/stripe-cli/pkg/ansi"
	"github.com/stripe/stripe-cli/pkg/fixtures"
	"github.com/stripe/stripe-cli/pkg/stripe"
	"github.com/stripe/stripe-cli/pkg/validators"
	"github.com/stripe/stripe-cli/pkg/version"
)

type triggerCmd struct {
	cmd *cobra.Command

	fs            afero.Fs
	stripeAccount string
	skip          []string
	override      []string
	add           []string
	remove        []string
	raw           string
	apiBaseURL    string
}

func newTriggerCmd() *triggerCmd {
	tc := &triggerCmd{}
	tc.fs = afero.NewOsFs()
	tc.cmd = &cobra.Command{
		Use:       "trigger <event>",
		Args:      validators.MaximumNArgs(1),
		ValidArgs: fixtures.EventNames(),
		Short:     "Trigger test webhook events",
		Long: fmt.Sprintf(`Trigger specific webhook events to be sent. Webhooks events created through
the trigger command will also create all necessary side-effect events that are
needed to create the triggered event as well as the corresponding API objects.

%s
%s
`,
			ansi.Bold("Supported events:"),
			fixtures.EventList(),
		),
		Example: `stripe trigger payment_intent.created`,
		RunE:    tc.runTriggerCmd,
	}

	tc.cmd.Flags().StringVar(&tc.stripeAccount, "stripe-account", "", "Set a header identifying the connected account")
	tc.cmd.Flags().StringArrayVar(&tc.skip, "skip", []string{}, "Skip specific steps in the trigger")
	tc.cmd.Flags().StringArrayVar(&tc.override, "override", []string{}, "Override params in the trigger")
	tc.cmd.Flags().StringArrayVar(&tc.add, "add", []string{}, "Add params to the trigger")
	tc.cmd.Flags().StringArrayVar(&tc.remove, "remove", []string{}, "Remove params from the trigger")
	tc.cmd.Flags().StringVar(&tc.raw, "raw", "", "Raw fixture in string format to replace all default fixtures")

	// Hidden configuration flags, useful for dev/debugging
	tc.cmd.Flags().StringVar(&tc.apiBaseURL, "api-base", stripe.DefaultAPIBaseURL, "Sets the API base URL")
	tc.cmd.Flags().MarkHidden("api-base") // #nosec G104

	return tc
}

func (tc *triggerCmd) runTriggerCmd(cmd *cobra.Command, args []string) error {
	version.CheckLatestVersion()

	if len(args) == 0 {
		cmd.Help()

		return nil
	}

	apiKey, err := Config.Profile.GetAPIKey(false)
	if err != nil {
		return err
	}

	event := args[0]

	_, err = fixtures.Trigger(cmd.Context(), event, tc.stripeAccount, tc.apiBaseURL, apiKey, tc.skip, tc.override, tc.add, tc.remove, tc.raw)
	if err != nil {
		return err
	}

	fmt.Println("Trigger succeeded! Check dashboard for event details.")
	return nil
}
