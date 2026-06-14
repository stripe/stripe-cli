package coop

import (
	"fmt"
	"strings"
)

// GenerateAPIRequestGuidance summarizes how an agent should use the blueprint
// request metadata for an app implementation step.
func GenerateAPIRequestGuidance(req *APIRequest) string {
	if req == nil {
		return ""
	}
	var b strings.Builder
	fmt.Fprintf(&b, "Use the blueprint API request as the canonical target: %s %s.", strings.ToUpper(req.Method), req.Path)
	if requestHasParams(req.Params) {
		b.WriteString(" The request params in this response are canonical; adapt placeholders and referenced IDs to the user's app state instead of inventing a different API shape.")
	} else if isMutatingMethod(req.Method) {
		b.WriteString(" This blueprint node currently provides endpoint and method only, so treat any SDK example as incomplete. Fill request params from the step intent, earlier blueprint outputs, and Stripe's official API docs, then report the exact app code path and params you used.")
	}
	if strings.Contains(req.Path, "/checkout/sessions") {
		b.WriteString(" For hosted Checkout, verify the app creates the Session, returns or redirects to the hosted URL, configures success/cancel URLs, and handles the relevant webhook state without automating card entry.")
	}
	return b.String()
}

// GenerateAsyncHandlerGuidance summarizes event handling and verification hints
// for a blueprint asyncHandler node.
func GenerateAsyncHandlerGuidance(events []string) string {
	events = normalizedEvents(events)
	if len(events) == 0 {
		return ""
	}
	var b strings.Builder
	b.WriteString("Implement one signed webhook/event handler, verify the Stripe signature on the raw body, branch on every blueprint event, and store or refresh the app state needed by later steps.")
	for _, event := range events {
		if note := asyncEventVerificationNote(event); note != "" {
			b.WriteString(" ")
			b.WriteString(note)
		}
	}
	return b.String()
}

func normalizedEvents(events []string) []string {
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

func asyncEventVerificationNote(event string) string {
	switch strings.TrimSpace(event) {
	case "entitlements.active_entitlement_summary.updated":
		return "The Stripe CLI might not support `stripe trigger entitlements.active_entitlement_summary.updated`; if it fails, keep `stripe listen` forwarding to the app and create or update subscription state for an entitlement-backed product so Stripe emits the event naturally."
	case "test_helpers.test_clock.ready":
		return "For test clock readiness, prefer advancing the clock through the app or Stripe test helpers, then poll/retrieve the clock and confirm the signed `test_helpers.test_clock.ready` event when available."
	default:
		return ""
	}
}
