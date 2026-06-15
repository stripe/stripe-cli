package coop

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGenerateWebhookExampleNodeHandlesSnapshotAndThinEvents(t *testing.T) {
	example := GenerateWebhookExample([]string{
		"invoice.paid",
		"customer.subscription.created",
		"invoice.paid",
	}, "node")

	assert.Contains(t, example, `express.raw({ type: "application/json" })`)
	assert.Contains(t, example, "stripe.webhooks.constructEvent")
	assert.Contains(t, example, "stripe.v2.core.events.retrieve(notification.id)")
	assert.Contains(t, example, "normalizeStripeEventType")
	assert.Contains(t, example, `case "invoice.paid"`)
	assert.Contains(t, example, `case "customer.subscription.created"`)
	assert.Contains(t, example, "mark the invoice/subscription entitlement state current idempotently")
	assert.Contains(t, example, "Persist subscription status")
	assert.Contains(t, example, "Use idempotencyKey")
	assert.Contains(t, example, "event.snapshot_event || event.id")
	assert.Contains(t, example, "v1.<event>")
	assert.Equal(t, 1, strings.Count(example, `case "invoice.paid"`))
}

func TestGenerateWebhookExampleNodeSteersCheckoutCompletionState(t *testing.T) {
	example := GenerateWebhookExample([]string{"checkout.session.completed"}, "node")

	assert.Contains(t, example, "Find the pending app record from Session metadata")
	assert.Contains(t, example, "mark it paid or fulfilled idempotently")
	assert.Contains(t, example, "do not rely on success-page query params")
}

func TestGenerateWebhookExamplePythonUsesSDKThinHelpers(t *testing.T) {
	example := GenerateWebhookExample([]string{"entitlements.active_entitlement_summary.updated"}, "python")

	assert.Contains(t, example, "stripe.StripeClient")
	assert.Contains(t, example, "parse_event_notification")
	assert.Contains(t, example, "notification.fetch_event()")
	assert.Contains(t, example, "stripe_client.construct_event")
	assert.Contains(t, example, `event_type == "entitlements.active_entitlement_summary.updated"`)
	assert.Contains(t, example, "Refresh the customer's server-side entitlement state")
}

func TestGenerateWebhookExampleGoUsesSDKThinHelpers(t *testing.T) {
	example := GenerateWebhookExample([]string{"v2.billing.pricing_plan_subscription.servicing_activated"}, "go")

	assert.Contains(t, example, "stripe.NewClient")
	assert.Contains(t, example, "stripeClient.ParseEventNotification")
	assert.Contains(t, example, "FetchEvent(r.Context())")
	assert.Contains(t, example, "stripeClient.ConstructEvent")
	assert.Contains(t, example, `case "v2.billing.pricing_plan_subscription.servicing_activated"`)
	assert.Contains(t, example, "Use the event or snapshot_event ID as an idempotency key")
}

func TestGenerateWebhookExampleGenericMentionsThinEvents(t *testing.T) {
	example := GenerateWebhookExample([]string{" checkout.session.completed "}, "ruby")

	assert.Contains(t, example, "checkout.session.completed")
	assert.Contains(t, example, "Events v2")
	assert.Contains(t, example, "v1.<event>")
	assert.Contains(t, example, "Find the pending app record from Session metadata")
}

func TestGenerateWebhookExamplePreservesExplicitV1ThinEvent(t *testing.T) {
	example := GenerateWebhookExample([]string{"v1.customer.created"}, "node")

	assert.Contains(t, example, `"v1.customer.created"`)
	assert.Contains(t, example, "handledStripeEventTypes.has(type)")
	assert.Contains(t, example, `case "v1.customer.created"`)
}
