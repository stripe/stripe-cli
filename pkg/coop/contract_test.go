package coop

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGenerateIntegrationContractDerivesObligationsFromBlueprint(t *testing.T) {
	bp := &Blueprint{
		ID: "checkout-connect",
		AppRoles: []AppRole{
			{ID: "payable_record", Kind: "domain_record", Required: true},
		},
		Steps: []BlueprintStep{
			{
				StepDefinition: StepDefinition{Key: "setup", Title: "Setup"},
				Nodes: []NodeDefinition{
					{
						Key:   "create-account",
						Type:  NodeAPIRequest,
						Title: "Create account",
						Request: &APIRequest{
							Path:   "/v2/core/accounts",
							Method: "post",
						},
					},
					{
						Key:   "create-checkout",
						Type:  NodeAPIRequest,
						Title: "Create checkout",
						Request: &APIRequest{
							Path:   "/v1/checkout/sessions",
							Method: "post",
							Params: map[string]interface{}{
								"line_items": []map[string]interface{}{{"price": "${node.setup.create-product:id}"}},
							},
						},
					},
					{
						Key:    "complete-checkout",
						Type:   NodeUIComponent,
						Title:  "Complete checkout",
						Events: nil,
					},
					{
						Key:    "webhook",
						Type:   NodeAsyncHandler,
						Title:  "Handle checkout",
						Events: []string{"checkout.session.completed", "v2.core.account[configuration.recipient].capability_status_updated"},
					},
				},
			},
		},
	}

	contract := GenerateIntegrationContract(bp)

	require.NotEmpty(t, contract)
	assert.Contains(t, contract, "For apiRequest nodes, preserve the blueprint method, path, params, and output references; adapt only concrete IDs, URLs, and app-owned values at the app boundary.")
	assert.Contains(t, contract, "Resolve blueprint references from earlier step outputs at runtime instead of creating unrelated Stripe resources: ${node.setup.create-product:id}.")
	assert.Contains(t, contract, "For uiComponent nodes, bind the described user or developer action to the app's existing UI, route, command, or setup surface; do not replace it with a sample-only path.")
	assert.Contains(t, contract, "For asyncHandler nodes, implement signed event handling for the blueprint event set and prove each event's app-visible effect: checkout.session.completed, v2.core.account[configuration.recipient].capability_status_updated.")
	assert.Contains(t, contract, "Because this blueprint starts a payment or billing flow, use the listed async event(s) as server-verified checkpoints before applying dependent durable app state changes: checkout.session.completed, v2.core.account[configuration.recipient].capability_status_updated.")
	assert.Contains(t, contract, "Because this blueprint uses account, capability, or account-link resources, bind the Stripe account owner and readiness state to trusted app state before executing dependent account, capability, money movement, financial-account, issuing, or transfer work represented by later blueprint steps.")
	assert.Contains(t, contract, "When blueprint app roles are present, bind each required role to concrete app code, data, UI, state, or the smallest app-native addition before implementing dependent steps.")
}

func TestGenerateAppMapRequirementsDerivesQuestionsFromBlueprint(t *testing.T) {
	bp := &Blueprint{
		ID: "subscription",
		Steps: []BlueprintStep{
			{
				StepDefinition: StepDefinition{Key: "main", Title: "Main"},
				Nodes: []NodeDefinition{
					{
						Key:   "create-customer",
						Type:  NodeAPIRequest,
						Title: "Create customer",
						Request: &APIRequest{
							Path:   "/v1/customers",
							Method: "post",
						},
					},
					{
						Key:   "create-subscription",
						Type:  NodeAPIRequest,
						Title: "Create subscription",
						Request: &APIRequest{
							Path:   "/v1/subscriptions",
							Method: "post",
						},
					},
					{
						Key:    "handle-invoice",
						Type:   NodeAsyncHandler,
						Title:  "Handle invoice",
						Events: []string{"invoice.payment_succeeded"},
					},
				},
			},
		},
	}

	require.Subset(t, GenerateAppMapRequirements(bp), []string{
		"Identify where money, currency, catalog, line items, and customer identity should come from in this app, using blueprint params when they are explicit.",
		"Identify where a signed webhook or async-event handler belongs, how raw request bodies are supported, and where idempotency or processed-event state can live.",
		"Identify where customer, setup, subscription, entitlement, invoice, or billing state should be persisted and read for later app decisions.",
	})
}

