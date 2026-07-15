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

func TestBuildRequestBody_Minimal(t *testing.T) {
	c := &dataMetricsRunCmd{
		metrics:     []string{"revenue.mrr"},
		startsAt:    "2026-01-01T00:00:00Z",
		endsAt:      "2026-01-31T23:59:59Z",
		granularity: "day",
	}

	body, err := c.buildRequestBody(false)
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

func TestBuildRequestBody_MultipleMetrics(t *testing.T) {
	c := &dataMetricsRunCmd{
		metrics:     []string{"revenue.mrr", "revenue.arr"},
		startsAt:    "2026-01-01T00:00:00Z",
		endsAt:      "2026-01-31T23:59:59Z",
		granularity: "month",
	}

	body, err := c.buildRequestBody(false)
	require.NoError(t, err)

	metrics := body["metrics"].([]map[string]interface{})
	require.Len(t, metrics, 2)
	assert.Equal(t, "revenue.mrr", metrics[0]["name"])
	assert.Equal(t, "revenue.arr", metrics[1]["name"])
}

func TestBuildRequestBody_MetricByID(t *testing.T) {
	c := &dataMetricsRunCmd{
		metrics:     []string{"metric_61Sud3n5oAGVCWiSr5", "metric_test_61UYChCcFUOh6ieln5"},
		startsAt:    "2026-01-01T00:00:00Z",
		endsAt:      "2026-01-31T23:59:59Z",
		granularity: "day",
	}

	body, err := c.buildRequestBody(false)
	require.NoError(t, err)

	metrics := body["metrics"].([]map[string]interface{})
	require.Len(t, metrics, 2)
	assert.Equal(t, "metric_61Sud3n5oAGVCWiSr5", metrics[0]["id"])
	assert.Nil(t, metrics[0]["name"])
	assert.Equal(t, "metric_test_61UYChCcFUOh6ieln5", metrics[1]["id"])
	assert.Nil(t, metrics[1]["name"])
}

func TestBuildRequestBody_MixedNameAndID(t *testing.T) {
	c := &dataMetricsRunCmd{
		metrics:     []string{"revenue.mrr", "metric_61Sud3n5oAGVCWiSr5"},
		startsAt:    "2026-01-01T00:00:00Z",
		endsAt:      "2026-01-31T23:59:59Z",
		granularity: "month",
	}

	body, err := c.buildRequestBody(false)
	require.NoError(t, err)

	metrics := body["metrics"].([]map[string]interface{})
	require.Len(t, metrics, 2)
	assert.Equal(t, "revenue.mrr", metrics[0]["name"])
	assert.Nil(t, metrics[0]["id"])
	assert.Equal(t, "metric_61Sud3n5oAGVCWiSr5", metrics[1]["id"])
	assert.Nil(t, metrics[1]["name"])
}

func TestBuildRequestBody_AllFields(t *testing.T) {
	c := &dataMetricsRunCmd{
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

	body, err := c.buildRequestBody(true)
	require.NoError(t, err)

	assert.Equal(t, "usd", body["currency"])
	assert.Equal(t, "America/New_York", body["timezone"])
	assert.Equal(t, 100, body["limit"])
	assert.Equal(t, []string{"product"}, body["group_by"])

	filters := body["filters"].(map[string][]string)
	assert.Equal(t, []string{"price_abc", "price_xyz"}, filters["price"])
}

func TestBuildRequestBody_LimitOmittedWhenNotSet(t *testing.T) {
	c := &dataMetricsRunCmd{
		metrics:     []string{"revenue.mrr"},
		startsAt:    "2026-01-01T00:00:00Z",
		endsAt:      "2026-01-31T23:59:59Z",
		granularity: "day",
		limit:       50,
	}

	body, err := c.buildRequestBody(false)
	require.NoError(t, err)
	assert.Nil(t, body["limit"], "limit should be omitted when the flag was not set")
}

func TestBuildRequestBody_LimitForwardedWhenSet(t *testing.T) {
	// When the user explicitly sets --limit we forward the value as-is (even a
	// nonsensical one like 0) so the API validates it rather than the CLI.
	c := &dataMetricsRunCmd{
		metrics:     []string{"revenue.mrr"},
		startsAt:    "2026-01-01T00:00:00Z",
		endsAt:      "2026-01-31T23:59:59Z",
		granularity: "day",
		limit:       0,
	}

	body, err := c.buildRequestBody(true)
	require.NoError(t, err)
	assert.Equal(t, 0, body["limit"], "explicitly set limit should be forwarded, even when 0")
}

// --- Unit tests: parseMetricFilters ---

func TestParseMetricFilters_SingleKey(t *testing.T) {
	result, err := parseMetricFilters([]string{"currency=usd"})
	require.NoError(t, err)
	assert.Equal(t, map[string][]string{"currency": {"usd"}}, result)
}

func TestParseMetricFilters_MultipleValuesForKey(t *testing.T) {
	result, err := parseMetricFilters([]string{"price=price_abc", "price=price_xyz"})
	require.NoError(t, err)
	assert.Equal(t, []string{"price_abc", "price_xyz"}, result["price"])
}

func TestParseMetricFilters_MultipleKeys(t *testing.T) {
	result, err := parseMetricFilters([]string{"currency=usd", "product=prod_123"})
	require.NoError(t, err)
	assert.Equal(t, []string{"usd"}, result["currency"])
	assert.Equal(t, []string{"prod_123"}, result["product"])
}

func TestParseMetricFilters_ValueWithEquals(t *testing.T) {
	// values that themselves contain = should be preserved
	result, err := parseMetricFilters([]string{"key=val=ue"})
	require.NoError(t, err)
	assert.Equal(t, []string{"val=ue"}, result["key"])
}

func TestParseMetricFilters_InvalidNoEquals(t *testing.T) {
	_, err := parseMetricFilters([]string{"noequals"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid filter")
}

func TestParseMetricFilters_InvalidEmptyKey(t *testing.T) {
	_, err := parseMetricFilters([]string{"=value"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid filter")
}

func TestParseMetricFilters_InvalidEmptyValue(t *testing.T) {
	_, err := parseMetricFilters([]string{"key="})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid filter")
}

// --- Integration tests: HTTP request shape ---

// newTestDataMetricsRunCmd creates a dataMetricsRunCmd wired to a test server URL
// with a fake API key and a background context (required so the telemetry goroutine
// spawned by stripe.Client.PerformRequest doesn't panic on a nil context).
func newTestDataMetricsRunCmd(t *testing.T, serverURL string) *dataMetricsRunCmd {
	t.Helper()
	c := newDataMetricsRunCmd()
	c.rb.Profile = &config.Profile{APIKey: "sk_test_1234567890abcdef"}
	c.rb.APIBaseURL = serverURL
	c.cmd.SetContext(context.Background())
	return c
}

func TestDataMetricsRunCmd_ForwardsToAPIWithoutClientValidation(t *testing.T) {
	var capturedBody []byte

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedBody, _ = io.ReadAll(r.Body)
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(`{"error":{"code":"metric_invalid_parameter_value","type":"invalid_request_error"}}`))
	}))
	defer ts.Close()

	c := newTestDataMetricsRunCmd(t, ts.URL)
	c.metrics = []string{"revenue.mrr"}
	// Omit starts-at and ends-at — the CLI forwards them (empty) to the API
	// instead of rejecting locally; only --metric is guarded client-side.
	err := c.runDataMetricsRunCmd(c.cmd, []string{})
	require.Error(t, err)

	var body map[string]interface{}
	require.NoError(t, json.Unmarshal(capturedBody, &body))

	assert.Equal(t, "", body["starts_at"])
	assert.Equal(t, "", body["ends_at"])
	assert.Equal(t, "day", body["granularity"])
}

func TestDataMetricsRunCmd_HTTPRequest(t *testing.T) {
	var capturedReq *http.Request
	var capturedBody []byte

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedReq = r
		capturedBody, _ = io.ReadAll(r.Body)
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"object":"v2.data.analytics.metric_query_result","data":[]}`))
	}))
	defer ts.Close()

	c := newTestDataMetricsRunCmd(t, ts.URL)
	c.metrics = []string{"revenue.mrr"}
	c.startsAt = "2026-01-01T00:00:00Z"
	c.endsAt = "2026-01-31T23:59:59Z"
	c.granularity = "month"

	err := c.runDataMetricsRunCmd(c.cmd, []string{})
	require.NoError(t, err)
	require.NotNil(t, capturedReq)

	assert.Equal(t, http.MethodPost, capturedReq.Method)
	assert.Equal(t, dataMetricsRunPath, capturedReq.URL.Path)
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

func TestDataMetricsRunCmd_HTTPRequest_WithFiltersAndGroupBy(t *testing.T) {
	var capturedBody []byte

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedBody, _ = io.ReadAll(r.Body)
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"object":"v2.data.analytics.metric_query_result","data":[]}`))
	}))
	defer ts.Close()

	c := newTestDataMetricsRunCmd(t, ts.URL)
	c.metrics = []string{"revenue.mrr"}
	c.startsAt = "2026-01-01T00:00:00Z"
	c.endsAt = "2026-01-31T23:59:59Z"
	c.granularity = "day"
	c.currency = "usd"
	c.groupBy = []string{"price"}
	c.filters = []string{"price=price_abc123", "price=price_xyz789"}

	err := c.runDataMetricsRunCmd(c.cmd, []string{})
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

