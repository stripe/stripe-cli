package cmd

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/stripe/stripe-cli/pkg/config"
	"github.com/stripe/stripe-cli/pkg/keyring"
	"github.com/stripe/stripe-cli/pkg/requests"
)

func runWhoami(t *testing.T, wc *whoamiCmd) (string, error) {
	t.Helper()
	buf := new(bytes.Buffer)
	wc.cmd.SetOut(buf)
	err := wc.runWhoamiCmd(wc.cmd, []string{})
	return buf.String(), err
}

func TestWhoamiNotAuthenticated(t *testing.T) {
	viper.Reset()
	config.KeyRing = keyring.NewMemoryStore(nil)

	wc := newWhoamiCmd()
	wc.profile = &config.Profile{
		ProfileName: "default",
		DeviceName:  "test-device",
	}

	out, err := runWhoami(t, wc)
	assert.ErrorIs(t, err, errNotAuthenticated)
	assert.Regexp(t, `Profile:\s+default`, out)
	assert.Regexp(t, `Authenticated:\s+false`, out)
}

func TestWhoamiNotAuthenticatedJSON(t *testing.T) {
	viper.Reset()
	config.KeyRing = keyring.NewMemoryStore(nil)

	wc := newWhoamiCmd()
	wc.profile = &config.Profile{
		ProfileName: "default",
		DeviceName:  "test-device",
	}
	wc.format = "json"

	out, err := runWhoami(t, wc)
	assert.ErrorIs(t, err, errNotAuthenticated)

	var result whoamiOutput
	require.NoError(t, json.Unmarshal([]byte(out), &result))

	assert.False(t, result.Authenticated)
	assert.Equal(t, "default", result.ProfileName)
	assert.False(t, result.TestModeKey.Available)
	assert.False(t, result.LiveModeKey.Available)
	assert.Nil(t, result.TestModeKey.ExpiresAt)
	assert.Nil(t, result.LiveModeKey.ExpiresAt)
	assert.Equal(t, requests.StripeVersionHeaderValue, result.APIVersion)
	assert.Equal(t, requests.StripePreviewVersionHeaderValue, result.PreviewAPIVersion)
}

func TestWhoamiWithTestKey(t *testing.T) {
	config.KeyRing = keyring.NewMemoryStore(nil)

	wc := newWhoamiCmd()
	wc.profile = &config.Profile{
		ProfileName: "default",
		DeviceName:  "test-device",
		APIKey:      "sk_test_1234567890abcdef",
		AccountID:   "acct_123",
	}
	wc.format = "json"

	out, err := runWhoami(t, wc)
	require.NoError(t, err)

	var result whoamiOutput
	require.NoError(t, json.Unmarshal([]byte(out), &result))

	assert.True(t, result.Authenticated)
	assert.True(t, result.TestModeKey.Available)
	assert.Nil(t, result.TestModeKey.ExpiresAt, "override keys have no expiry")
	assert.False(t, result.LiveModeKey.Available)
	assert.Equal(t, "acct_123", result.AccountID)
}

func TestWhoamiWithLiveModeAPIKey(t *testing.T) {
	config.KeyRing = keyring.NewMemoryStore(nil)

	wc := newWhoamiCmd()
	wc.profile = &config.Profile{
		ProfileName: "default",
		DeviceName:  "test-device",
		APIKey:      "sk_live_1234567890abcdef",
	}
	wc.format = "json"

	out, err := runWhoami(t, wc)
	require.NoError(t, err)

	var result whoamiOutput
	require.NoError(t, json.Unmarshal([]byte(out), &result))

	assert.True(t, result.Authenticated)
	assert.False(t, result.TestModeKey.Available)
	assert.True(t, result.LiveModeKey.Available)
	assert.Nil(t, result.LiveModeKey.ExpiresAt, "override keys have no expiry")
}

func TestWhoamiWithEnvVarKey(t *testing.T) {
	config.KeyRing = keyring.NewMemoryStore(nil)
	t.Setenv("STRIPE_API_KEY", "sk_test_envvar1234567890")

	wc := newWhoamiCmd()
	wc.profile = &config.Profile{
		ProfileName: "default",
		DeviceName:  "test-device",
	}
	wc.format = "json"

	out, err := runWhoami(t, wc)
	require.NoError(t, err)

	var result whoamiOutput
	require.NoError(t, json.Unmarshal([]byte(out), &result))

	assert.True(t, result.Authenticated)
	assert.True(t, result.TestModeKey.Available)
	assert.Nil(t, result.TestModeKey.ExpiresAt, "override keys have no expiry")
	assert.False(t, result.LiveModeKey.Available)
}

