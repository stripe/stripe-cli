package coop

import (
	"encoding/json"
	"fmt"
	"regexp"
	"sort"
	"strings"
)

var blueprintReferenceTokenRE = regexp.MustCompile(`\$\{[^}]+\}`)

// GenerateStepGuidance summarizes how an agent should follow one concrete
// blueprint step.
func GenerateStepGuidance(step StepInfo) string {
	var b strings.Builder
	fmt.Fprintf(&b, "Follow blueprint step %d", step.Number)
	if step.Key != "" {
		fmt.Fprintf(&b, " (%s)", step.Key)
	}
	fmt.Fprintf(&b, ": %s.", step.Title)
	if step.Description != "" {
		fmt.Fprintf(&b, " Step description: %s", step.Description)
	}
	if step.ReviewPrompt != "" {
		fmt.Fprintf(&b, " The review_prompt is the acceptance check to satisfy: %s", step.ReviewPrompt)
	}
	if step.ReviewCommand != "" {
		fmt.Fprintf(&b, " Use review_command during verification when applicable: %s", step.ReviewCommand)
	}

	switch step.Type {
	case NodeAPIRequest:
		if step.APIRequest != nil {
			b.WriteString(" ")
			b.WriteString(GenerateAPIRequestGuidance(step.APIRequest))
		}
	case NodeAsyncHandler:
		if len(step.Events) > 0 {
			b.WriteString(" ")
			b.WriteString(GenerateAsyncHandlerGuidance(step.Events))
		}
	case NodeUIComponent:
		b.WriteString(" Implement the user-facing app behavior described by this blueprint step; wire it to the specific API, redirect, webhook, or state produced by neighboring blueprint steps instead of building a detached demo UI.")
	case NodeCLICommand:
		b.WriteString(" This is a blueprint CLI step: run the command or command family described by the step and report concrete output. Do not turn non-CLI blueprint steps into CLI-only work.")
	case NodeTestHelper:
		b.WriteString(" This is a blueprint verification/setup step: use test helpers to exercise the app behavior required by the surrounding blueprint steps, and report the observed result.")
	case NodeDashboard, NodeSetUpWebhooks:
		b.WriteString(" Follow the blueprint setup intent exactly, then report the concrete app or Stripe configuration that satisfies it.")
	}
	return b.String()
}

// GenerateAPIRequestGuidance summarizes how an agent should use the blueprint
// request metadata for an app implementation step.
func GenerateAPIRequestGuidance(req *APIRequest) string {
	if req == nil {
		return ""
	}
	var b strings.Builder
	fmt.Fprintf(&b, "Use the blueprint API request as the canonical target: %s %s; do not change the endpoint or method.", strings.ToUpper(req.Method), req.Path)
	if refs := BlueprintReferences(req.Path, req.Params); len(refs) > 0 {
		fmt.Fprintf(&b, " Preserve blueprint output references and resolve them from prior steps at runtime: %s.", strings.Join(refs, ", "))
	}
	if requestHasParams(req.Params) {
		b.WriteString(" The request params in blueprint_step.api_request.params are canonical; use them as the request shape, adapting only placeholders and referenced IDs to the user's app state instead of inventing a different API shape.")
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
	fmt.Fprintf(&b, "Implement one signed webhook/event handler, verify the Stripe signature on the raw body, branch on every blueprint event (%s), and store or refresh the app state needed by later steps.", strings.Join(events, ", "))
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

// BlueprintReferences returns sorted unique ${...} references used by blueprint
// paths, params, and other structured values.
func BlueprintReferences(values ...interface{}) []string {
	seen := map[string]bool{}
	for _, value := range values {
		for _, ref := range blueprintReferenceTokenRE.FindAllString(blueprintReferenceSource(value), -1) {
			seen[ref] = true
		}
	}
	refs := make([]string, 0, len(seen))
	for ref := range seen {
		refs = append(refs, ref)
	}
	sort.Strings(refs)
	return refs
}

func hasBlueprintReferences(values ...interface{}) bool {
	return len(BlueprintReferences(values...)) > 0
}

func blueprintReferenceSource(value interface{}) string {
	switch v := value.(type) {
	case nil:
		return ""
	case string:
		return v
	default:
		data, err := json.Marshal(v)
		if err != nil {
			return fmt.Sprint(v)
		}
		return string(data)
	}
}
