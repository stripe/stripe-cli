package cmd

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"strconv"
	"testing"
	"time"

	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/stripe/stripe-cli/pkg/config"
	"github.com/stripe/stripe-cli/pkg/keyring"
	"github.com/stripe/stripe-cli/pkg/requests"
	"github.com/stripe/stripe-cli/pkg/stripe"
)

func runWhoami(t *testing.T, wc *whoamiCmd) (string, error) {
	t.Helper()
	if wc.apiBaseURL == stripe.DefaultAPIBaseURL {
		server := httptest.NewServer(http.NotFoundHandler())
		t.Cleanup(server.Close)
		wc.apiBaseURL = server.URL
	}
	buf := new(bytes.Buffer)
	wc.cmd.SetOut(buf)
	err := wc.runWhoamiCmd(wc.cmd, []string{})
	return buf.String(), err
}

func TestWhoamiNotAuthenticated(t *testing.T) {
	viper.Reset()
	config.KeyRing = keyring.NewMemoryStore(nil)

	wc := newWhoamiCmd()
	wc.cmd.SetContext(context.Background())
	wc.profile = &config.Profile{
		ProfileName: "default",
		DeviceName:  "test-device",
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("unauthenticated whoami must not call the API")
		http.NotFound(w, r)
	}))
	defer server.Close()
	wc.apiBaseURL = server.URL

	out, err := runWhoami(t, wc)
	assert.ErrorIs(t, err, errNotAuthenticated)
	assert.Regexp(t, `Profile:\s+default`, out)
	assert.Regexp(t, `Authenticated:\s+false`, out)
	assert.NotContains(t, out, "Network Business Profile")
}

func TestWhoamiNotAuthenticatedJSON(t *testing.T) {
	viper.Reset()
	config.KeyRing = keyring.NewMemoryStore(nil)

	wc := newWhoamiCmd()
	wc.cmd.SetContext(context.Background())
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
	wc.cmd.SetContext(context.Background())
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
	wc.cmd.SetContext(context.Background())
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
	wc.cmd.SetContext(context.Background())
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
	wc.cmd.SetContext(context.Background())
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
	wc.cmd.SetContext(context.Background())
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
	wc.cmd.SetContext(context.Background())
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
		case "/v2/network/business_profiles/me":
			w.WriteHeader(http.StatusNotFound)
		case "/v1/stripecli/context_statuses":
			w.WriteHeader(http.StatusInternalServerError)
		default:
			w.WriteHeader(http.StatusOK)
		}
	}))
	defer apiSrv.Close()

	wc := newWhoamiCmd()
	wc.cmd.SetContext(context.Background())
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

func TestWhoamiNetworkBusinessProfile(t *testing.T) {
	for _, key := range []string{"sk_test_1234567890abcdef", "sk_live_1234567890abcdef"} {
		for _, format := range []string{"", "json"} {
			for _, status := range []int{http.StatusOK, http.StatusNotFound, http.StatusInternalServerError} {
				t.Run(fmt.Sprintf("%s/format=%s/status=%d", key, format, status), func(t *testing.T) {
					config.KeyRing = keyring.NewMemoryStore(nil)
					t.Cleanup(func() { config.KeyRing = nil })
					calls := 0
					server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
						calls++
						assert.Equal(t, "/v2/network/business_profiles/me", r.URL.Path)
						assert.Equal(t, "Bearer "+key, r.Header.Get("Authorization"))
						assert.Equal(t, requests.StripePreviewVersionHeaderValue, r.Header.Get("Stripe-Version"))
						w.WriteHeader(status)
						w.Write([]byte(`{"id":"profile_test_123","username":"furever"}`)) //nolint:errcheck
					}))
					defer server.Close()

					wc := newWhoamiCmd()
					wc.cmd.SetContext(context.Background())
					wc.profile = &config.Profile{ProfileName: "default", APIKey: key}
					wc.apiBaseURL = server.URL
					wc.format = format
					out, err := runWhoami(t, wc)
					assert.Equal(t, 1, calls)
					assertWhoamiBusinessProfile(t, out, err, format, status)
				})
			}
		}
	}
}

