package requests

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/stripe/stripe-cli/pkg/config"
)

// requiresNewerCLIBody is the error body the plugin metadata endpoints send for a
// release whose min_core_version this CLI does not meet, copied from the shape the
// server's own tests assert on.
const requiresNewerCLIBody = `{
  "error": {
    "code": "plugin_requires_newer_cli",
    "message": "Version 2.0.1 of the appA plugin requires Stripe CLI 1.30.0 or later. Upgrade the Stripe CLI, or request a plugin version your CLI supports.",
    "min_core_version": "1.30.0",
    "param": "version",
    "type": "invalid_request_error"
  }
}`

// requiresNewerCLIWithoutVersionBody is the same answer for a request that named no
// version, where the endpoint reports the lowest floor among the plugin's releases.
// It blames no parameter, because the caller supplied none to blame, and names no
// plugin version. Both are absent from the wire, not merely unused here.
const requiresNewerCLIWithoutVersionBody = `{
  "error": {
    "code": "plugin_requires_newer_cli",
    "message": "The appA plugin requires Stripe CLI 1.30.0 or later. Upgrade the Stripe CLI to install it.",
    "min_core_version": "1.30.0",
    "type": "invalid_request_error"
  }
}`

// apiKeyExpiredBody is the API's answer for a key past its expiry date, such as
// one saved by `stripe login` long enough ago.
const apiKeyExpiredBody = `{
  "error": {
    "code": "api_key_expired",
    "message": "Expired API Key provided: rk_test_*************1234",
    "type": "invalid_request_error"
  }
}`

func TestPluginRequiresNewerCLI(t *testing.T) {
	tests := []struct {
		name               string
		err                error
		wantMinCoreVersion string
		wantOK             bool
	}{
		{
			name:               "typed response",
			err:                compileRequestError([]byte(requiresNewerCLIBody), http.StatusBadRequest),
			wantMinCoreVersion: "1.30.0",
			wantOK:             true,
		},
		{
			// The code is what identifies this answer, so the minimum version still
			// arrives for a request that named no version -- the case where the caller
			// has nothing else to tell the user to do.
			name:               "typed response for a request that named no version",
			err:                compileRequestError([]byte(requiresNewerCLIWithoutVersionBody), http.StatusBadRequest),
			wantMinCoreVersion: "1.30.0",
			wantOK:             true,
		},
		{
			// Still the same answer, so it is still recognized: the caller reports the
			// upgrade without naming a target rather than losing the reason for it.
			name:               "typed response without the extra attribute",
			err:                compileRequestError([]byte(`{"error":{"code":"plugin_requires_newer_cli"}}`), http.StatusBadRequest),
			wantMinCoreVersion: "",
			wantOK:             true,
		},
		{
			name:   "some other bad request",
			err:    compileRequestError([]byte(`{"error":{"code":"parameter_invalid_string_empty","param":"version"}}`), http.StatusBadRequest),
			wantOK: false,
		},
		{
			// The status is checked as well as the code so that a hidden release, which
			// answers 404 with no code at all, keeps being reported as a missing version.
			name:   "not found",
			err:    compileRequestError([]byte(`{"error":{"message":"not found"}}`), http.StatusNotFound),
			wantOK: false,
		},
		{
			name:   "right code on the wrong status",
			err:    compileRequestError([]byte(requiresNewerCLIBody), http.StatusInternalServerError),
			wantOK: false,
		},
		{
			name:   "not a request error",
			err:    errors.New("dial tcp: connection refused"),
			wantOK: false,
		},
		{
			name:               "wrapped request error",
			err:                fmt.Errorf("could not fetch plugin metadata: %w", compileRequestError([]byte(requiresNewerCLIBody), http.StatusBadRequest)),
			wantMinCoreVersion: "1.30.0",
			wantOK:             true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			minCoreVersion, ok := PluginRequiresNewerCLI(tt.err)
			require.Equal(t, tt.wantOK, ok)
			require.Equal(t, tt.wantMinCoreVersion, minCoreVersion)
		})
	}
}

// TestGetPluginMetadataSurfacesRequiresNewerCLI pins the whole round trip. The
// minimum version travels as an extra attribute on the error body, so it only
// survives as long as the request layer keeps that body intact.
func TestGetPluginMetadataSurfacesRequiresNewerCLI(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(res http.ResponseWriter, req *http.Request) {
		require.Equal(t, "/v1/stripecli/get-plugin-metadata", req.URL.Path)
		res.Header().Set("Content-Type", "application/json")
		res.WriteHeader(http.StatusBadRequest)
		_, _ = res.Write([]byte(requiresNewerCLIBody))
	}))
	defer server.Close()

	profile := &config.Profile{APIKey: "sk_test_1234"}

	_, err := GetPluginMetadata(context.Background(), server.URL, server.URL, "2020-08-27", "sk_test_1234", profile, "appA", "2.0.1", "darwin", "arm64", true)
	require.Error(t, err)

	minCoreVersion, ok := PluginRequiresNewerCLI(err)
	require.True(t, ok)
	require.Equal(t, "1.30.0", minCoreVersion)
}

