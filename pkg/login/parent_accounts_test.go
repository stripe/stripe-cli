package login

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/stripe/stripe-cli/pkg/config"
)

func TestResolveParentAccounts_OnlyQueriesSandboxes(t *testing.T) {
	var gotAccountIDs []string
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// The API 403s without an OAK context identifying the caller's
		// active account/livemode, so these must always be sent.
		assert.Equal(t, "acct_live1", r.Header.Get("Stripe-Context"))
		assert.Equal(t, "true", r.Header.Get("Stripe-Livemode"))

		require.NoError(t, r.ParseForm())
		gotAccountIDs = append(gotAccountIDs, r.Form.Get("contexts[0][account_id]"), r.Form.Get("contexts[1][account_id]"))
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"contexts": [
			{"account_id": "acct_sandbox1", "livemode": false, "valid": true, "parent_account_id": "acct_live1"},
			{"account_id": "acct_sandbox2", "livemode": false, "valid": true}
		]}`)) //nolint:errcheck
	}))
	defer ts.Close()

	accounts := []config.AuthorizedAccount{
		{ID: "acct_live1", Name: "Acme Inc", Modes: []string{"live"}},
		{ID: "acct_sandbox1", Name: "Acme Dev", Modes: []string{"test"}},
		{ID: "acct_sandbox2", Name: "Acme Staging", Modes: []string{"test"}},
	}
	ac := &config.ActiveContext{AccountID: "acct_live1", Livemode: true}

	got, ok := ResolveParentAccounts(context.Background(), ts.URL, nil, "oak_test_123", ac, accounts)

	assert.Equal(t, []string{"acct_sandbox1", "acct_sandbox2"}, gotAccountIDs, "accounts with a live mode shouldn't be queried for a parent")
	assert.True(t, ok)
	assert.Equal(t, map[string]string{"acct_sandbox1": "acct_live1"}, got)
}

func TestResolveParentAccounts_FailsOnError(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer ts.Close()

	accounts := []config.AuthorizedAccount{{ID: "acct_sandbox1", Name: "Acme Dev", Modes: []string{"test"}}}
	ac := &config.ActiveContext{AccountID: "acct_live1", Livemode: true}

	got, ok := ResolveParentAccounts(context.Background(), ts.URL, nil, "oak_test_123", ac, accounts)
	assert.False(t, ok, "a lookup failure should be reported so the caller can fall back to an ungrouped list")
	assert.Nil(t, got)
}

func TestResolveParentAccounts_NoActiveContextSkipsLookup(t *testing.T) {
	var requestReceived bool
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestReceived = true
		w.WriteHeader(http.StatusOK)
	}))
	defer ts.Close()

	accounts := []config.AuthorizedAccount{{ID: "acct_sandbox1", Name: "Acme Dev", Modes: []string{"test"}}}

	got, ok := ResolveParentAccounts(context.Background(), ts.URL, nil, "oak_test_123", nil, accounts)

	assert.True(t, ok, "no active context isn't itself a failure, just nothing to look up with")
	assert.Nil(t, got)
	assert.False(t, requestReceived, "without an active context there's nothing to authenticate the lookup with, so it shouldn't be attempted")
}

func TestPrintAuthorizedAccounts_FallsBackToFlatListOnError(t *testing.T) {
	cfg, cleanup := setupOAuthTestConfig(t)
	defer cleanup()
	require.NoError(t, cfg.Profile.SaveActiveContext("acct_live1", true))

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer ts.Close()

	accounts := []config.AuthorizedAccount{
		{ID: "acct_live1", Name: "Acme Inc", Modes: []string{"live"}},
		{ID: "acct_sandbox1", Name: "Acme Dev", Modes: []string{"test"}},
	}

	output := capturePrintedOutput(t, func() {
		PrintAuthorizedAccounts(context.Background(), ts.URL, &cfg.Profile, "oak_test_123", accounts)
	})

	assert.Contains(t, output, "Authorized (2 accounts):")
	assert.NotContains(t, output, "Other sandboxes", "a failed lookup shouldn't be reported as 'checked, no parent found'")
	assert.NotContains(t, output, "└─", "a failed lookup means we have no grouping info, so the list should be flat")
}
