package cmd

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/stripe/stripe-cli/pkg/config"
	"github.com/stripe/stripe-cli/pkg/requests"
)

// --- Unit tests: metricRef ---

func TestMetricRef_CommonName(t *testing.T) {
	assert.Equal(t, map[string]interface{}{"name": "revenue.mrr"}, metricRef("revenue.mrr"))
	assert.Equal(t, map[string]interface{}{"name": "revenue.arr"}, metricRef("revenue.arr"))
	assert.Equal(t, map[string]interface{}{"name": "usage_based_billing.gross_usage_revenue"}, metricRef("usage_based_billing.gross_usage_revenue"))
}

func TestMetricRef_LiveModeID(t *testing.T) {
	assert.Equal(t, map[string]interface{}{"id": "metric_61Sud3n5oAGVCWiSr5"}, metricRef("metric_61Sud3n5oAGVCWiSr5"))
}

func TestMetricRef_SandboxID(t *testing.T) {
	assert.Equal(t, map[string]interface{}{"id": "metric_test_61UYChCcFUOh6ieln5"}, metricRef("metric_test_61UYChCcFUOh6ieln5"))
}

func TestMetricRef_HypotheticalNameWithMetricPrefix(t *testing.T) {
	// "metric_x.foo" starts with "metric_" but contains a dot — must be treated
	// as a common name, not an ID.
	assert.Equal(t, map[string]interface{}{"name": "metric_x.foo"}, metricRef("metric_x.foo"))
}

// --- Unit tests: buildRequestBody ---

func TestAnalyticsBuildRequestBody_Minimal(t *testing.T) {
	aqc := &analyticsQueryCmd{
		metrics:     []string{"revenue.mrr"},
		startsAt:    "2026-01-01T00:00:00Z",
		endsAt:      "2026-01-31T23:59:59Z",
		granularity: "day",
	}

	body, err := aqc.buildRequestBody()
	require.NoError(t, err)

	metrics := body["metrics"].([]map[string]interface{})
	require.Len(t, metrics, 1)
	assert.Equal(t, "revenue.mrr", metrics[0]["name"])
	assert.Equal(t, "2026-01-01T00:00:00Z", body["starts_at"])
	assert.Equal(t, "2026-01-31T23:59:59Z", body["ends_at"])
	assert.Equal(t, "day", body["granularity"])

	assert.Nil(t, body["currency"])
	assert.Nil(t, body["timezone"])
	assert.Nil(t, body["limit"])
	assert.Nil(t, body["group_by"])
	assert.Nil(t, body["filters"])
}

func TestAnalyticsBuildRequestBody_MultipleMetrics(t *testing.T) {
	aqc := &analyticsQueryCmd{
		metrics:     []string{"revenue.mrr", "revenue.arr"},
		startsAt:    "2026-01-01T00:00:00Z",
		endsAt:      "2026-01-31T23:59:59Z",
		granularity: "month",
	}

	body, err := aqc.buildRequestBody()
	require.NoError(t, err)

	metrics := body["metrics"].([]map[string]interface{})
	require.Len(t, metrics, 2)
	assert.Equal(t, "revenue.mrr", metrics[0]["name"])
	assert.Equal(t, "revenue.arr", metrics[1]["name"])
}

func TestAnalyticsBuildRequestBody_MetricByID(t *testing.T) {
	aqc := &analyticsQueryCmd{
		metrics:     []string{"metric_61Sud3n5oAGVCWiSr5", "metric_test_61UYChCcFUOh6ieln5"},
		startsAt:    "2026-01-01T00:00:00Z",
		endsAt:      "2026-01-31T23:59:59Z",
		granularity: "day",
	}

	body, err := aqc.buildRequestBody()
	require.NoError(t, err)

	metrics := body["metrics"].([]map[string]interface{})
	require.Len(t, metrics, 2)
	assert.Equal(t, "metric_61Sud3n5oAGVCWiSr5", metrics[0]["id"])
	assert.Nil(t, metrics[0]["name"])
	assert.Equal(t, "metric_test_61UYChCcFUOh6ieln5", metrics[1]["id"])
	assert.Nil(t, metrics[1]["name"])
}

func TestAnalyticsBuildRequestBody_MixedNameAndID(t *testing.T) {
	aqc := &analyticsQueryCmd{
		metrics:     []string{"revenue.mrr", "metric_61Sud3n5oAGVCWiSr5"},
		startsAt:    "2026-01-01T00:00:00Z",
		endsAt:      "2026-01-31T23:59:59Z",
		granularity: "month",
	}

	body, err := aqc.buildRequestBody()
	require.NoError(t, err)

	metrics := body["metrics"].([]map[string]interface{})
	require.Len(t, metrics, 2)
	assert.Equal(t, "revenue.mrr", metrics[0]["name"])
	assert.Nil(t, metrics[0]["id"])
	assert.Equal(t, "metric_61Sud3n5oAGVCWiSr5", metrics[1]["id"])
	assert.Nil(t, metrics[1]["name"])
}

