package coop

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGenerateAPIRequestGuidanceUsesBlueprintParamsAsCanonical(t *testing.T) {
	guidance := GenerateAPIRequestGuidance(&APIRequest{
		Path:   "/v1/checkout/sessions",
		Method: "post",
		Params: map[string]interface{}{
			"mode": "payment",
		},
	})

	assert.Contains(t, guidance, "POST /v1/checkout/sessions")
	assert.Contains(t, guidance, "blueprint_step.api_request.params are canonical")
	assert.Contains(t, guidance, "hosted Checkout")
	assert.Contains(t, guidance, "Fallback guidance for thin blueprints")
}

func TestGenerateAPIRequestGuidanceSteersCheckoutToAppDomainLifecycle(t *testing.T) {
	guidance := GenerateAPIRequestGuidance(&APIRequest{
		Path:   "/v1/checkout/sessions",
		Method: "post",
		Params: map[string]interface{}{
			"mode": "payment",
		},
	})

	assert.Contains(t, guidance, "derive Checkout line items, amounts, customer identity, metadata, and return-state from the app's existing domain records")
	assert.Contains(t, guidance, "instead of hard-coded demo products or prices")
	assert.Contains(t, guidance, "Persist the Checkout Session or underlying PaymentIntent ID with a pending app record")
	assert.Contains(t, guidance, "rather than the success URL")
}

func TestGenerateAPIRequestGuidanceSteersPaymentIntentAndConnectSafety(t *testing.T) {
	guidance := GenerateAPIRequestGuidance(&APIRequest{
		Path:   "/v1/payment_intents",
		Method: "post",
	})

	assert.Contains(t, guidance, "derive amount, currency, customer identity, metadata, and idempotency from the existing app record")
	assert.Contains(t, guidance, "never by passing raw card numbers")
	assert.Contains(t, guidance, "signed payment completion event")
	assert.Contains(t, guidance, "resolve the connected account from trusted seller")
	assert.Contains(t, guidance, "do not accept an arbitrary destination account ID from the client")
}

func TestGenerateAPIRequestGuidancePreservesBlueprintReferences(t *testing.T) {
	guidance := GenerateAPIRequestGuidance(&APIRequest{
		Path:   "/v1/products/${node.main.create-product:id}/features",
		Method: "post",
		Params: map[string]interface{}{
			"entitlement_feature": "${node.main.create-feature:id}",
		},
	})

	assert.Contains(t, guidance, "${node.main.create-product:id}")
	assert.Contains(t, guidance, "${node.main.create-feature:id}")
	assert.Contains(t, guidance, "resolve them from prior steps")
}

func TestGenerateAPIRequestGuidanceFlagsEndpointOnlyMutatingCalls(t *testing.T) {
	guidance := GenerateAPIRequestGuidance(&APIRequest{
		Path:   "/v1/billing/meters",
		Method: "post",
	})

	assert.Contains(t, guidance, "POST /v1/billing/meters")
	assert.Contains(t, guidance, "endpoint and method only")
	assert.Contains(t, guidance, "exact app code path and params")
}

func TestGenerateAsyncHandlerGuidanceMentionsKnownTriggerGaps(t *testing.T) {
	guidance := GenerateAsyncHandlerGuidance([]string{
		"entitlements.active_entitlement_summary.updated",
		"test_helpers.test_clock.ready",
	})

	assert.Contains(t, guidance, "signed webhook/event handler")
	assert.Contains(t, guidance, "entitlements.active_entitlement_summary.updated, test_helpers.test_clock.ready")
	assert.Contains(t, guidance, "Fallback guidance for thin blueprints")
	assert.Contains(t, guidance, "might not support `stripe trigger entitlements.active_entitlement_summary.updated`")
	assert.Contains(t, guidance, "test clock readiness")
}

func TestGenerateAsyncHandlerGuidanceDescribesDurableCompletionState(t *testing.T) {
	guidance := GenerateAsyncHandlerGuidance([]string{
		"checkout.session.completed",
		"payment_intent.payment_failed",
		"customer.subscription.updated",
		"entitlements.active_entitlement_summary.updated",
		"account.updated",
	})

	assert.Contains(t, guidance, "Treat checkout.session.completed as durable completion events")
	assert.Contains(t, guidance, "apply fulfillment, inventory, access, paid-state, or entitlement changes idempotently inside the signed handler only")
	assert.Contains(t, guidance, "payment_intent.payment_failed")
	assert.Contains(t, guidance, "recoverable unpaid state")
	assert.Contains(t, guidance, "persist the subscription ID")
	assert.Contains(t, guidance, "refresh server-side entitlement state")
	assert.Contains(t, guidance, "refresh connected-account readiness")
}

