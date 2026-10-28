package cmd

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/stripe/stripe-cli/pkg/errorcategory"
	"github.com/stripe/stripe-cli/pkg/stripe"
)

const integHlthAlertsBody = `{
  "alerts": [
    {
      "health_insight_id": "hi_123",
      "predicate_name": "pinned_payment_method_types",
      "compartment_id": "cmpt_123",
      "created_at": "2026-10-01T00:00:00.000Z",
      "last_updated": "2026-10-20T12:00:00.000Z",
      "status": "open",
      "doctor_topic": "dpm",
      "health_insight": {
        "summary": "Your integration pins payment_method_types",
        "severity": "warning",
        "why_is_it_important": "New payment methods won't show up at checkout.",
        "recommended_action": {"title": "Use dynamic payment methods.", "description": "Remove payment_method_types."}
      }
    }
  ],
  "next_page_url": null
}`

type integHlthRequest struct {
	Method        string
	Path          string
	Query         string
	Auth          string
	Version       string
	StripeAccount string
	StripeContext string
}

func newIntegHlthServer(t *testing.T, statusCode int, body string, recs *[]integHlthRequest) *httptest.Server {
	t.Helper()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if recs != nil {
			*recs = append(*recs, integHlthRequest{
				Method:        r.Method,
				Path:          r.URL.Path,
				Query:         r.URL.RawQuery,
				Auth:          r.Header.Get("Authorization"),
				Version:       r.Header.Get("Stripe-Version"),
				StripeAccount: r.Header.Get("Stripe-Account"),
				StripeContext: r.Header.Get("Stripe-Context"),
			})
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(statusCode)
		_, _ = w.Write([]byte(body))
	}))
	t.Cleanup(server.Close)
	return server
}

func TestFetchIntegHlthAlerts_SendsPinnedVersionAndParsesAlerts(t *testing.T) {
	var recs []integHlthRequest
	server := newIntegHlthServer(t, http.StatusOK, integHlthAlertsBody, &recs)

	resp, err := fetchIntegHlthAlerts(context.Background(), server.URL, stripe.NewAPIKeyCredentials("sk_test_123"), "", 5)
	require.NoError(t, err)

	require.Len(t, recs, 1)
	assert.Equal(t, http.MethodGet, recs[0].Method)
	assert.Equal(t, "/v2/integ_hlth/alerts", recs[0].Path)
	assert.Equal(t, "limit=5", recs[0].Query)
	assert.Equal(t, "Bearer sk_test_123", recs[0].Auth)
	assert.Equal(t, integHlthAPIVersion, recs[0].Version)

	require.Len(t, resp.Alerts, 1)
	alert := resp.Alerts[0]
	assert.Equal(t, "pinned_payment_method_types", alert.PredicateName)
	assert.Equal(t, "open", alert.Status)
	require.NotNil(t, alert.DoctorTopic)
	assert.Equal(t, "dpm", *alert.DoctorTopic)
	require.NotNil(t, alert.HealthInsight)
	assert.Equal(t, "warning", *alert.HealthInsight.Severity)
}

func TestFetchIntegHlthAlerts_OmitsLimitWhenUnset(t *testing.T) {
	var recs []integHlthRequest
	server := newIntegHlthServer(t, http.StatusOK, `{"alerts": []}`, &recs)

	_, err := fetchIntegHlthAlerts(context.Background(), server.URL, stripe.NewAPIKeyCredentials("sk_test_123"), "", 0)
	require.NoError(t, err)

	require.Len(t, recs, 1)
	assert.Empty(t, recs[0].Query)
}

func TestFetchIntegHlthAlerts_ScopesOAKContextToConnectedAccount(t *testing.T) {
	var recs []integHlthRequest
	server := newIntegHlthServer(t, http.StatusOK, `{"alerts": []}`, &recs)

	creds := stripe.NewOAKCredentials("oak_123", "acct_platform", true)
	_, err := fetchIntegHlthAlerts(context.Background(), server.URL, creds, "acct_connected", 0)
	require.NoError(t, err)

	require.Len(t, recs, 1)
	assert.Equal(t, "Bearer oak_123", recs[0].Auth)
	assert.Equal(t, "acct_platform/acct_connected", recs[0].StripeAccount)
	assert.Empty(t, recs[0].StripeContext)
}

