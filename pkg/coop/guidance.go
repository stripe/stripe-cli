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
			b.WriteString(GenerateAPIRequestGuidanceForStep(step.APIRequest, step.Semantics))
			b.WriteString(" When sdk_example is present, use it as the generated SDK translation of blueprint_step.api_request; adapt it to the app's existing Stripe client pattern and resolve blueprint references instead of copying placeholders literally.")
		}
	case NodeAsyncHandler:
		if len(step.Events) > 0 {
			b.WriteString(" ")
			b.WriteString(GenerateAsyncHandlerGuidanceForStep(step.Events, step.Semantics))
			b.WriteString(" When webhook_example is present, use it as the generated handler translation of blueprint_step.events; adapt the route, framework, persistence, and side effects to the app without dropping or renaming blueprint events.")
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
	if semanticsGuidance := GenerateBlueprintSemanticsGuidance(step.Semantics); semanticsGuidance != "" {
		b.WriteString(" ")
		b.WriteString(semanticsGuidance)
	}
	if step.Semantics == nil {
		productGuidance := GenerateStepProductGuidance(step)
		if productGuidance == "" {
			return b.String()
		}
		b.WriteString(" ")
		b.WriteString(productGuidance)
	}
	return b.String()
}

// GenerateAPIRequestGuidance summarizes how an agent should use the blueprint
// request metadata for an app implementation step.
func GenerateAPIRequestGuidance(req *APIRequest) string {
	return GenerateAPIRequestGuidanceForStep(req, nil)
}

// GenerateAPIRequestGuidanceForStep summarizes request metadata plus structured
// blueprint semantics when available.
func GenerateAPIRequestGuidanceForStep(req *APIRequest, semantics *BlueprintSemantics) string {
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
	if !hasAPIProductSemantics(semantics) {
		for _, note := range APIRequestProductGuidance(req) {
			b.WriteString(" ")
			b.WriteString(note)
		}
	}
	return b.String()
}

// GenerateAsyncHandlerGuidance summarizes event handling and verification hints
// for a blueprint asyncHandler node.
func GenerateAsyncHandlerGuidance(events []string) string {
	return GenerateAsyncHandlerGuidanceForStep(events, nil)
}