func TestGenerateStepGuidanceUsesBlueprintStepFields(t *testing.T) {
	guidance := GenerateStepGuidance(StepInfo{
		Number:        4,
		Key:           "create-checkout-session",
		Title:         "Create a checkout session",
		Type:          NodeAPIRequest,
		Description:   "Create a hosted Checkout Session from the saved price.",
		ReviewPrompt:  "Confirm Checkout uses the saved price ID.",
		ReviewCommand: "npm test",
		APIRequest: &APIRequest{
			Path:   "/v1/checkout/sessions",
			Method: "post",
			Params: map[string]interface{}{
				"line_items": []map[string]interface{}{{"price": "${node.setup.create-product:default_price}"}},
			},
		},
	})

	assert.Contains(t, guidance, "Follow blueprint step 4 (create-checkout-session)")
	assert.Contains(t, guidance, "Create a hosted Checkout Session from the saved price.")
	assert.Contains(t, guidance, "Confirm Checkout uses the saved price ID.")
	assert.Contains(t, guidance, "npm test")
	assert.Contains(t, guidance, "${node.setup.create-product:default_price}")
	assert.Contains(t, guidance, "sdk_example")
	assert.Contains(t, guidance, "generated SDK translation")
}

func TestGenerateStepGuidanceCompilesStructuredBlueprintSemantics(t *testing.T) {
	guidance := GenerateStepGuidance(StepInfo{
		Number: 2,
		Key:    "create-checkout-session",
		Title:  "Create Checkout Session",
		Type:   NodeAPIRequest,
		APIRequest: &APIRequest{
			Path:   "/v1/checkout/sessions",
			Method: "post",
		},
		Semantics: &BlueprintSemantics{
			SourceOfTruth: &SourceOfTruthSemantics{
				Amount:    "app_domain",
				LineItems: "app_domain",
				Customer:  "authenticated_user",
			},
			PaymentLifecycle: &PaymentLifecycleSemantics{
				StartsPayment:                    true,
				CompletionEvent:                  "checkout.session.completed",
				PendingState:                     "pending",
				CompletedState:                   "paid",
				FulfillmentRequiresSignedWebhook: true,
			},
			Connect: &ConnectSemantics{
				RequiresConnectedAccount: true,
				ConnectedAccountOwner:    "seller",
				OnboardingRequired:       true,
				CapabilityGate:           "charges_enabled",
			},
			ServerVerification: &ServerVerificationSemantics{
				Required:    true,
				StateSource: "server",
				Reason:      "success page is not proof of payment",
			},
			Assertions: []string{"Checkout uses the app transaction total"},
		},
	})

	assert.Contains(t, guidance, "Blueprint source-of-truth semantics are canonical")
	assert.Contains(t, guidance, "amount=app_domain")
	assert.Contains(t, guidance, "Blueprint payment lifecycle semantics are canonical")
	assert.Contains(t, guidance, "completion_event=checkout.session.completed")
	assert.Contains(t, guidance, "fulfillment_requires_signed_webhook=true")
	assert.Contains(t, guidance, "Blueprint Connect semantics are canonical")
	assert.Contains(t, guidance, "connected_account_owner=seller")
	assert.Contains(t, guidance, "Blueprint server-verification semantics are canonical")
	assert.Contains(t, guidance, "Blueprint semantic assertions are acceptance criteria")
	assert.NotContains(t, guidance, "hard-coded demo products or prices")
}

func TestGenerateStepGuidanceCompilesStructuredEventRoles(t *testing.T) {
	guidance := GenerateStepGuidance(StepInfo{
		Number: 3,
		Key:    "handle-webhooks",
		Title:  "Handle webhooks",
		Type:   NodeAsyncHandler,
		Events: []string{"checkout.session.completed"},
		Semantics: &BlueprintSemantics{
			EventRoles: []EventRoleSemantics{
				{
					Event:          "checkout.session.completed",
					Role:           "payment_completion",
					StateUpdate:    "mark_transaction_paid",
					RequiresLookup: true,
				},
			},
		},
	})

	assert.Contains(t, guidance, "Blueprint event roles are canonical")
	assert.Contains(t, guidance, "checkout.session.completed:payment_completion->mark_transaction_paid(lookup_required)")
	assert.Contains(t, guidance, "rather than replacing events with lookup-only code")
	assert.NotContains(t, guidance, "Treat checkout.session.completed as durable completion events")
}

func TestGenerateStepGuidanceUsesWebhookExampleAsEventTranslation(t *testing.T) {
	guidance := GenerateStepGuidance(StepInfo{
		Number:       6,
		Key:          "handle-checkout-completed",
		Title:        "Handle checkout.session.completed",
		Type:         NodeAsyncHandler,
		ReviewPrompt: "Confirm fulfillment happens after signature verification.",
		Events:       []string{"checkout.session.completed"},
	})

	assert.Contains(t, guidance, "blueprint_step.events")
	assert.Contains(t, guidance, "webhook_example")
	assert.Contains(t, guidance, "generated handler translation")
	assert.Contains(t, guidance, "without dropping or renaming blueprint events")
}

