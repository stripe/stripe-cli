package login

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"runtime"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/kballard/go-shellquote"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/stripe/stripe-cli/pkg/config"
	"github.com/stripe/stripe-cli/pkg/keyring"
)

// Rewrites trusted production origins only inside tests. Production origin
// validation remains enabled; no localhost/endpoint override is added to auth.
type handoffTestTransport struct {
	target    *url.URL
	transport http.RoundTripper
}

func (t handoffTestTransport) RoundTrip(r *http.Request) (*http.Response, error) {
	if r.URL.Host != "access.stripe.com" && r.URL.Host != "qa-access.stripe.com" {
		return nil, errors.New("unexpected test origin")
	}
	c := r.Clone(r.Context())
	c.URL.Scheme, c.URL.Host = t.target.Scheme, t.target.Host
	return t.transport.RoundTrip(c)
}

type handoffFixture struct {
	blockRelease chan struct{}
	releaseOnce  sync.Once
	cfg          *config.Config
	now          time.Time
	issued       atomic.Int32
	polled       atomic.Int32
	approved     atomic.Bool
	accountsFail atomic.Bool
	blocking     atomic.Bool
	refreshError string
	refreshed    atomic.Int32
	tokenError   string
}

func newHandoffFixture(t *testing.T) *handoffFixture {
	t.Helper()
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	cfg, cleanup := setupOAuthTestConfig(t)
	t.Cleanup(cleanup)
	f := &handoffFixture{cfg: cfg, now: time.Now().UTC(), blockRelease: make(chan struct{})}
	s := httptest.NewServer(http.HandlerFunc(f.serve))
	t.Cleanup(s.Close)
	t.Cleanup(f.releaseBlocked)
	target, err := url.Parse(s.URL)
	require.NoError(t, err)
	originalClient, originalClock := accessSrvHTTPClient, handoffNow
	accessSrvHTTPClient = &http.Client{Transport: handoffTestTransport{target: target, transport: s.Client().Transport}}
	handoffNow = func() time.Time { return f.now }
	t.Cleanup(func() { accessSrvHTTPClient, handoffNow = originalClient, originalClock })
	return f
}

func (f *handoffFixture) serve(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	switch r.URL.Path {
	case accessAPNPath + "/device/authorization":
		n := f.issued.Add(1)
		_ = json.NewEncoder(w).Encode(DeviceAuthResponse{DeviceCode: fmt.Sprintf("secret-device-%d", n), UserCode: fmt.Sprintf("CODE-%d", n), VerificationURI: "https://access.stripe.com/verify", ExpiresIn: 600, Interval: 5})
	case accessAPNPath + "/token":
		if r.FormValue("grant_type") == "refresh_token" {
			f.refreshed.Add(1)
			if f.refreshError != "" {
				w.WriteHeader(400)
				_ = json.NewEncoder(w).Encode(tokenErrorResponse{Error: f.refreshError})
				return
			}
			_ = json.NewEncoder(w).Encode(OAuthTokenResponse{AccessToken: "oak_fixture_renewed", RefreshToken: "fixture_renewed_refresh", TokenType: "Bearer", ExpiresIn: 3600})
			return
		}
		f.polled.Add(1)
		if f.blocking.Load() {
			select {
			case <-r.Context().Done():
			case <-f.blockRelease:
			}
			return
		}
		if f.tokenError != "" {
			w.WriteHeader(400)
			_ = json.NewEncoder(w).Encode(tokenErrorResponse{Error: f.tokenError})
			return
		}
		if !f.approved.Load() {
			w.WriteHeader(400)
			_ = json.NewEncoder(w).Encode(tokenErrorResponse{Error: "authorization_pending"})
			return
		}
		_ = json.NewEncoder(w).Encode(OAuthTokenResponse{AccessToken: "oak_fixture_access", RefreshToken: "refresh_fixture", TokenType: "Bearer", ExpiresIn: 3600})
	case accessAPNPath + "/token/accounts":
		if f.accountsFail.Load() {
			w.WriteHeader(503)
			return
		}
		_ = json.NewEncoder(w).Encode(listAccountsResponse{Accounts: []config.AuthorizedAccount{{ID: "acct_fixture", Name: "Fixture", Modes: []string{"live"}}}})
	default:
		http.NotFound(w, r)
	}
}

