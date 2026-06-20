package coop

import (
	"fmt"
	"sort"
	"strings"
)

// GenerateIntegrationContract returns blueprint-derived obligations that apply
// across the whole integration. The contract is intentionally limited to facts
// available in the blueprint: node types, API requests, references, events,
// app roles, and structured semantics.
func GenerateIntegrationContract(bp *Blueprint) []string {
	if bp == nil {
		return nil
	}
	signals := blueprintSignalsFor(bp)
	var contract []string
	contract = append(contract, "Implement the blueprint as app behavior, not a detached demo: each apiRequest, uiComponent, and asyncHandler node should produce the corresponding app behavior or setup required by the blueprint.")
	if signals.hasAPI {
		contract = append(contract, "For apiRequest nodes, preserve the blueprint method, path, params, and output references; adapt only concrete IDs, URLs, and app-owned values at the app boundary.")
	}
	if len(signals.refs) > 0 {
		contract = append(contract, fmt.Sprintf("Resolve blueprint references from earlier step outputs at runtime instead of creating unrelated Stripe resources: %s.", strings.Join(signals.refs, ", ")))
	}
	if signals.hasUI {
		contract = append(contract, "For uiComponent nodes, bind the described user or developer action to the app's existing UI, route, command, or setup surface; do not replace it with a sample-only path.")
	}
	if len(signals.events) > 0 {
		contract = append(contract, fmt.Sprintf("For asyncHandler nodes, implement signed event handling for the blueprint event set and prove each event's app-visible effect: %s.", strings.Join(signals.events, ", ")))
	}
	if signals.startsPaymentOrBilling() {
		if len(signals.events) > 0 {
			contract = append(contract, fmt.Sprintf("Because this blueprint starts a payment or billing flow, use the listed async event(s) as server-verified checkpoints before applying dependent durable app state changes: %s.", strings.Join(signals.events, ", ")))
		} else {
			contract = append(contract, "Because this blueprint starts a payment or billing flow, do not treat creation, redirect, or return URLs as durable completion; verify the resulting Stripe state server-side before mutating durable app state.")
		}
	}
	if signals.usesAccountOrCapability() {
		contract = append(contract, "Because this blueprint uses account, capability, or account-link resources, bind the Stripe account owner and readiness state to trusted app state before executing dependent account, capability, money movement, financial-account, issuing, or transfer work represented by later blueprint steps.")
	}
	if signals.usesReusableCustomerOrSubscription() {
		contract = append(contract, "Because this blueprint uses reusable customer, setup, subscription, entitlement, or invoice state, identify where the app persists the relevant Stripe IDs and where later access or billing decisions read them.")
	}
	if signals.hasSemantics {
		contract = append(contract, "When blueprint semantics are present, treat them as canonical product intent and prefer them over inferred fallback guidance.")
	}
	if signals.hasAppRoles {
		contract = append(contract, "When blueprint app roles are present, bind each required role to concrete app code, data, UI, state, or the smallest app-native addition before implementing dependent steps.")
	}
	return dedupeStrings(contract)
}

