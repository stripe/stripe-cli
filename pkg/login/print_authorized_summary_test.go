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

func TestPrintAuthorizedSummary_NoContexts(t *testing.T) {
	cfg, cleanup := setupOAuthTestConfig(t)
	defer cleanup()

	output := capturePrintedOutput(t, func() {
		printAuthorizedSummary(context.Background(), "http://unused.example", &cfg.Profile, "oak_test_123", nil)
	})

	assert.Contains(t, output, "Done! The Stripe CLI is configured with your credentials.")
	assert.NotContains(t, output, "Authorized (")
}

func TestPrintAuthorizedSummary_SingleContext(t *testing.T) {
	cfg, cleanup := setupOAuthTestConfig(t)
	defer cleanup()

	accounts := []config.AuthorizedAccount{
		{ID: "acct_only", Name: "Solo Co", Modes: []string{"test"}},
	}

	output := capturePrintedOutput(t, func() {
		printAuthorizedSummary(context.Background(), "http://unused.example", &cfg.Profile, "oak_test_123", accounts)
	})

	assert.Contains(t, output, "Done! The Stripe CLI is authorized for Solo Co · sandbox (acct_only)")
	assert.NotContains(t, output, "Authorized (", "a single context doesn't need the full grouped list")
	assert.NotContains(t, output, "Run 'stripe switch'", "there's nothing to switch between with only one context")
}

func TestPrintAuthorizedSummary_MultipleContextsUsesGroupedList(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.NoError(t, r.ParseForm())
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"contexts": [
			{"account_id": "acct_sandbox1", "livemode": false, "valid": true, "parent_account_id": "acct_live1"}
		]}`)) //nolint:errcheck
	}))
	defer ts.Close()

	cfg, cleanup := setupOAuthTestConfig(t)
	defer cleanup()
	require.NoError(t, cfg.Profile.SaveActiveContext("acct_live1", true))

	accounts := []config.AuthorizedAccount{
		{ID: "acct_live1", Name: "Acme Inc", Modes: []string{"live"}},
		{ID: "acct_sandbox1", Name: "Acme Dev", Modes: []string{"test"}},
	}

	output := capturePrintedOutput(t, func() {
		printAuthorizedSummary(context.Background(), ts.URL, &cfg.Profile, "oak_test_123", accounts)
	})

	assert.Contains(t, output, "Done!")
	assert.Contains(t, output, "Authorized (2 accounts):", "more than one context should use the same grouped list as whoami/login list/switch")
	assert.Contains(t, output, "└─ Acme Dev", "the sandbox's resolved parent should nest it, same as everywhere else")
	assert.Contains(t, output, "● active", "the active row is marked inline, so no separate 'Currently active' line is needed")
	assert.Contains(t, output, "Run 'stripe switch'")
}

func TestPrintAuthorizedSummary_FallsBackToFlatListWhenLookupFails(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer ts.Close()

	cfg, cleanup := setupOAuthTestConfig(t)
	defer cleanup()
	require.NoError(t, cfg.Profile.SaveActiveContext("acct_live1", true))

	accounts := []config.AuthorizedAccount{
		{ID: "acct_live1", Name: "Acme Inc", Modes: []string{"live"}},
		{ID: "acct_sandbox1", Name: "Acme Dev", Modes: []string{"test"}},
	}

	output := capturePrintedOutput(t, func() {
		printAuthorizedSummary(context.Background(), ts.URL, &cfg.Profile, "oak_test_123", accounts)
	})

	assert.Contains(t, output, "Authorized (2 accounts):")
	assert.NotContains(t, output, "└─", "a failed lookup means no grouping info, so the list should be flat")
}