// TestGetPluginMetadataRetriesAnonymouslyWhenAPIKeyExpired covers a key the API
// rejects as expired, which the anonymous endpoint never needed.
func TestGetPluginMetadataRetriesAnonymouslyWhenAPIKeyExpired(t *testing.T) {
	const expiredKey = "rk_test_expired_1234"

	tests := []struct {
		name            string
		anonymousStatus int
		anonymousBody   string
		check           func(t *testing.T, metadata PluginMetadata, err error)
	}{
		{
			name:            "plugin anyone can install",
			anonymousStatus: http.StatusOK,
			anonymousBody:   `{"binary_url":"https://example.test/appA/2.0.1","plugin_manifest":"manifest"}`,
			check: func(t *testing.T, metadata PluginMetadata, err error) {
				require.NoError(t, err)
				require.Equal(t, "https://example.test/appA/2.0.1", metadata.BinaryURL)
				require.Equal(t, "manifest", metadata.PluginManifest)
			},
		},
		{
			// Possibly a plugin only a logged-in account can see, so the expired key
			// is still what gets reported: logging in again is the fix.
			name:            "plugin the anonymous endpoint cannot find",
			anonymousStatus: http.StatusNotFound,
			anonymousBody:   `{"error":{"message":"not found"}}`,
			check: func(t *testing.T, _ PluginMetadata, err error) {
				require.True(t, IsAPIKeyExpiredError(err), "want the expired key reported, got: %v", err)
			},
		},
		{
			// No login changes an outage, so it is reported instead of the key.
			name:            "anonymous endpoint down",
			anonymousStatus: http.StatusInternalServerError,
			anonymousBody:   `{"error":{"message":"boom"}}`,
			check: func(t *testing.T, _ PluginMetadata, err error) {
				var requestErr RequestError
				require.ErrorAs(t, err, &requestErr)
				require.Equal(t, http.StatusInternalServerError, requestErr.StatusCode)
				require.False(t, IsAPIKeyExpiredError(err))
			},
		},
		{
			// No login changes this answer, so it is reported instead of the key.
			name:            "plugin that needs a newer CLI",
			anonymousStatus: http.StatusBadRequest,
			anonymousBody:   requiresNewerCLIBody,
			check: func(t *testing.T, _ PluginMetadata, err error) {
				minCoreVersion, ok := PluginRequiresNewerCLI(err)
				require.True(t, ok, "want the newer-CLI refusal reported, got: %v", err)
				require.Equal(t, "1.30.0", minCoreVersion)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// The profile's key is the one that should be sent, and STRIPE_API_KEY
			// would win over it.
			t.Setenv("STRIPE_API_KEY", "")

			var apiRequests, dashboardRequests int
			apiServer := httptest.NewServer(http.HandlerFunc(func(res http.ResponseWriter, req *http.Request) {
				apiRequests++
				assert.Equal(t, "/v1/stripecli/get-plugin-metadata", req.URL.Path)
				assert.Equal(t, "Bearer "+expiredKey, req.Header.Get("Authorization"))
				res.WriteHeader(http.StatusUnauthorized)
				_, _ = res.Write([]byte(apiKeyExpiredBody))
			}))
			defer apiServer.Close()

			dashboardServer := httptest.NewServer(http.HandlerFunc(func(res http.ResponseWriter, req *http.Request) {
				dashboardRequests++
				assert.Equal(t, "/ajax/stripecli/plugins_metadata", req.URL.Path)
				assert.Empty(t, req.Header.Get("Authorization"), "the expired key must not be sent to the Dashboard host")
				assert.Equal(t, "appA", req.URL.Query().Get("plugin"))
				assert.Equal(t, "2.0.1", req.URL.Query().Get("version"))
				res.WriteHeader(tt.anonymousStatus)
				_, _ = res.Write([]byte(tt.anonymousBody))
			}))
			defer dashboardServer.Close()

			profile := &config.Profile{APIKey: expiredKey}
			metadata, err := GetPluginMetadata(context.Background(), apiServer.URL, dashboardServer.URL, "2020-08-27", expiredKey, profile, "appA", "2.0.1", "darwin", "arm64", true)
			tt.check(t, metadata, err)
			require.Equal(t, 1, apiRequests)
			require.Equal(t, 1, dashboardRequests)
		})
	}
}

