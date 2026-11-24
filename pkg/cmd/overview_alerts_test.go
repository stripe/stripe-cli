package cmd

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	genqlient "github.com/Khan/genqlient/graphql"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/stripe/stripe-cli/pkg/errorcategory"
)

const integHlthAlertsBody = `{
  "data": {
    "v2ListIntegHlthAlerts": {
      "alerts": [
        {
          "health_insight_id": "hi_123",
          "predicate_name": "LEGACY_PAYMENT_METHODS",
          "compartment_id": "cmpt_123",
          "created_at": "2026-10-01T00:00:00.000Z",
          "last_updated": "2026-10-20T12:00:00.000Z",
          "recent_integration_data": [],
          "status": "open",
          "doctor_topic": "dpm",
          "health_insight": {
            "summary": "Your integration pins payment_method_types",
            "severity": "warning",
            "why_is_it_important": "New payment methods won't show up at checkout.",
            "recommended_action": {"title": "Use dynamic payment methods.", "description": "Remove payment_method_types."}
          }
        }
      ]
    }
  }
}`

type integHlthGraphQLRequest struct {
	OperationName string         `json:"operationName"`
	Variables     map[string]any `json:"variables"`
}

func newIntegHlthGateway(t *testing.T, statusCode int, body string, recs *[]integHlthGraphQLRequest) *httptest.Server {
	t.Helper()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if recs != nil {
			var req integHlthGraphQLRequest
			assert.NoError(t, json.NewDecoder(r.Body).Decode(&req))
			*recs = append(*recs, req)
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(statusCode)
		_, _ = w.Write([]byte(body))
	}))
	t.Cleanup(server.Close)
	return server
}

// newIntegHlthPagedGateway serves pages[page] for the request's page variable
// ("" for the first page).
func newIntegHlthPagedGateway(t *testing.T, pages map[string]string, recs *[]integHlthGraphQLRequest) *httptest.Server {
	t.Helper()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req integHlthGraphQLRequest
		assert.NoError(t, json.NewDecoder(r.Body).Decode(&req))
		if recs != nil {
			*recs = append(*recs, req)
		}
		page, _ := req.Variables["page"].(string)
		body, ok := pages[page]
		assert.True(t, ok, "unexpected page %q", page)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(body))
	}))
	t.Cleanup(server.Close)
	return server
}

func integHlthAlertsPage(nextPage string, alerts ...string) string {
	return `{"data": {"v2ListIntegHlthAlerts": {"alerts": [` + strings.Join(alerts, ",") + `], "next_page": "` + nextPage + `"}}}`
}

func integHlthAlertJSON(id, predicateName string) string {
	return `{"health_insight_id": "` + id + `", "predicate_name": "` + predicateName + `", "status": "open"}`
}

func gatewayClient(server *httptest.Server) genqlient.Client {
	return genqlient.NewClient(server.URL, server.Client())
}

func TestFetchIntegHlthAlerts_QueriesAllStatusesAndConvertsAlerts(t *testing.T) {
	var recs []integHlthGraphQLRequest
	server := newIntegHlthGateway(t, http.StatusOK, integHlthAlertsBody, &recs)

	resp, err := fetchIntegHlthAlerts(context.Background(), gatewayClient(server), 0)
	require.NoError(t, err)

	require.Len(t, recs, 1)
	assert.Equal(t, "ListIntegHlthAlerts", recs[0].OperationName)
	assert.Equal(t, map[string]any{"statuses": []any{"open", "resolved"}}, recs[0].Variables)

	require.Len(t, resp.Alerts, 1)
	alert := resp.Alerts[0]
	assert.Equal(t, "hi_123", alert.HealthInsightID)
	assert.Equal(t, "LEGACY_PAYMENT_METHODS", alert.PredicateName)
	assert.Equal(t, "open", alert.Status)
	require.NotNil(t, alert.DoctorTopic)
	assert.Equal(t, "dpm", *alert.DoctorTopic)
	require.NotNil(t, alert.HealthInsight)
	assert.Equal(t, "warning", *alert.HealthInsight.Severity)
	require.NotNil(t, alert.HealthInsight.RecommendedAction)
	assert.Equal(t, "Use dynamic payment methods.", alert.HealthInsight.RecommendedAction.Title)
}

func TestFetchIntegHlthAlerts_OnlyReturnsLegacyPaymentMethodsAlerts(t *testing.T) {
	server := newIntegHlthPagedGateway(t, map[string]string{
		"": integHlthAlertsPage("", integHlthAlertJSON("heins_other", "API_KEY_EXPIRATION"), integHlthAlertJSON("heins_none", ""), integHlthAlertJSON("heins_dpm", "LEGACY_PAYMENT_METHODS")),
	}, nil)

	resp, err := fetchIntegHlthAlerts(context.Background(), gatewayClient(server), 0)
	require.NoError(t, err)

	require.Len(t, resp.Alerts, 1)
	assert.Equal(t, "heins_dpm", resp.Alerts[0].HealthInsightID)
}

func TestFetchIntegHlthAlerts_PagesThroughAlerts(t *testing.T) {
	var recs []integHlthGraphQLRequest
	server := newIntegHlthPagedGateway(t, map[string]string{
		"":       integHlthAlertsPage("page_2", integHlthAlertJSON("heins_other", "API_KEY_EXPIRATION")),
		"page_2": integHlthAlertsPage("", integHlthAlertJSON("heins_dpm", "LEGACY_PAYMENT_METHODS")),
	}, &recs)

	resp, err := fetchIntegHlthAlerts(context.Background(), gatewayClient(server), 0)
	require.NoError(t, err)

	require.Len(t, recs, 2)
	assert.Equal(t, "page_2", recs[1].Variables["page"])
	require.Len(t, resp.Alerts, 1)
	assert.Equal(t, "heins_dpm", resp.Alerts[0].HealthInsightID)
}

