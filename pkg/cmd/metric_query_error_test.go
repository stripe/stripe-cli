package cmd

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/stripe/stripe-cli/pkg/requests"
)

func TestFormatMetricQueryError_InvalidParameterValueDropsRetryWording(t *testing.T) {
	err := formatMetricQueryError(requests.RequestError{
		StatusCode: http.StatusBadRequest,
		ErrorCode:  "metric_invalid_parameter_value",
		Body:       `{"error":{"code":"metric_invalid_parameter_value","message":"An internal error occurred while running the metric query. Please try again later."}}`,
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "metric_invalid_parameter_value")
	assert.Contains(t, err.Error(), "--group-by")
	assert.Contains(t, err.Error(), "day, week, month, or year")
	assert.Contains(t, err.Error(), "client error")
	assert.NotContains(t, err.Error(), "internal error")
	assert.NotContains(t, err.Error(), "try again later")

	var reqErr requests.RequestError
	require.ErrorAs(t, err, &reqErr)
	assert.Equal(t, "metric_invalid_parameter_value", reqErr.ErrorCode)
}

func TestFormatMetricQueryError_NotFoundMetric(t *testing.T) {
	err := formatMetricQueryError(requests.RequestError{
		StatusCode: http.StatusNotFound,
		ErrorCode:  "not_found",
		Body:       `{"error":{"code":"not_found","message":"No metric found with id 'metric_61Sud3n5oAGVCWiSr5'"}}`,
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "No metric found")
	assert.Contains(t, err.Error(), "revenue.mrr")
	assert.Contains(t, err.Error(), "sandbox")
}

func TestFormatMetricQueryError_NotFoundAPIMethod(t *testing.T) {
	err := formatMetricQueryError(requests.RequestError{
		StatusCode: http.StatusNotFound,
		ErrorCode:  "not_found",
		Body:       `{"error":{"code":"not_found","message":"The API method cannot be found."}}`,
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "endpoint was not found")
	assert.Contains(t, err.Error(), "Private Preview")
	assert.NotContains(t, err.Error(), "revenue.mrr")
	assert.NotContains(t, err.Error(), "metric id")
}

func TestFormatMetricQueryError_PassthroughUnknown(t *testing.T) {
	orig := requests.RequestError{StatusCode: 500, ErrorCode: "other", Body: `{"error":{"message":"boom"}}`}
	err := formatMetricQueryError(orig)
	require.Error(t, err)
	assert.Equal(t, orig, err)
}
