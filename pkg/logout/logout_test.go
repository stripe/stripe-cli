package logout

import (
	"encoding/json"
	"fmt"
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

// All-project logout must clear readable legacy profiles, including names that
// Viper lowercases or interprets as nested config tables.
func TestAllClearsOAuthSessionOfLegacyProfiles(t *testing.T) {
	for _, version := range []int{1, 2} {
		for _, name := range []string{"example.project", "Example.Project", "plugin_configs", "plugin_configs.project", "profiles.work", "profiles"} {
			t.Run(fmt.Sprintf("v%d/%s", version, name), func(t *testing.T) {
				profilesFile := filepath.Join(t.TempDir(), "config.toml")
				contents := ""
				prefix := ""
				if version == 2 {
					contents = "config_version = 2\n"
					prefix = "profiles."
				}
				contents += fmt.Sprintf("[%s%q]\n  display_name = 'Legacy'\n", prefix, name)
				require.NoError(t, os.WriteFile(profilesFile, []byte(contents), 0600))

				cfg := &config.Config{
					LogLevel:     "info",
					Profile:      config.Profile{ProfileName: "normal"},
					ProfilesFile: profilesFile,
				}
				profile := &config.Profile{ProfileName: name}
				activeCtxJSON, err := json.Marshal(config.ActiveContext{AccountID: "acct_123", Livemode: true})
				require.NoError(t, err)
				config.KeyRing = keyring.NewMemoryStore(map[string][]byte{
					profile.KeyringKey(config.UATKeychainItemKey):            []byte("oak_test_access"),
					profile.KeyringKey(config.OAuthRefreshTokenKeychainKey):  []byte("oart_test_refresh"),
					profile.KeyringKey(config.OAuthActiveContextKeychainKey): activeCtxJSON,
					profile.KeyringKey(config.OAuthUATExpiresAtKeychainKey):  []byte(time.Now().Add(time.Hour).UTC().Format(time.RFC3339)),
				})
				t.Cleanup(func() {
					config.KeyRing = nil
					viper.Reset()
				})
				cfg.InitConfig()
				// Exercise logout after Viper has already rewritten the table
				// name in lowercase, losing the keyring identifier's casing.
				require.NoError(t, cfg.WriteConfigField(config.MachineUUIDKey, "test-machine"))

				creds, err := profile.ResolveCredentials(true)
				require.NoError(t, err)
				require.Equal(t, "oak_test_access", creds.Token)

				require.NoError(t, All(t.Context(), "https://access.stripe.com", cfg))

				// Reload and resolve through a fresh profile as the next CLI
				// invocation would, without relying on cached session state.
				cfg.InitConfig()
				profile = &config.Profile{ProfileName: name}
				_, err = profile.ResolveCredentials(true)
				require.Error(t, err)

				for _, key := range []string{
					config.UATKeychainItemKey,
					config.OAuthRefreshTokenKeychainKey,
					config.OAuthActiveContextKeychainKey,
					config.OAuthUATExpiresAtKeychainKey,
				} {
					_, err = config.KeyRing.Get(profile.KeyringKey(key))
					assert.ErrorIs(t, err, keyring.ErrKeyNotFound, "%s should have been removed from the keyring", key)
				}
			})
		}
	}
}
