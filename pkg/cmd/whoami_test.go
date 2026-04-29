package cmd

import (
	"bytes"
	"encoding/json"
	"testing"

	"github.com/99designs/keyring"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/stripe/stripe-cli/pkg/config"
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
	config.KeyRing = keyring.NewArrayKeyring([]keyring.Item{})

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
	config.KeyRing = keyring.NewArrayKeyring([]keyring.Item{})

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

func TestWhoamiNotAuthenticatedTOON(t *testing.T) {
	config.KeyRing = keyring.NewArrayKeyring([]keyring.Item{})

	wc := newWhoamiCmd()
	wc.profile = &config.Profile{
		ProfileName: "default",
		DeviceName:  "test-device",
	}
	wc.format = "toon"

	out, err := runWhoami(t, wc)
	assert.ErrorIs(t, err, errNotAuthenticated)
	assert.Contains(t, out, "authenticated: false")
	assert.Contains(t, out, "profile_name: default")
	assert.Contains(t, out, "test_mode_key:")
	assert.Contains(t, out, "live_mode_key:")
}

func TestWhoamiWithTestKey(t *testing.T) {
	config.KeyRing = keyring.NewArrayKeyring([]keyring.Item{})

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
	assert.False(t, result.LiveModeKey.Available)
	assert.Equal(t, "acct_123", result.AccountID)
}

func TestWhoamiWithLiveModeAPIKey(t *testing.T) {
	config.KeyRing = keyring.NewArrayKeyring([]keyring.Item{})

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
}

func TestWhoamiWithEnvVarKey(t *testing.T) {
	config.KeyRing = keyring.NewArrayKeyring([]keyring.Item{})
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
	assert.False(t, result.LiveModeKey.Available)
}

func TestAPIKeyIsLivemode(t *testing.T) {
	assert.False(t, apiKeyIsLivemode("sk_test_abc123"))
	assert.True(t, apiKeyIsLivemode("sk_live_abc123"))
	assert.False(t, apiKeyIsLivemode("rk_test_abc123"))
	assert.True(t, apiKeyIsLivemode("rk_live_abc123"))
}

func TestKeyAvailabilityText(t *testing.T) {
	expiresAt := "2026-05-01"

	assert.Equal(t, "not available", keyAvailabilityText(whoamiKeyInfo{Available: false}))
	assert.Equal(t, "available", keyAvailabilityText(whoamiKeyInfo{Available: true}))
	assert.Equal(t, "available (expires 2026-05-01)", keyAvailabilityText(whoamiKeyInfo{Available: true, ExpiresAt: &expiresAt}))
}
