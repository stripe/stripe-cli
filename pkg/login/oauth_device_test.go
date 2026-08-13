package login

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRequestDeviceCode_Success(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodPost, r.Method)
		assert.Equal(t, "/stripecli/oauth2/device/authorization", r.URL.Path)
		assert.Equal(t, "application/x-www-form-urlencoded", r.Header.Get("Content-Type"))
		require.NoError(t, r.ParseForm())
		assert.Equal(t, "test-client-id", r.FormValue("client_id"))
		assert.Equal(t, "stripecli", r.FormValue("scope"))

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(DeviceAuthResponse{ //nolint:errcheck
			DeviceCode:      "dev-code-123",
			UserCode:        "ABCD-EFGH",
			VerificationURI: "https://qa-access.stripe.com/stripecli/oauth2/device",
			ExpiresIn:       300,
			Interval:        5,
		})
	}))
	defer ts.Close()

	resp, err := RequestDeviceCode(context.Background(), ts.URL, "test-client-id")
	require.NoError(t, err)
	assert.Equal(t, "dev-code-123", resp.DeviceCode)
	assert.Equal(t, "ABCD-EFGH", resp.UserCode)
	assert.Equal(t, "https://qa-access.stripe.com/stripecli/oauth2/device", resp.VerificationURI)
	assert.Equal(t, 5, resp.Interval)
}

func TestRequestDeviceCode_NonOK(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte("not found")) //nolint:errcheck
	}))
	defer ts.Close()

	_, err := RequestDeviceCode(context.Background(), ts.URL, "test-client-id")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "404")
}

func TestPollDeviceToken_Success(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.NoError(t, r.ParseForm())
		assert.Equal(t, "urn:ietf:params:oauth:grant-type:device_code", r.FormValue("grant_type"))
		assert.Equal(t, "device-code", r.FormValue("device_code"))

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(OAuthTokenResponse{ //nolint:errcheck
			AccessToken:  "oaac_test_access",
			RefreshToken: "oart_test_refresh",
			TokenType:    "Bearer",
			ExpiresIn:    3600,
		})
	}))
	defer ts.Close()

	resp, err := PollDeviceToken(context.Background(), ts.URL, "client-id", "device-code", 1*time.Millisecond)
	require.NoError(t, err)
	assert.Equal(t, "oaac_test_access", resp.AccessToken)
	assert.Equal(t, "oart_test_refresh", resp.RefreshToken)
}

func TestPollDeviceToken_AuthorizationPending(t *testing.T) {
	var callCount int32

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		n := atomic.AddInt32(&callCount, 1)
		w.Header().Set("Content-Type", "application/json")
		if n < 3 {
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(tokenErrorResponse{Error: "authorization_pending"}) //nolint:errcheck
			return
		}
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(OAuthTokenResponse{AccessToken: "oaac_final", RefreshToken: "oart_final"}) //nolint:errcheck
	}))
	defer ts.Close()

	resp, err := PollDeviceToken(context.Background(), ts.URL, "client-id", "device-code", 1*time.Millisecond)
	require.NoError(t, err)
	assert.Equal(t, "oaac_final", resp.AccessToken)
	assert.Equal(t, int32(3), atomic.LoadInt32(&callCount))
}

func TestPollDeviceToken_SlowDown(t *testing.T) {
	var callCount int32

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		n := atomic.AddInt32(&callCount, 1)
		w.Header().Set("Content-Type", "application/json")
		if n == 1 {
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(tokenErrorResponse{Error: "slow_down"}) //nolint:errcheck
			return
		}
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(OAuthTokenResponse{AccessToken: "oaac_final", RefreshToken: "oart_final"}) //nolint:errcheck
	}))
	defer ts.Close()

	resp, err := PollDeviceToken(context.Background(), ts.URL, "client-id", "device-code", 1*time.Millisecond)
	require.NoError(t, err)
	assert.Equal(t, "oaac_final", resp.AccessToken)
	assert.Equal(t, int32(2), atomic.LoadInt32(&callCount))
}

func TestPollDeviceToken_ExpiredToken(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(tokenErrorResponse{Error: "expired_token"}) //nolint:errcheck
	}))
	defer ts.Close()

	_, err := PollDeviceToken(context.Background(), ts.URL, "client-id", "device-code", 1*time.Millisecond)
	require.Error(t, err)
	var oauthErr *OAuthError
	require.True(t, errors.As(err, &oauthErr))
	assert.Equal(t, "expired_token", oauthErr.Code)
	assert.Equal(t, http.StatusBadRequest, oauthErr.HTTPStatus)
}

func TestPollDeviceToken_AccessDenied(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(tokenErrorResponse{Error: "access_denied"}) //nolint:errcheck
	}))
	defer ts.Close()

	_, err := PollDeviceToken(context.Background(), ts.URL, "client-id", "device-code", 1*time.Millisecond)
	require.Error(t, err)
	var oauthErr *OAuthError
	require.True(t, errors.As(err, &oauthErr))
	assert.Equal(t, "access_denied", oauthErr.Code)
}

