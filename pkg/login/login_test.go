package login

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/stripe/stripe-cli/pkg/config"
)

// rewriteHostTransport redirects every request to target, regardless of the URL it was built
// with. This lets tests exercise code paths gated by ValidateAccessBaseURL (which only accepts
// the real access.stripe.com/qa-access.stripe.com origins) against a local httptest server,
// without weakening that validation.
type rewriteHostTransport struct {
	target *url.URL
	base   http.RoundTripper
}

func (t rewriteHostTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	req = req.Clone(req.Context())
	req.URL.Scheme = t.target.Scheme
	req.URL.Host = t.target.Host
	return t.base.RoundTrip(req)
}

// stubAccessSrv points accessSrvHTTPClient at ts for the duration of the test.
func stubAccessSrv(t *testing.T, ts *httptest.Server) {
	t.Helper()
	target, err := url.Parse(ts.URL)
	require.NoError(t, err)

	orig := accessSrvHTTPClient
	accessSrvHTTPClient = &http.Client{
		CheckRedirect: orig.CheckRedirect,
		Transport:     rewriteHostTransport{target: target, base: http.DefaultTransport},
	}
	t.Cleanup(func() { accessSrvHTTPClient = orig })
}

func TestInitiateOrResumeOAuthDeviceLogin_MintsWhenNoPending(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())

	var mintCount atomic.Int32
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mintCount.Add(1)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(DeviceAuthResponse{ //nolint:errcheck
			DeviceCode:      "device-code",
			UserCode:        "ABCD-EFGH",
			VerificationURI: "https://access.stripe.com/verify",
			ExpiresIn:       300,
			Interval:        5,
		})
	}))
	defer ts.Close()

	session, err := initiateOrResumeOAuthDeviceLogin(context.Background(), ts.URL)
	require.NoError(t, err)
	assert.Equal(t, "https://access.stripe.com/verify", session.BrowserURL)
	assert.Equal(t, "ABCD-EFGH", session.VerificationCode)
	assert.Equal(t, int32(1), mintCount.Load())
}

func TestInitiateOrResumeOAuthDeviceLogin_ResumesStillValidPending(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())

	var mintCount atomic.Int32
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mintCount.Add(1)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(DeviceAuthResponse{ //nolint:errcheck
			DeviceCode:      "device-code",
			UserCode:        "ABCD-EFGH",
			VerificationURI: "https://access.stripe.com/verify",
			ExpiresIn:       300,
			Interval:        5,
		})
	}))
	defer ts.Close()

	first, err := initiateOrResumeOAuthDeviceLogin(context.Background(), ts.URL)
	require.NoError(t, err)

	second, err := initiateOrResumeOAuthDeviceLogin(context.Background(), ts.URL)
	require.NoError(t, err)

	assert.Equal(t, int32(1), mintCount.Load(), "a second call within the expiry window must not mint a new device code")
	assert.Equal(t, first.BrowserURL, second.BrowserURL)
	assert.Equal(t, first.VerificationCode, second.VerificationCode)
}

func TestInitiateOrResumeOAuthDeviceLogin_MintsFreshWhenPendingExpired(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())

	var mintCount atomic.Int32
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mintCount.Add(1)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(DeviceAuthResponse{ //nolint:errcheck
			DeviceCode:      "new-device-code",
			UserCode:        "NEW1-CODE",
			VerificationURI: "https://access.stripe.com/verify",
			ExpiresIn:       300,
			Interval:        5,
		})
	}))
	defer ts.Close()

	require.NoError(t, savePendingDeviceAuth(&oauthContinuation{
		DeviceCode:      "stale-device-code",
		AccessBaseURL:   ts.URL,
		VerificationURI: "https://access.stripe.com/stale",
		UserCode:        "STALE-CODE",
		ExpiresIn:       1,
		IssuedAt:        time.Now().Add(-11 * time.Minute), // past the 10-minute deadline floor
	}))

	session, err := initiateOrResumeOAuthDeviceLogin(context.Background(), ts.URL)
	require.NoError(t, err)
	assert.Equal(t, int32(1), mintCount.Load())
	assert.Equal(t, "NEW1-CODE", session.VerificationCode)
}