func TestGenerateStepGuidanceSteersPaymentUIIntoExistingAppFlow(t *testing.T) {
	guidance := GenerateStepGuidance(StepInfo{
		Number:       5,
		Key:          "add-success-page",
		Title:        "Add a Checkout success page",
		Type:         NodeUIComponent,
		Description:  "Render the return page after payment redirect.",
		ReviewPrompt: "Confirm the success page reflects the completed payment.",
	})

	assert.Contains(t, guidance, "passing the current app record identity to the server endpoint")
	assert.Contains(t, guidance, "Fallback guidance for thin blueprints")
	assert.Contains(t, guidance, "do not create a separate sample-only payment path")
	assert.Contains(t, guidance, "domain flow for the thing being paid for or subscribed to")
	assert.Contains(t, guidance, "server-verified state tied to the current user and Stripe IDs")
	assert.Contains(t, guidance, "do not treat URL query params as proof")
}

func TestGenerateStepGuidanceSteersPaymentVerificationToAppLifecycle(t *testing.T) {
	guidance := GenerateStepGuidance(StepInfo{
		Number:       7,
		Key:          "test-payment-flow",
		Title:        "Test the payment flow",
		Type:         NodeTestHelper,
		Description:  "Run through Checkout and verify fulfillment.",
		ReviewPrompt: "Confirm the local order is paid after checkout.session.completed.",
	})

	assert.Contains(t, guidance, "Verify the app-level lifecycle")
	assert.Contains(t, guidance, "stay pending before the signed event")
	assert.Contains(t, guidance, "paid, active, fulfilled, or entitled state after the signed event")
}

func TestGenerateStructuredRequirementsForAPIRequest(t *testing.T) {
	step := StepInfo{
		Number: 2,
		Key:    "create-payment-intent",
		Title:  "Create PaymentIntent",
		Type:   NodeAPIRequest,
		APIRequest: &APIRequest{
			Path:   "/v1/payment_intents",
			Method: "post",
			Params: map[string]interface{}{"amount": "${app.amount_source}"},
		},
		AppRoles: []AppRole{
			{ID: "payable_record", Kind: "domain_record", Required: true},
		},
	}

	implementation := GenerateImplementationRequirements(step)
	verification := GenerateVerificationRequirements(step)
	warnings := GenerateQualityWarnings(step)

	assert.Contains(t, implementation, "Bind blueprint_step.app_roles to concrete app code, data, UI, or state before implementing this step; if a required role is missing, add the smallest app-native implementation and report it.")
	assert.Contains(t, implementation, "Use blueprint_step.api_request as the canonical API target: POST /v1/payment_intents.")
	assert.Contains(t, verification, "Verify that the app route, service, job, or handler creates or retrieves the Stripe object; direct Stripe CLI/API calls alone are not sufficient.")
	assert.Contains(t, verification, "Report the app code path and the resolved request params used at runtime, especially blueprint references and app-role bindings.")
	assert.Contains(t, warnings, "If you add or change persistent fields/tables and the project uses a migration system, add the matching migration artifact.")
	assert.Contains(t, warnings, "Preserve the existing package manager and lockfile format; do not rewrite or downgrade lockfiles while adding Stripe dependencies.")
	assert.Contains(t, warnings, "If the app has mock checkout, mock payment, or fake success paths, remove or gate bypasses so production app state cannot skip Stripe.")
}

func TestGenerateStructuredRequirementsForWebhook(t *testing.T) {
	step := StepInfo{
		Number: 3,
		Key:    "handle-payment",
		Title:  "Handle payment webhook",
		Type:   NodeAsyncHandler,
		Events: []string{"payment_intent.succeeded"},
	}

	implementation := GenerateImplementationRequirements(step)
	verification := GenerateVerificationRequirements(step)
	warnings := GenerateQualityWarnings(step)

	assert.Contains(t, implementation, "Implement a signed webhook or async-event handler using the raw request body and the official SDK signature helper.")
	assert.Contains(t, implementation, "Branch on every blueprint_step.events value without dropping, renaming, or replacing events with lookup-only work.")
	assert.Contains(t, verification, "Verify webhook handling through the app endpoint with signature verification enabled; use stripe listen/trigger or a documented fallback when trigger coverage is unavailable.")
	assert.Contains(t, verification, "Verify the app state or side effect produced by each handled event, not only that the event was received.")
	assert.Contains(t, warnings, "Make webhook side effects idempotent so repeated Stripe deliveries do not duplicate fulfillment, inventory, email, access, or payout work.")
}

func TestBlueprintReferencesReturnsSortedUniqueTokens(t *testing.T) {
	refs := BlueprintReferences(
		"/v1/invoices/${node.main.create-invoice:id}",
		map[string]interface{}{
			"customer": "${node.main.create-customer:id}",
			"invoice":  "${node.main.create-invoice:id}",
		},
	)

	assert.Equal(t, []string{
		"${node.main.create-customer:id}",
		"${node.main.create-invoice:id}",
	}, refs)
}
