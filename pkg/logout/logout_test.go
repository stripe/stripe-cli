package logout

import (
	"encoding/json"
	"path/filepath"
	"testing"

	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/stripe/stripe-cli/pkg/config"
	"github.com/stripe/stripe-cli/pkg/keyring"
)

func TestLogoutClearsOAuthCredentials(t *testing.T) {
	profilesFile := filepath.Join(t.TempDir(), "config.toml")
	cfg := &config.Config{
		LogLevel: "info",
		Profile: config.Profile{
			ProfileName: "default",
		},
		ProfilesFile: profilesFile,
	}
	cfg.InitConfig()

	activeCtxJSON, err := json.Marshal(config.ActiveContext{AccountID: "acct_123", Livemode: false})
	require.NoError(t, err)

	config.KeyRing = keyring.NewMemoryStore(map[string][]byte{
		config.UATKeychainItemKey:            []byte("oaac_test_access"),
		config.OAuthRefreshTokenKeychainKey:  []byte("oart_test_refresh"),
		config.OAuthActiveContextKeychainKey: activeCtxJSON,
	})
	t.Cleanup(func() {
		config.KeyRing = nil
		viper.Reset()
	})

	require.NoError(t, Logout(t.Context(), "https://access.stripe.com", cfg))

	_, err = config.KeyRing.Get(config.UATKeychainItemKey)
	assert.Error(t, err)
	_, err = config.KeyRing.Get(config.OAuthRefreshTokenKeychainKey)
	assert.Error(t, err)
	_, err = config.KeyRing.Get(config.OAuthActiveContextKeychainKey)
	assert.Error(t, err)
}
