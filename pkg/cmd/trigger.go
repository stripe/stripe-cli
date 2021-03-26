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

	// Hidden configuration flags, useful for dev/debugging
	tc.cmd.Flags().StringVar(&tc.apiBaseURL, "api-base", stripe.DefaultAPIBaseURL, "Sets the API base URL")
	tc.cmd.Flags().MarkHidden("api-base") // #nosec G104

	return tc
}

func (tc *triggerCmd) runTriggerCmd(cmd *cobra.Command, args []string) error {
	version.CheckLatestVersion()

	apiKey, err := Config.Profile.GetAPIKey(false)
	if err != nil {
		return err
	}

	if len(args) == 0 {
		cmd.Help()

		return nil
	}

	event := args[0]

	var fixture *fixtures.Fixture
	if file, ok := fixtures.Events[event]; ok {
		fixture, err = fixtures.BuildFromFixture(tc.fs, apiKey, tc.stripeAccount, tc.apiBaseURL, file)
		if err != nil {
			return err
		}
	} else {
		exists, _ := afero.Exists(tc.fs, event)
		if !exists {
			return fmt.Errorf(fmt.Sprintf("The event ‘%s’ is not supported by the Stripe CLI.", event))
		}

		fixture, err = fixtures.BuildFromFixture(tc.fs, apiKey, tc.stripeAccount, tc.apiBaseURL, event)
		if err != nil {
			return err
		}
	}

	err = fixture.Execute()
	if err == nil {
		fmt.Println("Trigger succeeded! Check dashboard for event details.")
	} else {
		fmt.Printf("Trigger failed: %s\n", err)
	}

	return err
}
