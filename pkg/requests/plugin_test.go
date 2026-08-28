package requests

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

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

	_, err := GetPluginMetadata(context.Background(), server.URL, server.URL, "2020-08-27", "sk_test_1234", profile, "appA", "2.0.1", "darwin", "arm64", "uuid")
	require.Error(t, err)

	minCoreVersion, ok := PluginRequiresNewerCLI(err)
	require.True(t, ok)
	require.Equal(t, "1.30.0", minCoreVersion)
}
