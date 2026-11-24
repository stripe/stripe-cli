package requests

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/stripe/stripe-cli/pkg/stripe"
)

func TestGetAccountFacts(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, http.MethodGet, r.Method)
		require.Equal(t, "/v1/stripecli/account_facts", r.URL.Path)
		require.Equal(t, "Bearer oak_test_123", r.Header.Get("Authorization"))
		require.Equal(t, "acct_123", r.Header.Get("Stripe-Context"))
		require.Equal(t, "true", r.Header.Get("Stripe-Livemode"))

		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"api_versions":["2025-03-31.basil","2024-06-20"],"default_api_version":"2024-06-20"}`))
	}))
	defer ts.Close()

	creds := stripe.NewOAKCredentials("oak_test_123", "acct_123", true)
	facts, err := GetAccountFacts(context.Background(), ts.URL, nil, creds, true)
	require.NoError(t, err)
	require.Equal(t, AccountFacts{
		APIVersions:       []string{"2025-03-31.basil", "2024-06-20"},
		DefaultAPIVersion: "2024-06-20",
	}, facts)
}

func TestGetAccountFacts_MalformedJSON(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"default_api_version":`))
	}))
	defer ts.Close()

	creds := stripe.NewOAKCredentials("oak_test_123", "acct_123", true)
	_, err := GetAccountFacts(context.Background(), ts.URL, nil, creds, true)
	require.ErrorContains(t, err, "failed to decode account facts response")
}

func TestGetAccountFacts_Non2xx(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte(`{"error":{"message":"Unrecognized request URL"}}`))
	}))
	defer ts.Close()

	creds := stripe.NewOAKCredentials("oak_test_123", "acct_123", true)
	_, err := GetAccountFacts(context.Background(), ts.URL, nil, creds, true)
	require.Error(t, err)
}
