package coop

import (
	"fmt"
	"strings"
)

// GenerateWebhookExample returns a compact, event-specific webhook handler
// skeleton for asyncHandler steps.
func GenerateWebhookExample(events []string, language string) string {
	events = normalizeWebhookEvents(events)
	if len(events) == 0 {
		return ""
	}
	switch normalizeExampleLanguage(language) {
	case "node":
		return generateNodeWebhookExample(events)
	case "python":
		return generatePythonWebhookExample(events)
	case "go":
		return generateGoWebhookExample(events)
	default:
		return generateGenericWebhookExample(events)
	}
}

func normalizeWebhookEvents(events []string) []string {
	seen := map[string]bool{}
	var normalized []string
	for _, event := range events {
		event = strings.TrimSpace(event)
		if event == "" || seen[event] {
			continue
		}
		seen[event] = true
		normalized = append(normalized, event)
	}
	return normalized
}

func normalizeExampleLanguage(language string) string {
	switch strings.ToLower(strings.TrimSpace(language)) {
	case "node", "javascript", "js", "typescript", "ts":
		return "node"
	case "python", "py":
		return "python"
	case "go", "golang":
		return "go"
	default:
		return ""
	}
}

func generateNodeWebhookExample(events []string) string {
	var b strings.Builder
	b.WriteString(`const express = require("express");
const stripe = require("stripe")(process.env.STRIPE_SECRET_KEY);
const app = express();

const handledStripeEventTypes = new Set([
`)
	writeNodeEventSet(&b, events)
	b.WriteString(`]);

app.post("/webhook", express.raw({ type: "application/json" }), async (req, res) => {
  const signature = req.headers["stripe-signature"];
  let notification;

  try {
    notification = stripe.webhooks.constructEvent(
      req.body,
      signature,
      process.env.STRIPE_WEBHOOK_SECRET,
    );
  } catch (err) {
    return res.status(400).send("Webhook Error: " + err.message);
  }

  const event = await resolveStripeEvent(notification);
  const eventType = normalizeStripeEventType(event.type);
  const data = event.data?.object ?? event.data ?? null;
  const relatedObject = event.related_object ?? null;
  const idempotencyKey = event.snapshot_event || event.id;

  switch (eventType) {
`)
	for _, event := range events {
		fmt.Fprintf(&b, "    case %q: {\n", event)
		b.WriteString("      // Snapshot events usually provide data.object. Thin events provide\n")
		b.WriteString("      // relatedObject and, after retrieval, any extra data/changes.\n")
		fmt.Fprintf(&b, "      // %s\n", webhookStateUpdateHint(event))
		b.WriteString("      // Use idempotencyKey to make repeated event delivery safe.\n")
		b.WriteString("      break;\n")
		b.WriteString("    }\n")
	}
	b.WriteString(`    default:
      break;
  }

  res.json({ received: true });
});

async function resolveStripeEvent(notification) {
  if (notification.object === "v2.core.event") {
    return await stripe.v2.core.events.retrieve(notification.id);
  }
  return notification;
}

function normalizeStripeEventType(type) {
  if (handledStripeEventTypes.has(type)) {
    return type;
  }
  const unprefixed = type && type.startsWith("v1.") ? type.slice(3) : type;
  return handledStripeEventTypes.has(unprefixed) ? unprefixed : type;
}

// Thin-event migration receives v1.<event> aliases, for example v1.customer.created.
// Use event.snapshot_event as the idempotency key when running snapshot and thin handlers in parallel.
// Lookup endpoints can supplement this handler, but they do not replace handling the signed event.
`)
	return b.String()
}

