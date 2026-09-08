package plugins

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"runtime"
	"testing"

	"github.com/spf13/afero"
	"github.com/stretchr/testify/require"

	"github.com/stripe/stripe-cli/pkg/version"
)

// withCoreVersion makes the CLI under test report a specific published version.
// Tests need it because a binary built from source reports "master", and the
// version the CLI reports is what these messages are about.
func withCoreVersion(t *testing.T, reportedVersion string) {
	t.Helper()

	previous := version.Version
	version.Version = reportedVersion
	t.Cleanup(func() { version.Version = previous })
}

// pluginWithRelease describes a plugin with a single release for this platform,
// which is enough for a cached lookup to succeed. Tests use it to make sure the
// fallback they are asserting does not happen was actually available.
func pluginWithRelease(pluginVersion string) Plugin {
	return Plugin{
		Shortname:        "appA",
		Binary:           "stripe-cli-app-a",
		MagicCookieValue: "APP-A-COOKIE",
		Releases: []Release{
			{
				Arch:    runtime.GOARCH,
				OS:      runtime.GOOS,
				Version: pluginVersion,
				Sum:     "abc123",
			},
		},
	}
}

// requiresNewerCLIMetadataServer stands in for the metadata endpoint refusing a
// release whose min_core_version this CLI does not meet. The body matches the shape
// the API sends, which is asserted on in pay-server's own tests.
//
// An empty pluginVersion sends the answer to a request that named no version, where
// minCoreVersion is the lowest floor among the plugin's releases. That body names no
// plugin version and blames no parameter, so passing "" here is what keeps a test of
// a versionless path from being handed more than the endpoint would give it.
func requiresNewerCLIMetadataServer(t *testing.T, pluginVersion, minCoreVersion string) *httptest.Server {
	t.Helper()

	return httptest.NewServer(http.HandlerFunc(func(res http.ResponseWriter, req *http.Request) {
		switch req.URL.Path {
		case "/v1/stripecli/get-plugin-metadata", "/ajax/stripecli/plugins_metadata":
			res.Header().Set("Content-Type", "application/json")
			res.WriteHeader(http.StatusBadRequest)

			if pluginVersion == "" {
				_, _ = fmt.Fprintf(res, `{
  "error": {
    "code": "plugin_requires_newer_cli",
    "message": "The appA plugin requires Stripe CLI %s or later. Upgrade the Stripe CLI to install it.",
    "min_core_version": %q,
    "type": "invalid_request_error"
  }
}`, minCoreVersion, minCoreVersion)

				return
			}

			_, _ = fmt.Fprintf(res, `{
  "error": {
    "code": "plugin_requires_newer_cli",
    "message": "Version %s of the appA plugin requires Stripe CLI %s or later. Upgrade the Stripe CLI, or request a plugin version your CLI supports.",
    "min_core_version": %q,
    "param": "version",
    "type": "invalid_request_error"
  }
}`, pluginVersion, minCoreVersion, minCoreVersion)
		default:
			t.Errorf("Received an unexpected request URL: %s", req.URL.String())
		}
	}))
}

func requireRequiresNewerCLI(t *testing.T, err error, wantMinCoreVersion string) {
	t.Helper()

	var requiresNewerCLI *ErrPluginRequiresNewerCLI
	require.ErrorAs(t, err, &requiresNewerCLI)
	require.Equal(t, wantMinCoreVersion, requiresNewerCLI.MinCoreVersion)
}

