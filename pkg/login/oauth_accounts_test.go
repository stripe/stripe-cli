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
	"unicode/utf8"

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
		{ID: "acct_orphan2", Name: "Second Mystery", Modes: []string{"test"}},
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
	orphanIdx := indexOf(t, output, "Mystery Sandbox")
	orphan2Idx := indexOf(t, output, "Second Mystery")

	assert.Less(t, liveIdx, sandbox1Idx, "parent account should print before its sandboxes")
	assert.Less(t, sandbox1Idx, sandbox2Idx)
	assert.Less(t, sandbox2Idx, orphanIdx, "standalone sandboxes should print after the grouped ones")
	assert.Less(t, orphanIdx, orphan2Idx)
	assert.Contains(t, output, "Authorized (5 accounts):")
	assert.NotContains(t, output, "Other sandboxes", "standalone sandboxes shouldn't be grouped under a catch-all heading")
	assert.Contains(t, output, "└─ Acme Dev", "sandboxes with a known parent should render with a tree connector showing they belong to it")
	assert.Contains(t, output, "└─ Acme Staging")
	assert.NotContains(t, output, "└─ Acme Inc", "the parent account's own row shouldn't be nested under itself")
	assert.NotContains(t, output, "└─ Mystery Sandbox", "a standalone sandbox has no parent to connect to, so no connector")
	assert.NotContains(t, output, "└─ Second Mystery")

	// The id column should line up across rows even though the header and
	// nested indents are different widths. Measure the display column (rune
	// count), not the byte offset, since the "└─" connector is multi-byte
	// UTF-8 but renders as single-width characters.
	idColumn := func(line string) int {
		idx := strings.Index(line, "acct_")
		require.NotEqual(t, -1, idx, "expected line %q to contain an account id", line)
		return utf8.RuneCountInString(line[:idx])
	}
	lines := strings.Split(strings.TrimRight(output, "\n"), "\n")
	var idColumns []int
	for _, line := range lines {
		if strings.Contains(line, "acct_") {
			idColumns = append(idColumns, idColumn(line))
		}
	}
	require.Len(t, idColumns, 5)
	for _, col := range idColumns[1:] {
		assert.Equal(t, idColumns[0], col, "the id column should be at the same offset on every row")
	}
}

func TestPrintGroupedAuthorizedAccounts_StandaloneSandboxesShownAsRegularAccounts(t *testing.T) {
	cfg, cleanup := setupOAuthTestConfig(t)
	defer cleanup()

	accounts := []config.AuthorizedAccount{
		{ID: "acct_sandbox1", Name: "Solo Sandbox", Modes: []string{"test"}},
		{ID: "acct_sandbox2", Name: "Another Sandbox", Modes: []string{"test"}},
	}

	output := capturePrintedOutput(t, func() {
		PrintGroupedAuthorizedAccounts(&cfg.Profile, accounts, nil)
	})

	assert.NotContains(t, output, "Other sandboxes", "standalone sandboxes shouldn't be grouped under a catch-all heading, no matter how many there are")
	assert.NotContains(t, output, "└─", "a standalone sandbox isn't nested under anything, so no connector")
	assert.Contains(t, output, "Solo Sandbox")
	assert.Contains(t, output, "Another Sandbox")
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
	assert.Contains(t, output, "└─ Test mode", "the legacy same-ID test row should be labeled Test mode and connected to its live row")
	assert.Contains(t, output, "└─ Acme Dev", "a distinct sandbox account should still show its own name, connected to its parent")
	assert.Equal(t, 1, strings.Count(output, "Acme Inc"), "the account name shouldn't be repeated for its own test-mode row")
}

func indexOf(t *testing.T, s, substr string) int {
	t.Helper()
	idx := strings.Index(s, substr)
	require.NotEqual(t, -1, idx, "expected %q to contain %q", s, substr)
	return idx
}