// TestGetPluginMetadataRemembersAnExpiredKey covers lookups after the first in one
// process, such as install's second metadata fetch: they go straight to the
// anonymous endpoint instead of repeating a request certain to fail, while a
// lookup that must not fall back still gets the expired key reported.
func TestGetPluginMetadataRemembersAnExpiredKey(t *testing.T) {
	const expiredKey = "rk_test_expired_1234"

	var apiRequests, dashboardRequests int
	apiServer := httptest.NewServer(http.HandlerFunc(func(res http.ResponseWriter, req *http.Request) {
		apiRequests++
		res.WriteHeader(http.StatusUnauthorized)
		_, _ = res.Write([]byte(apiKeyExpiredBody))
	}))
	defer apiServer.Close()

	dashboardServer := httptest.NewServer(http.HandlerFunc(func(res http.ResponseWriter, req *http.Request) {
		dashboardRequests++
		_, _ = res.Write([]byte(`{"binary_url":"https://example.test/appA/2.0.1","plugin_manifest":"manifest"}`))
	}))
	defer dashboardServer.Close()

	profile := &config.Profile{APIKey: expiredKey}
	for range 2 {
		_, err := GetPluginMetadata(context.Background(), apiServer.URL, dashboardServer.URL, "2020-08-27", expiredKey, profile, "appA", "2.0.1", "darwin", "arm64", true)
		require.NoError(t, err)
	}
	require.Equal(t, 1, apiRequests)
	require.Equal(t, 2, dashboardRequests)

	_, err := GetPluginMetadata(context.Background(), apiServer.URL, dashboardServer.URL, "2020-08-27", expiredKey, profile, "appA", "", "darwin", "arm64", false)
	require.True(t, IsAPIKeyExpiredError(err), "want the expired key reported, got: %v", err)
	require.Equal(t, 2, dashboardRequests)
}

// TestGetPluginMetadataRetriesAnonymouslyOnlyForAnExpiredKey keeps the retry to
// the failure it is for: anything else the authenticated endpoint answers is
// reported as it is.
func TestGetPluginMetadataRetriesAnonymouslyOnlyForAnExpiredKey(t *testing.T) {
	tests := []struct {
		name   string
		status int
		body   string
	}{
		{"unauthorized for another reason", http.StatusUnauthorized, `{"error":{"message":"Invalid API Key provided: sk_test_****1234","type":"invalid_request_error"}}`},
		{"plugin not found", http.StatusNotFound, `{"error":{"message":"not found"}}`},
		{"server error", http.StatusInternalServerError, `{"error":{"message":"boom"}}`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			apiServer := httptest.NewServer(http.HandlerFunc(func(res http.ResponseWriter, req *http.Request) {
				res.WriteHeader(tt.status)
				_, _ = res.Write([]byte(tt.body))
			}))
			defer apiServer.Close()

			dashboardServer := httptest.NewServer(http.HandlerFunc(func(res http.ResponseWriter, req *http.Request) {
				t.Errorf("unexpected anonymous retry: %s", req.URL.String())
			}))
			defer dashboardServer.Close()

			profile := &config.Profile{APIKey: "sk_test_1234"}
			_, err := GetPluginMetadata(context.Background(), apiServer.URL, dashboardServer.URL, "2020-08-27", "sk_test_1234", profile, "appA", "2.0.1", "darwin", "arm64", true)

			var requestErr RequestError
			require.ErrorAs(t, err, &requestErr)
			require.Equal(t, tt.status, requestErr.StatusCode)
		})
	}
}

// TestGetPluginMetadataReportsExpiredKeyWithoutOptIn covers the lookups that
// must not stand in the anonymous endpoint's view for the account's, such as an
// upgrade picking the latest release.
func TestGetPluginMetadataReportsExpiredKeyWithoutOptIn(t *testing.T) {
	const expiredKey = "rk_test_expired_1234"

	apiServer := httptest.NewServer(http.HandlerFunc(func(res http.ResponseWriter, req *http.Request) {
		res.WriteHeader(http.StatusUnauthorized)
		_, _ = res.Write([]byte(apiKeyExpiredBody))
	}))
	defer apiServer.Close()

	dashboardServer := httptest.NewServer(http.HandlerFunc(func(res http.ResponseWriter, req *http.Request) {
		t.Errorf("unexpected anonymous retry: %s", req.URL.String())
	}))
	defer dashboardServer.Close()

	profile := &config.Profile{APIKey: expiredKey}
	_, err := GetPluginMetadata(context.Background(), apiServer.URL, dashboardServer.URL, "2020-08-27", expiredKey, profile, "appA", "", "darwin", "arm64", false)
	require.True(t, IsAPIKeyExpiredError(err), "want the expired key reported, got: %v", err)
}

// TestGetPluginListReportsExpiredKey covers `stripe plugin list`, where the
// anonymous list would silently leave out plugins only the account can see.
func TestGetPluginListReportsExpiredKey(t *testing.T) {
	const expiredKey = "rk_test_expired_1234"

	apiServer := httptest.NewServer(http.HandlerFunc(func(res http.ResponseWriter, req *http.Request) {
		if req.URL.Path != "/v1/stripecli/list-plugins" {
			t.Errorf("unexpected path: %s", req.URL.Path)
		}
		res.WriteHeader(http.StatusUnauthorized)
		_, _ = res.Write([]byte(apiKeyExpiredBody))
	}))
	defer apiServer.Close()

	dashboardServer := httptest.NewServer(http.HandlerFunc(func(res http.ResponseWriter, req *http.Request) {
		t.Errorf("unexpected anonymous retry: %s", req.URL.String())
	}))
	defer dashboardServer.Close()

	profile := &config.Profile{APIKey: expiredKey}
	_, err := GetPluginList(context.Background(), apiServer.URL, dashboardServer.URL, "2020-08-27", expiredKey, profile, "darwin", "arm64")
	require.True(t, IsAPIKeyExpiredError(err), "want the expired key reported, got: %v", err)
}
