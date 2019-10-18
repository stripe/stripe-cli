package cmd

import (
	"fmt"

	"github.com/spf13/afero"
	"github.com/spf13/cobra"

	"github.com/stripe/stripe-cli/pkg/ansi"
	s "github.com/stripe/stripe-cli/pkg/samples"
	"github.com/stripe/stripe-cli/pkg/stripe"
	"github.com/stripe/stripe-cli/pkg/validators"
	"github.com/stripe/stripe-cli/pkg/version"
)

const apiVersion = "2019-03-14"

type triggerCmd struct {
	cmd *cobra.Command

	apiBaseURL string
}

func newTriggerCmd() *triggerCmd {
	tc := &triggerCmd{}
	tc.cmd = &cobra.Command{
		Use:  "trigger <event>",
		Args: validators.MaximumNArgs(1),
		ValidArgs: []string{
			"charge.captured",
			"charge.dispute.created",
			"charge.failed",
			"charge.refunded",
			"charge.succeeded",
			"checkout.session.completed",
			"customer.created",
			"customer.deleted",
			"customer.updated",
			"customer.source.created",
			"customer.source.updated",
			"customer.subscription.deleted",
			"customer.subscription.updated",
			"invoice.created",
			"invoice.finalized",
			"invoice.payment_failed",
			"invoice.payment_succeeded",
			"invoice.updated",
			"payment_intent.created",
			"payment_intent.payment_failed",
			"payment_intent.succeeded",
			"payment_intent.canceled",
			"payment_method.attached",
		},
		Short: "Trigger test webhook events to fire",
		Long: fmt.Sprintf(`%s

Cause a specific webhook event to be created and sent. Webhooks tested through
the trigger command will also create all necessary side-effect events that are
needed to create the triggered event.

%s
  charge.captured
  charge.dispute.created
  charge.failed
  charge.refunded
  charge.succeeded
  checkout.session.completed
  customer.created
  customer.deleted
  customer.updated
  customer.source.created
  customer.source.updated
  customer.subscription.deleted
  customer.subscription.updated
  invoice.created
  invoice.finalized
  invoice.payment_failed
  invoice.payment_succeeded
  invoice.updated
  payment_intent.created
  payment_intent.payment_failed
  payment_intent.succeeded
  payment_intent.canceled
  payment_method.attached
`,
			getBanner(),
			ansi.Bold("Supported events:"),
		),
		Example: `stripe trigger payment_intent.created`,
		RunE:    tc.runTriggerCmd,
	}

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
		cmd.Usage()

		return nil
	}

	event := args[0]
	supportedEvents := map[string]*s.Fixture{
		"charge.captured":               buildFromFixture(apiKey, "triggers/charge.captured.json"),
		"charge.dispute.created":        buildFromFixture(apiKey, "triggers/charge.disputed.created.json"),
		"charge.failed":                 buildFromFixture(apiKey, "triggers/charge.failed.json"),
		"charge.refunded":               buildFromFixture(apiKey, "triggers/charge.refunded.json"),
		"charge.succeeded":              buildFromFixture(apiKey, "triggers/charge.succeeded.json"),
		"checkout.session.completed":    buildFromFixture(apiKey, "triggers/checkout.session.completed.json"),
		"customer.created":              buildFromFixture(apiKey, "triggers/customer.created.json"),
		"customer.deleted":              buildFromFixture(apiKey, "triggers/customer.deleted.json"),
		"customer.updated":              buildFromFixture(apiKey, "triggers/customer.updated.json"),
		"customer.source.created":       buildFromFixture(apiKey, "triggers/customer.source.created.json"),
		"customer.source.updated":       buildFromFixture(apiKey, "triggers/customer.source.updated.json"),
		"customer.subscription.deleted": buildFromFixture(apiKey, "triggers/customer.subscription.deleted.json"),
		"customer.subscription.updated": buildFromFixture(apiKey, "triggers/customer.subscription.updated.json"),
		"invoice.created":               buildFromFixture(apiKey, "triggers/invoice.created.json"),
		"invoice.finalized":             buildFromFixture(apiKey, "triggers/invoice.finalized.json"),
		"invoice.payment_failed":        buildFromFixture(apiKey, "triggers/invoice.payment_failed.json"),
		"invoice.payment_succeeded":     buildFromFixture(apiKey, "triggers/invoice.payment_succeeded.json"),
		"invoice.updated":               buildFromFixture(apiKey, "triggers/invoice.updated.json"),
		"payment_intent.created":        buildFromFixture(apiKey, "triggers/payment_intent.created.json"),
		"payment_intent.payment_failed": buildFromFixture(apiKey, "triggers/payment_intent.payment_failed.json"),
		"payment_intent.succeeded":      buildFromFixture(apiKey, "triggers/payment_intent.succeeded.json"),
		"payment_intent.canceled":       buildFromFixture(apiKey, "triggers/payment_intent.canceled.json"),
		"payment_method.attached":       buildFromFixture(apiKey, "triggers/payment_method.attached.json"),
	}

	fixture, ok := supportedEvents[event]
	if !ok {
		return fmt.Errorf(fmt.Sprintf("event %s is not supported.", event))
	}

	err = fixture.Execute()
	if err == nil {
		fmt.Println("Trigger succeeded! Check dashboard for event details.")
	} else {
		fmt.Println(fmt.Sprintf("Trigger failed: %s", err))
	}

	return err
}

func buildFromFixture(apiKey, jsonFile string) *s.Fixture {
	fixture, _ := s.NewFixture(
		afero.NewOsFs(),
		apiKey,
		stripe.DefaultAPIBaseURL,
		jsonFile,
	)

	return fixture
}