func (f *handoffFixture) begin(t *testing.T) *LoginHandoff {
	t.Helper()
	r, err := BeginOrResumeLogin(context.Background(), DefaultAccessBaseURL, f.cfg)
	require.NoError(t, err)
	return r
}

func TestHandoffBeginReusesDurableState(t *testing.T) {
	f := newHandoffFixture(t)
	first := f.begin(t)
	f.now = f.now.Add(time.Minute)
	second := f.begin(t)
	assert.Equal(t, first.ID, second.ID)
	assert.Equal(t, first.BrowserURL, second.BrowserURL)
	assert.Equal(t, first.VerificationCode, second.VerificationCode)
	assert.Equal(t, first.ExpiresAt, second.ExpiresAt)
	assert.True(t, second.Reused)
	assert.EqualValues(t, 1, f.issued.Load())
	data, err := json.Marshal(first)
	require.NoError(t, err)
	assert.NotContains(t, string(data), "secret-device")
	assert.NotContains(t, string(data), "refresh_fixture")
	info, err := os.Stat(pendingDeviceAuthPath())
	require.NoError(t, err)
	if runtime.GOOS != "windows" {
		assert.Zero(t, info.Mode().Perm()&0077)
	}
}

func TestHandoffPendingAndSlowDownPersistRateLimit(t *testing.T) {
	f := newHandoffFixture(t)
	h := f.begin(t)
	r, err := CheckLogin(context.Background(), DefaultAccessBaseURL, f.cfg, h.ID)
	require.NoError(t, err)
	assert.Equal(t, LoginHandoffPending, r.State)
	_, err = CheckLogin(context.Background(), DefaultAccessBaseURL, f.cfg, h.ID)
	require.NoError(t, err)
	assert.EqualValues(t, 1, f.polled.Load())
	f.now = f.now.Add(6 * time.Second)
	f.tokenError = "slow_down"
	r, err = CheckLogin(context.Background(), DefaultAccessBaseURL, f.cfg, h.ID)
	require.NoError(t, err)
	assert.GreaterOrEqual(t, r.CheckAfterSeconds, 10)
	f.now = f.now.Add(6 * time.Second)
	_, err = CheckLogin(context.Background(), DefaultAccessBaseURL, f.cfg, h.ID)
	require.NoError(t, err)
	assert.EqualValues(t, 2, f.polled.Load())
}

func TestHandoffCompletesOnceAndClearsSecrets(t *testing.T) {
	f := newHandoffFixture(t)
	h := f.begin(t)
	f.approved.Store(true)
	r, err := CheckLogin(context.Background(), DefaultAccessBaseURL, f.cfg, h.ID)
	require.NoError(t, err)
	assert.Equal(t, LoginHandoffAuthenticated, r.State)
	assert.Equal(t, "acct_fixture", r.AccountID)
	assert.True(t, r.Livemode)
	assert.Empty(t, r.BrowserURL)
	_, err = CheckLogin(context.Background(), DefaultAccessBaseURL, f.cfg, h.ID)
	require.NoError(t, err)
	assert.EqualValues(t, 1, f.polled.Load())
	assert.Equal(t, LoginHandoffAuthenticated, f.begin(t).State)
	_, err = config.KeyRing.Get(handoffCompletionKey(h.ID))
	assert.ErrorIs(t, err, keyring.ErrKeyNotFound)
	data, err := os.ReadFile(pendingDeviceAuthPath())
	require.NoError(t, err)
	assert.NotContains(t, string(data), "secret-device")
	assert.NotContains(t, string(data), "oak_fixture_access")
	assert.NotContains(t, string(data), "refresh_fixture")
}