func generatePythonWebhookExample(events []string) string {
	var b strings.Builder
	b.WriteString(`import json
import os
import stripe
from flask import request, jsonify

stripe_client = stripe.StripeClient(os.environ["STRIPE_SECRET_KEY"])
HANDLED_STRIPE_EVENT_TYPES = {
`)
	writePythonEventSet(&b, events)
	b.WriteString(`}

@app.post("/webhook")
def stripe_webhook_handler():
    payload = request.get_data()
    signature = request.headers.get("Stripe-Signature")

    try:
        envelope = json.loads(payload)
        if envelope.get("object") == "v2.core.event":
            notification = stripe_client.parse_event_notification(
                payload, signature, os.environ["STRIPE_WEBHOOK_SECRET"]
            )
            event = notification.fetch_event()
        else:
            event = stripe_client.construct_event(
                payload, signature, os.environ["STRIPE_WEBHOOK_SECRET"]
            )
    except Exception as err:
        return jsonify(error=str(err)), 400

    event_type = normalize_stripe_event_type(event["type"])
    data = event.get("data", {}).get("object") or event.get("data")
    related_object = event.get("related_object")
    idempotency_key = event.get("snapshot_event") or event["id"]
`)
	for i, event := range events {
		prefix := "if"
		if i > 0 {
			prefix = "elif"
		}
		fmt.Fprintf(&b, "    %s event_type == %q:\n", prefix, event)
		b.WriteString("        # Snapshot events usually provide data.object. Thin events provide\n")
		b.WriteString("        # related_object and, after retrieval, any extra data/changes.\n")
		fmt.Fprintf(&b, "        # %s\n", webhookStateUpdateHint(event))
		b.WriteString("        # Use idempotency_key to make repeated event delivery safe.\n")
	}
	b.WriteString(`
    return jsonify(received=True)

def normalize_stripe_event_type(event_type):
    if event_type in HANDLED_STRIPE_EVENT_TYPES:
        return event_type
    unprefixed = event_type[3:] if event_type.startswith("v1.") else event_type
    return unprefixed if unprefixed in HANDLED_STRIPE_EVENT_TYPES else event_type

# Thin-event migration receives v1.<event> aliases, for example v1.customer.created.
# Use event.snapshot_event as the idempotency key when running snapshot and thin handlers in parallel.
# Lookup endpoints can supplement this handler, but they do not replace handling the signed event.
`)
	return b.String()
}

func generateGoWebhookExample(events []string) string {
	var b strings.Builder
	b.WriteString(`var stripeClient = stripe.NewClient(os.Getenv("STRIPE_SECRET_KEY"))
var handledStripeEventTypes = map[string]bool{
`)
	writeGoEventSet(&b, events)
	b.WriteString(`}

type stripeEventFetcher interface {
	FetchEvent(context.Context) (stripe.V2CoreEvent, error)
}

func stripeWebhookHandler(w http.ResponseWriter, r *http.Request) {
	const maxBodyBytes = int64(65536)
	r.Body = http.MaxBytesReader(w, r.Body, maxBodyBytes)
	payload, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, err.Error(), http.StatusServiceUnavailable)
		return
	}

	eventType := ""
	var envelope struct {
		Object string ` + "`json:\"object\"`" + `
	}
	if err := json.Unmarshal(payload, &envelope); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if envelope.Object == "v2.core.event" {
		notification, err := stripeClient.ParseEventNotification(
			payload,
			r.Header.Get("Stripe-Signature"),
			os.Getenv("STRIPE_WEBHOOK_SECRET"),
		)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		base := notification.GetEventNotification()
		eventType = normalizeStripeEventType(base.Type)

		// Retrieve the full Events v2 object before mutating app state.
		fetcher, ok := notification.(stripeEventFetcher)
		if !ok {
			http.Error(w, "unsupported Stripe thin event", http.StatusBadRequest)
			return
		}
		fullEvent, err := fetcher.FetchEvent(r.Context())
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		_ = fullEvent // Type switch on fullEvent or fetch the related object as needed.
	} else {
		event, err := stripeClient.ConstructEvent(
			payload,
			r.Header.Get("Stripe-Signature"),
			os.Getenv("STRIPE_WEBHOOK_SECRET"),
		)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		eventType = event.Type
	}

	switch eventType {
`)
	for _, event := range events {
		fmt.Fprintf(&b, "\tcase %q:\n", event)
		fmt.Fprintf(&b, "\t\t// %s\n", webhookStateUpdateHint(event))
		b.WriteString("\t\t// Snapshot events use the constructed event's Data.Raw; thin events use the retrieved Events v2 object or related object.\n")
		b.WriteString("\t\t// Use the event or snapshot_event ID as an idempotency key so repeated delivery is safe.\n")
	}
	b.WriteString(`	default:
	}

	w.WriteHeader(http.StatusOK)
}

func normalizeStripeEventType(eventType string) string {
	if handledStripeEventTypes[eventType] {
		return eventType
	}
	unprefixed := strings.TrimPrefix(eventType, "v1.")
	if handledStripeEventTypes[unprefixed] {
		return unprefixed
	}
	return eventType
}

// Thin-event migration receives v1.<event> aliases, for example v1.customer.created.
// Use snapshot_event as the idempotency key when running snapshot and thin handlers in parallel.
// Lookup endpoints can supplement this handler, but they do not replace handling the signed event.
`)
	return b.String()
}

