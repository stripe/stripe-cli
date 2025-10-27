package logtailing

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/stripe/stripe-cli/pkg/stripe"
	"github.com/stripe/stripe-cli/pkg/websocket"
)

func TestJsonifyFiltersAll(t *testing.T) {
	filters := &LogFilters{
		FilterAccount:        []string{"my-account"},
		FilterIPAddress:      []string{"my-ip-address"},
		FilterHTTPMethod:     []string{"my-http-method"},
		FilterRequestPath:    []string{"my-request-path"},
		FilterRequestStatus:  []string{"my-request-status"},
		FilterSource:         []string{"my-source"},
		FilterStatusCode:     []string{"my-status-code"},
		FilterStatusCodeType: []string{"my-status-code-type"},
	}
	expected := `{"filter_account":["my-account"],"filter_ip_address":["my-ip-address"],"filter_http_method":["my-http-method"],"filter_request_path":["my-request-path"],"filter_request_status":["my-request-status"],"filter_source":["my-source"],"filter_status_code":["my-status-code"],"filter_status_code_type":["my-status-code-type"]}`
	filtersStr, err := jsonifyFilters(filters)
	require.NoError(t, err)
	require.Equal(t, expected, filtersStr)
}

func TestJsonifyFiltersSome(t *testing.T) {
	filters := &LogFilters{
		FilterHTTPMethod: []string{"my-http-method"},
		FilterStatusCode: []string{"my-status-code"},
	}
	expected := `{"filter_http_method":["my-http-method"],"filter_status_code":["my-status-code"]}`
	filtersStr, err := jsonifyFilters(filters)
	require.NoError(t, err)
	require.Equal(t, expected, filtersStr)
}

func TestJsonifyFiltersEmpty(t *testing.T) {
	filters := &LogFilters{
		FilterAccount:        []string{},
		FilterIPAddress:      []string{},
		FilterHTTPMethod:     []string{},
		FilterRequestPath:    []string{},
		FilterRequestStatus:  []string{},
		FilterSource:         []string{},
		FilterStatusCode:     []string{},
		FilterStatusCodeType: []string{},
	}
	filtersStr, err := jsonifyFilters(filters)
	require.NoError(t, err)
	require.Equal(t, "{}", filtersStr)
}

func TestRun_RetryOnAuthorizationServerError(t *testing.T) {
	nAttempts := 0
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		nAttempts++
		require.Equal(t, http.MethodPost, r.Method)
		require.Equal(t, "/v1/stripecli/sessions", r.URL.Path)
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(`{"error":"internal_server_error"}`))
	}))
	defer ts.Close()

	baseURL, _ := url.Parse(ts.URL)

	cfg := Config{
		Client: &stripe.Client{APIKey: "sk_test_123", BaseURL: baseURL},
		OutCh:  make(chan websocket.IElement, 2),
	}
	tailer := New(&cfg)
	err := tailer.Run(context.Background())
	require.Error(t, err)
	require.Equal(t, 6, nAttempts)
}

func TestRun_NoRetryOnAuthorizationClientError(t *testing.T) {
	nAttempts := 0
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		nAttempts++
		require.Equal(t, http.MethodPost, r.Method)
		require.Equal(t, "/v1/stripecli/sessions", r.URL.Path)
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(`{"error":"bad_request"}`))
	}))
	defer ts.Close()

	baseURL, _ := url.Parse(ts.URL)

	cfg := Config{
		Client: &stripe.Client{APIKey: "sk_test_123", BaseURL: baseURL},
		OutCh:  make(chan websocket.IElement, 2),
	}
	tailer := New(&cfg)
	err := tailer.Run(context.Background())
	require.Error(t, err)
	require.Equal(t, 1, nAttempts)
}

func TestRun_NoRetryOnAuthorizationClientError_TooManyRequests(t *testing.T) {
	nAttempts := 0
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		nAttempts++
		require.Equal(t, http.MethodPost, r.Method)
		require.Equal(t, "/v1/stripecli/sessions", r.URL.Path)
		w.WriteHeader(http.StatusTooManyRequests)
		w.Write([]byte(`{"error":"too_many_requests"}`))
	}))
	defer ts.Close()

	baseURL, _ := url.Parse(ts.URL)

	cfg := Config{
		Client: &stripe.Client{APIKey: "sk_test_123", BaseURL: baseURL},
		OutCh:  make(chan websocket.IElement, 2),
	}
	tailer := New(&cfg)
	err := tailer.Run(context.Background())
	require.ErrorContains(t, err, "You have too many `stripe logs tail` sessions open. Please close some and try again.")
	require.Equal(t, 1, nAttempts)
}