func TestWhoamiWithLiveModeEnvVarKey(t *testing.T) {
	config.KeyRing = keyring.NewMemoryStore(nil)
	t.Setenv("STRIPE_API_KEY", "rk_live_envvar1234567890")

	wc := newWhoamiCmd()
	wc.profile = &config.Profile{
		ProfileName: "default",
		DeviceName:  "test-device",
	}
	wc.format = "json"

	out, err := runWhoami(t, wc)
	require.NoError(t, err)

	var result whoamiOutput
	require.NoError(t, json.Unmarshal([]byte(out), &result))

	assert.True(t, result.Authenticated)
	assert.False(t, result.TestModeKey.Available)
	assert.True(t, result.LiveModeKey.Available)
	assert.Nil(t, result.LiveModeKey.ExpiresAt, "override keys have no expiry")
}

func TestWhoamiLiveModeKeyDetected(t *testing.T) {
	config.KeyRing = keyring.NewMemoryStore(map[string][]byte{
		"default.live_mode_api_key": []byte("rk_live_1234567890abcdef"),
	})

	wc := newWhoamiCmd()
	wc.profile = &config.Profile{
		ProfileName: "default",
	}
	wc.format = "json"

	out, err := runWhoami(t, wc)
	require.NoError(t, err)

	var result whoamiOutput
	require.NoError(t, json.Unmarshal([]byte(out), &result))

	assert.True(t, result.Authenticated)
	assert.True(t, result.LiveModeKey.Available)
}

func TestWhoamiOAuth_RejectsArbitraryAccessBaseWithoutSendingUAT(t *testing.T) {
	config.KeyRing = keyring.NewMemoryStore(map[string][]byte{
		config.UATKeychainItemKey: []byte("oak_live_secret_token"),
	})
	t.Cleanup(func() { config.KeyRing = nil })

	var requestReceived bool
	attacker := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestReceived = true
		assert.Empty(t, r.Header.Get("Authorization"), "attacker-controlled server must never see the UAT")
		w.WriteHeader(http.StatusOK)
	}))
	defer attacker.Close()

	wc := newWhoamiCmd()
	wc.profile = &config.Profile{
		ProfileName: "default",
		DeviceName:  "test-device",
	}
	wc.accessBaseURL = attacker.URL

	_, err := runWhoami(t, wc)
	require.Error(t, err)
	assert.False(t, requestReceived, "an arbitrary --access-base must be rejected before any request is sent")
}

func TestBuildOAuthWhoamiOutput_NoActiveContext(t *testing.T) {
	accounts := []config.AuthorizedAccount{{ID: "acct_123", Name: "Acme", Modes: []string{"test", "live"}}}

	out := buildOAuthWhoamiOutput(accounts, nil, requests.UserInfo{}, time.Time{}, false)

	assert.Empty(t, out.AccountID)
	assert.Empty(t, out.Mode)
	assert.Nil(t, out.ExpiresAt)
	assert.Equal(t, accounts, out.AuthorizedAccounts)
}

func TestBuildOAuthWhoamiOutput_TestModeActiveContext(t *testing.T) {
	accounts := []config.AuthorizedAccount{{ID: "acct_123", Name: "Acme", Modes: []string{"test", "live"}}}
	ac := &config.ActiveContext{AccountID: "acct_123", Livemode: false}
	info := requests.UserInfo{Email: "user@example.com", Role: "admin"}
	expiresAt := time.Date(2026, 5, 1, 0, 0, 0, 0, time.FixedZone("PDT", -7*60*60))

	out := buildOAuthWhoamiOutput(accounts, ac, info, expiresAt, true)

	assert.Equal(t, "acct_123", out.AccountID)
	assert.Equal(t, "Acme", out.DisplayName)
	assert.Equal(t, "user@example.com", out.Email)
	assert.Equal(t, "admin", out.Role)
	assert.Equal(t, "test", out.Mode)
	require.NotNil(t, out.ExpiresAt)
	assert.Equal(t, expiresAt.Unix(), *out.ExpiresAt, "expires_at is rendered as a unix timestamp")
}

func TestBuildOAuthWhoamiOutput_LiveModeActiveContextUnknownAccount(t *testing.T) {
	accounts := []config.AuthorizedAccount{{ID: "acct_123", Name: "Acme", Modes: []string{"test", "live"}}}
	ac := &config.ActiveContext{AccountID: "acct_999", Livemode: true}

	out := buildOAuthWhoamiOutput(accounts, ac, requests.UserInfo{}, time.Time{}, false)

	assert.Equal(t, "acct_999", out.AccountID)
	assert.Equal(t, "acct_999", out.DisplayName, "falls back to account ID when not found among authorized accounts")
	assert.Equal(t, "live", out.Mode)
	assert.Nil(t, out.ExpiresAt)
}

func TestDisplayModeText(t *testing.T) {
	assert.Equal(t, "sandbox", displayModeText("test"))
	assert.Equal(t, "live", displayModeText("live"))
}