// GenerateAppMapRequirements returns concrete scan prompts for the prepended
// project-context step. These prompts are derived from the blueprint and ask the
// agent to bind unknown app facts instead of hardcoding app-specific knowledge.
func GenerateAppMapRequirements(bp *Blueprint) []string {
	if bp == nil {
		return nil
	}
	signals := blueprintSignalsFor(bp)
	requirements := []string{
		"Identify language/framework, frontend/backend boundaries, package manager and lockfile, migration tooling, env/config pattern, existing Stripe SDK usage, existing webhook routes, mock payment paths, and test/Docker commands.",
	}
	if signals.startsPaymentOrBilling() || signals.hasUI {
		requirements = append(requirements, "Identify the app-owned record, action, or state that this blueprint affects, plus the route/UI surface that starts or displays the flow.")
	}
	if signals.usesMoneyOrCatalog() {
		requirements = append(requirements, "Identify where money, currency, catalog, line items, and customer identity should come from in this app, using blueprint params when they are explicit.")
	}
	if signals.usesHostedRedirect() {
		requirements = append(requirements, "Identify the app route or UI surface that starts the hosted redirect flow and where return or cancel URLs should land.")
	}
	if len(signals.events) > 0 {
		requirements = append(requirements, "Identify where a signed webhook or async-event handler belongs, how raw request bodies are supported, and where idempotency or processed-event state can live.")
	}
	if signals.usesAccountOrCapability() {
		requirements = append(requirements, "Identify the app actor that owns each Stripe account-like resource and where account IDs, onboarding links, and readiness/capability state should be stored or read.")
	}
	if signals.usesReusableCustomerOrSubscription() {
		requirements = append(requirements, "Identify where customer, setup, subscription, entitlement, invoice, or billing state should be persisted and read for later app decisions.")
	}
	if signals.usesIssuingOrFinancialAccounts() {
		requirements = append(requirements, "Identify the app owner and state for financial accounts, cardholders, cards, funding, authorizations, and captures represented by the blueprint.")
	}
	return dedupeStrings(requirements)
}

// GenerateAcceptanceCriteria returns step-level completion criteria for the
// agent. The criteria are derived only from the current blueprint step.
func GenerateAcceptanceCriteria(step StepInfo) []string {
	var criteria []string
	switch step.Type {
	case NodeAPIRequest:
		if step.APIRequest == nil {
			break
		}
		method := strings.ToUpper(strings.TrimSpace(step.APIRequest.Method))
		path := strings.TrimSpace(step.APIRequest.Path)
		criteria = append(criteria, fmt.Sprintf("App code calls the blueprint API target %s %s through the official Stripe SDK or the app's existing Stripe client pattern.", method, path))
		if requestHasParams(step.APIRequest.Params) {
			criteria = append(criteria, "Runtime request params follow blueprint_step.api_request.params, with placeholders resolved from prior blueprint outputs or trusted app state.")
		} else if isMutatingMethod(step.APIRequest.Method) {
			criteria = append(criteria, "Any required request params not present in the blueprint are chosen from the step intent, earlier blueprint outputs, app state, and official Stripe docs, then reported explicitly.")
		}
		if refs := BlueprintReferences(step.APIRequest.Path, step.APIRequest.Params); len(refs) > 0 {
			criteria = append(criteria, "Every blueprint reference used by this request is resolved from the referenced prior step output at runtime: "+strings.Join(refs, ", ")+".")
		}
		criteria = append(criteria, apiRequestAcceptanceCriteria(step.APIRequest)...)
	case NodeAsyncHandler:
		events := normalizedEvents(step.Events)
		if len(events) > 0 {
			criteria = append(criteria, "A signed handler verifies the Stripe signature from the raw request body and branches on every blueprint event: "+strings.Join(events, ", ")+".")
			criteria = append(criteria, "Verification proves each listed event changes or refreshes the app state or side effect the later blueprint flow depends on.")
		} else {
			criteria = append(criteria, "A signed handler verifies the Stripe signature from the raw request body and reports the handled event set.")
		}
		criteria = append(criteria,
			"Duplicate delivery is safe: event processing is idempotent for app state and any blueprint-dependent side effects.",
			"Invalid signatures are rejected during verification.",
		)
		if hasThinLookingEvent(events) {
			criteria = append(criteria, "For lightweight or v2 event notifications, retrieve the full event or related Stripe object before mutating durable app state.")
		}
	case NodeUIComponent:
		criteria = append(criteria, "The described UI behavior is wired into an existing app route, page, command, or setup surface rather than a detached demo.")
		if semanticsRequiresServerVerification(step.Semantics) {
			criteria = append(criteria, "User-facing success, return, or cancel state is rendered from server-verified app or Stripe state as required by blueprint_step.semantics.")
		}
	case NodeTestHelper:
		if step.Key == "scan-project" {
			criteria = append(criteria, "The project scan reports the app facts and blueprint-derived app map needed before implementation.")
		} else {
			criteria = append(criteria, "The helper verifies app-visible behavior required by surrounding blueprint steps, not only raw Stripe object creation.")
		}
	case NodeCLICommand:
		criteria = append(criteria, "The reported command output corresponds to the CLI command or command family described by this blueprint step.")
	case NodeDashboard, NodeSetUpWebhooks:
		criteria = append(criteria, "The reported setup maps the blueprint requirement to concrete app or Stripe configuration.")
	}
	if len(step.AppRoles) > 0 {
		criteria = append(criteria, "Every required blueprint_step.app_roles entry used by this step has a reported app binding or the smallest app-native addition needed.")
	}
	if step.Semantics != nil {
		criteria = append(criteria, "Any blueprint_step.semantics values are satisfied directly or explicitly called out if the app cannot support them yet.")
	}
	return dedupeStrings(criteria)
}

