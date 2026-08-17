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

func TestFetchAuthorizedAccounts_paginates(t *testing.T) {
	requests := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests++
		assert.Equal(t, "100", r.URL.Query().Get("limit"))
		w.Header().Set("Content-Type", "application/json")
		switch requests {
		case 1:
			assert.Empty(t, r.URL.Query().Get("starting_after"))
			json.NewEncoder(w).Encode(listAccountsResponse{
				Accounts: []config.AuthorizedAccount{{ID: "acct_first", Name: "First", Modes: []string{"live"}}},
				HasMore:  true,
			})
		case 2:
			assert.Equal(t, "acct_first", r.URL.Query().Get("starting_after"))
			json.NewEncoder(w).Encode(listAccountsResponse{
				Accounts: []config.AuthorizedAccount{{ID: "acct_second", Name: "Second", Modes: []string{"test"}}},
			})
		default:
			t.Fatalf("unexpected request %d", requests)
		}
	}))
	defer srv.Close()

	got, err := fetchAuthorizedAccounts(context.Background(), srv.URL, "oak_test")
	require.NoError(t, err)
	assert.Equal(t, []config.AuthorizedAccount{
		{ID: "acct_first", Name: "First", Modes: []string{"live"}},
		{ID: "acct_second", Name: "Second", Modes: []string{"test"}},
	}, got)
	assert.Equal(t, 2, requests)
}

func TestIsAuthorizedAccountsUnauthorized(t *testing.T) {
	assert.True(t, IsAuthorizedAccountsUnauthorized(&accountsRequestError{statusCode: http.StatusUnauthorized}))
	assert.False(t, IsAuthorizedAccountsUnauthorized(&accountsRequestError{statusCode: http.StatusForbidden}))
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