func TestPollDeviceToken_InvalidGrant(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(tokenErrorResponse{Error: "invalid_grant", ErrorDescription: "permissions no longer valid"}) //nolint:errcheck
	}))
	defer ts.Close()

	_, err := PollDeviceToken(context.Background(), ts.URL, "client-id", "device-code", 1*time.Millisecond)
	require.Error(t, err)
	var oauthErr *OAuthError
	require.True(t, errors.As(err, &oauthErr))
	assert.Equal(t, "invalid_grant", oauthErr.Code)
	assert.Equal(t, "permissions no longer valid", oauthErr.Description)
}

func TestPollDeviceToken_InvalidClient(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(tokenErrorResponse{Error: "invalid_client"}) //nolint:errcheck
	}))
	defer ts.Close()

	_, err := PollDeviceToken(context.Background(), ts.URL, "client-id", "device-code", 1*time.Millisecond)
	require.Error(t, err)
	var oauthErr *OAuthError
	require.True(t, errors.As(err, &oauthErr))
	assert.Equal(t, "invalid_client", oauthErr.Code)
	assert.Equal(t, http.StatusUnauthorized, oauthErr.HTTPStatus)
}

func TestPollDeviceToken_ContextCancelled(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(tokenErrorResponse{Error: "authorization_pending"}) //nolint:errcheck
	}))
	defer ts.Close()

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := PollDeviceToken(ctx, ts.URL, "client-id", "device-code", 1*time.Millisecond)
	require.Error(t, err)
	assert.ErrorIs(t, err, context.Canceled)
}

func TestRefreshAccessToken_Success(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/stripecli/oauth2/token", r.URL.Path)
		require.NoError(t, r.ParseForm())
		assert.Equal(t, "refresh_token", r.FormValue("grant_type"))
		assert.Equal(t, "oart_old_refresh", r.FormValue("refresh_token"))
		assert.Equal(t, "test-client-id", r.FormValue("client_id"))

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

	resp, err := RefreshAccessToken(context.Background(), ts.URL, "test-client-id", "oart_old_refresh")
	require.NoError(t, err)
	assert.Equal(t, "oaac_new_access", resp.AccessToken)
	assert.Equal(t, "oart_new_refresh", resp.RefreshToken)
}

func TestLoginWithDeviceCodeFailsWhenAccountsFetchFails(t *testing.T) {
	origCanOpenBrowser := canOpenBrowser
	canOpenBrowser = func() bool { return false }
	t.Cleanup(func() { canOpenBrowser = origCanOpenBrowser })

	cfg, cleanup := setupOAuthTestConfig(t)
	defer cleanup()

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/stripecli/oauth2/device/authorization":
			json.NewEncoder(w).Encode(DeviceAuthResponse{ //nolint:errcheck
				DeviceCode:      "device-code",
				UserCode:        "ABCD-EFGH",
				VerificationURI: "https://example.com/verify",
				ExpiresIn:       30,
				Interval:        1,
			})
		case "/stripecli/oauth2/token":
			json.NewEncoder(w).Encode(OAuthTokenResponse{ //nolint:errcheck
				AccessToken:  "oaac_test_access",
				RefreshToken: "oart_test_refresh",
				TokenType:    "Bearer",
				ExpiresIn:    3600,
			})
		case "/stripecli/oauth2/token/accounts":
			http.Error(w, "unavailable", http.StatusInternalServerError)
		default:
			http.NotFound(w, r)
		}
	}))
	defer ts.Close()

	err := LoginWithDeviceCode(context.Background(), ts.URL, cfg)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to fetch account info")
}

func TestRefreshAccessToken_NoRefreshTokenInResponse(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(OAuthTokenResponse{ //nolint:errcheck
			AccessToken: "oaac_new_access",
			TokenType:   "Bearer",
			ExpiresIn:   3600,
			// RefreshToken intentionally absent
		})
	}))
	defer ts.Close()

	resp, err := RefreshAccessToken(context.Background(), ts.URL, "test-client-id", "oart_old_refresh")
	require.NoError(t, err)
	assert.Equal(t, "oaac_new_access", resp.AccessToken)
	assert.Empty(t, resp.RefreshToken)
}

func TestRefreshAccessToken_InvalidGrant(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(tokenErrorResponse{Error: "invalid_grant", ErrorDescription: "Invalid refresh token"}) //nolint:errcheck
	}))
	defer ts.Close()

	_, err := RefreshAccessToken(context.Background(), ts.URL, "test-client-id", "oart_bad")
	require.Error(t, err)
	var oauthErr *OAuthError
	require.True(t, errors.As(err, &oauthErr))
	assert.Equal(t, "invalid_grant", oauthErr.Code)
	assert.Equal(t, "Invalid refresh token", oauthErr.Description)
}