func TestGenerateAcceptanceCriteriaForCheckoutAPI(t *testing.T) {
	criteria := GenerateAcceptanceCriteria(StepInfo{
		Type: NodeAPIRequest,
		APIRequest: &APIRequest{
			Path:   "/v1/checkout/sessions",
			Method: "post",
			Params: map[string]interface{}{
				"mode": "payment",
				"line_items": []map[string]interface{}{
					{"price": "${node.setup.create-product.request:default_price}"},
				},
			},
		},
	})

	assert.Contains(t, criteria, "App code calls the blueprint API target POST /v1/checkout/sessions through the official Stripe SDK or the app's existing Stripe client pattern.")
	assert.Contains(t, criteria, "Runtime request params follow blueprint_step.api_request.params, with placeholders resolved from prior blueprint outputs or trusted app state.")
	assert.Contains(t, criteria, "Every blueprint reference used by this request is resolved from the referenced prior step output at runtime: ${node.setup.create-product.request:default_price}.")
	assert.Contains(t, criteria, "The created Checkout Session is correlated to a current app-owned record or action using a trusted server-side value.")
	assert.Contains(t, criteria, "The success or return URL does not mark durable app payment or billing state complete without server-side verification.")
}

func TestGenerateAcceptanceCriteriaForAsyncHandler(t *testing.T) {
	criteria := GenerateAcceptanceCriteria(StepInfo{
		Type:   NodeAsyncHandler,
		Events: []string{"checkout.session.completed", "v2.core.account[configuration.recipient].capability_status_updated"},
	})

	assert.Contains(t, criteria, "A signed handler verifies the Stripe signature from the raw request body and branches on every blueprint event: checkout.session.completed, v2.core.account[configuration.recipient].capability_status_updated.")
	assert.Contains(t, criteria, "Duplicate delivery is safe: event processing is idempotent for app state and any blueprint-dependent side effects.")
	assert.Contains(t, criteria, "Invalid signatures are rejected during verification.")
	assert.Contains(t, criteria, "For lightweight or v2 event notifications, retrieve the full event or related Stripe object before mutating durable app state.")
}

func TestGenerateAcceptanceCriteriaForUIUsesSemanticsNotFreeText(t *testing.T) {
	criteria := GenerateAcceptanceCriteria(StepInfo{
		Type:        NodeUIComponent,
		Key:         "checkout-return",
		Title:       "Render checkout success",
		Description: "Show the customer that payment completed.",
	})

	assert.Contains(t, criteria, "The described UI behavior is wired into an existing app route, page, command, or setup surface rather than a detached demo.")
	assert.NotContains(t, criteria, "User-facing success, return, or cancel state is rendered from server-verified app or Stripe state as required by blueprint_step.semantics.")

	criteria = GenerateAcceptanceCriteria(StepInfo{
		Type: NodeUIComponent,
		Semantics: &BlueprintSemantics{
			ServerVerification: &ServerVerificationSemantics{Required: true},
		},
	})

	assert.Contains(t, criteria, "User-facing success, return, or cancel state is rendered from server-verified app or Stripe state as required by blueprint_step.semantics.")
}

func TestGenerateAcceptanceCriteriaForContextScan(t *testing.T) {
	criteria := GenerateAcceptanceCriteria(StepInfo{
		Type: NodeTestHelper,
		Key:  "scan-project",
	})

	assert.Contains(t, criteria, "The project scan reports the app facts and blueprint-derived app map needed before implementation.")
	assert.NotContains(t, criteria, "The helper verifies app-visible behavior required by surrounding blueprint steps, not only raw Stripe object creation.")
}
