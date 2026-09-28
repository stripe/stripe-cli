package plugins

import (
	"context"
	"errors"
	"path/filepath"
	"runtime"
	"strings"
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
	// now is what maybeAutoUpgrade reads the clock as, pinned so a test can place a
	// check stamp at an exact age instead of depending on the wall clock.
	now time.Time

	// Recorded calls.
	settingReads     []string
	resolveCalls     []autoUpgradeResolveCall
	installCalls     []autoUpgradeInstallCall
	postInstallCalls []autoUpgradePostInstallCall
}

func stubAutoUpgrade(t *testing.T) *autoUpgradeStubs {
	t.Helper()

	stubs := &autoUpgradeStubs{
		updatesEnabled: true,
		now:            time.Date(2026, 4, 1, 12, 0, 0, 0, time.UTC),
	}

	origUpdatesEnabled := pluginUpdatesEnabled
	origResolver := autoUpgradeResolver
	origInstaller := autoUpgradeInstaller
	origPostInstall := autoUpgradePostInstall
	origNow := autoUpgradeNow
	origPluginsPath := PluginsPath
	// Every test here runs as a normal, non-local-dev install unless it says otherwise.
	// Left set by another test in this package, either of these would skip auto-upgrade
	// outright and every assertion below would pass for the wrong reason. Both spellings,
	// because the guard now asks about both.
	PluginsPath = ""
	t.Setenv("STRIPE_PLUGINS_PATH", "")

	t.Cleanup(func() {
		pluginUpdatesEnabled = origUpdatesEnabled
		autoUpgradeResolver = origResolver
		autoUpgradeInstaller = origInstaller
		autoUpgradePostInstall = origPostInstall
		autoUpgradeNow = origNow
		PluginsPath = origPluginsPath
	})

	autoUpgradeNow = func() time.Time { return stubs.now }

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

// autoUpgradeCachedResolution mirrors what ResolvePluginForUpgrade returns once the
// metadata endpoint has failed and it falls back to cached metadata: full plugin
// metadata and a version, but no binary URL, because only a live response carries
// one.
func autoUpgradeCachedResolution(version string) *ResolvedPluginVersion {
	resolved := autoUpgradeResolvedPlugin(version)
	resolved.BinaryURL = ""

	return resolved
}

func autoUpgradeTestConfig() *cfgpkg.Config {
	cfg := &TestConfig{}
	cfg.InitConfig()

	return &cfg.Config
}

// writeAutoUpgradeCheckStamp records a check as having happened at the given time, in
// the format maybeAutoUpgrade writes rather than by calling the writer, so a test that
// breaks the reader cannot be rescued by a matching break in the writer.
func writeAutoUpgradeCheckStamp(t *testing.T, cfg cfgpkg.IConfig, fs afero.Fs, pluginName string, at time.Time) {
	t.Helper()

	path, err := autoUpgradeCheckStampPath(cfg, pluginName)
	require.NoError(t, err)
	require.NoError(t, fs.MkdirAll(filepath.Dir(path), 0755))
	require.NoError(t, afero.WriteFile(fs, path, []byte(at.UTC().Format(time.RFC3339)+"\n"), 0644))
}

// readAutoUpgradeCheckStamp returns the recorded check time, and fails the test if
// there is no stamp to read.
func readAutoUpgradeCheckStamp(t *testing.T, cfg cfgpkg.IConfig, fs afero.Fs, pluginName string) time.Time {
	t.Helper()

	path, err := autoUpgradeCheckStampPath(cfg, pluginName)
	require.NoError(t, err)
	body, err := afero.ReadFile(fs, path)
	require.NoError(t, err)

	at, err := time.Parse(time.RFC3339, strings.TrimSpace(string(body)))
	require.NoError(t, err)

	return at
}

func autoUpgradeCheckStampExists(t *testing.T, cfg cfgpkg.IConfig, fs afero.Fs, pluginName string) bool {
	t.Helper()

	path, err := autoUpgradeCheckStampPath(cfg, pluginName)
	require.NoError(t, err)
	exists, err := afero.Exists(fs, path)
	require.NoError(t, err)

	return exists
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

	// The user did not ask for this, so it has to say what it did and how to stop it
	// doing it again.
	require.Contains(t, output, "Updated the apps plugin to v1.3.0")
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
		pluginsPathEnv   string
		installedVersion string
		updatesDisabled  bool
		resolved         *ResolvedPluginVersion
		resolveErr       error
		// lastCheckedAgo places a check stamp that far in the past. Zero leaves the
		// plugin unstamped, which is how every case but the throttle one runs.
		lastCheckedAgo time.Duration
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
			// The same thing said the other way. A plugin developer is far more likely to
			// point the CLI at their build with this than to compile a path into it, and
			// the check used to miss them entirely -- installing over the directory, and
			// deleting every other version in it on the way out.
			name:             "a plugin loaded from a local path set in the environment",
			pluginsPathEnv:   "/some/local/dev/path",
			installedVersion: "1.2.0",
		},
		{
			// Whichever way it is set, before the setting is read: it costs nothing, and
			// a developer who once turned updates on for a plugin they now have a build of
			// should not have that decision reach it.
			name:             "a local path with updates turned on",
			pluginsPathEnv:   "/some/local/dev/path",
			installedVersion: "1.2.0",
			resolved:         autoUpgradeResolvedPlugin("1.3.0"),
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
			// The other half of what the feature costs someone who left it on: one
			// config read and one stat, for all but the first command in a few hours.
			name:             "checked for an upgrade recently",
			installedVersion: "1.2.0",
			lastCheckedAgo:   autoUpgradeCheckInterval - time.Minute,
			resolved:         autoUpgradeResolvedPlugin("1.3.0"),
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
		{
			// What ResolvePluginForUpgrade returns when it falls back to cached metadata:
			// a newer version, and no binary URL. Installing it would send install back
			// to the endpoint that just missed the deadline, this time without one, so
			// the budget above has to require the URL to mean anything.
			name:             "the newest release came back without a binary URL",
			installedVersion: "1.2.0",
			resolved:         autoUpgradeCachedResolution("1.3.0"),
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
			if tt.pluginsPathEnv != "" {
				t.Setenv("STRIPE_PLUGINS_PATH", tt.pluginsPathEnv)
			}

			cfg := autoUpgradeTestConfig()
			fs := afero.NewMemMapFs()
			if tt.lastCheckedAgo != 0 {
				writeAutoUpgradeCheckStamp(t, cfg, fs, "apps", stubs.now.Add(-tt.lastCheckedAgo))
			}

			installed := autoUpgradeTestPlugin("1.2.0")

			var gotPlugin *Plugin
			var gotVersion string
			output := captureStderr(t, func() {
				gotPlugin, gotVersion = maybeAutoUpgrade(context.Background(), cfg, fs,
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

			// A check that spent a request is stamped however it turned out; one that
			// bailed before the lookup leaves the next command free to make it.
			require.Equal(t, tt.wantLookup || tt.lastCheckedAgo != 0,
				autoUpgradeCheckStampExists(t, cfg, fs, "apps"))
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
	output := captureStderr(t, func() {
		gotPlugin, gotVersion = maybeAutoUpgrade(ctx, autoUpgradeTestConfig(), afero.NewMemMapFs(), installed, "1.2.0", "", "", "")
	})

	require.Same(t, installed, gotPlugin)
	require.Equal(t, "1.2.0", gotVersion)

	require.Len(t, stubs.installCalls, 1)
	// Nothing was installed, so nothing should have run the new version's hook or
	// reported an upgrade that did not happen. install says what went wrong itself.
	require.Empty(t, stubs.postInstallCalls)
	require.Empty(t, client.events)
	require.NotContains(t, output, "Updated the apps plugin")
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

// The throttle is meant to be forgotten about: once its interval is up, the check
// happens exactly as it would have without one.
func TestMaybeAutoUpgradeChecksAgainOnceTheIntervalIsUp(t *testing.T) {
	stubs := stubAutoUpgrade(t)
	stubs.resolved = autoUpgradeResolvedPlugin("1.3.0")

	cfg := autoUpgradeTestConfig()
	fs := afero.NewMemMapFs()
	lastCheck := stubs.now.Add(-autoUpgradeCheckInterval)
	writeAutoUpgradeCheckStamp(t, cfg, fs, "apps", lastCheck)

	var gotVersion string
	captureStderr(t, func() {
		_, gotVersion = maybeAutoUpgrade(context.Background(), cfg, fs, autoUpgradeTestPlugin("1.2.0"), "1.2.0", "", "", "")
	})

	require.Equal(t, "1.3.0", gotVersion)
	require.Len(t, stubs.resolveCalls, 1)

	// Moved forward, not left at the previous check. A stamp that never advanced would
	// leave the plugin permanently due and the throttle would do nothing at all.
	require.Equal(t, stubs.now, readAutoUpgradeCheckStamp(t, cfg, fs, "apps").UTC())
}

// The case the throttle exists for. Without it, a machine that cannot reach the
// metadata endpoint pays the full lookup timeout in front of every plugin command.
func TestMaybeAutoUpgradeThrottlesAFailedCheck(t *testing.T) {
	stubs := stubAutoUpgrade(t)
	stubs.resolveErr = errors.New("metadata endpoint unreachable")

	cfg := autoUpgradeTestConfig()
	fs := afero.NewMemMapFs()

	captureStderr(t, func() {
		for range 3 {
			maybeAutoUpgrade(context.Background(), cfg, fs, autoUpgradeTestPlugin("1.2.0"), "1.2.0", "", "", "")
		}
	})

	require.Len(t, stubs.resolveCalls, 1, "a failed check should be throttled like any other")
	require.Equal(t, stubs.now, readAutoUpgradeCheckStamp(t, cfg, fs, "apps").UTC())
}

// A successful upgrade is throttled the same way, so that the command right after one
// does not go straight back to the endpoint to be told it is up to date.
func TestMaybeAutoUpgradeThrottlesAfterUpgrading(t *testing.T) {
	stubs := stubAutoUpgrade(t)
	stubs.resolved = autoUpgradeResolvedPlugin("1.3.0")

	cfg := autoUpgradeTestConfig()
	fs := afero.NewMemMapFs()

	var secondVersion string
	captureStderr(t, func() {
		maybeAutoUpgrade(context.Background(), cfg, fs, autoUpgradeTestPlugin("1.2.0"), "1.2.0", "", "", "")
		// As Run would call it next time: the upgraded version is what is installed now.
		_, secondVersion = maybeAutoUpgrade(context.Background(), cfg, fs, stubs.resolved.Plugin, "1.3.0", "", "", "")
	})

	require.Len(t, stubs.resolveCalls, 1)
	require.Len(t, stubs.installCalls, 1)
	require.Equal(t, "1.3.0", secondVersion)
}

// The stamp is state on disk that anything could have written, so every way of
// reading it wrong has to fall the same way: check, rather than never check again.
func TestMaybeAutoUpgradeChecksWhenTheStampIsUnusable(t *testing.T) {
	tests := []struct {
		name string
		body string
	}{
		{name: "empty", body: ""},
		{name: "not a timestamp", body: "yesterday\n"},
		{name: "truncated", body: "2026-04-0"},
		// A clock corrected backwards. Waiting for the recorded time to arrive could
		// park the check for as long as the correction was large.
		{name: "in the future", body: "2027-04-01T12:00:00Z\n"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			stubs := stubAutoUpgrade(t)
			stubs.resolved = autoUpgradeResolvedPlugin("1.3.0")

			cfg := autoUpgradeTestConfig()
			fs := afero.NewMemMapFs()
			path, err := autoUpgradeCheckStampPath(cfg, "apps")
			require.NoError(t, err)
			require.NoError(t, fs.MkdirAll(filepath.Dir(path), 0755))
			require.NoError(t, afero.WriteFile(fs, path, []byte(tt.body), 0644))

			var gotVersion string
			captureStderr(t, func() {
				_, gotVersion = maybeAutoUpgrade(context.Background(), cfg, fs, autoUpgradeTestPlugin("1.2.0"), "1.2.0", "", "", "")
			})

			require.Equal(t, "1.3.0", gotVersion)
			require.Len(t, stubs.resolveCalls, 1)
			// Overwritten with something readable, so the throttle works from here on.
			require.Equal(t, stubs.now, readAutoUpgradeCheckStamp(t, cfg, fs, "apps").UTC())
		})
	}
}

// A stamp that cannot be written is the throttle failing, not the upgrade failing.
func TestMaybeAutoUpgradeUpgradesWhenTheStampCannotBeWritten(t *testing.T) {
	stubs := stubAutoUpgrade(t)
	stubs.resolved = autoUpgradeResolvedPlugin("1.3.0")

	cfg := autoUpgradeTestConfig()
	fs := afero.NewReadOnlyFs(afero.NewMemMapFs())

	var gotVersion string
	captureStderr(t, func() {
		_, gotVersion = maybeAutoUpgrade(context.Background(), cfg, fs, autoUpgradeTestPlugin("1.2.0"), "1.2.0", "", "", "")
	})

	require.Equal(t, "1.3.0", gotVersion)
	require.Len(t, stubs.installCalls, 1)
}

// The setting is per-plugin, so the throttle has to be too: one plugin's check must
// not stand in for another's.
func TestMaybeAutoUpgradeThrottlesEachPluginSeparately(t *testing.T) {
	stubs := stubAutoUpgrade(t)
	stubs.resolved = autoUpgradeResolvedPlugin("1.3.0")

	cfg := autoUpgradeTestConfig()
	fs := afero.NewMemMapFs()
	writeAutoUpgradeCheckStamp(t, cfg, fs, "apps", stubs.now)

	other := autoUpgradeTestPlugin("1.2.0")
	other.Shortname = "projects"

	captureStderr(t, func() {
		maybeAutoUpgrade(context.Background(), cfg, fs, autoUpgradeTestPlugin("1.2.0"), "1.2.0", "", "", "")
		maybeAutoUpgrade(context.Background(), cfg, fs, other, "1.2.0", "", "", "")
	})

	require.Len(t, stubs.resolveCalls, 1)
	require.Equal(t, "projects", stubs.resolveCalls[0].pluginName)
	require.True(t, autoUpgradeCheckStampExists(t, cfg, fs, "projects"))
}

// The interval has to be claimed before the request goes out rather than when it comes
// back. The gap between the two is what a concurrently starting CLI slips through, and
// it is as wide as the lookup -- up to the whole resolve timeout.
func TestMaybeAutoUpgradeClaimsTheIntervalBeforeLookingUp(t *testing.T) {
	stubs := stubAutoUpgrade(t)
	stubs.resolved = autoUpgradeResolvedPlugin("1.3.0")

	cfg := autoUpgradeTestConfig()
	fs := afero.NewMemMapFs()

	// Wrapping the stub rather than replacing it keeps its call recording intact.
	// stubAutoUpgrade's cleanup restores this along with everything else.
	recording := autoUpgradeResolver
	var claimedDuringLookup bool
	autoUpgradeResolver = func(ctx context.Context, c cfgpkg.IConfig, f afero.Fs, pluginName, apiBaseURL, dashboardBaseURL string) (*ResolvedPluginVersion, error) {
		claimedDuringLookup = autoUpgradeCheckStampExists(t, cfg, fs, pluginName)
		return recording(ctx, c, f, pluginName, apiBaseURL, dashboardBaseURL)
	}

	var gotVersion string
	captureStderr(t, func() {
		_, gotVersion = maybeAutoUpgrade(context.Background(), cfg, fs, autoUpgradeTestPlugin("1.2.0"), "1.2.0", "", "", "")
	})

	require.Len(t, stubs.resolveCalls, 1)
	require.Equal(t, "1.3.0", gotVersion, "claiming the interval must not cost the upgrade")
	require.True(t, claimedDuringLookup,
		"a second CLI starting while this lookup was in flight would have made the same request")
}
