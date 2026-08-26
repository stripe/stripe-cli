package plugins

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"runtime"
	"testing"

	"github.com/spf13/afero"
	"github.com/stretchr/testify/require"

	"github.com/stripe/stripe-cli/pkg/requests"
	"github.com/stripe/stripe-cli/pkg/version"
)

// withCoreVersion makes the CLI under test report a specific published version.
// Tests need it because a binary built from source reports "master", which
// satisfies every MinCoreVersion by design.
func withCoreVersion(t *testing.T, reportedVersion string) {
	t.Helper()

	previous := version.Version
	version.Version = reportedVersion
	t.Cleanup(func() { version.Version = previous })
}

// restrictedPlugin describes a plugin whose only release for this platform needs
// a newer core CLI than any test here pretends to be.
func restrictedPlugin(pluginVersion, minCoreVersion string) Plugin {
	return Plugin{
		Shortname:        "appA",
		Binary:           "stripe-cli-app-a",
		MagicCookieValue: "APP-A-COOKIE",
		Releases: []Release{
			{
				Arch:           runtime.GOARCH,
				OS:             runtime.GOOS,
				Version:        pluginVersion,
				Sum:            "abc123",
				MinCoreVersion: minCoreVersion,
			},
		},
	}
}

// failingMetadataServer stands in for the metadata endpoint hiding a release from
// a core CLI too old for it, which is what sends the install down the cached path.
func failingMetadataServer(t *testing.T) *httptest.Server {
	t.Helper()

	return httptest.NewServer(http.HandlerFunc(func(res http.ResponseWriter, req *http.Request) {
		switch req.URL.Path {
		case "/v1/stripecli/get-plugin-metadata", "/ajax/stripecli/plugins_metadata":
			res.WriteHeader(http.StatusNotFound)
			_, _ = res.Write([]byte(`{"error":{"message":"not found"}}`))
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

func TestCoreVersionSupports(t *testing.T) {
	tests := []struct {
		name           string
		coreVersion    string
		minCoreVersion string
		supported      bool
	}{
		{name: "no constraint", coreVersion: "1.20.0", minCoreVersion: "", supported: true},
		{name: "no constraint tolerates an odd core version", coreVersion: "not-a-version", minCoreVersion: "", supported: true},
		{name: "newer core version", coreVersion: "1.31.0", minCoreVersion: "1.30.0", supported: true},
		{name: "same core version", coreVersion: "1.30.0", minCoreVersion: "1.30.0", supported: true},
		{name: "older patch", coreVersion: "1.30.0", minCoreVersion: "1.30.1", supported: false},
		// Compared as semver, not as strings: "1.9.0" sorts after "1.10.0" lexically.
		{name: "older minor", coreVersion: "1.9.0", minCoreVersion: "1.10.0", supported: false},
		{name: "development build satisfies anything", coreVersion: "master", minCoreVersion: "99.0.0", supported: true},
		{name: "leading v on the core version", coreVersion: "v1.30.0", minCoreVersion: "1.30.0", supported: true},
		{name: "leading v on the constraint", coreVersion: "1.30.0", minCoreVersion: "v1.30.0", supported: true},
		// Fails closed: an unevaluable constraint is exactly when installing anyway
		// risks a binary this CLI cannot drive.
		{name: "prerelease core version", coreVersion: "1.30.0-beta.1", minCoreVersion: "1.29.0", supported: false},
		{name: "unparseable core version", coreVersion: "not-a-version", minCoreVersion: "1.30.0", supported: false},
		{name: "unparseable constraint", coreVersion: "1.30.0", minCoreVersion: "latest", supported: false},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			withCoreVersion(t, test.coreVersion)
			require.Equal(t, test.supported, coreVersionSupports(test.minCoreVersion))
		})
	}
}

func TestMinCoreVersionParsedFromManifest(t *testing.T) {
	manifest := fmt.Sprintf(`[[Plugin]]
  Shortname = "appA"
  Binary = "stripe-cli-app-a"
  MagicCookieValue = "APP-A-COOKIE"

  [[Plugin.Release]]
    Arch = "%s"
    OS = "%s"
    Version = "2.0.1"
    Sum = "abc123"
    MinCoreVersion = "1.30.0"
`, runtime.GOARCH, runtime.GOOS)

	pluginList, err := validatePluginManifest([]byte(manifest))
	require.NoError(t, err)
	plugin, err := findPlugin(*pluginList, "appA")
	require.NoError(t, err)

	release := plugin.getReleaseForVersion("2.0.1")
	require.NotNil(t, release)
	require.Equal(t, "1.30.0", release.MinCoreVersion)
}

// TestMinCoreVersionPreservedInLocalMetadataRoundTrip is what makes offline
// enforcement possible at all: local metadata is re-encoded from the Release
// struct, so a field that does not survive the round trip is not there to enforce
// when the endpoint is unreachable.
func TestMinCoreVersionPreservedInLocalMetadataRoundTrip(t *testing.T) {
	fs := afero.NewMemMapFs()
	config := &TestConfig{}
	config.InitConfig()

	require.NoError(t, writeLocalPluginMetadata(config, fs, restrictedPlugin("2.0.1", "1.30.0")))

	cached, err := readLocalPluginMetadata(config, fs, "appA")
	require.NoError(t, err)
	release := cached.getReleaseForVersion("2.0.1")
	require.NotNil(t, release)
	require.Equal(t, "1.30.0", release.MinCoreVersion)
}

func TestLookUpLatestVersionSkipsReleasesRequiringNewerCoreCLI(t *testing.T) {
	plugin := Plugin{
		Shortname: "appA",
		Releases: []Release{
			{Arch: runtime.GOARCH, OS: runtime.GOOS, Version: "1.0.1"},
			{Arch: runtime.GOARCH, OS: runtime.GOOS, Version: "2.0.1", MinCoreVersion: "1.30.0"},
		},
	}

	t.Run("core CLI too old falls back to the newest release it can run", func(t *testing.T) {
		withCoreVersion(t, "1.29.0")
		require.Equal(t, "1.0.1", plugin.LookUpLatestVersion())
	})

	t.Run("core CLI new enough gets the newest release", func(t *testing.T) {
		withCoreVersion(t, "1.30.0")
		require.Equal(t, "2.0.1", plugin.LookUpLatestVersion())
	})

	t.Run("ignoring the constraint reports the release that needs a newer CLI", func(t *testing.T) {
		withCoreVersion(t, "1.29.0")
		require.Equal(t, "2.0.1", plugin.lookUpLatestVersion(false))
	})
}

// TestLookUpLatestVersionReturnsEmptyWhenEveryReleaseRequiresNewerCoreCLI covers
// the case the callers have to tell apart from a plugin with no releases at all.
func TestLookUpLatestVersionReturnsEmptyWhenEveryReleaseRequiresNewerCoreCLI(t *testing.T) {
	withCoreVersion(t, "1.29.0")
	plugin := restrictedPlugin("2.0.1", "1.30.0")

	require.Empty(t, plugin.LookUpLatestVersion())
	require.NoError(t, plugin.checkCoreVersionForRelease(""))
	requireRequiresNewerCLI(t, plugin.checkCoreVersionForLatestRelease(), "1.30.0")
}

// TestCheckCoreVersionForReleaseSkipsLocalDevelopmentVersion covers a locally
// built plugin, which is not something the registry published a constraint for.
func TestCheckCoreVersionForReleaseSkipsLocalDevelopmentVersion(t *testing.T) {
	withCoreVersion(t, "1.29.0")
	plugin := restrictedPlugin(localDevelopmentVersion, "1.30.0")

	require.NoError(t, plugin.checkCoreVersionForRelease(localDevelopmentVersion))
}

func TestErrPluginRequiresNewerCLIMessageNamesBothVersions(t *testing.T) {
	withCoreVersion(t, "1.29.0")

	plugin := restrictedPlugin("2.0.1", "1.30.0")
	err := plugin.checkCoreVersionForRelease("2.0.1")
	require.Error(t, err)
	require.Contains(t, err.Error(), "appA plugin v2.0.1")
	require.Contains(t, err.Error(), "Stripe CLI v1.30.0 or newer")
	require.Contains(t, err.Error(), "1.29.0")
}

// TestInstallRefusesReleaseRequiringNewerCoreCLI checks the last line of defense.
// The endpoint served the release here, so this is the case where filtering by
// User-Agent did not happen or did not survive the request path.
func TestInstallRefusesReleaseRequiringNewerCoreCLI(t *testing.T) {
	withCoreVersion(t, "1.29.0")

	fs := afero.NewMemMapFs()
	config := &TestConfig{}
	config.InitConfig()

	metadataManifest := fmt.Sprintf(`[[Plugin]]
  Shortname = "appA"
  Binary = "stripe-cli-app-a"
  MagicCookieValue = "APP-A-COOKIE"

  [[Plugin.Release]]
    Arch = "%s"
    OS = "%s"
    Version = "2.0.1"
    Sum = "abc123"
    MinCoreVersion = "1.30.0"
`, runtime.GOARCH, runtime.GOOS)

	stripeServer := httptest.NewServer(http.HandlerFunc(func(res http.ResponseWriter, req *http.Request) {
		switch req.URL.Path {
		case "/v1/stripecli/get-plugin-metadata":
			body, err := json.Marshal(requests.PluginMetadata{
				BinaryURL:      "https://example.test/appA/2.0.1",
				PluginManifest: metadataManifest,
			})
			require.NoError(t, err)
			_, _ = res.Write(body)
		default:
			t.Errorf("Received an unexpected request URL: %s", req.URL.String())
		}
	}))
	defer stripeServer.Close()

	plugin := restrictedPlugin("2.0.1", "1.30.0")
	err := plugin.Install(context.Background(), config, fs, "2.0.1", stripeServer.URL, stripeServer.URL)
	requireRequiresNewerCLI(t, err, "1.30.0")

	// Refused before anything was downloaded or recorded.
	binaryExists, existsErr := afero.Exists(fs, fmt.Sprintf("/plugins/appA/2.0.1/stripe-cli-app-a%s", GetBinaryExtension()))
	require.NoError(t, existsErr)
	require.False(t, binaryExists)
}

func TestResolvePluginForInstallRejectsExplicitVersionRequiringNewerCoreCLI(t *testing.T) {
	withCoreVersion(t, "1.29.0")

	fs := afero.NewMemMapFs()
	config := &TestConfig{}
	config.InitConfig()
	require.NoError(t, writeLocalPluginMetadata(config, fs, restrictedPlugin("2.0.1", "1.30.0")))

	server := failingMetadataServer(t)
	defer server.Close()

	resolvedPlugin, err := ResolvePluginForInstall(context.Background(), config, fs, "appA", "2.0.1", server.URL, server.URL)
	require.Nil(t, resolvedPlugin)
	requireRequiresNewerCLI(t, err, "1.30.0")
}

// TestResolvePluginForInstallReportsRequiresNewerCoreCLIForLatest guards the
// message rather than the refusal: with every release filtered out, the resolve
// would otherwise report the plugin as having no release for this platform.
func TestResolvePluginForInstallReportsRequiresNewerCoreCLIForLatest(t *testing.T) {
	withCoreVersion(t, "1.29.0")

	fs := afero.NewMemMapFs()
	config := &TestConfig{}
	config.InitConfig()
	require.NoError(t, writeLocalPluginMetadata(config, fs, restrictedPlugin("2.0.1", "1.30.0")))

	server := failingMetadataServer(t)
	defer server.Close()

	resolvedPlugin, err := ResolvePluginForInstall(context.Background(), config, fs, "appA", "", server.URL, server.URL)
	require.Nil(t, resolvedPlugin)
	requireRequiresNewerCLI(t, err, "1.30.0")
}

func TestResolvePluginForInstallAllowsCachedReleaseThisCoreCLISupports(t *testing.T) {
	withCoreVersion(t, "1.30.0")

	fs := afero.NewMemMapFs()
	config := &TestConfig{}
	config.InitConfig()
	require.NoError(t, writeLocalPluginMetadata(config, fs, restrictedPlugin("2.0.1", "1.30.0")))

	server := failingMetadataServer(t)
	defer server.Close()

	resolvedPlugin, err := ResolvePluginForInstall(context.Background(), config, fs, "appA", "2.0.1", server.URL, server.URL)
	require.NoError(t, err)
	require.Equal(t, "2.0.1", resolvedPlugin.Version)
}

// TestResolvePluginForInstallSkipsRestrictedReleaseWhenAnotherIsSupported keeps
// the constraint from reading as "this plugin is unavailable" when an older
// release is still installable.
func TestResolvePluginForInstallSkipsRestrictedReleaseWhenAnotherIsSupported(t *testing.T) {
	withCoreVersion(t, "1.29.0")

	fs := afero.NewMemMapFs()
	config := &TestConfig{}
	config.InitConfig()
	require.NoError(t, writeLocalPluginMetadata(config, fs, Plugin{
		Shortname:        "appA",
		Binary:           "stripe-cli-app-a",
		MagicCookieValue: "APP-A-COOKIE",
		Releases: []Release{
			{Arch: runtime.GOARCH, OS: runtime.GOOS, Version: "1.0.1", Sum: "abc123"},
			{Arch: runtime.GOARCH, OS: runtime.GOOS, Version: "2.0.1", Sum: "def456", MinCoreVersion: "1.30.0"},
		},
	}))

	server := failingMetadataServer(t)
	defer server.Close()

	resolvedPlugin, err := ResolvePluginForInstall(context.Background(), config, fs, "appA", "", server.URL, server.URL)
	require.NoError(t, err)
	require.Equal(t, "1.0.1", resolvedPlugin.Version)
}

// TestResolvePluginForInstallRejectsRestrictedReleaseFromLiveMetadata covers the
// endpoint answering with a release it should have filtered out. The cache here
// holds the same release without the constraint -- which is how a cache written
// before the field existed looks -- so this also pins that the fallback cannot
// overturn a definitive answer.
func TestResolvePluginForInstallRejectsRestrictedReleaseFromLiveMetadata(t *testing.T) {
	withCoreVersion(t, "1.29.0")

	fs := afero.NewMemMapFs()
	config := &TestConfig{}
	config.InitConfig()
	require.NoError(t, writeLocalPluginMetadata(config, fs, restrictedPlugin("2.0.1", "")))

	metadataManifest := fmt.Sprintf(`[[Plugin]]
  Shortname = "appA"
  Binary = "stripe-cli-app-a"
  MagicCookieValue = "APP-A-COOKIE"

  [[Plugin.Release]]
    Arch = "%s"
    OS = "%s"
    Version = "2.0.1"
    Sum = "abc123"
    MinCoreVersion = "1.30.0"
`, runtime.GOARCH, runtime.GOOS)

	server := httptest.NewServer(http.HandlerFunc(func(res http.ResponseWriter, req *http.Request) {
		switch req.URL.Path {
		case "/v1/stripecli/get-plugin-metadata":
			body, err := json.Marshal(requests.PluginMetadata{
				BinaryURL:      "https://example.test/appA/2.0.1",
				PluginManifest: metadataManifest,
			})
			require.NoError(t, err)
			_, _ = res.Write(body)
		default:
			t.Errorf("Received an unexpected request URL: %s", req.URL.String())
		}
	}))
	defer server.Close()

	resolvedPlugin, err := ResolvePluginForInstall(context.Background(), config, fs, "appA", "2.0.1", server.URL, server.URL)
	require.Nil(t, resolvedPlugin)
	requireRequiresNewerCLI(t, err, "1.30.0")
}

func TestResolvePluginForUpgradeReportsRequiresNewerCoreCLI(t *testing.T) {
	withCoreVersion(t, "1.29.0")

	fs := afero.NewMemMapFs()
	config := &TestConfig{}
	config.InitConfig()
	require.NoError(t, writeLocalPluginMetadata(config, fs, restrictedPlugin("2.0.1", "1.30.0")))

	server := failingMetadataServer(t)
	defer server.Close()

	resolvedPlugin, err := ResolvePluginForUpgrade(context.Background(), config, fs, "appA", server.URL, server.URL)
	require.Nil(t, resolvedPlugin)
	requireRequiresNewerCLI(t, err, "1.30.0")
}

// TestResolvePluginForAutoInstallReportsRequiresNewerCoreCLI checks that the
// version the user needs is not buried in the auto-install path's combined
// "latest lookup failed; cached lookup failed" wrapper.
func TestResolvePluginForAutoInstallReportsRequiresNewerCoreCLI(t *testing.T) {
	withCoreVersion(t, "1.29.0")

	fs := afero.NewMemMapFs()
	config := &TestConfig{}
	config.InitConfig()
	require.NoError(t, writeLocalPluginMetadata(config, fs, restrictedPlugin("2.0.1", "1.30.0")))

	server := failingMetadataServer(t)
	defer server.Close()

	resolvedPlugin, err := resolvePluginForAutoInstall(context.Background(), config, fs, "appA", server.URL, server.URL)
	require.Nil(t, resolvedPlugin)
	requireRequiresNewerCLI(t, err, "1.30.0")
	require.NotContains(t, err.Error(), "cached lookup failed")

	var requiresNewerCLI *ErrPluginRequiresNewerCLI
	require.True(t, errors.As(err, &requiresNewerCLI))
}
