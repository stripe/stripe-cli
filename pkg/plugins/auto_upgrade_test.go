package plugins

import (
	"context"
	"errors"
	"runtime"
	"testing"
	"time"

	"github.com/spf13/afero"
	"github.com/stretchr/testify/require"

	cfgpkg "github.com/stripe/stripe-cli/pkg/config"
	"github.com/stripe/stripe-cli/pkg/stripe"
)

type autoUpgradePostInstallCall struct {
	version          string
	previousVersion  string
	apiBaseURL       string
	dashboardBaseURL string
	accessBaseURL    string
}

type autoUpgradeResolveCall struct {
	pluginName       string
	apiBaseURL       string
	dashboardBaseURL string
	hasDeadline      bool
}

type autoUpgradeInstallCall struct {
	version          string
	apiBaseURL       string
	dashboardBaseURL string
}

// autoUpgradeStubs stands in for everything maybeAutoUpgrade reaches outside the
// package, and records what each seam was asked to do. Tests set the answers, run
// maybeAutoUpgrade, then assert on the calls: "no install recorded" is how a test
// says the upgrade was skipped, rather than inferring it from the return value.
type autoUpgradeStubs struct {
	// Answers.
	updatesEnabled bool
	resolved       *ResolvedPluginVersion
	resolveErr     error
	installErr     error
	// blockUntilCanceled makes the resolver wait for its context instead of
	// answering, so a test can prove the lookup is actually bounded.
	blockUntilCanceled bool

	// Recorded calls.
	settingReads     []string
	resolveCalls     []autoUpgradeResolveCall
	installCalls     []autoUpgradeInstallCall
	postInstallCalls []autoUpgradePostInstallCall
}

func stubAutoUpgrade(t *testing.T) *autoUpgradeStubs {
	t.Helper()

	stubs := &autoUpgradeStubs{updatesEnabled: true}

	origUpdatesEnabled := pluginUpdatesEnabled
	origResolver := autoUpgradeResolver
	origInstaller := autoUpgradeInstaller
	origPostInstall := autoUpgradePostInstall
	origPluginsPath := PluginsPath
	// Every test here runs as a normal, non-local-dev install unless it says otherwise.
	// Left set by another test in this package, it would skip auto-upgrade outright and
	// every assertion below would pass for the wrong reason.
	PluginsPath = ""

	t.Cleanup(func() {
		pluginUpdatesEnabled = origUpdatesEnabled
		autoUpgradeResolver = origResolver
		autoUpgradeInstaller = origInstaller
		autoUpgradePostInstall = origPostInstall
		PluginsPath = origPluginsPath
	})

	pluginUpdatesEnabled = func(pluginName string) bool {
		stubs.settingReads = append(stubs.settingReads, pluginName)
		return stubs.updatesEnabled
	}

	autoUpgradeResolver = func(ctx context.Context, _ cfgpkg.IConfig, _ afero.Fs, pluginName, apiBaseURL, dashboardBaseURL string) (*ResolvedPluginVersion, error) {
		_, hasDeadline := ctx.Deadline()
		stubs.resolveCalls = append(stubs.resolveCalls, autoUpgradeResolveCall{
			pluginName:       pluginName,
			apiBaseURL:       apiBaseURL,
			dashboardBaseURL: dashboardBaseURL,
			hasDeadline:      hasDeadline,
		})

		if stubs.blockUntilCanceled {
			<-ctx.Done()
			return nil, ctx.Err()
		}

		return stubs.resolved, stubs.resolveErr
	}

	autoUpgradeInstaller = func(_ context.Context, resolved *ResolvedPluginVersion, _ cfgpkg.IConfig, _ afero.Fs, apiBaseURL, dashboardBaseURL string) error {
		stubs.installCalls = append(stubs.installCalls, autoUpgradeInstallCall{
			version:          resolved.Version,
			apiBaseURL:       apiBaseURL,
			dashboardBaseURL: dashboardBaseURL,
		})

		return stubs.installErr
	}

	autoUpgradePostInstall = func(_ context.Context, _ *cfgpkg.Config, _ afero.Fs, _ *Plugin, version, previousVersion, apiBaseURL, dashboardBaseURL, accessBaseURL string) {
		stubs.postInstallCalls = append(stubs.postInstallCalls, autoUpgradePostInstallCall{
			version:          version,
			previousVersion:  previousVersion,
			apiBaseURL:       apiBaseURL,
			dashboardBaseURL: dashboardBaseURL,
			accessBaseURL:    accessBaseURL,
		})
	}

	return stubs
}

