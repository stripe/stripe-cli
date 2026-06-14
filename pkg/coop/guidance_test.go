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
	assert.Contains(t, guidance, "request params in this response are canonical")
	assert.Contains(t, guidance, "hosted Checkout")
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
	assert.Contains(t, guidance, "might not support `stripe trigger entitlements.active_entitlement_summary.updated`")
	assert.Contains(t, guidance, "test clock readiness")
}