func TestHandoffInterruptedPollResumesOriginal(t *testing.T) {
	f := newHandoffFixture(t)
	h := f.begin(t)
	f.blocking.Store(true)
	ctx, cancel := context.WithTimeout(context.Background(), 40*time.Millisecond)
	defer cancel()
	_, err := CheckLogin(ctx, DefaultAccessBaseURL, f.cfg, h.ID)
	require.Error(t, err)
	assert.Equal(t, h.ID, f.begin(t).ID)
	f.releaseBlocked()
	f.blocking.Store(false)
	f.approved.Store(true)
	f.now = f.now.Add(6 * time.Second)
	r, err := CheckLogin(context.Background(), DefaultAccessBaseURL, f.cfg, h.ID)
	require.NoError(t, err)
	assert.Equal(t, LoginHandoffAuthenticated, r.State)
	assert.EqualValues(t, 1, f.issued.Load())
}

func TestHandoffCheckpointSurvivesAccountLookupFailure(t *testing.T) {
	f := newHandoffFixture(t)
	h := f.begin(t)
	f.approved.Store(true)
	f.accountsFail.Store(true)
	_, err := CheckLogin(context.Background(), DefaultAccessBaseURL, f.cfg, h.ID)
	require.Error(t, err)
	assert.Equal(t, LoginHandoffCompleting, f.begin(t).State)
	token, err := f.cfg.Profile.GetUAT()
	require.NoError(t, err)
	assert.Empty(t, token)
	f.accountsFail.Store(false)
	r, err := CheckLogin(context.Background(), DefaultAccessBaseURL, f.cfg, h.ID)
	require.NoError(t, err)
	assert.Equal(t, LoginHandoffAuthenticated, r.State)
	assert.EqualValues(t, 1, f.polled.Load(), "retry must use the checkpoint, not redeem the one-use code again")
}

type failHandoffInstallStore struct {
	keyring.SecureStore
	fail bool
}

func (s *failHandoffInstallStore) Set(key string, data []byte, description string) error {
	if s.fail && key == config.UATKeychainItemKey {
		return errors.New("injected write failure")
	}
	return s.SecureStore.Set(key, data, description)
}

func TestHandoffCheckpointSurvivesCredentialWriteFailure(t *testing.T) {
	f := newHandoffFixture(t)
	h := f.begin(t)
	f.approved.Store(true)
	store := &failHandoffInstallStore{SecureStore: config.KeyRing, fail: true}
	config.KeyRing = store
	_, err := CheckLogin(context.Background(), DefaultAccessBaseURL, f.cfg, h.ID)
	require.Error(t, err)
	store.fail = false
	r, err := CheckLogin(context.Background(), DefaultAccessBaseURL, f.cfg, h.ID)
	require.NoError(t, err)
	assert.Equal(t, LoginHandoffAuthenticated, r.State)
	assert.EqualValues(t, 1, f.polled.Load())
}

func TestHandoffTerminalResultsNeverCreateReplacement(t *testing.T) {
	for _, reason := range []string{"access_denied", "expired_token", "ambiguous_exchange", "local_expiry"} {
		t.Run(reason, func(t *testing.T) {
			f := newHandoffFixture(t)
			h := f.begin(t)
			want := LoginHandoffExpired
			switch reason {
			case "local_expiry":
				f.now = f.now.Add(11 * time.Minute)
			case "ambiguous_exchange":
				cont, err := loadPendingDeviceAuth()
				require.NoError(t, err)
				cont.PollInFlight = true
				require.NoError(t, savePendingDeviceAuth(cont))
				f.tokenError = "expired_token"
				want = LoginHandoffRecoveryRequired
			case "access_denied":
				f.tokenError = reason
				want = LoginHandoffDenied
			default:
				f.tokenError = reason
			}
			r, err := CheckLogin(context.Background(), DefaultAccessBaseURL, f.cfg, h.ID)
			require.NoError(t, err)
			assert.Equal(t, want, r.State)
			assert.Equal(t, want, f.begin(t).State)
			assert.EqualValues(t, 1, f.issued.Load())
		})
	}
}