// autoUpgradeTestPlugin returns a plugin as Run would hold it: the installed
// release's metadata, and nothing about what may exist upstream.
func autoUpgradeTestPlugin(version string) *Plugin {
	return &Plugin{
		Shortname:        "apps",
		Binary:           "stripe-cli-apps",
		MagicCookieValue: "APPS-COOKIE",
		Releases: []Release{
			{Arch: runtime.GOARCH, OS: runtime.GOOS, Version: version, Sum: "installedsum"},
		},
	}
}

func autoUpgradeResolvedPlugin(version string) *ResolvedPluginVersion {
	return &ResolvedPluginVersion{
		Plugin: &Plugin{
			Shortname:        "apps",
			Binary:           "stripe-cli-apps",
			MagicCookieValue: "APPS-COOKIE",
			Releases: []Release{
				{Arch: runtime.GOARCH, OS: runtime.GOOS, Version: version, Sum: "upgradedsum"},
			},
		},
		Version:   version,
		BinaryURL: "https://artifacts.example/apps/" + version,
	}
}

func autoUpgradeTestConfig() *cfgpkg.Config {
	cfg := &TestConfig{}
	cfg.InitConfig()

	return &cfg.Config
}

func TestMaybeAutoUpgradeInstallsNewerRelease(t *testing.T) {
	stubs := stubAutoUpgrade(t)
	stubs.resolved = autoUpgradeResolvedPlugin("1.3.0")

	client := &recordingTelemetryClient{}
	metadata := &stripe.CLIAnalyticsEventMetadata{}
	ctx := stripe.WithEventMetadata(stripe.WithTelemetryClient(context.Background(), client), metadata)

	installed := autoUpgradeTestPlugin("1.2.0")

	var gotPlugin *Plugin
	var gotVersion string
	output := captureStderr(t, func() {
		gotPlugin, gotVersion = maybeAutoUpgrade(ctx, autoUpgradeTestConfig(), afero.NewMemMapFs(), installed, "1.2.0", "", "", "")
	})

	require.Equal(t, "1.3.0", gotVersion)
	// The resolved plugin has to replace the caller's copy, not just the version: the
	// new binary is dispensed against the new release's checksum.
	require.Same(t, stubs.resolved.Plugin, gotPlugin)

	require.Equal(t, []autoUpgradeInstallCall{{
		version:          "1.3.0",
		apiBaseURL:       "https://api.stripe.com",
		dashboardBaseURL: "https://dashboard.stripe.com",
	}}, stubs.installCalls)

	// The previous version is what a plugin's PostInstall hook migrates from, so it has
	// to be the version that was actually running, not the one just installed.
	require.Equal(t, []autoUpgradePostInstallCall{{version: "1.3.0", previousVersion: "1.2.0"}}, stubs.postInstallCalls)

	require.Equal(t, []recordedTelemetryEvent{{name: "Plugin Upgraded", value: "1.3.0"}}, client.events)
	require.Equal(t, "1.3.0", metadata.PluginVersion)

	// The user did not ask for this and is waiting on it, so it has to say what it is
	// doing and how to stop it doing it.
	require.Contains(t, output, "Upgrading the apps plugin to v1.3.0")
	require.Contains(t, output, "stripe plugin auto-update apps --disable")
}

