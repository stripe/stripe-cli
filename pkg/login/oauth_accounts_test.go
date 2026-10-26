package login

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/stripe/stripe-cli/pkg/config"
)

// capturePrintedOutput redirects os.Stdout for the duration of fn and returns
// everything written to it. PrintGroupedAuthorizedAccounts and its siblings
// write via fmt.Print* directly to os.Stdout rather than an injectable
// io.Writer, so tests must intercept it at that level.
func capturePrintedOutput(t *testing.T, fn func()) string {
	t.Helper()
	orig := os.Stdout
	r, w, err := os.Pipe()
	require.NoError(t, err)
	os.Stdout = w

	fn()

	w.Close()
	os.Stdout = orig
	var buf bytes.Buffer
	buf.ReadFrom(r) //nolint:errcheck
	return buf.String()
}

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

	require.NoError(t, config.KeyRing.Set(cfg.Profile.KeyringKey(config.UATKeychainItemKey), []byte("oak_test"), ""))
	require.NoError(t, cfg.Profile.SaveActiveContext("acct_456", true))

	result, err := ListAuthorizedAccountsForActiveSession(context.Background(), srv.URL, cfg)
	require.NoError(t, err)
	assert.Equal(t, want, result.Accounts)
	assert.Equal(t, "acct_456", result.ActiveAccountID)
	assert.True(t, result.ActiveLivemode)
}

func TestPrintGroupedAuthorizedAccounts_GroupsSandboxesUnderParent(t *testing.T) {
	cfg, cleanup := setupOAuthTestConfig(t)
	defer cleanup()

	accounts := []config.AuthorizedAccount{
		{ID: "acct_live1", Name: "Acme Inc", Modes: []string{"live"}},
		{ID: "acct_sandbox1", Name: "Acme Dev", Modes: []string{"test"}},
		{ID: "acct_sandbox2", Name: "Acme Staging", Modes: []string{"test"}},
		{ID: "acct_orphan", Name: "Mystery Sandbox", Modes: []string{"test"}},
	}
	parentAccountIDs := map[string]string{
		"acct_sandbox1": "acct_live1",
		"acct_sandbox2": "acct_live1",
	}

	output := capturePrintedOutput(t, func() {
		PrintGroupedAuthorizedAccounts(&cfg.Profile, accounts, parentAccountIDs)
	})

	liveIdx := indexOf(t, output, "Acme Inc")
	sandbox1Idx := indexOf(t, output, "Acme Dev")
	sandbox2Idx := indexOf(t, output, "Acme Staging")
	otherHeadingIdx := indexOf(t, output, "Other sandboxes")
	orphanIdx := indexOf(t, output, "Mystery Sandbox")

	assert.Less(t, liveIdx, sandbox1Idx, "parent account should print before its sandboxes")
	assert.Less(t, sandbox1Idx, sandbox2Idx)
	assert.Less(t, sandbox2Idx, otherHeadingIdx, "grouped sandboxes should print before the Other sandboxes section")
	assert.Less(t, otherHeadingIdx, orphanIdx, "sandboxes without a known parent should print under Other sandboxes")
	assert.Contains(t, output, "Authorized (4 accounts):")
}

func TestPrintGroupedAuthorizedAccounts_NoParentsKnown(t *testing.T) {
	cfg, cleanup := setupOAuthTestConfig(t)
	defer cleanup()

	accounts := []config.AuthorizedAccount{
		{ID: "acct_sandbox1", Name: "Solo Sandbox", Modes: []string{"test"}},
	}

	output := capturePrintedOutput(t, func() {
		PrintGroupedAuthorizedAccounts(&cfg.Profile, accounts, nil)
	})

	assert.Contains(t, output, "Other sandboxes:")
	assert.Contains(t, output, "Solo Sandbox")
}

func TestPrintGroupedAuthorizedAccounts_LegacyCombinedAccountShowsTestMode(t *testing.T) {
	cfg, cleanup := setupOAuthTestConfig(t)
	defer cleanup()

	accounts := []config.AuthorizedAccount{
		{ID: "acct_legacy", Name: "Acme Inc", Modes: []string{"live", "test"}},
		{ID: "acct_sandbox1", Name: "Acme Dev", Modes: []string{"test"}},
	}
	parentAccountIDs := map[string]string{"acct_sandbox1": "acct_legacy"}

	output := capturePrintedOutput(t, func() {
		PrintGroupedAuthorizedAccounts(&cfg.Profile, accounts, parentAccountIDs)
	})

	assert.Contains(t, output, "Acme Inc", "the live row should still show the account name")
	assert.Contains(t, output, "Test mode", "the legacy same-ID test row should be labeled Test mode, not repeat the account name")
	assert.Contains(t, output, "Acme Dev", "a distinct sandbox account should still show its own name")
	assert.Equal(t, 1, strings.Count(output, "Acme Inc"), "the account name shouldn't be repeated for its own test-mode row")
}

func indexOf(t *testing.T, s, substr string) int {
	t.Helper()
	idx := strings.Index(s, substr)
	require.NotEqual(t, -1, idx, "expected %q to contain %q", s, substr)
	return idx
}
