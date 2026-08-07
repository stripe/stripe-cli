package login

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/stripe/stripe-cli/pkg/config"
	"github.com/stripe/stripe-cli/pkg/keyring"
)

// OAuthRefreshTokenKeychainKey is the keyring key for the OAuth refresh token.
const OAuthRefreshTokenKeychainKey = config.OAuthRefreshTokenKeychainKey

func saveOAuthCredentials(cfg *config.Config, resp *OAuthTokenResponse) error {
	return UpdateOAuthTokens(cfg, resp)
}

// UpdateOAuthTokens replaces the stored access token and refresh token after a
// successful token refresh. If resp.RefreshToken is empty, the refresh token is
// left unchanged.
func UpdateOAuthTokens(cfg *config.Config, resp *OAuthTokenResponse) error {
	cfg.Profile.UAT = resp.AccessToken
	if err := cfg.Profile.CreateProfile(); err != nil {
		return err
	}

	if resp.RefreshToken != "" {
		if err := config.KeyRing.Set(OAuthRefreshTokenKeychainKey, []byte(resp.RefreshToken), "Stripe CLI OAuth refresh token"); err != nil {
			return err
		}
	}

	return nil
}

// RevokeToken revokes the stored refresh token via the revocation endpoint.
// Returns nil if no refresh token is stored. Errors from the server are
// returned to callers, who should log and continue with credential cleanup.
func RevokeToken(ctx context.Context, accessBaseURL string) error {
	if config.KeyRing == nil {
		return nil
	}
	refreshTokenBytes, err := config.KeyRing.Get(config.OAuthRefreshTokenKeychainKey)
	if err != nil {
		if errors.Is(err, keyring.ErrKeyNotFound) {
			return nil
		}
		return err
	}

	data := url.Values{}
	data.Set("token", string(refreshTokenBytes))
	data.Set("client_id", clientIDForAccessBaseURL(accessBaseURL))

	endpoint := accessBaseURL + accessAPNPath + "/revoke"
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, strings.NewReader(data.Encode()))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent {
		return fmt.Errorf("token revocation failed (status %d)", resp.StatusCode)
	}
	return nil
}

// ClearOAuthCredentials removes the stored access token, refresh token, and
// compartment data. Call this when the server returns invalid_grant on a
// refresh attempt.
func ClearOAuthCredentials(cfg *config.Config) error {
	cfg.Profile.UAT = ""
	if err := cfg.Profile.CreateProfile(); err != nil {
		return err
	}

	if err := config.KeyRing.Remove(OAuthRefreshTokenKeychainKey); err != nil && !errors.Is(err, keyring.ErrKeyNotFound) {
		return err
	}

	if err := config.KeyRing.Remove(config.OAuthActiveContextKeychainKey); err != nil && !errors.Is(err, keyring.ErrKeyNotFound) {
		return err
	}

	return nil
}