func TestInitiateOrResumeOAuthDeviceLogin_MintsFreshForDifferentAccessBaseURL(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())

	var mintCount atomic.Int32
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mintCount.Add(1)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(DeviceAuthResponse{ //nolint:errcheck
			DeviceCode:      "new-device-code",
			UserCode:        "NEW1-CODE",
			VerificationURI: "https://access.stripe.com/verify",
			ExpiresIn:       300,
			Interval:        5,
		})
	}))
	defer ts.Close()

	require.NoError(t, savePendingDeviceAuth(&oauthContinuation{
		DeviceCode:      "other-env-device-code",
		AccessBaseURL:   "https://a-different-env.example.com",
		VerificationURI: "https://access.stripe.com/other",
		UserCode:        "OTHER-CODE",
		ExpiresIn:       300,
		IssuedAt:        time.Now(),
	}))

	session, err := initiateOrResumeOAuthDeviceLogin(context.Background(), ts.URL)
	require.NoError(t, err)
	assert.Equal(t, int32(1), mintCount.Load())
	assert.Equal(t, "NEW1-CODE", session.VerificationCode)
}

func TestMintOAuthDeviceLogin_AlwaysMintsFreshEvenWithValidPending(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())

	var mintCount atomic.Int32
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mintCount.Add(1)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(DeviceAuthResponse{ //nolint:errcheck
			DeviceCode:      "new-device-code",
			UserCode:        "NEW1-CODE",
			VerificationURI: "https://access.stripe.com/verify",
			ExpiresIn:       300,
			Interval:        5,
		})
	}))
	defer ts.Close()

	require.NoError(t, savePendingDeviceAuth(&oauthContinuation{
		DeviceCode:      "still-valid-device-code",
		AccessBaseURL:   ts.URL,
		VerificationURI: "https://access.stripe.com/still-valid",
		UserCode:        "STILL-VALID",
		ExpiresIn:       300,
		IssuedAt:        time.Now(),
	}))

	// `stripe login --non-interactive` (initiateOAuthDeviceLogin) calls mintOAuthDeviceLogin
	// directly, not initiateOrResumeOAuthDeviceLogin, so it must mint a new device code even
	// though the pending one saved above is still well within its expiry window.
	session, err := mintOAuthDeviceLogin(context.Background(), ts.URL)
	require.NoError(t, err)
	assert.Equal(t, int32(1), mintCount.Load())
	assert.Equal(t, "NEW1-CODE", session.VerificationCode)

	cont, err := loadPendingDeviceAuth()
	require.NoError(t, err)
	assert.Equal(t, "new-device-code", cont.DeviceCode, "the fresh device code must overwrite the still-valid pending one")
}

func TestFindPendingOAuthLogin_NoneExists(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())

	session, err := FindPendingOAuthLogin("https://example.com")
	require.NoError(t, err)
	assert.Nil(t, session)
}

func TestFindPendingOAuthLogin_FindsStillValidPending(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())

	require.NoError(t, savePendingDeviceAuth(&oauthContinuation{
		DeviceCode:      "device-code",
		AccessBaseURL:   "https://example.com",
		VerificationURI: "https://access.stripe.com/verify",
		UserCode:        "ABCD-EFGH",
		ExpiresIn:       300,
		IssuedAt:        time.Now(),
	}))

	session, err := FindPendingOAuthLogin("https://example.com")
	require.NoError(t, err)
	require.NotNil(t, session)
	assert.Equal(t, "https://access.stripe.com/verify", session.BrowserURL)
	assert.Equal(t, "ABCD-EFGH", session.VerificationCode)
}

func TestFindPendingOAuthLogin_IgnoresExpiredPending(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())

	require.NoError(t, savePendingDeviceAuth(&oauthContinuation{
		DeviceCode:    "device-code",
		AccessBaseURL: "https://example.com",
		ExpiresIn:     1,
		IssuedAt:      time.Now().Add(-11 * time.Minute), // past the 10-minute deadline floor
	}))

	session, err := FindPendingOAuthLogin("https://example.com")
	require.NoError(t, err)
	assert.Nil(t, session)
}

func TestFindPendingOAuthLogin_IgnoresDifferentAccessBaseURL(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())

	require.NoError(t, savePendingDeviceAuth(&oauthContinuation{
		DeviceCode:    "device-code",
		AccessBaseURL: "https://a-different-env.example.com",
		ExpiresIn:     300,
		IssuedAt:      time.Now(),
	}))

	session, err := FindPendingOAuthLogin("https://example.com")
	require.NoError(t, err)
	assert.Nil(t, session)
}

func TestPollPendingOAuthLogin_CallerTimeoutLeavesPendingState(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	cfg, cleanup := setupOAuthTestConfig(t)
	defer cleanup()

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(tokenErrorResponse{Error: "authorization_pending"}) //nolint:errcheck
	}))
	defer ts.Close()
	stubAccessSrv(t, ts)

	require.NoError(t, savePendingDeviceAuth(&oauthContinuation{
		DeviceCode:    "device-code",
		Interval:      5,
		ExpiresIn:     300,
		AccessBaseURL: QAAccessBaseURL,
		IssuedAt:      time.Now(),
	}))

	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	result, err := PollPendingOAuthLogin(ctx, cfg)
	require.NoError(t, err)
	assert.Nil(t, result)

	_, loadErr := loadPendingDeviceAuth()
	assert.NoError(t, loadErr, "pending state must survive a caller-side timeout so a later poll can resume")
}