func TestKeyAvailabilityText(t *testing.T) {
	expiresAt := "2026-05-01"

	assert.Equal(t, "not available", keyAvailabilityText(whoamiKeyInfo{Available: false}))
	assert.Equal(t, "available", keyAvailabilityText(whoamiKeyInfo{Available: true}))
	assert.Equal(t, "available (expires 2026-05-01)", keyAvailabilityText(whoamiKeyInfo{Available: true, ExpiresAt: &expiresAt}))
}

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

	wc := newWhoamiCmd()
	wc.profile = &config.Profile{ProfileName: "default"}
	wc.apiBaseURL = ts.URL

	accounts := []config.AuthorizedAccount{
		{ID: "acct_live1", Name: "Acme Inc", Modes: []string{"live"}},
		{ID: "acct_sandbox1", Name: "Acme Dev", Modes: []string{"test"}},
		{ID: "acct_sandbox2", Name: "Acme Staging", Modes: []string{"test"}},
	}
	ac := &config.ActiveContext{AccountID: "acct_live1", Livemode: true}

	got, ok := wc.resolveParentAccounts(context.Background(), "oak_test_123", ac, accounts)

	assert.Equal(t, []string{"acct_sandbox1", "acct_sandbox2"}, gotAccountIDs, "accounts with a live mode shouldn't be queried for a parent")
	assert.True(t, ok)
	assert.Equal(t, map[string]string{"acct_sandbox1": "acct_live1"}, got)
}

func TestResolveParentAccounts_FailsOnError(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer ts.Close()

	wc := newWhoamiCmd()
	wc.profile = &config.Profile{ProfileName: "default"}
	wc.apiBaseURL = ts.URL

	accounts := []config.AuthorizedAccount{{ID: "acct_sandbox1", Name: "Acme Dev", Modes: []string{"test"}}}
	ac := &config.ActiveContext{AccountID: "acct_live1", Livemode: true}

	got, ok := wc.resolveParentAccounts(context.Background(), "oak_test_123", ac, accounts)
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

	wc := newWhoamiCmd()
	wc.profile = &config.Profile{ProfileName: "default"}
	wc.apiBaseURL = ts.URL

	accounts := []config.AuthorizedAccount{{ID: "acct_sandbox1", Name: "Acme Dev", Modes: []string{"test"}}}

	got, ok := wc.resolveParentAccounts(context.Background(), "oak_test_123", nil, accounts)

	assert.True(t, ok, "no active context isn't itself a failure, just nothing to look up with")
	assert.Nil(t, got)
	assert.False(t, requestReceived, "without an active context there's nothing to authenticate the lookup with, so it shouldn't be attempted")
}

// TestRunWhoamiOAuth_FallsBackToFlatListWhenContextStatusLookupFails exercises
// the OAuth text-output path end to end: when the context-status endpoint
// itself fails, whoami should print the plain, ungrouped account list rather
// than dumping every sandbox under "Other sandboxes" (which would otherwise
// look like we checked and found no parent, rather than couldn't check at all).
func TestRunWhoamiOAuth_FallsBackToFlatListWhenContextStatusLookupFails(t *testing.T) {
	config.KeyRing = keyring.NewMemoryStore(nil)
	t.Cleanup(func() { config.KeyRing = nil })

	accessSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, "/stripecli/oauth2/token/accounts", r.URL.Path)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{ //nolint:errcheck
			"accounts": []config.AuthorizedAccount{
				{ID: "acct_live1", Name: "Acme Inc", Modes: []string{"live"}},
				{ID: "acct_sandbox1", Name: "Acme Dev", Modes: []string{"test"}},
			},
		})
	}))
	defer accessSrv.Close()

	apiSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/v1/stripecli/context_statuses":
			w.WriteHeader(http.StatusInternalServerError)
		default:
			w.WriteHeader(http.StatusOK)
		}
	}))
	defer apiSrv.Close()

	wc := newWhoamiCmd()
	wc.profile = &config.Profile{ProfileName: "default"}
	wc.accessBaseURL = accessSrv.URL
	wc.apiBaseURL = apiSrv.URL
	require.NoError(t, wc.profile.SaveActiveContext("acct_live1", true))

	wc.cmd.SetContext(context.Background())
	wc.cmd.SetOut(new(bytes.Buffer))

	// PrintAuthorizedContextsList/PrintGroupedAuthorizedAccounts write
	// directly to os.Stdout rather than cmd's writer, so that's what must be
	// captured here.
	origStdout := os.Stdout
	r, pipeW, err := os.Pipe()
	require.NoError(t, err)
	os.Stdout = pipeW

	runErr := wc.runWhoamiOAuth(wc.cmd, "oak_test_123")

	pipeW.Close()
	os.Stdout = origStdout
	var stdoutBuf bytes.Buffer
	stdoutBuf.ReadFrom(r) //nolint:errcheck
	require.NoError(t, runErr)

	stdout := stdoutBuf.String()
	assert.Contains(t, stdout, "Authorized (2 accounts):")
	assert.NotContains(t, stdout, "Other sandboxes", "a failed lookup shouldn't be reported as 'checked, no parent found'")
	assert.NotContains(t, stdout, "└─", "a failed lookup means we have no grouping info, so the list should be flat")
}
