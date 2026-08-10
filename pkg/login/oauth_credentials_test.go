package login

import (
	"path/filepath"
	"testing"

	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/stripe/stripe-cli/pkg/config"
	"github.com/stripe/stripe-cli/pkg/keyring"
)

func setupOAuthTestConfig(t *testing.T) (*config.Config, func()) {
	t.Helper()
	profilesFile := filepath.Join(t.TempDir(), "config.toml")
	viper.SetConfigFile(profilesFile)
	t.Cleanup(viper.Reset)

	cfg := &config.Config{
		Color:    "auto",
		LogLevel: "info",
		Profile: config.Profile{
			ProfileName: "test",
		},
		ProfilesFile: profilesFile,
	}
	cfg.InitConfig()
	config.KeyRing = keyring.NewMemoryStore(nil)
	return cfg, func() {
		config.KeyRing = nil
	}
}

func TestSaveOAuthCredentials(t *testing.T) {
	cfg, cleanup := setupOAuthTestConfig(t)
	defer cleanup()

	resp := &OAuthTokenResponse{
		AccessToken:  "oaac_test_access",
		RefreshToken: "oart_test_refresh",
		TokenType:    "Bearer",
		ExpiresIn:    3600,
	}

	err := saveOAuthCredentials(cfg, resp)
	require.NoError(t, err)

	uat, err := config.KeyRing.Get(config.UATKeychainItemKey)
	require.NoError(t, err)
	assert.Equal(t, "oaac_test_access", string(uat))

	rt, err := config.KeyRing.Get(OAuthRefreshTokenKeychainKey)
	require.NoError(t, err)
	assert.Equal(t, "oart_test_refresh", string(rt))
}

func TestUpdateOAuthTokens_WithNewRefreshToken(t *testing.T) {
	cfg, cleanup := setupOAuthTestConfig(t)
	defer cleanup()

	// Seed initial tokens.
	require.NoError(t, config.KeyRing.Set(config.UATKeychainItemKey, []byte("oaac_old"), ""))
	require.NoError(t, config.KeyRing.Set(OAuthRefreshTokenKeychainKey, []byte("oart_old"), ""))

	resp := &OAuthTokenResponse{
		AccessToken:  "oaac_new",
		RefreshToken: "oart_new",
	}
	require.NoError(t, UpdateOAuthTokens(cfg, resp))

	uat, err := config.KeyRing.Get(config.UATKeychainItemKey)
	require.NoError(t, err)
	assert.Equal(t, "oaac_new", string(uat))

	rt, err := config.KeyRing.Get(OAuthRefreshTokenKeychainKey)
	require.NoError(t, err)
	assert.Equal(t, "oart_new", string(rt))
}

func TestUpdateOAuthTokens_NoRefreshToken(t *testing.T) {
	cfg, cleanup := setupOAuthTestConfig(t)
	defer cleanup()

	// Seed initial tokens.
	require.NoError(t, config.KeyRing.Set(config.UATKeychainItemKey, []byte("oaac_old"), ""))
	require.NoError(t, config.KeyRing.Set(OAuthRefreshTokenKeychainKey, []byte("oart_old"), ""))

	// Response with no refresh token — old RT must be preserved.
	resp := &OAuthTokenResponse{
		AccessToken: "oaac_new",
		// RefreshToken intentionally absent
	}
	require.NoError(t, UpdateOAuthTokens(cfg, resp))

	uat, err := config.KeyRing.Get(config.UATKeychainItemKey)
	require.NoError(t, err)
	assert.Equal(t, "oaac_new", string(uat))

	rt, err := config.KeyRing.Get(OAuthRefreshTokenKeychainKey)
	require.NoError(t, err)
	assert.Equal(t, "oart_old", string(rt), "refresh token must not change when response omits it")
}

func TestClearOAuthCredentials(t *testing.T) {
	cfg, cleanup := setupOAuthTestConfig(t)
	defer cleanup()

	// Seed tokens.
	require.NoError(t, config.KeyRing.Set(config.UATKeychainItemKey, []byte("oaac_to_clear"), ""))
	require.NoError(t, config.KeyRing.Set(OAuthRefreshTokenKeychainKey, []byte("oart_to_clear"), ""))

	require.NoError(t, ClearOAuthCredentials(cfg))

	_, err := config.KeyRing.Get(config.UATKeychainItemKey)
	assert.Error(t, err, "access token should be removed")

	_, err = config.KeyRing.Get(OAuthRefreshTokenKeychainKey)
	assert.Error(t, err, "refresh token should be removed")
}

func TestClearOAuthCredentials_NoExistingTokens(t *testing.T) {
	cfg, cleanup := setupOAuthTestConfig(t)
	defer cleanup()

	// Should not error when nothing is present.
	assert.NoError(t, ClearOAuthCredentials(cfg))
}