func TestPollPendingOAuthLogin_DeviceCodeExpiryClearsPendingState(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	cfg, cleanup := setupOAuthTestConfig(t)
	defer cleanup()

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(tokenErrorResponse{Error: "authorization_pending"}) //nolint:errcheck
	}))
	defer ts.Close()
	stubAccessSrv(t, ts)

	require.NoError(t, savePendingDeviceAuth(&oauthContinuation{
		DeviceCode:    "device-code",
		Interval:      5,
		ExpiresIn:     1,
		AccessBaseURL: QAAccessBaseURL,
		IssuedAt:      time.Now().Add(-11 * time.Minute), // past the 10-minute deadline floor
	}))

	_, err := PollPendingOAuthLogin(context.Background(), cfg)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "expired")

	_, loadErr := loadPendingDeviceAuth()
	assert.Error(t, loadErr, "pending state must be cleared once the device code has actually expired")
}

func TestPollPendingOAuthLogin_TerminalOAuthErrorClearsPendingState(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	cfg, cleanup := setupOAuthTestConfig(t)
	defer cleanup()

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(tokenErrorResponse{Error: "access_denied"}) //nolint:errcheck
	}))
	defer ts.Close()
	stubAccessSrv(t, ts)

	require.NoError(t, savePendingDeviceAuth(&oauthContinuation{
		DeviceCode:    "device-code",
		Interval:      5,
		ExpiresIn:     300,
		AccessBaseURL: QAAccessBaseURL,
		IssuedAt:      time.Now(),
	}))

	_, err := PollPendingOAuthLogin(context.Background(), cfg)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "access_denied")

	_, loadErr := loadPendingDeviceAuth()
	assert.Error(t, loadErr, "pending state must be cleared once the server returns a terminal error")
}

func TestPollPendingOAuthLogin_SuccessClearsPendingStateAndReturnsResult(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	cfg, cleanup := setupOAuthTestConfig(t)
	defer cleanup()

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/stripecli/oauth2/token":
			json.NewEncoder(w).Encode(OAuthTokenResponse{ //nolint:errcheck
				AccessToken:  "oaac_test_access",
				RefreshToken: "oart_test_refresh",
				TokenType:    "Bearer",
				ExpiresIn:    3600,
			})
		case "/stripecli/oauth2/token/accounts":
			json.NewEncoder(w).Encode(listAccountsResponse{Accounts: []config.AuthorizedAccount{ //nolint:errcheck
				{ID: "acct_123", Name: "Test Account", Modes: []string{"test", "live"}},
			}})
		default:
			http.NotFound(w, r)
		}
	}))
	defer ts.Close()
	stubAccessSrv(t, ts)

	require.NoError(t, savePendingDeviceAuth(&oauthContinuation{
		DeviceCode:    "device-code",
		Interval:      1,
		ExpiresIn:     300,
		AccessBaseURL: QAAccessBaseURL,
		IssuedAt:      time.Now(),
	}))

	result, err := PollPendingOAuthLogin(context.Background(), cfg)
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Equal(t, "acct_123", result.ActiveAccountID)

	_, loadErr := loadPendingDeviceAuth()
	assert.Error(t, loadErr, "pending state must be cleared once login succeeds")
}

func TestCheckPendingOAuthLogin_NotYetLoggedInLeavesPendingState(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	cfg, cleanup := setupOAuthTestConfig(t)
	defer cleanup()

	var callCount atomic.Int32
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount.Add(1)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(tokenErrorResponse{Error: "authorization_pending"}) //nolint:errcheck
	}))
	defer ts.Close()
	stubAccessSrv(t, ts)

	require.NoError(t, savePendingDeviceAuth(&oauthContinuation{
		DeviceCode:    "device-code",
		Interval:      5,
		ExpiresIn:     300,
		AccessBaseURL: QAAccessBaseURL,
		IssuedAt:      time.Now(),
	}))

	result, err := CheckPendingOAuthLogin(context.Background(), cfg)
	require.NoError(t, err)
	assert.Nil(t, result)
	assert.Equal(t, int32(1), callCount.Load(), "a single check must make exactly one request, never loop or sleep")

	_, loadErr := loadPendingDeviceAuth()
	assert.NoError(t, loadErr, "pending state must survive a not-yet-completed check")
}

