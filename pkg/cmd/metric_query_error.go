package cmd

import (
	"encoding/json"
	"errors"
	"strings"

	"github.com/stripe/stripe-cli/pkg/errorcategory"
	"github.com/stripe/stripe-cli/pkg/requests"
)

const supportedMetricsURL = "https://docs.stripe.com/data/analytics/supported-metrics"

// formatMetricQueryError replaces opaque API 4xx bodies with an actionable
// message. The Analytics API sometimes returns metric_invalid_parameter_value
// with "An internal error occurred... try again later", which is a client
// error and must not tell the user to retry.
func formatMetricQueryError(err error) error {
	if err == nil {
		return nil
	}

	var reqErr requests.RequestError
	if !errors.As(err, &reqErr) {
		return err
	}

	msg := metricQueryErrorMessage(reqErr)
	if msg == "" {
		return err
	}

	return errorcategory.With(metricQueryError{RequestError: reqErr, userMsg: msg}, errorcategory.UserInput)
}

type metricQueryError struct {
	requests.RequestError
	userMsg string
}

func (e metricQueryError) Error() string { return e.userMsg }

func (e metricQueryError) Unwrap() error { return e.RequestError }

func metricQueryErrorMessage(reqErr requests.RequestError) string {
	apiMessage := stripeErrorMessage(reqErr)

	switch reqErr.ErrorCode {
	case "metric_invalid_parameter_value":
		// The API sometimes names the offending parameter ("limit cannot be
		// greater than 1,000"), which beats anything we can guess. Only fall
		// back to the checklist when it returns the useless retry boilerplate.
		if apiMessage != "" && !isGenericRetryMessage(apiMessage) {
			return apiMessage + "\nThis is a client error; retrying the same request will not succeed."
		}
		return strings.Join([]string{
			"Invalid metric query parameter (metric_invalid_parameter_value).",
			"Check:",
			"  --granularity: day, week, month, or year",
			"  --group-by: at most one dimension; valid names depend on the metric (" + supportedMetricsURL + ")",
			"  --filter: at most two dimensions and 10 values per dimension",
			"  --limit: 1–1000",
			"  --currency / --timezone: must be supported for the account",
			"This is a client error; retrying the same request will not succeed.",
		}, "\n")
	case "metric_invalid_parameter_use":
		return "Invalid metric query parameter (metric_invalid_parameter_use). Metrics in one request must share a namespace, and --group-by / --filter dimensions must exist on the metric. See " + supportedMetricsURL
	case "metric_invalid_time_range":
		return "Invalid metric query time range (metric_invalid_time_range). starts-at must be before ends-at, both must be in the past, and the range must be within the maximum for the requested granularity."
	case "not_found":
		if isAPIMethodNotFound(apiMessage) {
			return "The Analytics metric query endpoint was not found for this account (HTTP 404).\n" +
				"stripe data metrics run is a Private Preview API and returns this error until the account is enrolled.\n" +
				"This is not caused by --metric or --group-by."
		}
		if apiMessage != "" && strings.Contains(strings.ToLower(apiMessage), "metric") {
			return apiMessage + " Use a common name such as revenue.mrr, or a metric id from this account and mode. Live-mode ids are not valid in a sandbox."
		}
		if apiMessage != "" {
			return apiMessage
		}
		return "No metric found. Use a common name such as revenue.mrr, or a metric id from this account and mode."
	default:
		if reqErr.StatusCode >= 400 && reqErr.StatusCode < 500 && isGenericRetryMessage(apiMessage) {
			return "Metric query failed with a client error. Retrying the same request will not succeed. " + reqErr.Error()
		}
		return ""
	}
}

func stripeErrorMessage(reqErr requests.RequestError) string {
	body, ok := reqErr.Body.(string)
	if !ok || body == "" {
		return ""
	}
	var parsed struct {
		Error struct {
			Message string `json:"message"`
		} `json:"error"`
	}
	if err := json.Unmarshal([]byte(body), &parsed); err != nil {
		return ""
	}
	return parsed.Error.Message
}

func isGenericRetryMessage(message string) bool {
	lower := strings.ToLower(message)
	return strings.Contains(lower, "internal error") || strings.Contains(lower, "try again later")
}

func isAPIMethodNotFound(message string) bool {
	return strings.Contains(strings.ToLower(message), "api method cannot be found")
}