func TestAnalyticsBuildRequestBody_AllFields(t *testing.T) {
	aqc := &analyticsQueryCmd{
		metrics:     []string{"revenue.mrr"},
		startsAt:    "2026-01-01T00:00:00Z",
		endsAt:      "2026-01-31T23:59:59Z",
		granularity: "month",
		currency:    "usd",
		timezone:    "America/New_York",
		limit:       100,
		groupBy:     []string{"product"},
		filters:     []string{"price=price_abc", "price=price_xyz"},
	}

	body, err := aqc.buildRequestBody()
	require.NoError(t, err)

	assert.Equal(t, "usd", body["currency"])
	assert.Equal(t, "America/New_York", body["timezone"])
	assert.Equal(t, 100, body["limit"])
	assert.Equal(t, []string{"product"}, body["group_by"])

	filters := body["filters"].(map[string][]string)
	assert.Equal(t, []string{"price_abc", "price_xyz"}, filters["price"])
}

func TestAnalyticsBuildRequestBody_LimitZeroOmitted(t *testing.T) {
	aqc := &analyticsQueryCmd{
		metrics:     []string{"revenue.mrr"},
		startsAt:    "2026-01-01T00:00:00Z",
		endsAt:      "2026-01-31T23:59:59Z",
		granularity: "day",
		limit:       0,
	}

	body, err := aqc.buildRequestBody()
	require.NoError(t, err)
	assert.Nil(t, body["limit"], "limit=0 should be omitted from the request")
}

// --- Unit tests: parseAnalyticsFilters ---

func TestParseAnalyticsFilters_SingleKey(t *testing.T) {
	result, err := parseAnalyticsFilters([]string{"currency=usd"})
	require.NoError(t, err)
	assert.Equal(t, map[string][]string{"currency": {"usd"}}, result)
}

func TestParseAnalyticsFilters_MultipleValuesForKey(t *testing.T) {
	result, err := parseAnalyticsFilters([]string{"price=price_abc", "price=price_xyz"})
	require.NoError(t, err)
	assert.Equal(t, []string{"price_abc", "price_xyz"}, result["price"])
}

func TestParseAnalyticsFilters_MultipleKeys(t *testing.T) {
	result, err := parseAnalyticsFilters([]string{"currency=usd", "product=prod_123"})
	require.NoError(t, err)
	assert.Equal(t, []string{"usd"}, result["currency"])
	assert.Equal(t, []string{"prod_123"}, result["product"])
}

func TestParseAnalyticsFilters_ValueWithEquals(t *testing.T) {
	// values that themselves contain = should be preserved
	result, err := parseAnalyticsFilters([]string{"key=val=ue"})
	require.NoError(t, err)
	assert.Equal(t, []string{"val=ue"}, result["key"])
}