func TestFetchIntegHlthAlerts_AppliesLimitAfterFiltering(t *testing.T) {
	var recs []integHlthGraphQLRequest
	server := newIntegHlthPagedGateway(t, map[string]string{
		"":       integHlthAlertsPage("page_2", integHlthAlertJSON("heins_other", "API_KEY_EXPIRATION"), integHlthAlertJSON("heins_lpm_1", "LEGACY_PAYMENT_METHODS"), integHlthAlertJSON("heins_lpm_2", "LEGACY_PAYMENT_METHODS")),
		"page_2": integHlthAlertsPage("", integHlthAlertJSON("heins_lpm_3", "LEGACY_PAYMENT_METHODS")),
	}, &recs)

	resp, err := fetchIntegHlthAlerts(context.Background(), gatewayClient(server), 1)
	require.NoError(t, err)

	require.Len(t, recs, 1, "stops paging once the limit is reached")
	require.Len(t, resp.Alerts, 1)
	assert.Equal(t, "heins_lpm_1", resp.Alerts[0].HealthInsightID)
}

func TestFetchIntegHlthAlerts_StopsWhenNextPageRepeats(t *testing.T) {
	var recs []integHlthGraphQLRequest
	server := newIntegHlthPagedGateway(t, map[string]string{
		"":       integHlthAlertsPage("page_2"),
		"page_2": integHlthAlertsPage("page_2", integHlthAlertJSON("heins_dpm", "LEGACY_PAYMENT_METHODS")),
	}, &recs)

	resp, err := fetchIntegHlthAlerts(context.Background(), gatewayClient(server), 0)
	require.NoError(t, err)

	assert.Len(t, recs, 2)
	assert.Len(t, resp.Alerts, 1)
}

func TestFetchIntegHlthAlerts_NoMatchingAlertsReturnsEmptyList(t *testing.T) {
	server := newIntegHlthPagedGateway(t, map[string]string{
		"": integHlthAlertsPage("", integHlthAlertJSON("heins_other", "API_KEY_EXPIRATION")),
	}, nil)

	resp, err := fetchIntegHlthAlerts(context.Background(), gatewayClient(server), 0)
	require.NoError(t, err)

	assert.NotNil(t, resp.Alerts)
	assert.Empty(t, resp.Alerts)
}

func TestFetchIntegHlthAlerts_LeavesMissingFieldsUnset(t *testing.T) {
	body := `{"data": {"v2ListIntegHlthAlerts": {"alerts": [{"health_insight_id": "heins_123", "predicate_name": "LEGACY_PAYMENT_METHODS", "status": "open"}]}}}`
	server := newIntegHlthGateway(t, http.StatusOK, body, nil)

	resp, err := fetchIntegHlthAlerts(context.Background(), gatewayClient(server), 0)
	require.NoError(t, err)

	require.Len(t, resp.Alerts, 1)
	assert.Nil(t, resp.Alerts[0].DoctorTopic)
	assert.Nil(t, resp.Alerts[0].HealthInsight)
}

func TestFetchIntegHlthAlerts_Unauthorized(t *testing.T) {
	server := newIntegHlthGateway(t, http.StatusUnauthorized, `{"errors": [{"message": "unauthorized"}]}`, nil)

	_, err := fetchIntegHlthAlerts(context.Background(), gatewayClient(server), 0)
	require.Error(t, err)
	category, ok := errorcategory.Get(err)
	require.True(t, ok)
	assert.Equal(t, errorcategory.Auth, category)
	assert.Contains(t, err.Error(), "stripe login")
}

func TestFetchIntegHlthAlerts_NonOKStatus(t *testing.T) {
	server := newIntegHlthGateway(t, http.StatusInternalServerError, `{"errors": [{"message": "gateway unavailable"}]}`, nil)

	_, err := fetchIntegHlthAlerts(context.Background(), gatewayClient(server), 0)
	require.Error(t, err)
	category, ok := errorcategory.Get(err)
	require.True(t, ok)
	assert.Equal(t, errorcategory.API, category)
	assert.Equal(t, "failed to retrieve integration health alerts (status 500): gateway unavailable", err.Error())
}

func TestFetchIntegHlthAlerts_TestmodeUnsupportedSuggestsSwitch(t *testing.T) {
	body := `{"data": null, "errors": [{"message": "This API is not included in the public preview of API v2 support in test mode.", "extensions": {"code": "v2_api_not_supported_in_testmode"}}]}`
	server := newIntegHlthGateway(t, http.StatusOK, body, nil)

	_, err := fetchIntegHlthAlerts(context.Background(), gatewayClient(server), 0)
	require.Error(t, err)
	category, ok := errorcategory.Get(err)
	require.True(t, ok)
	assert.Equal(t, errorcategory.UserInput, category)
	assert.Contains(t, err.Error(), "stripe switch --live")
}

func TestFetchIntegHlthAlerts_SurfacesGraphQLErrorMessage(t *testing.T) {
	body := `{"data": null, "errors": [{"message": "You do not have access."}]}`
	server := newIntegHlthGateway(t, http.StatusOK, body, nil)

	_, err := fetchIntegHlthAlerts(context.Background(), gatewayClient(server), 0)
	require.Error(t, err)
	assert.Equal(t, "failed to retrieve integration health alerts: You do not have access.", err.Error())
}