// TestResolvePluginForInstallReportsEndpointRequiresNewerCLI covers the endpoint's
// answer for a version the caller named. Nothing on this side knows the constraint,
// so reading the answer is the whole feature: without it the install reports `no
// plugin named "appA" exists`, which is what a hidden release's 404 looks like.
func TestResolvePluginForInstallReportsEndpointRequiresNewerCLI(t *testing.T) {
	// Both endpoints, because the version a caller runs is the whole question here and
	// being logged out must not change the answer.
	for _, tt := range []struct {
		name   string
		apiKey string
	}{
		{name: "logged in", apiKey: "sk_test_1234"},
		{name: "not logged in", apiKey: ""},
	} {
		t.Run(tt.name, func(t *testing.T) {
			withCoreVersion(t, "1.29.0")

			fs := afero.NewMemMapFs()
			config := &TestConfig{}
			config.InitConfig()
			config.Profile.APIKey = tt.apiKey

			server := requiresNewerCLIMetadataServer(t, "2.0.1", "1.30.0")
			defer server.Close()

			resolvedPlugin, err := ResolvePluginForInstall(context.Background(), config, fs, "appA", "2.0.1", server.URL, server.URL)
			require.Nil(t, resolvedPlugin)
			requireRequiresNewerCLI(t, err, "1.30.0")

			var requiresNewerCLI *ErrPluginRequiresNewerCLI
			require.ErrorAs(t, err, &requiresNewerCLI)
			require.Equal(t, "2.0.1", requiresNewerCLI.Version)
			require.Equal(t, "appA", requiresNewerCLI.Name)
			require.Contains(t, err.Error(), "Stripe CLI v1.30.0 or newer")
			require.NotContains(t, err.Error(), "no plugin named")
			require.NotContains(t, err.Error(), "cached lookup failed")
		})
	}
}

// TestResolvePluginForInstallEndpointRequiresNewerCLIOutranksCache pins that the
// cached fallback cannot overturn the answer. The cache records no constraint of its
// own, so resolving from it would happily hand back the very version the endpoint
// just refused.
func TestResolvePluginForInstallEndpointRequiresNewerCLIOutranksCache(t *testing.T) {
	withCoreVersion(t, "1.29.0")

	fs := afero.NewMemMapFs()
	config := &TestConfig{}
	config.InitConfig()
	require.NoError(t, writeLocalPluginMetadata(config, fs, pluginWithRelease("2.0.1")))

	server := requiresNewerCLIMetadataServer(t, "2.0.1", "1.30.0")
	defer server.Close()

	resolvedPlugin, err := ResolvePluginForInstall(context.Background(), config, fs, "appA", "2.0.1", server.URL, server.URL)
	require.Nil(t, resolvedPlugin)
	requireRequiresNewerCLI(t, err, "1.30.0")
}

// TestResolvePluginForInstallWithoutAVersionReportsEndpointRequiresNewerCLI covers an
// install that named no version, where every release the plugin has needs a newer CLI
// than this one. The endpoint reports the lowest of their floors, since that is the
// nearest version that would make any of them installable.
//
// It is a separate case from the named-version one because the answer arrives in a
// different shape: no plugin version to attribute it to, and no parameter to blame.
// Reading it still has to yield the minimum, which is the only actionable part.
func TestResolvePluginForInstallWithoutAVersionReportsEndpointRequiresNewerCLI(t *testing.T) {
	withCoreVersion(t, "1.29.0")

	fs := afero.NewMemMapFs()
	config := &TestConfig{}
	config.InitConfig()

	server := requiresNewerCLIMetadataServer(t, "", "1.30.0")
	defer server.Close()

	resolvedPlugin, err := ResolvePluginForInstall(context.Background(), config, fs, "appA", "", server.URL, server.URL)
	require.Nil(t, resolvedPlugin)
	requireRequiresNewerCLI(t, err, "1.30.0")

	// The plugin is still named, and the sentence still reads, without a version for it.
	require.Contains(t, err.Error(), "the appA plugin requires Stripe CLI v1.30.0 or newer")
	require.NotContains(t, err.Error(), "plugin v ")
	require.NotContains(t, err.Error(), "no plugin named")
}

