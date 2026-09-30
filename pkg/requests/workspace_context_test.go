package requests

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/stripe/stripe-cli/pkg/stripe"
)

func TestGetWorkspaceContext(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, http.MethodGet, r.Method)
		require.Equal(t, "/v1/stripecli/workspace_context", r.URL.Path)
		require.Empty(t, r.URL.RawQuery)
		require.Equal(t, "Bearer oak_test_123", r.Header.Get("Authorization"))
		require.Equal(t, "acct_123", r.Header.Get("Stripe-Context"))
		require.Equal(t, "true", r.Header.Get("Stripe-Livemode"))

		body, err := io.ReadAll(r.Body)
		require.NoError(t, err)
		require.Empty(t, body)

		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"workspace_id":"wksp_live_123"}`))
	}))
	defer ts.Close()

	creds := stripe.NewOAKCredentials("oak_test_123", "acct_123", true)
	workspaceContext, err := GetWorkspaceContext(context.Background(), ts.URL, nil, creds, true)
	require.NoError(t, err)
	require.Equal(t, WorkspaceContext{WorkspaceID: "wksp_live_123"}, workspaceContext)
}

func TestGetWorkspaceContext_MalformedJSON(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"workspace_id":`))
	}))
	defer ts.Close()

	creds := stripe.NewOAKCredentials("oak_test_123", "acct_123", true)
	_, err := GetWorkspaceContext(context.Background(), ts.URL, nil, creds, true)
	require.Error(t, err)
	require.ErrorContains(t, err, "failed to decode workspace context response")
}

func TestGetWorkspaceContext_Non2xx(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadGateway)
		_, _ = w.Write([]byte(`{"error":{"message":"backend unavailable"}}`))
	}))
	defer ts.Close()

	creds := stripe.NewOAKCredentials("oak_test_123", "acct_123", true)
	_, err := GetWorkspaceContext(context.Background(), ts.URL, nil, creds, true)
	require.Error(t, err)

	var requestErr RequestError
	require.True(t, errors.As(err, &requestErr))
	require.Equal(t, http.StatusBadGateway, requestErr.StatusCode)
}
