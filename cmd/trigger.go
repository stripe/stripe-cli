package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/stripe/stripe-cli/requests"
	"github.com/stripe/stripe-cli/validators"
)

const apiVersion = "2019-03-14"
const stripeURL = "https://api.stripe.com"

type triggerCmd struct {
	cmd *cobra.Command
}

func newTriggerCmd() *triggerCmd {
	tc := &triggerCmd{}
	tc.cmd = &cobra.Command{
		Use:   "trigger",
		Args:  validators.ExactArgs(1),
		Short: "Trigger test webhook events to fire",
		Long: `Cause a specific webhook event to be created and sent. Webhooks tested through
the trigger command will also create all necessary side-effect events that are
needed to create the triggered event.

Trigger a payment_intent.created event:
$ stripe trigger payment_intent.created

Supported events:
	charge.captured
	charge.failed
	charge.succeeded
	customer.created
	customer.updated
	customer.source.created
	customer.source.updated
	customer.subscription.updated
	invoice.created
	invoice.finalized
	invoice.payment_succeeded
	invoice.updated
	payment_intent.created
	payment_intent.payment_failed
	payment_intent.succeeded
	payment_method.attached`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return triggerEvent(args[0])
		},
	}

	return tc
}

func triggerEvent(event string) error {
	secretKey, err := profile.GetSecretKey()
	if err != nil {
		return err
	}

	examples := requests.Examples{
		Profile:    profile,
		APIUrl: stripeURL,
		APIVersion: apiVersion,
		SecretKey:  secretKey,
	}

	supportedEvents := map[string]interface{}{
		"charge.captured": examples.ChargeCaptured,
		"charge.failed": examples.ChargeFailed,
		"charge.succeeded": examples.ChargeSucceeded,
		"customer.created": examples.CustomerCreated,
		"customer.updated": examples.CustomerUpdated,
		"customer.source.created": examples.CustomerSourceCreated,
		"customer.source.updated": examples.CustomerSourceUpdated,
		"customer.subscription.updated": examples.CustomerSubscriptionUpdated,
		"invoice.created": examples.InvoiceCreated,
		"invoice.finalized": examples.InvoiceFinalized,
		"invoice.payment_succeeded": examples.InvoicePaymentSucceeded,
		"invoice.updated": examples.InvoiceUpdated,
		"payment_intent.created": examples.PaymentIntentCreated,
		"payment_intent.payment_failed": examples.PaymentIntentFailed,
		"payment_intent.succeeded": examples.PaymentIntentSucceeded,
		"payment_method.attached": examples.PaymentMethodAttached,
	}
	function, ok := supportedEvents[event]
	if !ok {
		return fmt.Errorf(fmt.Sprintf("event %s is not supported.", event))
	}
	err = function.(func() error)()
	return err
}
