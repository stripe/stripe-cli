package login

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/stripe/stripe-cli/pkg/config"
	"github.com/stripe/stripe-cli/pkg/keyring"
)

func (f *handoffFixture) oauthSession(t *testing.T, expiry time.Time) {
	t.Helper()
	require.NoError(t, config.KeyRing.Set(config.UATKeychainItemKey, []byte("oak_existing"), ""))
	require.NoError(t, config.KeyRing.Set(config.OAuthRefreshTokenKeychainKey, []byte("existing_refresh"), ""))
	require.NoError(t, config.SaveActiveContext("acct_existing", true))
	require.NoError(t, config.SaveUATExpiresAt(expiry))
}

func TestOAuthResolutionIgnoresAllAPIKeySources(t *testing.T) {
	f := newHandoffFixture(t)
	t.Setenv("STRIPE_API_KEY", "sk_test_override")
	f.cfg.Profile.APIKey = "sk_live_override"
	f.cfg.Profile.TestModeAPIKey = "sk_test_legacy"
	options := OAuthResolutionOptions{Livemode: true, AllowRefresh: true}
	r, err := ResolveOAuthCredentials(context.Background(), DefaultAccessBaseURL, f.cfg, options)
	require.NoError(t, err)
	assert.Equal(t, OAuthLoginRequired, r.State)
	assert.Empty(t, r.Token)
	f.oauthSession(t, time.Now().Add(time.Hour))
	r, err = ResolveOAuthCredentials(context.Background(), DefaultAccessBaseURL, f.cfg, options)
	require.NoError(t, err)
	assert.Equal(t, OAuthResolved, r.State)
	assert.Equal(t, "oak_existing", r.Token)
	assert.Equal(t, "acct_existing", r.StripeContext)
	assert.True(t, r.Livemode)
	assert.Zero(t, f.issued.Load())
	assert.Zero(t, f.refreshed.Load())
}

func TestOAuthResolutionContextAndModeRemainBound(t *testing.T) {
	f := newHandoffFixture(t)
	f.oauthSession(t, time.Now().Add(-time.Minute))
	for _, tc := range []struct {
		options OAuthResolutionOptions
		want    OAuthResolutionState
	}{
		{OAuthResolutionOptions{Livemode: false, AllowRefresh: true}, OAuthModeMismatch},
		{OAuthResolutionOptions{Livemode: true, ExpectedContext: "acct_different", AllowRefresh: true}, OAuthContextChanged},
	} {
		r, err := ResolveOAuthCredentials(context.Background(), DefaultAccessBaseURL, f.cfg, tc.options)
		require.NoError(t, err)
		assert.Equal(t, tc.want, r.State)
		assert.Empty(t, r.Token)
	}
	require.NoError(t, config.KeyRing.Remove(config.OAuthActiveContextKeychainKey))
	r, err := ResolveOAuthCredentials(context.Background(), DefaultAccessBaseURL, f.cfg, OAuthResolutionOptions{AllowRefresh: true})
	require.NoError(t, err)
	assert.Equal(t, OAuthContextRequired, r.State)
	assert.Zero(t, f.refreshed.Load())
	assert.Zero(t, f.issued.Load())
}

func TestOAuthResolutionRefreshAndReadOnly(t *testing.T) {
	f := newHandoffFixture(t)
	f.oauthSession(t, time.Now().Add(-time.Minute))
	options := OAuthResolutionOptions{Livemode: true, ExpectedContext: "acct_existing"}
	r, err := ResolveOAuthCredentials(context.Background(), DefaultAccessBaseURL, f.cfg, options)
	require.NoError(t, err)
	assert.Equal(t, OAuthRefreshRequired, r.State)
	assert.Zero(t, f.refreshed.Load())
	options.AllowRefresh = true
	r, err = ResolveOAuthCredentials(context.Background(), DefaultAccessBaseURL, f.cfg, options)
	require.NoError(t, err)
	assert.Equal(t, OAuthResolved, r.State)
	assert.Equal(t, "oak_fixture_renewed", r.Token)
	assert.Equal(t, "acct_existing", r.StripeContext)
	assert.EqualValues(t, 1, f.refreshed.Load())
}

func TestOAuthResolutionDistinguishesRenewalFailures(t *testing.T) {
	for _, code := range []string{"temporarily_unavailable", "invalid_grant", "missing_refresh"} {
		t.Run(code, func(t *testing.T) {
			f := newHandoffFixture(t)
			f.oauthSession(t, time.Now().Add(-time.Minute))
			f.refreshError = code
			if code == "missing_refresh" {
				require.NoError(t, config.KeyRing.Remove(config.OAuthRefreshTokenKeychainKey))
			}
			r, err := ResolveOAuthCredentials(context.Background(), DefaultAccessBaseURL, f.cfg, OAuthResolutionOptions{Livemode: true, AllowRefresh: true})
			require.NoError(t, err)
			want := OAuthLoginRequired
			if code == "temporarily_unavailable" {
				want = OAuthRefreshError
			}
			assert.Equal(t, want, r.State)
			assert.Empty(t, r.Token)
			assert.Zero(t, f.issued.Load())
			if want == OAuthLoginRequired {
				assert.Equal(t, LoginHandoffPending, f.begin(t).State, "terminal renewal failure must permit new authentication")
			}
		})
	}
}

type unreadableOAuthStore struct{ keyring.SecureStore }

func (s unreadableOAuthStore) Get(string) ([]byte, error) {
	return nil, errors.New("private store failure")
}

func TestOAuthResolutionStorageFailureDoesNotStartLogin(t *testing.T) {
	f := newHandoffFixture(t)
	config.KeyRing = unreadableOAuthStore{config.KeyRing}
	r, err := ResolveOAuthCredentials(context.Background(), DefaultAccessBaseURL, f.cfg, OAuthResolutionOptions{AllowRefresh: true})
	require.NoError(t, err)
	assert.Equal(t, OAuthStorageError, r.State)
	assert.Empty(t, r.Token)
	assert.Zero(t, f.issued.Load())
}

func TestOAuthResolutionIncompleteInstallResumesBeforeSelectingContext(t *testing.T) {
	f := newHandoffFixture(t)
	h := f.begin(t)
	f.approved.Store(true)
	f.accountsFail.Store(true)
	_, err := CheckLogin(context.Background(), DefaultAccessBaseURL, f.cfg, h.ID)
	require.Error(t, err)
	r, err := ResolveOAuthCredentials(context.Background(), DefaultAccessBaseURL, f.cfg, OAuthResolutionOptions{Livemode: true, AllowRefresh: true})
	require.NoError(t, err)
	assert.Equal(t, OAuthCompletionRequired, r.State)
	assert.Empty(t, r.Token)
	f.accountsFail.Store(false)
	_, err = CheckLogin(context.Background(), DefaultAccessBaseURL, f.cfg, h.ID)
	require.NoError(t, err)
	r, err = ResolveOAuthCredentials(context.Background(), DefaultAccessBaseURL, f.cfg, OAuthResolutionOptions{Livemode: true, AllowRefresh: true})
	require.NoError(t, err)
	assert.Equal(t, OAuthResolved, r.State)
	assert.EqualValues(t, 1, f.polled.Load())
}