func TestParseAnalyticsFilters_InvalidNoEquals(t *testing.T) {
	_, err := parseAnalyticsFilters([]string{"noequals"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid filter")
}

func TestParseAnalyticsFilters_InvalidEmptyKey(t *testing.T) {
	_, err := parseAnalyticsFilters([]string{"=value"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid filter")
}

func TestParseAnalyticsFilters_InvalidEmptyValue(t *testing.T) {
	_, err := parseAnalyticsFilters([]string{"key="})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid filter")
}

// --- Unit tests: runAnalyticsQueryCmd validation ---

func TestAnalyticsQueryCmd_MissingMetric(t *testing.T) {
	aqc := newAnalyticsQueryCmd()
	aqc.rb.Profile = &config.Profile{APIKey: "sk_test_1234567890abcdef"}
	aqc.startsAt = "2026-01-01T00:00:00Z"
	aqc.endsAt = "2026-01-31T23:59:59Z"

	err := aqc.runAnalyticsQueryCmd(aqc.cmd, []string{})
	require.Error(t, err)
	assert.Equal(t, "--metric is required", err.Error())
}

func TestAnalyticsQueryCmd_MissingStartsAt(t *testing.T) {
	aqc := newAnalyticsQueryCmd()
	aqc.rb.Profile = &config.Profile{APIKey: "sk_test_1234567890abcdef"}
	aqc.metrics = []string{"revenue.mrr"}
	aqc.endsAt = "2026-01-31T23:59:59Z"

	err := aqc.runAnalyticsQueryCmd(aqc.cmd, []string{})
	require.Error(t, err)
	assert.Equal(t, "--starts-at is required", err.Error())
}

func TestAnalyticsQueryCmd_TooManyGroupBy(t *testing.T) {
	aqc := newAnalyticsQueryCmd()
	aqc.rb.Profile = &config.Profile{APIKey: "sk_test_1234567890abcdef"}
	aqc.metrics = []string{"revenue.mrr"}
	aqc.startsAt = "2026-01-01T00:00:00Z"
	aqc.endsAt = "2026-01-31T23:59:59Z"
	aqc.groupBy = []string{"price", "product"}

	err := aqc.runAnalyticsQueryCmd(aqc.cmd, []string{})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "--group-by accepts at most one dimension")
	assert.Contains(t, err.Error(), "price, product")
}

func TestAnalyticsQueryCmd_MissingEndsAt(t *testing.T) {
	aqc := newAnalyticsQueryCmd()
	aqc.rb.Profile = &config.Profile{APIKey: "sk_test_1234567890abcdef"}
	aqc.metrics = []string{"revenue.mrr"}
	aqc.startsAt = "2026-01-01T00:00:00Z"

	err := aqc.runAnalyticsQueryCmd(aqc.cmd, []string{})
	require.Error(t, err)
	assert.Equal(t, "--ends-at is required", err.Error())
}

// --- Integration tests: HTTP request shape ---

// newTestAnalyticsCmd creates an analyticsQueryCmd wired to a test server URL with a
// fake API key and a background context (required so the telemetry goroutine spawned
// by stripe.Client.PerformRequest doesn't panic on a nil context).
func newTestAnalyticsCmd(t *testing.T, serverURL string) *analyticsQueryCmd {
	t.Helper()
	aqc := newAnalyticsQueryCmd()
	aqc.rb.Profile = &config.Profile{APIKey: "sk_test_1234567890abcdef"}
	aqc.rb.APIBaseURL = serverURL
	aqc.cmd.SetContext(context.Background())
	return aqc
}

func TestAnalyticsQueryCmd_HTTPRequest(t *testing.T) {
	var capturedReq *http.Request
	var capturedBody []byte

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedReq = r
		capturedBody, _ = io.ReadAll(r.Body)
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"object":"v2.data.analytics.metric_query_result","data":[]}`))
	}))
	defer ts.Close()

	aqc := newTestAnalyticsCmd(t, ts.URL)
	aqc.metrics = []string{"revenue.mrr"}
	aqc.startsAt = "2026-01-01T00:00:00Z"
	aqc.endsAt = "2026-01-31T23:59:59Z"
	aqc.granularity = "month"

	err := aqc.runAnalyticsQueryCmd(aqc.cmd, []string{})
	require.NoError(t, err)
	require.NotNil(t, capturedReq)

	assert.Equal(t, http.MethodPost, capturedReq.Method)
	assert.Equal(t, analyticsQueryPath, capturedReq.URL.Path)
	assert.Equal(t, "Bearer sk_test_1234567890abcdef", capturedReq.Header.Get("Authorization"))
	assert.Equal(t, "application/json", capturedReq.Header.Get("Content-Type"))
	assert.Equal(t, requests.StripePreviewVersionHeaderValue, capturedReq.Header.Get("Stripe-Version"))

	var body map[string]interface{}
	require.NoError(t, json.Unmarshal(capturedBody, &body))

	metrics := body["metrics"].([]interface{})
	require.Len(t, metrics, 1)
	assert.Equal(t, "revenue.mrr", metrics[0].(map[string]interface{})["name"])
	assert.Equal(t, "2026-01-01T00:00:00Z", body["starts_at"])
	assert.Equal(t, "2026-01-31T23:59:59Z", body["ends_at"])
	assert.Equal(t, "month", body["granularity"])
}

func TestAnalyticsQueryCmd_HTTPRequest_WithFiltersAndGroupBy(t *testing.T) {
	var capturedBody []byte

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedBody, _ = io.ReadAll(r.Body)
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"object":"v2.data.analytics.metric_query_result","data":[]}`))
	}))
	defer ts.Close()

	aqc := newTestAnalyticsCmd(t, ts.URL)
	aqc.metrics = []string{"revenue.mrr"}
	aqc.startsAt = "2026-01-01T00:00:00Z"
	aqc.endsAt = "2026-01-31T23:59:59Z"
	aqc.granularity = "day"
	aqc.currency = "usd"
	aqc.groupBy = []string{"price"}
	aqc.filters = []string{"price=price_abc123", "price=price_xyz789"}

	err := aqc.runAnalyticsQueryCmd(aqc.cmd, []string{})
	require.NoError(t, err)

	var body map[string]interface{}
	require.NoError(t, json.Unmarshal(capturedBody, &body))

	assert.Equal(t, "usd", body["currency"])

	groupBy := body["group_by"].([]interface{})
	assert.Equal(t, "price", groupBy[0])

	filters := body["filters"].(map[string]interface{})
	priceFilters := filters["price"].([]interface{})
	assert.Equal(t, "price_abc123", priceFilters[0])
	assert.Equal(t, "price_xyz789", priceFilters[1])
}

