package requests

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/stripe/stripe-cli/pkg/stripe"
)

func TestGetUserInfo(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, http.MethodGet, r.Method)
		require.Equal(t, "/v1/stripecli/user_info", r.URL.Path)
		require.Equal(t, "Bearer oak_test_123", r.Header.Get("Authorization"))
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"id": "usr_123", "email": "dwood@stripe.com", "role": "Administrator"}`)) //nolint:errcheck
	}))
	defer ts.Close()

	creds := stripe.NewOAKCredentials("oak_test_123", "acct_123", true)
	info, err := GetUserInfo(context.Background(), ts.URL, nil, creds, true)
	require.NoError(t, err)
	require.Equal(t, UserInfo{ID: "usr_123", Email: "dwood@stripe.com", Role: "Administrator"}, info)
}

func TestGetUserInfo_OmitsOptionalFields(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"id": "usr_123"}`)) //nolint:errcheck
	}))
	defer ts.Close()

	creds := stripe.NewOAKCredentials("oak_test_123", "acct_123", true)
	info, err := GetUserInfo(context.Background(), ts.URL, nil, creds, true)
	require.NoError(t, err)
	require.Equal(t, UserInfo{ID: "usr_123"}, info)
}

func TestGetUserInfo_ErrOnStatus(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte(`{"error": {"message": "invalid token"}}`)) //nolint:errcheck
	}))
	defer ts.Close()

	creds := stripe.NewOAKCredentials("oak_test_123", "acct_123", true)
	_, err := GetUserInfo(context.Background(), ts.URL, nil, creds, true)
	require.Error(t, err)
}
