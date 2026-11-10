package requests

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	log "github.com/sirupsen/logrus"
	"github.com/stretchr/testify/require"

	"github.com/stripe/stripe-cli/pkg/errorcategory"
	"github.com/stripe/stripe-cli/pkg/stripe"
)

func TestRequestDiagnosticsControls(t *testing.T) {
	for _, test := range []struct {
		name           string
		level          log.Level
		headers, debug bool
	}{
		{"default", log.InfoLevel, false, false},
		{"headers", log.InfoLevel, true, false},
		{"debug", log.DebugLevel, false, true},
		{"both", log.DebugLevel, true, true},
		{"trace", log.TraceLevel, false, true},
	} {
		t.Run(test.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Request-Id", "req_final123")
				w.Header().Set("X-Private", "header_secret")
				w.WriteHeader(http.StatusBadRequest)
				_, _ = w.Write([]byte(`{"error":{"code":"compartments_max_tree_size_reached","message":"body_secret oak_secret sk_secret"}}`))
			}))
			defer server.Close()
			var debug bytes.Buffer
			logger := log.StandardLogger()
			oldOut, oldLevel, oldFormatter := logger.Out, logger.Level, logger.Formatter
			log.SetOutput(&debug)
			log.SetLevel(test.level)
			log.SetFormatter(&log.JSONFormatter{})
			defer func() { log.SetOutput(oldOut); log.SetLevel(oldLevel); log.SetFormatter(oldFormatter) }()
			diagnostics := &Diagnostics{ShowHeaders: test.headers, Operation: "sandbox_create", Stage: "create"}
			base := &Base{APIBaseURL: server.URL, Method: http.MethodPost, SuppressOutput: true, Diagnostics: diagnostics}
			var requestErr error
			headers := captureStderr(t, func() {
				_, requestErr = base.MakeRequest(context.Background(), stripe.NewOAKCredentials("oak_secret", "acct_parent", true), "/v2/sandboxes", &RequestParameters{}, map[string]interface{}{"name": "request_secret"}, true, func(r *http.Request) error {
					r.Header.Set("Stripe-Context", "play_parent")
					return nil
				})
			})
			require.Error(t, requestErr)
			require.Equal(t, "req_final123", RequestIDFromError(requestErr))
			require.Equal(t, "req_final123", diagnostics.RequestID)
			require.Equal(t, test.headers, strings.Contains(headers, "< Request-Id: req_final123"))
			require.Equal(t, test.debug, debug.Len() > 0)
			if test.headers {
				require.Equal(t, 1, strings.Count(headers, "> POST "))
				require.Equal(t, 1, strings.Count(headers, "< HTTP 400"))
				require.Contains(t, headers, "Bearer [REDACTED]")
			}
			if test.debug {
				var fields map[string]interface{}
				require.NoError(t, json.Unmarshal(debug.Bytes(), &fields))
				for key, value := range map[string]interface{}{
					"operation": "sandbox_create", "stage": "create", "method": "POST", "endpoint": "/v2/sandboxes", "status": float64(400),
					"request_id": "req_final123", "stripe_context": "play_parent", "active_account": "acct_parent", "livemode": "true", "error_code": "compartments_max_tree_size_reached", "outcome": "http_error",
				} {
					require.Equal(t, value, fields[key], key)
				}
				elapsed, ok := fields["elapsed"].(float64)
				require.True(t, ok)
				require.GreaterOrEqual(t, elapsed, float64(0))
			}
			for _, secret := range []string{"body_secret", "request_secret", "oak_secret", "sk_secret", "header_secret"} {
				require.NotContains(t, headers+debug.String(), secret)
			}
		})
	}
}

func TestRequestIDCorrelation(t *testing.T) {
	original := errorcategory.New(errorcategory.API, "safe failure")
	for _, id := range []string{"", "req_", "req_bad\ninjected", "oak_secret", strings.Repeat("a", 200)} {
		require.Equal(t, original, WithRequestID(original, id))
	}
	wrapped := WithRequestID(original, "req_safe123")
	require.EqualError(t, wrapped, "safe failure (request ID: req_safe123)")
	require.ErrorIs(t, wrapped, original)
	category, ok := errorcategory.Get(wrapped)
	require.True(t, ok)
	require.Equal(t, errorcategory.API, category)
	require.Equal(t, wrapped, WithRequestID(wrapped, "req_safe123"))
	for _, err := range []error{RequestError{RequestID: "req_safe123"}, &RequestError{RequestID: "req_safe123"}} {
		require.Equal(t, "req_safe123", RequestIDFromError(fmt.Errorf("wrapped: %w", err)))
	}
	require.Empty(t, RequestIDFromError(errors.New("req_secret")))
	require.Nil(t, WithRequestID(nil, "req_safe123"))
}

func TestDiagnosticsSanitizeMetadata(t *testing.T) {
	require.Empty(t, safeDiagnosticValue("req_123\nsecret", safeRequestID))
	require.Empty(t, safeDiagnosticValue("bad code body_secret", safeErrorCode))
	require.Empty(t, safeDiagnosticValue("oak_secret", safeContext))
	require.Empty(t, safeDiagnosticValue("play_parent\nsecret", safeContext))
}
