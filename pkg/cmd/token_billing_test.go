package cmd

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/stripe/stripe-cli/pkg/config"
)

func newTestTokenBillingInitCmd(t *testing.T, serverURL string) *tokenBillingInitCmd {
	t.Helper()
	t.Setenv("STRIPE_API_KEY", "")
	ic := newTokenBillingInitCmd()
	ic.rb.Profile = &config.Profile{APIKey: "sk_test_1234567890abcdef"}
	ic.rb.APIBaseURL = serverURL
	ic.cmd.SetContext(context.Background())
	return ic
}

func TestTokenBillingInitBuildRequestBody(t *testing.T) {
	ic := newTokenBillingInitCmd()
	ic.planName = "Custom AI plan"
	ic.models = []string{"openai/gpt-4o-mini", "anthropic/claude-3-5-sonnet"}
	ic.defaultMarkupPercent = "15.5"
	ic.priceTrackingPreference = "new_customers_only"
	ic.subscriptionFeeAmount = "1000"
	ic.creditGrantPerPeriodAmount = "500"
	ic.interval = "month"

	body := ic.buildRequestBody()

	assert.Equal(t, "Custom AI plan", body["plan_name"])
	assert.Equal(t, []interface{}{"openai/gpt-4o-mini", "anthropic/claude-3-5-sonnet"}, body["models"])
	assert.Equal(t, "15.5", body["default_markup_percent"])
	assert.Equal(t, "new_customers_only", body["price_tracking_preference"])
	assert.Equal(t, "1000", body["subscription_fee_amount"])
	assert.Equal(t, "500", body["credit_grant_per_period_amount"])
	assert.Equal(t, "month", body["interval"])
}

func TestTokenBillingInitCmd_HTTPRequest(t *testing.T) {
	var capturedReq *http.Request
	var capturedBody []byte

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedReq = r
		capturedBody, _ = io.ReadAll(r.Body)
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{
			"status":"initialized",
			"message":"Token Billing is initialized for testmode.",
			"ai_gateway_enabled":true,
			"test_request_completed":false,
			"pricing_configured":true,
			"token_meters_configured":true,
			"zero_credit_rejection_enabled":false,
			"webhook_healthy":false,
			"inference_mode":"not_tested",
			"gateway_endpoint":"https://llm.stripe.com/v1",
			"environment_variables":{
				"STRIPE_TOKEN_BILLING_GATEWAY_URL":"https://llm.stripe.com/v1",
				"STRIPE_TOKEN_BILLING_PRICING_PLAN_ID":"bpp_test_123"
			},
			"resources":{
				"pricing_plan_id":"bpp_test_123",
				"rate_card_id":"rc_test_123",
				"meter_id":"mtr_test_123",
				"settings_id":"tbset_test_123"
			},
			"next_steps":["Run one AI Gateway test request."]
		}`))
	}))
	defer ts.Close()

	ic := newTestTokenBillingInitCmd(t, ts.URL)
	ic.planName = "Custom AI plan"
	ic.models = []string{"openai/gpt-4o-mini", "anthropic/claude-3-5-sonnet"}
	ic.defaultMarkupPercent = "15.5"
	ic.subscriptionFeeAmount = "1000"
	ic.creditGrantPerPeriodAmount = "500"

	var output strings.Builder
	ic.cmd.SetOut(&output)

	err := ic.runTokenBillingInitCmd(ic.cmd, []string{})
	require.NoError(t, err)
	require.NotNil(t, capturedReq)

	assert.Equal(t, http.MethodPost, capturedReq.Method)
	assert.Equal(t, tokenBillingOnboardingInitPath, capturedReq.URL.Path)
	assert.Equal(t, "Bearer sk_test_1234567890abcdef", capturedReq.Header.Get("Authorization"))

	values, err := url.ParseQuery(string(capturedBody))
	require.NoError(t, err)
	assert.Equal(t, "Custom AI plan", values.Get("plan_name"))
	assert.Equal(t, "15.5", values.Get("default_markup_percent"))
	assert.Equal(t, "1000", values.Get("subscription_fee_amount"))
	assert.Equal(t, "500", values.Get("credit_grant_per_period_amount"))
	assert.Equal(t, "month", values.Get("interval"))
	assert.Equal(t, "disabled", values.Get("price_tracking_preference"))
	assert.Equal(t, []string{"openai/gpt-4o-mini", "anthropic/claude-3-5-sonnet"}, values["models[]"])

	assert.Contains(t, output.String(), "Token Billing initialized")
	assert.Contains(t, output.String(), "STRIPE_TOKEN_BILLING_GATEWAY_URL=https://llm.stripe.com/v1")
	assert.Contains(t, output.String(), "pricing_plan_id=bpp_test_123")
	assert.Contains(t, output.String(), "[x] Pricing configured")
	assert.Contains(t, output.String(), "Run one AI Gateway test request.")
}

func TestTokenBillingInitCmd_DryRun(t *testing.T) {
	ic := newTestTokenBillingInitCmd(t, "https://api.stripe.com")
	ic.rb.DryRun = true

	var output strings.Builder
	ic.cmd.SetOut(&output)

	err := ic.runTokenBillingInitCmd(ic.cmd, []string{})
	require.NoError(t, err)

	var body map[string]interface{}
	require.NoError(t, json.Unmarshal([]byte(output.String()), &body))
	dryRun := body["dry_run"].(map[string]interface{})
	assert.Equal(t, http.MethodPost, dryRun["method"])
	assert.Equal(t, "https://api.stripe.com/v1/token_billing/onboarding/init", dryRun["url"])
}

func TestTokenBillingInitCmd_RequiresModel(t *testing.T) {
	ic := newTestTokenBillingInitCmd(t, "https://api.stripe.com")
	ic.models = []string{}

	err := ic.runTokenBillingInitCmd(ic.cmd, []string{})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "at least one --model is required")
}