func TestDataMetricsRunCmd_HTTPRequest_MultipleMetrics(t *testing.T) {
	var capturedBody []byte

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedBody, _ = io.ReadAll(r.Body)
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"object":"v2.data.analytics.metric_query_result","data":[]}`))
	}))
	defer ts.Close()

	c := newTestDataMetricsRunCmd(t, ts.URL)
	c.metrics = []string{"revenue.mrr", "revenue.arr"}
	c.startsAt = "2026-01-01T00:00:00Z"
	c.endsAt = "2026-01-31T23:59:59Z"
	c.granularity = "month"

	err := c.runDataMetricsRunCmd(c.cmd, []string{})
	require.NoError(t, err)

	var body map[string]interface{}
	require.NoError(t, json.Unmarshal(capturedBody, &body))

	metrics := body["metrics"].([]interface{})
	require.Len(t, metrics, 2)
	assert.Equal(t, "revenue.mrr", metrics[0].(map[string]interface{})["name"])
	assert.Equal(t, "revenue.arr", metrics[1].(map[string]interface{})["name"])
}

func TestDataMetricsRunCmd_HTTPError(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(`{"error":{"code":"metric_invalid_parameter_value","type":"invalid_request_error"}}`))
	}))
	defer ts.Close()

	c := newTestDataMetricsRunCmd(t, ts.URL)
	c.metrics = []string{"revenue.mrr"}
	c.startsAt = "2026-01-01T00:00:00Z"
	c.endsAt = "2026-01-31T23:59:59Z"
	c.granularity = "day"

	// errOnStatus=true: 4xx responses are returned as a Go error (exit 1).
	err := c.runDataMetricsRunCmd(c.cmd, []string{})
	require.Error(t, err)

	var reqErr requests.RequestError
	require.ErrorAs(t, err, &reqErr)
	assert.Equal(t, http.StatusBadRequest, reqErr.StatusCode)
	assert.Equal(t, "metric_invalid_parameter_value", reqErr.ErrorCode)
}

func TestDataMetricsRunCmd_StripeVersionHeader(t *testing.T) {
	var versionHeader string

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		versionHeader = r.Header.Get("Stripe-Version")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{}`))
	}))
	defer ts.Close()

	c := newTestDataMetricsRunCmd(t, ts.URL)
	c.metrics = []string{"revenue.mrr"}
	c.startsAt = "2026-01-01T00:00:00Z"
	c.endsAt = "2026-01-31T23:59:59Z"

	err := c.runDataMetricsRunCmd(c.cmd, []string{})
	require.NoError(t, err)

	assert.Equal(t, requests.StripePreviewVersionHeaderValue, versionHeader,
		"Stripe-Version header must be the preview version")
}