// TestResolvePluginForUpgradeReportsEndpointRequiresNewerCLI and its auto-install
// counterpart cover the same answer reaching the paths that never name a version, and
// pin that the cache is not allowed to contradict it there either.
func TestResolvePluginForUpgradeReportsEndpointRequiresNewerCLI(t *testing.T) {
	withCoreVersion(t, "1.29.0")

	fs := afero.NewMemMapFs()
	config := &TestConfig{}
	config.InitConfig()
	require.NoError(t, writeLocalPluginMetadata(config, fs, pluginWithRelease("2.0.1")))

	server := requiresNewerCLIMetadataServer(t, "", "1.30.0")
	defer server.Close()

	resolvedPlugin, err := ResolvePluginForUpgrade(context.Background(), config, fs, "appA", server.URL, server.URL)
	require.Nil(t, resolvedPlugin)
	requireRequiresNewerCLI(t, err, "1.30.0")
}

func TestResolvePluginForAutoInstallReportsEndpointRequiresNewerCLI(t *testing.T) {
	withCoreVersion(t, "1.29.0")

	fs := afero.NewMemMapFs()
	config := &TestConfig{}
	config.InitConfig()
	require.NoError(t, writeLocalPluginMetadata(config, fs, pluginWithRelease("2.0.1")))

	server := requiresNewerCLIMetadataServer(t, "", "1.30.0")
	defer server.Close()

	resolvedPlugin, err := resolvePluginForAutoInstall(context.Background(), config, fs, "appA", server.URL, server.URL)
	require.Nil(t, resolvedPlugin)
	requireRequiresNewerCLI(t, err, "1.30.0")

	// Not buried in this path's combined "latest lookup failed; cached lookup failed".
	require.NotContains(t, err.Error(), "cached lookup failed")
}

// TestInstallReportsEndpointRequiresNewerCLI covers the install path, which fetches
// metadata again for a version resolved without a binary URL. This is the only thing
// standing between the endpoint's refusal and a download; without it the lookup
// failure surfaces as a missing download URL.
func TestInstallReportsEndpointRequiresNewerCLI(t *testing.T) {
	withCoreVersion(t, "1.29.0")

	fs := afero.NewMemMapFs()
	config := &TestConfig{}
	config.InitConfig()

	server := requiresNewerCLIMetadataServer(t, "2.0.1", "1.30.0")
	defer server.Close()

	plugin := pluginWithRelease("2.0.1")
	err := plugin.Install(context.Background(), config, fs, "2.0.1", server.URL, server.URL)
	requireRequiresNewerCLI(t, err, "1.30.0")
	require.NotContains(t, err.Error(), "could not resolve download URL")

	binaryExists, existsErr := afero.Exists(fs, fmt.Sprintf("/plugins/appA/2.0.1/stripe-cli-app-a%s", GetBinaryExtension()))
	require.NoError(t, existsErr)
	require.False(t, binaryExists)
}

func TestErrPluginRequiresNewerCLIMessageNamesBothVersions(t *testing.T) {
	withCoreVersion(t, "1.29.0")

	err := newErrPluginRequiresNewerCLI("appA", "2.0.1", "1.30.0")
	require.Contains(t, err.Error(), "appA plugin v2.0.1")
	require.Contains(t, err.Error(), "Stripe CLI v1.30.0 or newer")
	require.Contains(t, err.Error(), "this is Stripe CLI 1.29.0")
}

// TestErrPluginRequiresNewerCLIMessageDegradesWithoutAMinimumVersion guards the
// wording for an answer that names the problem without naming a target, which is
// all a response missing the extra attribute can give.
func TestErrPluginRequiresNewerCLIMessageDegradesWithoutAMinimumVersion(t *testing.T) {
	withCoreVersion(t, "1.29.0")

	err := newErrPluginRequiresNewerCLI("appA", "2.0.1", "")
	require.Contains(t, err.Error(), "the appA plugin v2.0.1 requires a newer Stripe CLI")
	require.Contains(t, err.Error(), "this is Stripe CLI 1.29.0")
	require.NotContains(t, err.Error(), "v or newer")
}