func writeNodeEventSet(b *strings.Builder, events []string) {
	for _, event := range events {
		fmt.Fprintf(b, "  %q,\n", event)
	}
}

func writePythonEventSet(b *strings.Builder, events []string) {
	for _, event := range events {
		fmt.Fprintf(b, "    %q,\n", event)
	}
}

func writeGoEventSet(b *strings.Builder, events []string) {
	for _, event := range events {
		fmt.Fprintf(b, "\t%q: true,\n", event)
	}
}

func generateGenericWebhookExample(events []string) string {
	var b strings.Builder
	b.WriteString("Verify the Stripe signature using STRIPE_WEBHOOK_SECRET, then branch on every required event:\n\n")
	for _, event := range events {
		fmt.Fprintf(&b, "- %s: read event.data.object for snapshot events, or retrieve the full Events v2 object/related object for thin events. %s\n", event, webhookStateUpdateHint(event))
	}
	b.WriteString("\nFor v1 thin-event migration, handle the v1.<event> alias as the same logical event and use snapshot_event for idempotency when both destinations run in parallel.\n")
	b.WriteString("Lookup endpoints can supplement this handler, but they do not replace handling the signed event.\n")
	return b.String()
}

func webhookStateUpdateHint(event string) string {
	logical := logicalStripeEvent(event)
	switch logical {
	case "checkout.session.completed":
		return "Find the pending app record from Session metadata, client_reference_id, customer, or session ID, then mark it paid or fulfilled idempotently; do not rely on success-page query params."
	case "checkout.session.async_payment_succeeded":
		return "Find the pending app record from the Checkout Session and complete fulfillment idempotently after the asynchronous payment succeeds."
	case "checkout.session.async_payment_failed":
		return "Find the pending app record from the Checkout Session and keep it unpaid or recoverable without fulfillment."
	case "payment_intent.succeeded":
		return "Find the pending app record from PaymentIntent metadata, customer, or ID, then mark it paid or fulfilled idempotently."
	case "payment_intent.payment_failed":
		return "Find the pending app record from the PaymentIntent and keep it unpaid or recoverable without fulfillment."
	case "invoice.paid", "invoice.payment_succeeded":
		return "Find the customer, subscription, or invoice record and mark the invoice/subscription entitlement state current idempotently."
	case "invoice.payment_failed":
		return "Find the customer or subscription record and mark the invoice unpaid or past_due without granting new access."
	case "entitlements.active_entitlement_summary.updated":
		return "Refresh the customer's server-side entitlement state and make access checks read that stored state."
	}
	if strings.HasPrefix(logical, "customer.subscription.") {
		return "Persist subscription status, current period, cancellation fields, customer mapping, and plan or price references used for access decisions."
	}
	if strings.HasPrefix(logical, "account.") || strings.Contains(logical, ".capability.") {
		return "Refresh connected-account readiness and capability state before enabling charges, payouts, or destination transfers."
	}
	return fmt.Sprintf("Update the app state required by %s idempotently using data, related object, and the event ID.", logical)
}