func apiRequestAcceptanceCriteria(req *APIRequest) []string {
	if req == nil {
		return nil
	}
	path := strings.ToLower(req.Path)
	var criteria []string
	if strings.Contains(path, "/checkout/sessions") {
		criteria = append(criteria,
			"The created Checkout Session is correlated to a current app-owned record or action using a trusted server-side value.",
			"The app stores or passes through the Checkout Session or resulting payment/subscription identifier needed by later blueprint steps.",
			"The success or return URL does not mark durable app payment or billing state complete without server-side verification.",
		)
	}
	if strings.Contains(path, "/payment_intents") {
		criteria = append(criteria,
			"Any amount, currency, customer identity, metadata, and idempotency values used by this request come from blueprint params or trusted app state.",
			"The server does not collect or pass raw card numbers to Stripe APIs.",
		)
	}
	if strings.Contains(path, "/subscriptions") || strings.Contains(path, "/entitlements") {
		criteria = append(criteria, "Subscription, entitlement, customer, product, or price IDs needed by later access or billing decisions are persisted or explicitly justified as transient.")
	}
	if strings.Contains(path, "/invoices") {
		criteria = append(criteria, "Invoice identifiers, hosted invoice URLs, and paid/unpaid state needed by the app are persisted, displayed, or refreshed server-side as appropriate.")
	}
	if strings.Contains(path, "/accounts") || strings.Contains(path, "account_links") {
		criteria = append(criteria, "Account identifiers, account-link URLs, and readiness/capability state are tied to trusted app-owned account context, not arbitrary client input.")
	}
	if strings.Contains(path, "/billing/") || strings.Contains(path, "/v2/billing/") {
		criteria = append(criteria, "Billing setup resources and identifiers that later nodes reuse are persisted or made discoverable by stable app-owned lookup state.")
	}
	if strings.Contains(path, "/issuing/") || strings.Contains(path, "financial_accounts") || strings.Contains(path, "money_management") || strings.Contains(path, "/treasury/") {
		criteria = append(criteria, "Financial account, funding, cardholder, card, authorization, or capture state is tied to the app owner represented by the blueprint flow.")
	}
	return criteria
}

type blueprintSignals struct {
	hasAPI       bool
	hasUI        bool
	hasSemantics bool
	hasAppRoles  bool
	paths        []string
	events       []string
	refs         []string
}

