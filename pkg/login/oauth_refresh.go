package login

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/url"
	"time"

	"github.com/stripe/stripe-cli/pkg/config"
	"github.com/stripe/stripe-cli/pkg/errorcategory"
	"github.com/stripe/stripe-cli/pkg/keyring"
)

func init() {
	config.OAuthTokenRefresher = refreshOAuthToken
}

// refreshOAuthToken exchanges the stored refresh token for a new access token
// and updates p in-place. On invalid_grant it clears all OAuth credentials so
// the next ResolveCredentials call surfaces an actionable re-login error.
func refreshOAuthToken(p *config.Profile) error {
	if config.KeyRing == nil {
		return errorcategory.New(errorcategory.Auth, "keyring unavailable; run 'stripe login' to re-authenticate")
	}

	refreshTokenBytes, err := config.KeyRing.Get(config.OAuthRefreshTokenKeychainKey)
	if err != nil {
		if errors.Is(err, keyring.ErrKeyNotFound) {
			return errorcategory.New(errorcategory.Auth, "session expired; run 'stripe login' to re-authenticate")
		}
		return err
	}

	accessBaseURL := p.OAuthAccessBaseURL
	if accessBaseURL == "" {
		accessBaseURL = DefaultAccessBaseURL
	}
	clientID := clientIDForAccessBaseURL(accessBaseURL)

	tokenResp, err := doRefreshToken(context.Background(), accessBaseURL, clientID, string(refreshTokenBytes))
	if err != nil {
		var oauthErr *OAuthError
		if errors.As(err, &oauthErr) && oauthErr.Code == "invalid_grant" {
			// Token is invalid or consumed; clear credentials so the user gets a
			// clear "run stripe login" error on the next attempt rather than an
			// opaque 401 from the Stripe API.
			_ = clearOAuthCredentialsForProfile(p)
			return errorcategory.New(errorcategory.Auth, "session expired; run 'stripe login' to re-authenticate")
		}
		return fmt.Errorf("failed to refresh session: %w", err)
	}

	p.UAT = tokenResp.AccessToken
	if err := p.CreateProfile(); err != nil {
		return err
	}

	if tokenResp.RefreshToken != "" {
		if err := config.KeyRing.Set(config.OAuthRefreshTokenKeychainKey, []byte(tokenResp.RefreshToken), "Stripe CLI OAuth refresh token"); err != nil {
			return err
		}
	} else {
		// Per spec: no refresh token in response means the submitted token was
		// consumed and no replacement was issued. Remove it so the next expiry
		// triggers a re-login rather than another invalid_grant attempt.
		_ = config.KeyRing.Remove(config.OAuthRefreshTokenKeychainKey)
	}

	if tokenResp.ExpiresIn > 0 {
		expiresAt := time.Now().Add(time.Duration(tokenResp.ExpiresIn) * time.Second)
		if err := config.SaveUATExpiresAt(expiresAt); err != nil {
			return err
		}
	}

	return nil
}

// doRefreshToken calls the token endpoint with grant_type=refresh_token.
func doRefreshToken(ctx context.Context, accessBaseURL, clientID, refreshToken string) (*OAuthTokenResponse, error) {
	endpoint := accessBaseURL + accessAPNPath + "/token"

	data := url.Values{}
	data.Set("grant_type", "refresh_token")
	data.Set("refresh_token", refreshToken)
	data.Set("client_id", clientID)

	resp, err := doPostForm(ctx, endpoint, data)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != 200 {
		var errResp tokenErrorResponse
		if jsonErr := json.Unmarshal(body, &errResp); jsonErr == nil && errResp.Error != "" {
			return nil, &OAuthError{Code: errResp.Error, Description: errResp.ErrorDescription, HTTPStatus: resp.StatusCode}
		}
		return nil, errorcategory.Errorf(errorcategory.Auth, "token refresh failed (status %d)", resp.StatusCode)
	}

	var tokenResp OAuthTokenResponse
	if err := json.Unmarshal(body, &tokenResp); err != nil {
		return nil, fmt.Errorf("failed to parse refresh response: %w", err)
	}
	return &tokenResp, nil
}
