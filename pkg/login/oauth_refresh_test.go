package login

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/stripe/stripe-cli/pkg/config"
)

// TestRefreshOAuthToken_PreservesProfileMetadata guards against a regression
// where refreshing the UAT wiped display_name/account_id/user_id: the refresh
// path only learns the new access token, but CreateProfile deletes those
// fields before rewriting whatever is (or isn't) set on the Profile struct.
func TestRefreshOAuthToken_PreservesProfileMetadata(t *testing.T) {
	cfg, cleanup := setupOAuthTestConfig(t)
	defer cleanup()

	// Simulate a completed login: display name, account ID, and user ID are
	// persisted alongside the access token.
	cfg.Profile.UAT = "oaac_old_access"
	cfg.Profile.DisplayName = "Acme Inc"
	cfg.Profile.AccountID = "acct_123"
	cfg.Profile.UserID = "user_123"
	require.NoError(t, cfg.Profile.CreateProfile())

	require.NoError(t, config.KeyRing.Set(config.OAuthRefreshTokenKeychainKey, []byte("oart_old_refresh"), "test"))

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(OAuthTokenResponse{ //nolint:errcheck
			AccessToken:  "oaac_new_access",
			RefreshToken: "oart_new_refresh",
			TokenType:    "Bearer",
			ExpiresIn:    3600,
		})
	}))
	defer ts.Close()

	p := &config.Profile{
		ProfileName:        cfg.Profile.ProfileName,
		OAuthAccessBaseURL: ts.URL,
	}
	require.NoError(t, refreshOAuthToken(p))

	assert.Equal(t, "oaac_new_access", p.UAT)
	assert.Equal(t, "Acme Inc", viper.GetString(p.GetConfigField(config.DisplayNameName)))
	assert.Equal(t, "acct_123", viper.GetString(p.GetConfigField(config.AccountIDName)))
	assert.Equal(t, "user_123", viper.GetString(p.GetConfigField(config.UserIDName)))
}