// --- Unit tests: command construction ---

// TestNewDataMetricsRunCmd_Flags only checks that each flag is registered, not
// that it is required. group-by, filter, currency, timezone, and limit are all
// optional; requiredness is enforced by the API, not the CLI.
func TestNewDataMetricsRunCmd_Flags(t *testing.T) {
	c := newDataMetricsRunCmd()

	require.NotNil(t, c.cmd.Flags().Lookup("metric"))
	require.NotNil(t, c.cmd.Flags().Lookup("starts-at"))
	require.NotNil(t, c.cmd.Flags().Lookup("ends-at"))
	require.NotNil(t, c.cmd.Flags().Lookup("granularity"))
	require.NotNil(t, c.cmd.Flags().Lookup("group-by"))
	require.NotNil(t, c.cmd.Flags().Lookup("filter"))
	require.NotNil(t, c.cmd.Flags().Lookup("currency"))
	require.NotNil(t, c.cmd.Flags().Lookup("timezone"))
	require.NotNil(t, c.cmd.Flags().Lookup("limit"))

	granularity, err := c.cmd.Flags().GetString("granularity")
	require.NoError(t, err)
	assert.Equal(t, "day", granularity, "granularity should default to 'day'")
}

func TestNewDataMetricsRunCmd_IsPreview(t *testing.T) {
	c := newDataMetricsRunCmd()
	assert.True(t, c.rb.IsPreviewCommand, "data metrics run must use the preview Stripe-Version header")
	assert.Equal(t, http.MethodPost, c.rb.Method)
}

func TestDataMetricsRunCmd_RequiresMetric(t *testing.T) {
	c := newDataMetricsRunCmd()
	c.startsAt = "2026-01-01T00:00:00Z"
	c.endsAt = "2026-01-31T23:59:59Z"

	err := c.runDataMetricsRunCmd(c.cmd, []string{})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "--metric is required")
}

func TestNewDataCmd_HasRunSubcommand(t *testing.T) {
	dc := newDataCmd()
	_, _, err := dc.cmd.Find([]string{"metrics", "run"})
	require.NoError(t, err)
}

func TestDataMetricsRunCmd_CommandPath(t *testing.T) {
	dc := newDataCmd()
	runCmd, _, err := dc.cmd.Find([]string{"metrics", "run"})
	require.NoError(t, err)
	assert.Equal(t, "data metrics run", runCmd.CommandPath())
}

func TestNewDataCmd_HiddenForPrivatePreview(t *testing.T) {
	dc := newDataCmd()
	assert.True(t, dc.cmd.Hidden, "data command is a Private Preview API and must be hidden")

	metricsCmd, _, err := dc.cmd.Find([]string{"metrics"})
	require.NoError(t, err)
	assert.True(t, metricsCmd.Hidden, "metrics command is a Private Preview API and must be hidden")

	runCmd, _, err := dc.cmd.Find([]string{"metrics", "run"})
	require.NoError(t, err)
	assert.True(t, runCmd.Hidden, "run command is a Private Preview API and must be hidden")
}
