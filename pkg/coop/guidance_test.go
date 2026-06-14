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
	assert.Contains(t, guidance, "might not support `stripe trigger entitlements.active_entitlement_summary.updated`")
	assert.Contains(t, guidance, "test clock readiness")
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