func blueprintSignalsFor(bp *Blueprint) blueprintSignals {
	var signals blueprintSignals
	if bp == nil {
		return signals
	}
	if bp.Semantics != nil {
		signals.hasSemantics = true
	}
	if len(bp.AppRoles) > 0 {
		signals.hasAppRoles = true
	}
	refs := map[string]bool{}
	events := map[string]bool{}
	paths := map[string]bool{}
	for _, ch := range bp.Chapters {
		if ch.Semantics != nil {
			signals.hasSemantics = true
		}
		if len(ch.AppRoles) > 0 {
			signals.hasAppRoles = true
		}
		for _, n := range ch.Nodes {
			if n.Semantics != nil {
				signals.hasSemantics = true
			}
			if len(n.AppRoles) > 0 {
				signals.hasAppRoles = true
			}
			switch n.Type {
			case NodeAPIRequest:
				signals.hasAPI = true
			case NodeUIComponent:
				signals.hasUI = true
			}
			if n.Request != nil {
				if path := strings.TrimSpace(n.Request.Path); path != "" {
					paths[path] = true
				}
				for _, ref := range BlueprintReferences(n.Request.Path, n.Request.Params) {
					refs[ref] = true
				}
			}
			for _, event := range n.Events {
				event = strings.TrimSpace(event)
				if event != "" {
					events[event] = true
				}
			}
		}
	}
	signals.paths = sortedMapKeys(paths)
	signals.events = sortedMapKeys(events)
	signals.refs = sortedMapKeys(refs)
	return signals
}

func (s blueprintSignals) startsPaymentOrBilling() bool {
	return s.hasPathContaining("/checkout/sessions", "/payment_intents", "/invoices", "/subscriptions") ||
		s.hasEventContaining("payment_intent.", "checkout.session.", "invoice.", "customer.subscription.")
}

func (s blueprintSignals) usesMoneyOrCatalog() bool {
	return s.startsPaymentOrBilling() ||
		s.hasPathContaining("/products", "/prices", "/invoiceitems", "/billing/", "/v2/billing/")
}

func (s blueprintSignals) usesHostedRedirect() bool {
	return s.hasPathContaining("/checkout/sessions", "/billing_portal/sessions", "account_links") ||
		s.hasEventContaining("checkout.session.")
}

func (s blueprintSignals) usesAccountOrCapability() bool {
	return s.hasPathContaining("/accounts", "account_links") ||
		s.hasEventContaining("account", "capability")
}

func (s blueprintSignals) usesReusableCustomerOrSubscription() bool {
	return s.hasPathContaining("/customers", "/setup_intents", "/subscriptions", "/entitlements", "/invoices", "/billing/", "/v2/billing/") ||
		s.hasEventContaining("customer.", "subscription", "entitlements.", "invoice.", "billing.")
}

func (s blueprintSignals) usesIssuingOrFinancialAccounts() bool {
	return s.hasPathContaining("/issuing/", "financial_accounts", "/treasury/", "money_management") ||
		s.hasEventContaining("financial_account", "inbound_transfer", "issuing.")
}

func (s blueprintSignals) hasPathContaining(parts ...string) bool {
	for _, path := range s.paths {
		path = strings.ToLower(path)
		for _, part := range parts {
			if strings.Contains(path, strings.ToLower(part)) {
				return true
			}
		}
	}
	return false
}

func (s blueprintSignals) hasEventContaining(parts ...string) bool {
	for _, event := range s.events {
		event = strings.ToLower(event)
		for _, part := range parts {
			if strings.Contains(event, strings.ToLower(part)) {
				return true
			}
		}
	}
	return false
}

func hasThinLookingEvent(events []string) bool {
	for _, event := range events {
		event = strings.TrimSpace(event)
		if strings.HasPrefix(event, "v2.") || strings.Contains(event, "[") {
			return true
		}
	}
	return false
}

func sortedMapKeys(values map[string]bool) []string {
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

func semanticsRequiresServerVerification(semantics *BlueprintSemantics) bool {
	if semantics == nil {
		return false
	}
	if semantics.ServerVerification != nil && semantics.ServerVerification.Required {
		return true
	}
	return semantics.PaymentLifecycle != nil && semantics.PaymentLifecycle.FulfillmentRequiresSignedWebhook
}

func dedupeStrings(values []string) []string {
	seen := map[string]bool{}
	deduped := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" || seen[value] {
			continue
		}
		seen[value] = true
		deduped = append(deduped, value)
	}
	return deduped
}