func TestAnalyticsQueryCmd_HTTPRequest_MultipleMetrics(t *testing.T) {
	var capturedBody []byte

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedBody, _ = io.ReadAll(r.Body)
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"object":"v2.data.analytics.metric_query_result","data":[]}`))
	}))
	defer ts.Close()

	aqc := newTestAnalyticsCmd(t, ts.URL)
	aqc.metrics = []string{"revenue.mrr", "revenue.arr"}
	aqc.startsAt = "2026-01-01T00:00:00Z"
	aqc.endsAt = "2026-01-31T23:59:59Z"
	aqc.granularity = "month"

	err := aqc.runAnalyticsQueryCmd(aqc.cmd, []string{})
	require.NoError(t, err)

	var body map[string]interface{}
	require.NoError(t, json.Unmarshal(capturedBody, &body))

	metrics := body["metrics"].([]interface{})
	require.Len(t, metrics, 2)
	assert.Equal(t, "revenue.mrr", metrics[0].(map[string]interface{})["name"])
	assert.Equal(t, "revenue.arr", metrics[1].(map[string]interface{})["name"])
}

func TestAnalyticsQueryCmd_HTTPError(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(`{"error":{"code":"metric_invalid_parameter_value","type":"invalid_request_error"}}`))
	}))
	defer ts.Close()

	aqc := newTestAnalyticsCmd(t, ts.URL)
	aqc.metrics = []string{"revenue.mrr"}
	aqc.startsAt = "2026-01-01T00:00:00Z"
	aqc.endsAt = "2026-01-31T23:59:59Z"
	aqc.granularity = "day"

	// errOnStatus=true: 4xx responses are returned as a Go error (exit 1).
	err := aqc.runAnalyticsQueryCmd(aqc.cmd, []string{})
	require.Error(t, err)

	var reqErr requests.RequestError
	require.ErrorAs(t, err, &reqErr)
	assert.Equal(t, http.StatusBadRequest, reqErr.StatusCode)
	assert.Equal(t, "metric_invalid_parameter_value", reqErr.ErrorCode)
}

func TestAnalyticsQueryCmd_StripeVersionHeader(t *testing.T) {
	var versionHeader string

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		versionHeader = r.Header.Get("Stripe-Version")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{}`))
	}))
	defer ts.Close()

	aqc := newTestAnalyticsCmd(t, ts.URL)
	aqc.metrics = []string{"revenue.mrr"}
	aqc.startsAt = "2026-01-01T00:00:00Z"
	aqc.endsAt = "2026-01-31T23:59:59Z"

	err := aqc.runAnalyticsQueryCmd(aqc.cmd, []string{})
	require.NoError(t, err)

	assert.Equal(t, requests.StripePreviewVersionHeaderValue, versionHeader,
		"Stripe-Version header must be the preview version")
}

// --- Unit tests: command construction ---

func TestNewAnalyticsQueryCmd_Flags(t *testing.T) {
	aqc := newAnalyticsQueryCmd()

	require.NotNil(t, aqc.cmd.Flags().Lookup("metric"))
	require.NotNil(t, aqc.cmd.Flags().Lookup("starts-at"))
	require.NotNil(t, aqc.cmd.Flags().Lookup("ends-at"))
	require.NotNil(t, aqc.cmd.Flags().Lookup("granularity"))
	require.NotNil(t, aqc.cmd.Flags().Lookup("group-by"))
	require.NotNil(t, aqc.cmd.Flags().Lookup("filter"))
	require.NotNil(t, aqc.cmd.Flags().Lookup("currency"))
	require.NotNil(t, aqc.cmd.Flags().Lookup("timezone"))
	require.NotNil(t, aqc.cmd.Flags().Lookup("limit"))

	granularity, err := aqc.cmd.Flags().GetString("granularity")
	require.NoError(t, err)
	assert.Equal(t, "day", granularity, "granularity should default to 'day'")
}

func TestNewAnalyticsQueryCmd_IsPreview(t *testing.T) {
	aqc := newAnalyticsQueryCmd()
	assert.True(t, aqc.rb.IsPreviewCommand, "analytics query must use the preview Stripe-Version header")
	assert.Equal(t, http.MethodPost, aqc.rb.Method)
}

func TestNewAnalyticsCmd_HasQuerySubcommand(t *testing.T) {
	ac := newAnalyticsCmd()
	_, _, err := ac.cmd.Find([]string{"query"})
	require.NoError(t, err)
}

func TestAnalyticsQueryCmd_CommandPath(t *testing.T) {
	ac := newAnalyticsCmd()
	queryCmd, _, err := ac.cmd.Find([]string{"query"})
	require.NoError(t, err)
	assert.Equal(t, "analytics query", queryCmd.CommandPath())
}