func TestFetchIntegHlthAlerts_Unauthorized(t *testing.T) {
	server := newIntegHlthServer(t, http.StatusUnauthorized, `{"error": {"type": "invalid_request_error"}}`, nil)

	_, err := fetchIntegHlthAlerts(context.Background(), server.URL, stripe.NewAPIKeyCredentials("sk_test_123"), "", 0)
	require.Error(t, err)
	category, ok := errorcategory.Get(err)
	require.True(t, ok)
	assert.Equal(t, errorcategory.Auth, category)
	assert.Contains(t, err.Error(), "stripe login")
}

func TestFetchIntegHlthAlerts_NonOKStatus(t *testing.T) {
	server := newIntegHlthServer(t, http.StatusNotFound, `{"error": {"code": "not_found"}}`, nil)

	_, err := fetchIntegHlthAlerts(context.Background(), server.URL, stripe.NewAPIKeyCredentials("sk_test_123"), "", 0)
	require.Error(t, err)
	category, ok := errorcategory.Get(err)
	require.True(t, ok)
	assert.Equal(t, errorcategory.API, category)
	assert.Contains(t, err.Error(), "status 404")
}

func TestFetchIntegHlthAlerts_TestmodeUnsupportedSuggestsLive(t *testing.T) {
	server := newIntegHlthServer(t, http.StatusBadRequest, `{"error": {"code": "v2_api_not_supported_in_testmode", "message": "This API is not included in the public preview of API v2 support in test mode."}}`, nil)

	_, err := fetchIntegHlthAlerts(context.Background(), server.URL, stripe.NewAPIKeyCredentials("sk_test_123"), "", 0)
	require.Error(t, err)
	category, ok := errorcategory.Get(err)
	require.True(t, ok)
	assert.Equal(t, errorcategory.UserInput, category)
	assert.Contains(t, err.Error(), "--live")
}

func TestFetchIntegHlthAlerts_SurfacesAPIErrorMessage(t *testing.T) {
	server := newIntegHlthServer(t, http.StatusForbidden, `{"error": {"code": "forbidden", "message": "You do not have access."}}`, nil)

	_, err := fetchIntegHlthAlerts(context.Background(), server.URL, stripe.NewAPIKeyCredentials("sk_test_123"), "", 0)
	require.Error(t, err)
	assert.Equal(t, "failed to retrieve integration health alerts (status 403): You do not have access.", err.Error())
}

func TestFetchIntegHlthAlerts_InvalidJSON(t *testing.T) {
	server := newIntegHlthServer(t, http.StatusOK, `not json`, nil)

	_, err := fetchIntegHlthAlerts(context.Background(), server.URL, stripe.NewAPIKeyCredentials("sk_test_123"), "", 0)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to parse response")
}

func TestRenderIntegHlthAlerts_PointsToDoctorWhenTopicSet(t *testing.T) {
	server := newIntegHlthServer(t, http.StatusOK, integHlthAlertsBody, nil)
	resp, err := fetchIntegHlthAlerts(context.Background(), server.URL, stripe.NewAPIKeyCredentials("sk_test_123"), "", 0)
	require.NoError(t, err)

	var buf bytes.Buffer
	renderIntegHlthAlerts(&buf, resp.Alerts)

	assert.Equal(t, "1 recommended action:\n1. Use dynamic payment methods. Run `stripe doctor dpm` for more information.\n", buf.String())
}

func TestRenderIntegHlthAlerts_PointsToHealthWithoutDoctorTopic(t *testing.T) {
	title := "Add the Stripe-Version header to your requests"
	var buf bytes.Buffer
	renderIntegHlthAlerts(&buf, []integHlthAlert{
		{HealthInsightID: "heins_123", HealthInsight: &integHlthAlertInsight{RecommendedAction: &integHlthRecommendedAction{Title: title}}},
		{HealthInsightID: "heins_456", PredicateName: "SOME_PREDICATE"},
	})

	assert.Equal(t, "2 recommended actions:\n1. Add the Stripe-Version header to your requests. Run `stripe health heins_123` for more information.\n"+
		"2. SOME_PREDICATE. Run `stripe health heins_456` for more information.\n", buf.String())
}

func TestRenderIntegHlthAlerts_Empty(t *testing.T) {
	var buf bytes.Buffer
	renderIntegHlthAlerts(&buf, nil)

	assert.Contains(t, buf.String(), "No integration health alerts")
}