func TestCheckPendingOAuthLogin_DeviceCodeExpiryClearsPendingState(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	cfg, cleanup := setupOAuthTestConfig(t)
	defer cleanup()

	require.NoError(t, savePendingDeviceAuth(&oauthContinuation{
		DeviceCode:    "device-code",
		Interval:      5,
		ExpiresIn:     1,
		AccessBaseURL: QAAccessBaseURL,
		IssuedAt:      time.Now().Add(-11 * time.Minute), // past the 10-minute deadline floor
	}))

	_, err := CheckPendingOAuthLogin(context.Background(), cfg)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "expired")

	_, loadErr := loadPendingDeviceAuth()
	assert.Error(t, loadErr, "pending state must be cleared once the device code has actually expired")
}

func TestCheckPendingOAuthLogin_TerminalOAuthErrorClearsPendingState(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	cfg, cleanup := setupOAuthTestConfig(t)
	defer cleanup()

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(tokenErrorResponse{Error: "access_denied"}) //nolint:errcheck
	}))
	defer ts.Close()
	stubAccessSrv(t, ts)

	require.NoError(t, savePendingDeviceAuth(&oauthContinuation{
		DeviceCode:    "device-code",
		Interval:      5,
		ExpiresIn:     300,
		AccessBaseURL: QAAccessBaseURL,
		IssuedAt:      time.Now(),
	}))

	_, err := CheckPendingOAuthLogin(context.Background(), cfg)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "access_denied")

	_, loadErr := loadPendingDeviceAuth()
	assert.Error(t, loadErr, "pending state must be cleared once the server returns a terminal error")
}

func TestCheckPendingOAuthLogin_SuccessClearsPendingStateAndReturnsResult(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	cfg, cleanup := setupOAuthTestConfig(t)
	defer cleanup()

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/stripecli/oauth2/token":
			json.NewEncoder(w).Encode(OAuthTokenResponse{ //nolint:errcheck
				AccessToken:  "oaac_test_access",
				RefreshToken: "oart_test_refresh",
				TokenType:    "Bearer",
				ExpiresIn:    3600,
			})
		case "/stripecli/oauth2/token/accounts":
			json.NewEncoder(w).Encode(listAccountsResponse{Accounts: []config.AuthorizedAccount{ //nolint:errcheck
				{ID: "acct_123", Name: "Test Account", Modes: []string{"test", "live"}},
			}})
		default:
			http.NotFound(w, r)
		}
	}))
	defer ts.Close()
	stubAccessSrv(t, ts)

	require.NoError(t, savePendingDeviceAuth(&oauthContinuation{
		DeviceCode:    "device-code",
		Interval:      1,
		ExpiresIn:     300,
		AccessBaseURL: QAAccessBaseURL,
		IssuedAt:      time.Now(),
	}))

	result, err := CheckPendingOAuthLogin(context.Background(), cfg)
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Equal(t, "acct_123", result.ActiveAccountID)

	_, loadErr := loadPendingDeviceAuth()
	assert.Error(t, loadErr, "pending state must be cleared once login succeeds")
}

func TestCheckPendingOAuthLogin_SaveFailureAfterTokenExchangeClearsPendingState(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	cfg, cleanup := setupOAuthTestConfig(t)
	defer cleanup()

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/stripecli/oauth2/token":
			json.NewEncoder(w).Encode(OAuthTokenResponse{ //nolint:errcheck
				AccessToken:  "oaac_test_access",
				RefreshToken: "oart_test_refresh",
				TokenType:    "Bearer",
				ExpiresIn:    3600,
			})
		case "/stripecli/oauth2/token/accounts":
			// The token exchange succeeded (the device code is now consumed), but fetching
			// account info fails.
			w.WriteHeader(http.StatusInternalServerError)
		default:
			http.NotFound(w, r)
		}
	}))
	defer ts.Close()
	stubAccessSrv(t, ts)

	require.NoError(t, savePendingDeviceAuth(&oauthContinuation{
		DeviceCode:    "device-code",
		Interval:      1,
		ExpiresIn:     300,
		AccessBaseURL: QAAccessBaseURL,
		IssuedAt:      time.Now(),
	}))

	result, err := CheckPendingOAuthLogin(context.Background(), cfg)
	require.Error(t, err)
	assert.Nil(t, result)

	_, loadErr := loadPendingDeviceAuth()
	assert.Error(t, loadErr, "pending state must be cleared once the (single-use) device code has been consumed, even if saving the resulting credentials failed")
}
