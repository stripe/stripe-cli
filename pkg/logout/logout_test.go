package logout

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

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

// A legacy profile with a period in its name is invisible to the profile
// listing, but `stripe logout --all` still has to end its session: afterwards,
// selecting that profile must not resolve to a usable credential.
func TestAllClearsOAuthSessionOfDottedProfile(t *testing.T) {
	profilesFile := filepath.Join(t.TempDir(), "config.toml")
	require.NoError(t, os.WriteFile(profilesFile, []byte(`[normal]
  display_name = 'Normal'

["example.project"]
  display_name = 'Legacy'
`), 0600))

	cfg := &config.Config{
		LogLevel: "info",
		Profile: config.Profile{
			ProfileName: "normal",
		},
		ProfilesFile: profilesFile,
	}

	dotted := &config.Profile{ProfileName: "example.project"}
	activeCtxJSON, err := json.Marshal(config.ActiveContext{AccountID: "acct_123", Livemode: true})
	require.NoError(t, err)

	config.KeyRing = keyring.NewMemoryStore(map[string][]byte{
		dotted.KeyringKey(config.UATKeychainItemKey):            []byte("oak_test_access"),
		dotted.KeyringKey(config.OAuthRefreshTokenKeychainKey):  []byte("oart_test_refresh"),
		dotted.KeyringKey(config.OAuthActiveContextKeychainKey): activeCtxJSON,
		dotted.KeyringKey(config.OAuthUATExpiresAtKeychainKey):  []byte(time.Now().Add(time.Hour).UTC().Format(time.RFC3339)),
	})
	t.Cleanup(func() {
		config.KeyRing = nil
		viper.Reset()
	})
	cfg.InitConfig()

	creds, err := dotted.ResolveCredentials(true)
	require.NoError(t, err)
	require.Equal(t, "oak_test_access", creds.Token)

	require.NoError(t, All(t.Context(), "https://access.stripe.com", cfg))

	_, err = dotted.ResolveCredentials(true)
	require.Error(t, err)

	for _, key := range []string{
		config.UATKeychainItemKey,
		config.OAuthRefreshTokenKeychainKey,
		config.OAuthActiveContextKeychainKey,
		config.OAuthUATExpiresAtKeychainKey,
	} {
		_, err = config.KeyRing.Get(dotted.KeyringKey(key))
		assert.ErrorIs(t, err, keyring.ErrKeyNotFound, "%s should have been removed from the keyring", key)
	}
}