// The base URLs Run holds carry only what the user explicitly passed, because a
// plugin reads an empty one as "use your own default". The lookup and download need
// a real host, so they get the filled-in pair while the plugin still gets the
// user's -- an override here would silently point plugins at the CLI's environment.
func TestMaybeAutoUpgradeBaseURLs(t *testing.T) {
	tests := []struct {
		name             string
		apiBaseURL       string
		dashboardBaseURL string
		accessBaseURL    string
		wantLookupAPI    string
		wantLookupBoard  string
	}{
		{
			name:            "no overrides fall back to this CLI's defaults",
			wantLookupAPI:   "https://api.stripe.com",
			wantLookupBoard: "https://dashboard.stripe.com",
		},
		{
			name:            "an api override carries the dashboard with it",
			apiBaseURL:      "https://qa-api.stripe.com",
			accessBaseURL:   "https://qa-access.stripe.com",
			wantLookupAPI:   "https://qa-api.stripe.com",
			wantLookupBoard: "https://qa-dashboard.stripe.com",
		},
		{
			name:             "explicit overrides are used as given",
			apiBaseURL:       "https://qa-api.stripe.com",
			dashboardBaseURL: "https://custom-dashboard.stripe.com",
			wantLookupAPI:    "https://qa-api.stripe.com",
			wantLookupBoard:  "https://custom-dashboard.stripe.com",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			stubs := stubAutoUpgrade(t)
			stubs.resolved = autoUpgradeResolvedPlugin("1.3.0")

			captureStderr(t, func() {
				maybeAutoUpgrade(context.Background(), autoUpgradeTestConfig(), afero.NewMemMapFs(),
					autoUpgradeTestPlugin("1.2.0"), "1.2.0", tt.apiBaseURL, tt.dashboardBaseURL, tt.accessBaseURL)
			})

			require.Equal(t, []autoUpgradeResolveCall{{
				pluginName:       "apps",
				apiBaseURL:       tt.wantLookupAPI,
				dashboardBaseURL: tt.wantLookupBoard,
				hasDeadline:      true,
			}}, stubs.resolveCalls)

			require.Equal(t, []autoUpgradeInstallCall{{
				version:          "1.3.0",
				apiBaseURL:       tt.wantLookupAPI,
				dashboardBaseURL: tt.wantLookupBoard,
			}}, stubs.installCalls)

			require.Equal(t, []autoUpgradePostInstallCall{{
				version:          "1.3.0",
				previousVersion:  "1.2.0",
				apiBaseURL:       tt.apiBaseURL,
				dashboardBaseURL: tt.dashboardBaseURL,
				accessBaseURL:    tt.accessBaseURL,
			}}, stubs.postInstallCalls)
		})
	}
}

func TestMaybeAutoUpgradeSkips(t *testing.T) {
	tests := []struct {
		name             string
		pluginsPath      string
		installedVersion string
		updatesDisabled  bool
		resolved         *ResolvedPluginVersion
		resolveErr       error
		// wantSettingRead is false for the checks that come before it, which is the
		// point of ordering them that way.
		wantSettingRead bool
		wantLookup      bool
	}{
		{
			// Overwriting a plugin loaded from a local path would throw away the build
			// under development, and the endpoint knows nothing about it anyway.
			name:             "a plugin loaded from a local path",
			pluginsPath:      "/some/local/dev/path",
			installedVersion: "1.2.0",
		},
		{
			// Run's auto-install already handles a missing binary, and resolves the
			// newest release itself while doing so.
			name:             "nothing installed to upgrade",
			installedVersion: "",
		},
		{
			name:             "a locally built version",
			installedVersion: localDevelopmentVersion,
		},
		{
			// The whole cost of the feature to someone who left it off: one config read,
			// no request.
			name:             "updates turned off for the plugin",
			installedVersion: "1.2.0",
			updatesDisabled:  true,
			wantSettingRead:  true,
		},
		{
			name:             "the lookup failed",
			installedVersion: "1.2.0",
			resolveErr:       errors.New("metadata endpoint unreachable"),
			wantSettingRead:  true,
			wantLookup:       true,
		},
		{
			// The endpoint withheld every release because this CLI is too old, so there
			// is nothing to upgrade to. Reporting it would interrupt a command that runs
			// fine on the installed version.
			name:             "no release this CLI is new enough to run",
			installedVersion: "1.2.0",
			resolveErr:       newErrPluginRequiresNewerCLI("apps", "", "3.0.0"),
			wantSettingRead:  true,
			wantLookup:       true,
		},
		{
			name:             "the newest release is the installed one",
			installedVersion: "1.2.0",
			resolved:         autoUpgradeResolvedPlugin("1.2.0"),
			wantSettingRead:  true,
			wantLookup:       true,
		},
		{
			// Load-bearing rather than defensive. The endpoint withholds releases this
			// CLI cannot run, so its newest can be older than what is installed once the
			// CLI itself is downgraded. Anything but a strictly-newer test would roll the
			// plugin back, and would do it again on every command.
			name:             "the newest release is older than the installed one",
			installedVersion: "1.3.0",
			resolved:         autoUpgradeResolvedPlugin("1.2.0"),
			wantSettingRead:  true,
			wantLookup:       true,
		},
		{
			name:             "the lookup answered with nothing",
			installedVersion: "1.2.0",
			resolved:         nil,
			wantSettingRead:  true,
			wantLookup:       true,
		},
		{
			name:             "the lookup answered without a version",
			installedVersion: "1.2.0",
			resolved:         &ResolvedPluginVersion{Plugin: autoUpgradeTestPlugin("1.3.0")},
			wantSettingRead:  true,
			wantLookup:       true,
		},
		{
			name:             "the lookup answered without plugin metadata",
			installedVersion: "1.2.0",
			resolved:         &ResolvedPluginVersion{Version: "1.3.0"},
			wantSettingRead:  true,
			wantLookup:       true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			stubs := stubAutoUpgrade(t)
			stubs.updatesEnabled = !tt.updatesDisabled
			stubs.resolved = tt.resolved
			stubs.resolveErr = tt.resolveErr
			PluginsPath = tt.pluginsPath

			installed := autoUpgradeTestPlugin("1.2.0")

			var gotPlugin *Plugin
			var gotVersion string
			output := captureStderr(t, func() {
				gotPlugin, gotVersion = maybeAutoUpgrade(context.Background(), autoUpgradeTestConfig(), afero.NewMemMapFs(),
					installed, tt.installedVersion, "", "", "")
			})

			require.Same(t, installed, gotPlugin)
			require.Equal(t, tt.installedVersion, gotVersion)

			require.Empty(t, stubs.installCalls)
			require.Empty(t, stubs.postInstallCalls)
			// Nothing was upgraded, so the user's command should look exactly as it would
			// with the feature turned off.
			require.Empty(t, output)

			require.Equal(t, tt.wantSettingRead, len(stubs.settingReads) > 0)
			require.Equal(t, tt.wantLookup, len(stubs.resolveCalls) > 0)
		})
	}
}