func TestHandoffContextChangeCannotOverwriteNewSession(t *testing.T) {
	f := newHandoffFixture(t)
	h := f.begin(t)
	f.approved.Store(true)
	require.NoError(t, config.KeyRing.Set(config.UATKeychainItemKey, []byte("oak_other"), ""))
	require.NoError(t, config.SaveActiveContext("acct_other", false))
	r, err := CheckLogin(context.Background(), DefaultAccessBaseURL, f.cfg, h.ID)
	require.NoError(t, err)
	assert.Equal(t, LoginHandoffSuperseded, r.State)
	assert.EqualValues(t, 0, f.polled.Load())
	assert.Equal(t, LoginHandoffSessionPresent, f.begin(t).State)
}

func TestHandoffExistingOAuthIsNotReplacedByAPIKeyOverrides(t *testing.T) {
	f := newHandoffFixture(t)
	t.Setenv("STRIPE_API_KEY", "sk_test_override")
	require.NoError(t, config.KeyRing.Set(config.UATKeychainItemKey, []byte("oak_existing"), ""))
	assert.Equal(t, LoginHandoffSessionPresent, f.begin(t).State)
	assert.EqualValues(t, 0, f.issued.Load())
}

func TestHandoffConcurrentBeginsAndChecks(t *testing.T) {
	f := newHandoffFixture(t)
	results := make(chan *LoginHandoff, 8)
	errs := make(chan error, 8)
	var wg sync.WaitGroup
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			r, err := BeginOrResumeLogin(context.Background(), DefaultAccessBaseURL, f.cfg)
			results <- r
			errs <- err
		}()
	}
	wg.Wait()
	close(results)
	close(errs)
	for err := range errs {
		require.NoError(t, err)
	}
	var id string
	for r := range results {
		if id == "" {
			id = r.ID
		}
		assert.Equal(t, id, r.ID)
	}
	assert.EqualValues(t, 1, f.issued.Load())
	f.approved.Store(true)
	errs = make(chan error, 8)
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			r, err := CheckLogin(context.Background(), DefaultAccessBaseURL, f.cfg, id)
			if err == nil && r.State != LoginHandoffAuthenticated {
				err = fmt.Errorf("unexpected state %s", r.State)
			}
			errs <- err
		}()
	}
	wg.Wait()
	close(errs)
	for err := range errs {
		require.NoError(t, err)
	}
	assert.EqualValues(t, 1, f.polled.Load())
}

func TestHandoffRestartIsExplicit(t *testing.T) {
	f := newHandoffFixture(t)
	first := f.begin(t)
	require.NoError(t, ForgetPendingLogin(context.Background()))
	second := f.begin(t)
	assert.NotEqual(t, first.ID, second.ID)
	r, err := CheckLogin(context.Background(), DefaultAccessBaseURL, f.cfg, first.ID)
	require.NoError(t, err)
	assert.Equal(t, LoginHandoffSuperseded, r.State)
	assert.EqualValues(t, 2, f.issued.Load())
}

func TestHandoffRejectsWrongContextAndUnknownSchema(t *testing.T) {
	f := newHandoffFixture(t)
	h := f.begin(t)
	_, err := CheckLogin(context.Background(), QAAccessBaseURL, f.cfg, h.ID)
	require.Error(t, err)
	_, err = BeginOrResumeLogin(context.Background(), "https://untrusted.example", f.cfg)
	require.Error(t, err)
	cont, err := loadPendingDeviceAuth()
	require.NoError(t, err)
	cont.Version = 999
	require.NoError(t, savePendingDeviceAuth(cont))
	_, err = BeginOrResumeLogin(context.Background(), DefaultAccessBaseURL, f.cfg)
	require.Error(t, err)
	assert.EqualValues(t, 1, f.issued.Load())
	assert.EqualValues(t, 0, f.polled.Load())
	cont, err = loadPendingDeviceAuth()
	require.NoError(t, err)
	assert.Equal(t, 999, cont.Version)
}

