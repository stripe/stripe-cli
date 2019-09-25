package cmd

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/stripe/stripe-cli/pkg/ansi"
	"github.com/stripe/stripe-cli/pkg/requests"
	"github.com/stripe/stripe-cli/pkg/stripe"
	"github.com/stripe/stripe-cli/pkg/validators"
)

const apiVersion = "2019-03-14"

type triggerCmd struct {
	cmd *cobra.Command

	apiBaseURL string
	eventID    string
}

func newTriggerCmd() *triggerCmd {
	tc := &triggerCmd{}
	tc.cmd = &cobra.Command{
		Use:  "trigger <event>",
		Args: validators.MaximumNArgs(1),
		ValidArgs: []string{
			"charge.captured",
			"charge.failed",
			"charge.succeeded",
			"customer.created",
			"customer.delete",
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
			"payment_method.attached",
		},
		Short: "Trigger test webhook events to fire",
		Long: fmt.Sprintf(`%s

Cause a specific webhook event to be created and sent. Webhooks tested through
the trigger command will also create all necessary side-effect events that are
needed to create the triggered event.

%s
  charge.captured
  charge.failed
  charge.succeeded
  customer.created
  customer.delete
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
  payment_method.attached

  You can also resend a past event using the --event flag:
  e.g. stripe trigger --event evt_123`,
			getBanner(),
			ansi.Bold("Supported events:"),
		),
		Example: `stripe trigger payment_intent.created`,
		RunE:    tc.runTriggerCmd,
	}

	// Hidden configuration flags, useful for dev/debugging
	tc.cmd.Flags().StringVar(&tc.apiBaseURL, "api-base", stripe.DefaultAPIBaseURL, "Sets the API base URL")
	tc.cmd.Flags().MarkHidden("api-base") // #nosec G104
	tc.cmd.Flags().StringVar(&tc.eventID, "event", "", "ID of the event to resend")

	return tc
}

func (tc *triggerCmd) runTriggerCmd(cmd *cobra.Command, args []string) error {
	apiKey, err := Config.Profile.GetAPIKey(false)
	if err != nil {
		return err
	}

	examples := requests.Examples{
		Profile:    Config.Profile,
		APIBaseURL: tc.apiBaseURL,
		APIVersion: apiVersion,
		APIKey:     apiKey,
	}

	if len(args) == 0 {
		if tc.eventID != "" {
			return examples.ResendEvent(tc.eventID)
		}

		cmd.Usage()
		return nil
	}

	event := args[0]
	supportedEvents := map[string]interface{}{
		"charge.captured":               examples.ChargeCaptured,
		"charge.failed":                 examples.ChargeFailed,
		"charge.succeeded":              examples.ChargeSucceeded,
		"customer.created":              examples.CustomerCreated,
		"customer.deleted":              examples.CustomerDeleted,
		"customer.updated":              examples.CustomerUpdated,
		"customer.source.created":       examples.CustomerSourceCreated,
		"customer.source.updated":       examples.CustomerSourceUpdated,
		"customer.subscription.deleted": examples.CustomerSubscriptionDeleted,
		"customer.subscription.updated": examples.CustomerSubscriptionUpdated,
		"invoice.created":               examples.InvoiceCreated,
		"invoice.finalized":             examples.InvoiceFinalized,
		"invoice.payment_failed":        examples.InvoicePaymentFailed,
		"invoice.payment_succeeded":     examples.InvoicePaymentSucceeded,
		"invoice.updated":               examples.InvoiceUpdated,
		"payment_intent.created":        examples.PaymentIntentCreated,
		"payment_intent.payment_failed": examples.PaymentIntentFailed,
		"payment_intent.succeeded":      examples.PaymentIntentSucceeded,
		"payment_method.attached":       examples.PaymentMethodAttached,
	}
	function, ok := supportedEvents[event]
	if !ok {
		return fmt.Errorf(fmt.Sprintf("event %s is not supported.", event))
	}
	err = function.(func() error)()

	if err == nil {
		fmt.Println("Trigger succeeded! Check dashboard for event details.")
	} else {
		fmt.Println(fmt.Sprintf("Trigger failed: %s", err))
	}

	return err
}
