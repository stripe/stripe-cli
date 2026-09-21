package login

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/stripe/stripe-cli/pkg/config"
)

func TestFetchAuthorizedAccounts_success(t *testing.T) {
	want := []config.AuthorizedAccount{
		{ID: "acct_123", Name: "Test Co", Modes: []string{"test"}},
		{ID: "acct_456", Name: "Live Co", Modes: []string{"live", "test"}},
	}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodGet, r.Method)
		assert.Equal(t, "/stripecli/oauth2/token/accounts", r.URL.Path)
		assert.Equal(t, "Bearer oak_test", r.Header.Get("Authorization"))
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(listAccountsResponse{Accounts: want}) //nolint:errcheck
	}))
	defer srv.Close()

	got, err := fetchAuthorizedAccounts(context.Background(), srv.URL, "oak_test")
	require.NoError(t, err)
	assert.Equal(t, want, got)
}

func TestFetchAuthorizedAccounts_non200(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "not found", http.StatusNotFound)
	}))
	defer srv.Close()

	_, err := fetchAuthorizedAccounts(context.Background(), srv.URL, "oak_test")
	assert.ErrorContains(t, err, "404")
}

func TestListAuthorizedAccounts_returnsRealDataWhenAvailable(t *testing.T) {
	want := []config.AuthorizedAccount{
		{ID: "acct_real", Name: "Real Business", Modes: []string{"live", "test"}},
	}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(listAccountsResponse{Accounts: want}) //nolint:errcheck
	}))
	defer srv.Close()

	got, err := ListAuthorizedAccounts(context.Background(), srv.URL, "oak_test")
	require.NoError(t, err)
	assert.Equal(t, want, got)
}

func TestListAuthorizedAccountsForActiveSession_NotLoggedIn(t *testing.T) {
	cfg, cleanup := setupOAuthTestConfig(t)
	defer cleanup()

	_, err := ListAuthorizedAccountsForActiveSession(context.Background(), "https://access.example", cfg)
	assert.ErrorContains(t, err, "not logged in")
}

func TestListAuthorizedAccountsForActiveSession_ReturnsAccountsAndActiveContext(t *testing.T) {
	cfg, cleanup := setupOAuthTestConfig(t)
	defer cleanup()

	want := []config.AuthorizedAccount{
		{ID: "acct_123", Name: "Test Co", Modes: []string{"test"}},
		{ID: "acct_456", Name: "Live Co", Modes: []string{"live", "test"}},
	}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(listAccountsResponse{Accounts: want}) //nolint:errcheck
	}))
	defer srv.Close()

	require.NoError(t, config.KeyRing.Set(config.UATKeychainItemKey, []byte("oak_test"), ""))
	require.NoError(t, config.SaveActiveContext("acct_456", true))

	result, err := ListAuthorizedAccountsForActiveSession(context.Background(), srv.URL, cfg)
	require.NoError(t, err)
	assert.Equal(t, want, result.Accounts)
	assert.Equal(t, "acct_456", result.ActiveAccountID)
	assert.True(t, result.ActiveLivemode)
}