func (f *handoffFixture) releaseBlocked() { f.releaseOnce.Do(func() { close(f.blockRelease) }) }

type cleanupFailHandoffStore struct {
	keyring.SecureStore
	fail bool
}

func (s *cleanupFailHandoffStore) Remove(key string) error {
	if s.fail && strings.HasPrefix(key, "oauth_login_completion.") {
		return errors.New("injected cleanup failure")
	}
	return s.SecureStore.Remove(key)
}
func TestHandoffCleanupFailureDoesNotReinstall(t *testing.T) {
	f := newHandoffFixture(t)
	h := f.begin(t)
	store := &cleanupFailHandoffStore{SecureStore: config.KeyRing, fail: true}
	config.KeyRing = store
	f.approved.Store(true)
	_, err := CheckLogin(context.Background(), DefaultAccessBaseURL, f.cfg, h.ID)
	require.ErrorContains(t, err, "completion_checkpoint_cleanup_failed")
	// Even when the account endpoint subsequently fails, the installed result
	// remains usable; retry only removes the old checkpoint.
	f.accountsFail.Store(true)
	store.fail = false
	r, err := CheckLogin(context.Background(), DefaultAccessBaseURL, f.cfg, h.ID)
	require.NoError(t, err)
	assert.Equal(t, LoginHandoffAuthenticated, r.State)
	assert.EqualValues(t, 1, f.polled.Load())
}

func TestHandoffCheckpointCannotOverwriteChangedContext(t *testing.T) {
	f := newHandoffFixture(t)
	h := f.begin(t)
	f.approved.Store(true)
	f.accountsFail.Store(true)
	_, err := CheckLogin(context.Background(), DefaultAccessBaseURL, f.cfg, h.ID)
	require.Error(t, err)
	require.NoError(t, config.KeyRing.Set(config.UATKeychainItemKey, []byte("oak_newer"), ""))
	require.NoError(t, config.SaveActiveContext("acct_newer", false))
	f.accountsFail.Store(false)
	r, err := CheckLogin(context.Background(), DefaultAccessBaseURL, f.cfg, h.ID)
	require.NoError(t, err)
	assert.Equal(t, LoginHandoffSuperseded, r.State)
	token, err := f.cfg.Profile.GetUAT()
	require.NoError(t, err)
	assert.Equal(t, "oak_newer", token)
}

func TestHandoffCredentialMutationFencesOldApproval(t *testing.T) {
	f := newHandoffFixture(t)
	h := f.begin(t)
	f.approved.Store(true)
	require.NoError(t, MutateLoginCredentials(context.Background(), func(context.Context) error {
		return config.KeyRing.Set(config.UATKeychainItemKey, []byte("oak_replacement"), "")
	}))
	r, err := CheckLogin(context.Background(), DefaultAccessBaseURL, f.cfg, h.ID)
	require.NoError(t, err)
	assert.Equal(t, LoginHandoffSuperseded, r.State)
	assert.EqualValues(t, 0, f.polled.Load())
}

func TestPendingLoginCommandRetainsContext(t *testing.T) {
	cfg := &config.Config{Profile: config.Profile{ProfileName: "team"}, ProfilesFile: "/config with spaces/cli.toml"}
	args, err := shellquote.Split(pendingLoginCommand(cfg))
	require.NoError(t, err)
	assert.Equal(t, []string{"stripe", "login", "--complete-device", "--project-name", "team", "--config", "/config with spaces/cli.toml"}, args)
}
