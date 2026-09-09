package cmd

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
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