func TestWhoamiOAuthNetworkBusinessProfile(t *testing.T) {
	for _, livemode := range []bool{false, true} {
		for _, format := range []string{"", "json"} {
			for _, status := range []int{http.StatusOK, http.StatusNotFound, http.StatusInternalServerError} {
				t.Run(fmt.Sprintf("live=%t/format=%s/status=%d", livemode, format, status), func(t *testing.T) {
					config.KeyRing = keyring.NewMemoryStore(nil)
					t.Cleanup(func() { config.KeyRing = nil })
					calls := 0
					server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
						switch r.URL.Path {
						case "/stripecli/oauth2/token/accounts":
							w.Write([]byte(`{"accounts":[]}`)) //nolint:errcheck
						case "/v1/stripecli/user_info":
							w.Write([]byte(`{"email":"user@example.com"}`)) //nolint:errcheck
						case "/v2/network/business_profiles/me":
							calls++
							assert.Equal(t, "Bearer oak_test_123", r.Header.Get("Authorization"))
							assert.Equal(t, "acct_123", r.Header.Get("Stripe-Context"))
							assert.Equal(t, strconv.FormatBool(livemode), r.Header.Get("Stripe-Livemode"))
							w.WriteHeader(status)
							w.Write([]byte(`{"id":"profile_test_123","username":"furever"}`)) //nolint:errcheck
						default:
							http.NotFound(w, r)
						}
					}))
					defer server.Close()

					wc := newWhoamiCmd()
					wc.cmd.SetContext(context.Background())
					wc.profile = &config.Profile{ProfileName: "default"}
					require.NoError(t, wc.profile.SaveActiveContext("acct_123", livemode))
					wc.apiBaseURL = server.URL
					wc.accessBaseURL = server.URL
					wc.format = format
					buf := new(bytes.Buffer)
					wc.cmd.SetOut(buf)
					err := wc.runWhoamiOAuth(wc.cmd, "oak_test_123")
					assert.Equal(t, 1, calls)
					assertWhoamiBusinessProfile(t, buf.String(), err, format, status)
				})
			}
		}
	}
}

func assertWhoamiBusinessProfile(t *testing.T, out string, err error, format string, status int) {
	t.Helper()
	if status == http.StatusInternalServerError {
		require.ErrorContains(t, err, "failed to fetch network business profile")
		return
	}
	require.NoError(t, err)
	switch {
	case format == "json":
		var result map[string]json.RawMessage
		require.NoError(t, json.Unmarshal([]byte(out), &result))
		if status == http.StatusNotFound {
			assert.NotContains(t, result, "network_business_profile")
		} else {
			assert.JSONEq(t, `{"id":"profile_test_123","username":"furever"}`, string(result["network_business_profile"]))
		}
	case status == http.StatusNotFound:
		assert.NotContains(t, out, "Network Business Profile")
	default:
		assert.Regexp(t, `(?m)^Network Business Profile\s+@furever \(profile_test_123\)$`, out)
	}
}

func TestWhoamiOAuthNetworkBusinessProfileNoActiveContext(t *testing.T) {
	config.KeyRing = keyring.NewMemoryStore(nil)
	t.Cleanup(func() { config.KeyRing = nil })
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/stripecli/oauth2/token/accounts" {
			t.Errorf("unexpected request without an active context: %s", r.URL.Path)
			http.NotFound(w, r)
			return
		}
		w.Write([]byte(`{"accounts":[]}`)) //nolint:errcheck
	}))
	defer server.Close()

	wc := newWhoamiCmd()
	wc.cmd.SetContext(context.Background())
	wc.profile = &config.Profile{ProfileName: "default"}
	wc.accessBaseURL = server.URL
	wc.apiBaseURL = server.URL
	wc.format = "json"
	buf := new(bytes.Buffer)
	wc.cmd.SetOut(buf)
	require.NoError(t, wc.runWhoamiOAuth(wc.cmd, "oak_test_123"))
	assert.NotContains(t, buf.String(), "network_business_profile")
}