// GenerateAsyncHandlerGuidanceForStep summarizes event handling plus structured
// event/lifecycle semantics when available.
func GenerateAsyncHandlerGuidanceForStep(events []string, semantics *BlueprintSemantics) string {
	events = normalizedEvents(events)
	if len(events) == 0 {
		return ""
	}
	var b strings.Builder
	fmt.Fprintf(&b, "Implement one signed webhook/event handler, verify the Stripe signature on the raw body, branch on every blueprint event (%s), and store or refresh the app state needed by later steps.", strings.Join(events, ", "))
	if !hasAsyncProductSemantics(semantics) {
		for _, note := range AsyncEventProductGuidance(events) {
			b.WriteString(" ")
			b.WriteString(note)
		}
	}
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

// GenerateBlueprintSemanticsGuidance compiles remote/local blueprint semantics
// into concise agent-facing implementation constraints.
func GenerateBlueprintSemanticsGuidance(semantics *BlueprintSemantics) string {
	if semantics == nil {
		return ""
	}
	var notes []string
	if source := semantics.SourceOfTruth; source != nil {
		var parts []string
		appendKeyValue := func(key, value string) {
			if strings.TrimSpace(value) != "" {
				parts = append(parts, fmt.Sprintf("%s=%s", key, value))
			}
		}
		appendKeyValue("amount", source.Amount)
		appendKeyValue("line_items", source.LineItems)
		appendKeyValue("catalog", source.Catalog)
		appendKeyValue("customer", source.Customer)
		appendKeyValue("connected_account", source.ConnectedAccount)
		if len(parts) > 0 {
			notes = append(notes, fmt.Sprintf("Blueprint source-of-truth semantics are canonical (%s); bind these values to the app's matching runtime state instead of inventing demo values.", strings.Join(parts, ", ")))
		}
	}
	if lifecycle := semantics.PaymentLifecycle; lifecycle != nil {
		var parts []string
		if lifecycle.StartsPayment {
			parts = append(parts, "starts_payment=true")
		}
		if lifecycle.CompletionEvent != "" {
			parts = append(parts, "completion_event="+lifecycle.CompletionEvent)
		}
		if len(lifecycle.FailureEvents) > 0 {
			parts = append(parts, "failure_events="+strings.Join(lifecycle.FailureEvents, ","))
		}
		if lifecycle.PendingState != "" {
			parts = append(parts, "pending_state="+lifecycle.PendingState)
		}
		if lifecycle.CompletedState != "" {
			parts = append(parts, "completed_state="+lifecycle.CompletedState)
		}
		if lifecycle.FulfillmentRequiresSignedWebhook {
			parts = append(parts, "fulfillment_requires_signed_webhook=true")
		}
		if len(parts) > 0 {
			notes = append(notes, fmt.Sprintf("Blueprint payment lifecycle semantics are canonical (%s); keep app state pending until the signed completion event before fulfillment, access, inventory, or durable paid-state changes.", strings.Join(parts, ", ")))
		}
	}
	if connect := semantics.Connect; connect != nil {
		var parts []string
		if connect.RequiresConnectedAccount {
			parts = append(parts, "requires_connected_account=true")
		}
		if connect.ConnectedAccountOwner != "" {
			parts = append(parts, "connected_account_owner="+connect.ConnectedAccountOwner)
		}
		if connect.OnboardingRequired {
			parts = append(parts, "onboarding_required=true")
		}
		if connect.AccountLinkRequired {
			parts = append(parts, "account_link_required=true")
		}
		if connect.CapabilityGate != "" {
			parts = append(parts, "capability_gate="+connect.CapabilityGate)
		}
		if connect.Source != "" {
			parts = append(parts, "source="+connect.Source)
		}
		if len(parts) > 0 {
			notes = append(notes, fmt.Sprintf("Blueprint Connect semantics are canonical (%s); resolve connected accounts from trusted app state and verify onboarding/capabilities before enabling charges, payouts, or destination transfers.", strings.Join(parts, ", ")))
		}
	}
	if len(semantics.EventRoles) > 0 {
		var parts []string
		for _, role := range semantics.EventRoles {
			event := strings.TrimSpace(role.Event)
			if event == "" {
				continue
			}
			roleText := event
			if role.Role != "" {
				roleText += ":" + role.Role
			}
			if role.StateUpdate != "" {
				roleText += "->" + role.StateUpdate
			}
			if role.RequiresLookup {
				roleText += "(lookup_required)"
			}
			parts = append(parts, roleText)
		}
		if len(parts) > 0 {
			notes = append(notes, fmt.Sprintf("Blueprint event roles are canonical (%s); implement each role in the signed handler rather than replacing events with lookup-only code.", strings.Join(parts, ", ")))
		}
	}
	if verification := semantics.ServerVerification; verification != nil && (verification.Required || verification.StateSource != "" || verification.Reason != "") {
		var parts []string
		if verification.Required {
			parts = append(parts, "required=true")
		}
		if verification.StateSource != "" {
			parts = append(parts, "state_source="+verification.StateSource)
		}
		if verification.Reason != "" {
			parts = append(parts, "reason="+verification.Reason)
		}
		notes = append(notes, fmt.Sprintf("Blueprint server-verification semantics are canonical (%s); render user-facing completion state from server-verified Stripe/app state, not URL params.", strings.Join(parts, ", ")))
	}
	if len(semantics.Assertions) > 0 {
		notes = append(notes, "Blueprint semantic assertions are acceptance criteria: "+strings.Join(semantics.Assertions, "; ")+".")
	}
	return strings.Join(notes, " ")
}

// GenerateStepProductGuidance returns product-safety guidance inferred from a
// step's current blueprint fields. It intentionally stays generic; richer
// semantics should eventually come from upstream blueprint metadata.
func GenerateStepProductGuidance(step StepInfo) string {
	context := stepGuidanceContext(step)
	var notes []string
	switch step.Type {
	case NodeUIComponent:
		if containsAny(context, "checkout", "payment", "invoice", "subscription", "billing portal") {
			notes = append(notes, "Wire the UI into the existing business flow by passing the current app record identity to the server endpoint; do not create a separate sample-only payment path when the app already has a domain flow for the thing being paid for or subscribed to.")
		}
		if containsAny(context, "success", "return", "redirect", "cancel") {
			notes = append(notes, "On success, return, or cancel pages, render server-verified state tied to the current user and Stripe IDs; do not treat URL query params as proof of payment, subscription, or entitlement status.")
		}
	case NodeTestHelper:
		if containsAny(context, "checkout", "payment", "invoice", "subscription", "entitlement") {
			notes = append(notes, "Verify the app-level lifecycle, not only that a Stripe object exists: the local record should stay pending before the signed event and move to the intended paid, active, fulfilled, or entitled state after the signed event.")
		}
	}
	return strings.Join(notes, " ")
}

// APIRequestProductGuidance returns endpoint-specific implementation guidance
// for product correctness. It is used in prose guidance and SDK fallback
// comments so agents see the same constraints even when docs snippets are not
// available.
func APIRequestProductGuidance(req *APIRequest) []string {
	if req == nil {
		return nil
	}
	path := strings.ToLower(strings.TrimSpace(req.Path))
	var notes []string
	if strings.Contains(path, "/checkout/sessions") {
		notes = append(notes, "For existing apps, derive Checkout line items, amounts, customer identity, metadata, and return-state from the app's existing domain records for the transaction instead of hard-coded demo products or prices unless the blueprint params explicitly define that catalog.")
		notes = append(notes, "Persist the Checkout Session or underlying PaymentIntent ID with a pending app record, then finalize paid, fulfilled, active, or entitled state from the signed completion webhook rather than the success URL.")
	}
	if strings.Contains(path, "/payment_intents") {
		notes = append(notes, "For PaymentIntents, derive amount, currency, customer identity, metadata, and idempotency from the existing app record; collect and confirm payment with a hosted or client-side Stripe integration, never by passing raw card numbers through server-side API calls.")
		notes = append(notes, "Keep local payment-related app state pending until a signed payment completion event confirms the PaymentIntent.")
		notes = append(notes, "If this is a Connect flow, resolve the connected account from trusted seller, provider, or platform-user state, ensure onboarding and capabilities are ready, and do not accept an arbitrary destination account ID from the client.")
	}
	if strings.Contains(path, "/billing_portal/sessions") {
		notes = append(notes, "Create portal sessions for the authenticated app customer and return to a server-owned route that reloads current subscription state; do not use return URLs as proof that billing state changed.")
	}
	return notes
}

// AsyncEventProductGuidance adds app-state semantics for common Stripe event
// roles until upstream blueprints provide explicit event-role metadata.
func AsyncEventProductGuidance(events []string) []string {
	events = normalizedEvents(events)
	var completion, failure []string
	hasSubscription := false
	hasEntitlement := false
	hasConnectAccount := false
	for _, event := range events {
		logical := logicalStripeEvent(event)
		switch logical {
		case "checkout.session.completed", "checkout.session.async_payment_succeeded", "payment_intent.succeeded", "invoice.paid", "invoice.payment_succeeded":
			completion = append(completion, event)
		case "checkout.session.async_payment_failed", "payment_intent.payment_failed", "invoice.payment_failed":
			failure = append(failure, event)
		}
		if strings.HasPrefix(logical, "customer.subscription.") {
			hasSubscription = true
		}
		if strings.HasPrefix(logical, "entitlements.") {
			hasEntitlement = true
		}
		if strings.HasPrefix(logical, "account.") || strings.Contains(logical, ".capability.") {
			hasConnectAccount = true
		}
	}

	var notes []string
	if len(completion) > 0 {
		notes = append(notes, fmt.Sprintf("Treat %s as durable completion events: find the pending app record via metadata, client_reference_id, customer, subscription, PaymentIntent, or Session ID, then apply fulfillment, inventory, access, paid-state, or entitlement changes idempotently inside the signed handler only.", strings.Join(completion, ", ")))
	}
	if len(failure) > 0 {
		notes = append(notes, fmt.Sprintf("Treat %s as failure signals: keep or move the app record to a recoverable unpaid state and avoid fulfillment or access grants.", strings.Join(failure, ", ")))
	}
	if hasSubscription {
		notes = append(notes, "For subscription events, persist the subscription ID, customer mapping, status, current period, cancellation fields, and plan or price references that the app uses for access decisions.")
	}
	if hasEntitlement {
		notes = append(notes, "For entitlement events, refresh server-side entitlement state and make access checks read that stored state instead of trusting client-side assumptions.")
	}
	if hasConnectAccount {
		notes = append(notes, "For Connect account or capability events, refresh connected-account readiness from Stripe before enabling charges, payouts, or destination transfers in the app.")
	}
	return notes
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

func stepGuidanceContext(step StepInfo) string {
	values := []string{
		string(step.Type),
		step.Key,
		step.Title,
		step.Description,
		step.ReviewPrompt,
		step.ReviewCommand,
	}
	if step.APIRequest != nil {
		values = append(values, step.APIRequest.Path, step.APIRequest.Method, blueprintReferenceSource(step.APIRequest.Params))
	}
	values = append(values, step.Events...)
	return strings.ToLower(strings.Join(values, " "))
}

func containsAny(value string, needles ...string) bool {
	for _, needle := range needles {
		if strings.Contains(value, needle) {
			return true
		}
	}
	return false
}

func hasAPIProductSemantics(semantics *BlueprintSemantics) bool {
	return semantics != nil && (semantics.SourceOfTruth != nil || semantics.PaymentLifecycle != nil || semantics.Connect != nil || semantics.ServerVerification != nil || len(semantics.Assertions) > 0)
}

func hasAsyncProductSemantics(semantics *BlueprintSemantics) bool {
	return semantics != nil && (semantics.PaymentLifecycle != nil || len(semantics.EventRoles) > 0 || semantics.Connect != nil || len(semantics.Assertions) > 0)
}

func logicalStripeEvent(event string) string {
	event = strings.TrimSpace(event)
	if strings.HasPrefix(event, "v1.") {
		return strings.TrimPrefix(event, "v1.")
	}
	return event
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
