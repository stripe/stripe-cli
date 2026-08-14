package cmd

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/stripe/stripe-cli/pkg/config"
	"github.com/stripe/stripe-cli/pkg/keyring"
)

// TestLoginNewSessionRevokesPreviousToken verifies that `stripe login --new-session`
// revokes the previously stored OAuth token, same as `stripe logout`, before
// starting a new login flow.
//
// revokeToken and initiateLogin are stubbed rather than backed by an
// httptest server: --access-base is validated against a hardcoded
// production/QA allowlist before any of this logic runs (it carries OAuth
// credentials), so a mock server URL can never be assigned to it.
func TestLoginNewSessionRevokesPreviousToken(t *testing.T) {
	origRevokeToken, origInitiateLogin := revokeToken, initiateLogin
	t.Cleanup(func() {
		revokeToken = origRevokeToken
		initiateLogin = origInitiateLogin
	})

	var revokeCalled, initiateLoginCalled bool
	revokeToken = func(ctx context.Context, accessBaseURL string) error {
		revokeCalled = true
		rt, err := config.KeyRing.Get(config.OAuthRefreshTokenKeychainKey)
		require.NoError(t, err)
		assert.Equal(t, "oart_previous_refresh", string(rt))
		return nil
	}
	initiateLogin = func(ctx context.Context, dashboardBaseURL, accessBaseURL string, cfg *config.Config) error {
		initiateLoginCalled = true
		assert.True(t, revokeCalled, "expected the previous session to be revoked before continuing")
		return nil
	}

	profilesFile := filepath.Join(t.TempDir(), "config.toml")
	Config = config.Config{
		LogLevel: "info",
		Profile: config.Profile{
			ProfileName: "default",
			DeviceName:  "test-device",
		},
		ProfilesFile: profilesFile,
	}
	Config.InitConfig()

	config.KeyRing = keyring.NewMemoryStore(map[string][]byte{
		config.UATKeychainItemKey:           []byte("oak_previous_uat"),
		config.OAuthRefreshTokenKeychainKey: []byte("oart_previous_refresh"),
	})
	t.Cleanup(func() {
		config.KeyRing = nil
		viper.Reset()
	})

	lc := newLoginCmd()
	lc.newSession = true
	lc.nonInteractive = true
	lc.cmd.SetContext(context.Background())

	require.NoError(t, lc.runLoginCmd(lc.cmd, []string{}))
	assert.True(t, revokeCalled, "expected --new-session to revoke the previous OAuth token")
	assert.True(t, initiateLoginCalled, "expected login to continue after revoking")
}