// A download that breaks leaves a working plugin behind, so the command the user
// actually ran still has something to run.
func TestMaybeAutoUpgradeKeepsInstalledVersionWhenInstallFails(t *testing.T) {
	stubs := stubAutoUpgrade(t)
	stubs.resolved = autoUpgradeResolvedPlugin("1.3.0")
	stubs.installErr = errors.New("checksum mismatch")

	client := &recordingTelemetryClient{}
	ctx := stripe.WithTelemetryClient(context.Background(), client)
	installed := autoUpgradeTestPlugin("1.2.0")

	var gotPlugin *Plugin
	var gotVersion string
	captureStderr(t, func() {
		gotPlugin, gotVersion = maybeAutoUpgrade(ctx, autoUpgradeTestConfig(), afero.NewMemMapFs(), installed, "1.2.0", "", "", "")
	})

	require.Same(t, installed, gotPlugin)
	require.Equal(t, "1.2.0", gotVersion)

	require.Len(t, stubs.installCalls, 1)
	// Nothing was installed, so nothing should have run the new version's hook or
	// reported an upgrade that did not happen.
	require.Empty(t, stubs.postInstallCalls)
	require.Empty(t, client.events)
}

// The lookup runs before the command the user typed, so a metadata endpoint that
// will not answer has to stop mattering quickly.
func TestMaybeAutoUpgradeBoundsTheLookup(t *testing.T) {
	stubs := stubAutoUpgrade(t)
	stubs.blockUntilCanceled = true

	origTimeout := autoUpgradeResolveTimeout
	autoUpgradeResolveTimeout = 10 * time.Millisecond
	defer func() { autoUpgradeResolveTimeout = origTimeout }()

	installed := autoUpgradeTestPlugin("1.2.0")

	start := time.Now()
	gotPlugin, gotVersion := maybeAutoUpgrade(context.Background(), autoUpgradeTestConfig(), afero.NewMemMapFs(), installed, "1.2.0", "", "", "")
	elapsed := time.Since(start)

	require.Same(t, installed, gotPlugin)
	require.Equal(t, "1.2.0", gotVersion)
	require.Empty(t, stubs.installCalls)
	require.Less(t, elapsed, time.Second, "the lookup should have been abandoned at the timeout")
}

// Callers thread a context down from Cobra, but not every path that reaches a plugin
// has one to thread.
func TestMaybeAutoUpgradeWithNilContext(t *testing.T) {
	stubs := stubAutoUpgrade(t)
	stubs.resolved = autoUpgradeResolvedPlugin("1.3.0")

	var gotVersion string
	captureStderr(t, func() {
		//nolint:staticcheck // a nil context is exactly what this guards against
		_, gotVersion = maybeAutoUpgrade(nil, autoUpgradeTestConfig(), afero.NewMemMapFs(), autoUpgradeTestPlugin("1.2.0"), "1.2.0", "", "", "")
	})

	require.Equal(t, "1.3.0", gotVersion)
	require.Len(t, stubs.resolveCalls, 1)
	require.True(t, stubs.resolveCalls[0].hasDeadline)
}
