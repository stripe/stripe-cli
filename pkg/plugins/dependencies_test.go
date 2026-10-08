package plugins

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/BurntSushi/toml"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/require"

	"github.com/stripe/stripe-cli/pkg/config"
	"github.com/stripe/stripe-cli/pkg/errorcategory"
	"github.com/stripe/stripe-cli/pkg/requests"
	"github.com/stripe/stripe-cli/pkg/stripe"
)

func TestPluginVersionSatisfiesMinimum(t *testing.T) {
	cases := []struct {
		name             string
		installedVersion string
		minimum          string
		satisfied        bool
	}{
		{"no minimum declared", "1.0.0", "", true},
		{"not installed", "", "1.17.0", false},
		{"below the minimum", "1.16.2", "1.17.0", false},
		{"exactly the minimum", "1.17.0", "1.17.0", true},
		{"above the minimum: a floor, not a pin", "2.3.0", "1.17.0", true},
		{"local development build", localDevelopmentVersion, "99.0.0", true},
		{"unparseable installed version is a developer's build", "my-own-build", "1.17.0", true},
		{"unparseable minimum cannot be enforced", "1.0.0", "not-a-version", true},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			require.Equal(t, tc.satisfied, pluginVersionSatisfiesMinimum(tc.installedVersion, tc.minimum))
		})
	}
}

// dependencyTestRelease is one downloadable release of a dependencyTestPlugin.
// The manifest checksum always matches body; corrupt serves different bytes so
// the download fails verification.
type dependencyTestRelease struct {
	version string
	body    string
	corrupt bool
}

// dependencyTestPlugin describes a plugin the dependency test servers know about.
// releases must be in ascending version order, matching the real manifest format.
type dependencyTestPlugin struct {
	name     string
	releases []dependencyTestRelease
	minPeers map[string]string
}

type dependencyTestEnv struct {
	fs        afero.Fs
	config    *TestConfig
	stripeURL string
	// downloads records each binary download as "<name>@<version>", in order.
	downloads []string
}

func (p dependencyTestPlugin) binaryName() string {
	return "stripe-cli-" + p.name
}

func (p dependencyTestPlugin) manifestPlugin() Plugin {
	releases := make([]Release, 0, len(p.releases))
	for _, release := range p.releases {
		sum := sha256.Sum256([]byte(release.body))
		releases = append(releases, Release{
			Arch:    runtime.GOARCH,
			OS:      runtime.GOOS,
			Version: release.version,
			Sum:     hex.EncodeToString(sum[:]),
		})
	}

	return Plugin{
		Shortname:        p.name,
		Binary:           p.binaryName(),
		MagicCookieValue: p.name + "-cookie",
		Releases:         releases,
	}
}

// setUpDependencyTest builds metadata and artifact servers for a set of plugins
// whose metadata responses carry min_plugin_versions, so dependency installation
// can be exercised end to end against the real install path.
func setUpDependencyTest(t *testing.T, testPlugins []dependencyTestPlugin) *dependencyTestEnv {
	t.Helper()

	env := &dependencyTestEnv{
		fs:     afero.NewMemMapFs(),
		config: &TestConfig{},
	}
	env.config.InitConfig()

	pluginsByName := map[string]dependencyTestPlugin{}
	for _, testPlugin := range testPlugins {
		pluginsByName[testPlugin.name] = testPlugin
	}

	artifactoryServer := httptest.NewServer(http.HandlerFunc(func(res http.ResponseWriter, req *http.Request) {
		parts := strings.Split(strings.TrimPrefix(req.URL.Path, "/"), "/")
		if len(parts) < 2 {
			t.Errorf("unexpected artifact request URL: %s", req.URL.String())
			return
		}

		testPlugin, found := pluginsByName[parts[0]]
		if !found {
			t.Errorf("artifact request for unknown plugin: %s", req.URL.String())
			return
		}

		for _, release := range testPlugin.releases {
			if release.version != parts[1] {
				continue
			}

			env.downloads = append(env.downloads, testPlugin.name+"@"+release.version)
			body := release.body
			if release.corrupt {
				body += "-corrupted-in-transit"
			}
			res.Write([]byte(body))
			return
		}

		t.Errorf("artifact request for unknown release: %s", req.URL.String())
	}))
	t.Cleanup(artifactoryServer.Close)

	stripeServer := httptest.NewServer(http.HandlerFunc(func(res http.ResponseWriter, req *http.Request) {
		switch req.URL.Path {
		case "/v1/stripecli/get-plugin-metadata", "/ajax/stripecli/plugins_metadata":
			pluginName := req.URL.Query().Get("plugin")
			testPlugin, found := pluginsByName[pluginName]
			if !found {
				http.NotFound(res, req)
				return
			}

			resolvedVersion := req.URL.Query().Get("version")
			if resolvedVersion == "" {
				resolvedVersion = testPlugin.releases[len(testPlugin.releases)-1].version
			}

			manifest := new(bytes.Buffer)
			requireNoError(t, toml.NewEncoder(manifest).Encode(PluginList{Plugins: []Plugin{testPlugin.manifestPlugin()}}))

			body, err := json.Marshal(requests.PluginMetadata{
				BinaryURL:         artifactoryServer.URL + "/" + pluginName + "/" + resolvedVersion + "/" + testPlugin.binaryName(),
				PluginManifest:    manifest.String(),
				MinPluginVersions: testPlugin.minPeers,
			})
			requireNoError(t, err)
			res.Write(body)
		default:
			t.Errorf("unexpected stripe request URL: %s", req.URL.String())
		}
	}))
	t.Cleanup(stripeServer.Close)

	env.stripeURL = stripeServer.URL
	return env
}

func (env *dependencyTestEnv) installedBinaryExists(t *testing.T, pluginName, version string) bool {
	t.Helper()

	exists, err := afero.Exists(env.fs, filepath.Join("/plugins", pluginName, version, "stripe-cli-"+pluginName+GetBinaryExtension()))
	require.NoError(t, err)
	return exists
}

func (env *dependencyTestEnv) placeInstalledBinary(t *testing.T, pluginName, version string) {
	t.Helper()
	placeFakeBinary(t, env.fs, "/plugins", pluginName, "stripe-cli-"+pluginName, version)
}

func TestInstallInstallsMissingDependencyFirst(t *testing.T) {
	env := setUpDependencyTest(t, []dependencyTestPlugin{
		{name: "parent", releases: []dependencyTestRelease{{version: "1.0.0", body: "parent-one"}}, minPeers: map[string]string{"child": "1.17.0"}},
		{name: "child", releases: []dependencyTestRelease{{version: "1.17.0", body: "child-one"}}},
	})

	err := (&Plugin{Shortname: "parent"}).Install(context.Background(), env.config, env.fs, "1.0.0", env.stripeURL, env.stripeURL)
	require.NoError(t, err)

	require.Equal(t, []string{"child@1.17.0", "parent@1.0.0"}, env.downloads)
	require.True(t, env.installedBinaryExists(t, "child", "1.17.0"))
	require.True(t, env.installedBinaryExists(t, "parent", "1.0.0"))
	require.Equal(t, []string{"child", "parent"}, env.config.GetInstalledPlugins())

	// The requirement is persisted with the installed plugin, which is what lets
	// RunPeerPlugin enforce it without a network request.
	parentMetadata, err := readLocalPluginMetadata(env.config, env.fs, "parent")
	require.NoError(t, err)
	require.Equal(t, map[string]string{"child": "1.17.0"}, parentMetadata.MinPluginVersions)

	childMetadata, err := readLocalPluginMetadata(env.config, env.fs, "child")
	require.NoError(t, err)
	require.Empty(t, childMetadata.MinPluginVersions)
}

func TestInstallSkipsDependencyThatSatisfiesTheMinimum(t *testing.T) {
	env := setUpDependencyTest(t, []dependencyTestPlugin{
		{name: "parent", releases: []dependencyTestRelease{{version: "1.0.0", body: "parent-one"}}, minPeers: map[string]string{"child": "1.17.0"}},
		{name: "child", releases: []dependencyTestRelease{{version: "1.17.0", body: "child-one"}}},
	})
	env.placeInstalledBinary(t, "child", "1.18.5")

	err := (&Plugin{Shortname: "parent"}).Install(context.Background(), env.config, env.fs, "1.0.0", env.stripeURL, env.stripeURL)
	require.NoError(t, err)

	require.Equal(t, []string{"parent@1.0.0"}, env.downloads)
}

func TestInstallUpgradesDependencyBelowTheMinimumToTheNewestRelease(t *testing.T) {
	env := setUpDependencyTest(t, []dependencyTestPlugin{
		{name: "parent", releases: []dependencyTestRelease{{version: "1.0.0", body: "parent-one"}}, minPeers: map[string]string{"child": "1.17.0"}},
		{name: "child", releases: []dependencyTestRelease{
			{version: "1.17.0", body: "child-one"},
			{version: "1.18.0", body: "child-two"},
		}},
	})
	env.placeInstalledBinary(t, "child", "1.16.2")

	err := (&Plugin{Shortname: "parent"}).Install(context.Background(), env.config, env.fs, "1.0.0", env.stripeURL, env.stripeURL)
	require.NoError(t, err)

	// 1.18.0, not 1.17.0: the declared minimum is a floor, and what gets installed
	// is the newest release that clears it.
	require.Equal(t, []string{"child@1.18.0", "parent@1.0.0"}, env.downloads)
	require.True(t, env.installedBinaryExists(t, "child", "1.18.0"))
	require.False(t, env.installedBinaryExists(t, "child", "1.16.2"))
}

func TestInstallLeavesRequesterAloneWhenDependencyFails(t *testing.T) {
	env := setUpDependencyTest(t, []dependencyTestPlugin{
		{name: "parent", releases: []dependencyTestRelease{{version: "1.0.0", body: "parent-one"}}, minPeers: map[string]string{"child": "1.17.0"}},
		{name: "child", releases: []dependencyTestRelease{{version: "1.17.0", body: "child-one", corrupt: true}}},
	})
	// A previous parent version stays in place when the new one cannot go in.
	env.placeInstalledBinary(t, "parent", "0.9.0")

	err := (&Plugin{Shortname: "parent"}).Install(context.Background(), env.config, env.fs, "1.0.0", env.stripeURL, env.stripeURL)
	require.Error(t, err)
	require.Contains(t, err.Error(), "could not install the child plugin, which the parent plugin depends on")

	require.Equal(t, []string{"child@1.17.0"}, env.downloads)
	require.False(t, env.installedBinaryExists(t, "parent", "1.0.0"))
	require.True(t, env.installedBinaryExists(t, "parent", "0.9.0"))
	require.False(t, env.installedBinaryExists(t, "child", "1.17.0"))
	require.Empty(t, env.config.GetInstalledPlugins())
}

func TestInstallFailsWhenNewestDependencyReleaseIsBelowTheMinimum(t *testing.T) {
	env := setUpDependencyTest(t, []dependencyTestPlugin{
		{name: "parent", releases: []dependencyTestRelease{{version: "1.0.0", body: "parent-one"}}, minPeers: map[string]string{"child": "2.0.0"}},
		{name: "child", releases: []dependencyTestRelease{{version: "1.17.0", body: "child-one"}}},
	})

	err := (&Plugin{Shortname: "parent"}).Install(context.Background(), env.config, env.fs, "1.0.0", env.stripeURL, env.stripeURL)
	require.Error(t, err)
	require.Contains(t, err.Error(), "could not install the child plugin, which the parent plugin depends on")
	require.Contains(t, err.Error(), "v2.0.0 or newer is required")
	require.Contains(t, err.Error(), "newest release available to this CLI is v1.17.0")

	require.Empty(t, env.downloads)
	require.False(t, env.installedBinaryExists(t, "parent", "1.0.0"))
}

func TestInstallResolvesDependencyCyclesWithoutRecursing(t *testing.T) {
	env := setUpDependencyTest(t, []dependencyTestPlugin{
		{name: "parent", releases: []dependencyTestRelease{{version: "1.0.0", body: "parent-one"}}, minPeers: map[string]string{"child": "1.0.0"}},
		{name: "child", releases: []dependencyTestRelease{{version: "1.0.0", body: "child-one"}}, minPeers: map[string]string{"parent": "1.0.0"}},
	})

	err := (&Plugin{Shortname: "parent"}).Install(context.Background(), env.config, env.fs, "1.0.0", env.stripeURL, env.stripeURL)
	require.NoError(t, err)

	// child's own requirement on parent ends at the cycle guard: parent is already
	// being installed up the chain, so each binary downloads exactly once.
	require.Equal(t, []string{"child@1.0.0", "parent@1.0.0"}, env.downloads)
	require.True(t, env.installedBinaryExists(t, "child", "1.0.0"))
	require.True(t, env.installedBinaryExists(t, "parent", "1.0.0"))
}

func TestInstallFailsWhenACycleCannotMeetTheFloor(t *testing.T) {
	// parent v1.0.0 is being installed, but child declares it needs parent v2.0.0.
	// The cycle must stop the recursion without waving that requirement through.
	env := setUpDependencyTest(t, []dependencyTestPlugin{
		{name: "parent", releases: []dependencyTestRelease{{version: "1.0.0", body: "parent-one"}}, minPeers: map[string]string{"child": "1.0.0"}},
		{name: "child", releases: []dependencyTestRelease{{version: "1.0.0", body: "child-one"}}, minPeers: map[string]string{"parent": "2.0.0"}},
	})

	err := (&Plugin{Shortname: "parent"}).Install(context.Background(), env.config, env.fs, "1.0.0", env.stripeURL, env.stripeURL)
	require.Error(t, err)
	require.Contains(t, err.Error(), "could not install the parent plugin, which the child plugin depends on")
	require.Contains(t, err.Error(), "v2.0.0 or newer is required, but v1.0.0 is already being installed")

	require.False(t, env.installedBinaryExists(t, "parent", "1.0.0"))
	require.False(t, env.installedBinaryExists(t, "child", "1.0.0"))
	require.Empty(t, env.config.GetInstalledPlugins())
}

func TestInstallReChecksAFloorForDependenciesInstalledEarlierInTheOperation(t *testing.T) {
	// parent needs b (which installs c on its own, satisfied at v1.0.0) and then
	// needs c itself at v5.0.0, which no release can meet. Having installed c for
	// b earlier in the same operation must not exempt it from parent's own floor.
	env := setUpDependencyTest(t, []dependencyTestPlugin{
		{name: "parent", releases: []dependencyTestRelease{{version: "1.0.0", body: "parent-one"}}, minPeers: map[string]string{"b": "1.0.0", "c": "5.0.0"}},
		{name: "b", releases: []dependencyTestRelease{{version: "1.0.0", body: "b-one"}}, minPeers: map[string]string{"c": "1.0.0"}},
		{name: "c", releases: []dependencyTestRelease{{version: "1.0.0", body: "c-one"}}},
	})

	err := (&Plugin{Shortname: "parent"}).Install(context.Background(), env.config, env.fs, "1.0.0", env.stripeURL, env.stripeURL)
	require.Error(t, err)
	require.Contains(t, err.Error(), "could not install the c plugin, which the parent plugin depends on")
	require.Contains(t, err.Error(), "v5.0.0 or newer is required")

	require.Equal(t, []string{"c@1.0.0", "b@1.0.0"}, env.downloads)
	require.False(t, env.installedBinaryExists(t, "parent", "1.0.0"))
}

func TestInstallDefersDependencyHooksUntilTheChainCompletes(t *testing.T) {
	env := setUpDependencyTest(t, []dependencyTestPlugin{
		{name: "parent", releases: []dependencyTestRelease{{version: "1.0.0", body: "parent-one"}}, minPeers: map[string]string{"child": "1.17.0"}},
		{name: "child", releases: []dependencyTestRelease{{version: "1.17.0", body: "child-one"}}},
	})

	// A hook is plugin code that can call back into the CLI, so it must only run
	// once every plugin in the chain is fully installed — here, parent must be on
	// disk by the time child's hook fires, even though child installed first.
	parentInstalledAtHookTime := false
	previous := dependencyPostInstall
	dependencyPostInstall = func(ctx context.Context, cfg config.IConfig, fs afero.Fs, p *Plugin, version, previousVersion, apiBaseURL, dashboardBaseURL, accessBaseURL string) {
		parentInstalledAtHookTime = env.installedBinaryExists(t, "parent", "1.0.0")
	}
	t.Cleanup(func() { dependencyPostInstall = previous })

	err := (&Plugin{Shortname: "parent"}).Install(context.Background(), env.config, env.fs, "1.0.0", env.stripeURL, env.stripeURL)
	require.NoError(t, err)
	require.True(t, parentInstalledAtHookTime)
}

func TestInstallNeverReplacesALocalDependencyBuild(t *testing.T) {
	env := setUpDependencyTest(t, []dependencyTestPlugin{
		{name: "parent", releases: []dependencyTestRelease{{version: "1.0.0", body: "parent-one"}}, minPeers: map[string]string{"child": "99.0.0"}},
		{name: "child", releases: []dependencyTestRelease{{version: "1.17.0", body: "child-one"}}},
	})
	env.placeInstalledBinary(t, "child", localDevelopmentVersion)

	err := (&Plugin{Shortname: "parent"}).Install(context.Background(), env.config, env.fs, "1.0.0", env.stripeURL, env.stripeURL)
	require.NoError(t, err)

	require.Equal(t, []string{"parent@1.0.0"}, env.downloads)
	require.True(t, env.installedBinaryExists(t, "child", localDevelopmentVersion))
}

func TestHookBaseURLOverrides(t *testing.T) {
	qaDashboard := stripe.DashboardBaseURLForAPIBaseURL("https://qa-api.stripe.com")

	cases := []struct {
		name                  string
		apiBaseURL            string
		dashboardBaseURL      string
		expectedHookAPI       string
		expectedHookDashboard string
	}{
		{
			name:             "defaults resolve back to no overrides",
			apiBaseURL:       stripe.DefaultAPIBaseURL,
			dashboardBaseURL: stripe.DashboardBaseURLForAPIBaseURL(stripe.DefaultAPIBaseURL),
		},
		{
			name:                  "non-default API URL can only be the user's",
			apiBaseURL:            "https://qa-api.stripe.com",
			dashboardBaseURL:      qaDashboard,
			expectedHookAPI:       "https://qa-api.stripe.com",
			expectedHookDashboard: "", // derived from the API URL, so resolution filled it in
		},
		{
			name:                  "dashboard URL that is not the derived one is the user's",
			apiBaseURL:            stripe.DefaultAPIBaseURL,
			dashboardBaseURL:      "https://dashboard.example.test",
			expectedHookDashboard: "https://dashboard.example.test",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			hookAPI, hookDashboard := hookBaseURLOverrides(tc.apiBaseURL, tc.dashboardBaseURL)
			require.Equal(t, tc.expectedHookAPI, hookAPI)
			require.Equal(t, tc.expectedHookDashboard, hookDashboard)
		})
	}
}

type dependencyPostInstallCall struct {
	plugin           string
	version          string
	previousVersion  string
	apiBaseURL       string
	dashboardBaseURL string
	accessBaseURL    string
}

func stubDependencyPostInstall(t *testing.T) *[]dependencyPostInstallCall {
	t.Helper()

	calls := &[]dependencyPostInstallCall{}
	previous := dependencyPostInstall
	dependencyPostInstall = func(ctx context.Context, cfg config.IConfig, fs afero.Fs, p *Plugin, version, previousVersion, apiBaseURL, dashboardBaseURL, accessBaseURL string) {
		*calls = append(*calls, dependencyPostInstallCall{
			plugin:           p.Shortname,
			version:          version,
			previousVersion:  previousVersion,
			apiBaseURL:       apiBaseURL,
			dashboardBaseURL: dashboardBaseURL,
			accessBaseURL:    accessBaseURL,
		})
	}
	t.Cleanup(func() { dependencyPostInstall = previous })

	return calls
}

func TestInstallForwardsExplicitBaseURLsToDependencyHooks(t *testing.T) {
	hookCalls := stubDependencyPostInstall(t)

	env := setUpDependencyTest(t, []dependencyTestPlugin{
		{name: "parent", releases: []dependencyTestRelease{{version: "1.0.0", body: "parent-one"}}, minPeers: map[string]string{"child": "1.17.0"}},
		{name: "child", releases: []dependencyTestRelease{{version: "1.17.0", body: "child-one"}}},
	})

	// The test server URL stands in for an explicit --api-base: it is not the
	// default, so only the user could have put it there. The dashboard URL differs
	// from the one derived from the API URL, so it counts as explicit too. (The
	// metadata requests authenticate with the test API key, so the dashboard URL is
	// only threaded through, never contacted.)
	explicitDashboardBaseURL := env.stripeURL + "/dashboard"
	err := (&Plugin{Shortname: "parent"}).Install(context.Background(), env.config, env.fs, "1.0.0", env.stripeURL, explicitDashboardBaseURL)
	require.NoError(t, err)

	require.Equal(t, []dependencyPostInstallCall{{
		plugin:           "child",
		version:          "1.17.0",
		apiBaseURL:       env.stripeURL,
		dashboardBaseURL: explicitDashboardBaseURL,
	}}, *hookCalls)
}

// placeFakeBinary writes an installed-looking plugin binary so version lookups
// find it, without going through a real install.
func placeFakeBinary(t *testing.T, fs afero.Fs, pluginsDir, pluginName, binaryName, version string) {
	t.Helper()

	installDir := filepath.Join(pluginsDir, pluginName, version)
	require.NoError(t, fs.MkdirAll(installDir, 0755))
	require.NoError(t, afero.WriteFile(fs, filepath.Join(installDir, binaryName+GetBinaryExtension()), []byte("bin"), 0755))
}

type peerEnforcementStubs struct {
	resolveCalls []string
	resolveErr   error
	resolved     *ResolvedPluginVersion
	installCalls []string
	installErr   error
	// installWrites simulates the install landing on disk, so the subsequent run
	// finds a version instead of wandering into the auto-install network path.
	installWrites func(resolved *ResolvedPluginVersion) error
}

func stubPeerPluginEnforcement(t *testing.T) *peerEnforcementStubs {
	t.Helper()

	stubs := &peerEnforcementStubs{}
	previousResolver := minimumVersionResolver
	previousInstaller := minimumVersionInstaller

	minimumVersionResolver = func(ctx context.Context, cfg config.IConfig, fs afero.Fs, pluginName, apiBaseURL, dashboardBaseURL string) (*ResolvedPluginVersion, error) {
		stubs.resolveCalls = append(stubs.resolveCalls, pluginName)
		if stubs.resolveErr != nil {
			return nil, stubs.resolveErr
		}
		return stubs.resolved, nil
	}
	minimumVersionInstaller = func(ctx context.Context, resolved *ResolvedPluginVersion, cfg config.IConfig, fs afero.Fs, apiBaseURL, dashboardBaseURL string, chain *installChain) error {
		stubs.installCalls = append(stubs.installCalls, resolved.Plugin.Shortname+"@"+resolved.Version)
		if stubs.installErr != nil {
			return stubs.installErr
		}
		if stubs.installWrites != nil {
			return stubs.installWrites(resolved)
		}
		return nil
	}

	t.Cleanup(func() {
		minimumVersionResolver = previousResolver
		minimumVersionInstaller = previousInstaller
	})

	return stubs
}

// setUpPeerEnforcement wires appB as the calling plugin, recording the given minimum
// peer versions in its local metadata (nil means appB has no local metadata at all,
// like a plugin installed before requirements were recorded), and installs appA (the
// peer) at peerInstalledVersion when non-empty. It returns a helper created on appB's
// behalf.
func setUpPeerEnforcement(t *testing.T, callerMinimums map[string]string, peerInstalledVersion string) (CoreCLIHelper, *TestConfig, afero.Fs, *peerEnforcementStubs) {
	t.Helper()

	stubs := stubPeerPluginEnforcement(t)
	stubs.resolved = runAutoUpgradeResolution("3.0.0")

	fs := setUpFS()
	cfg := &TestConfig{}
	cfg.InitConfig()

	// RunPeerPlugin needs a real *config.Config, whose config folder is not
	// TestConfig's "/"; see setUpRunAutoUpgrade for why XDG_CONFIG_HOME is the knob.
	t.Setenv("XDG_CONFIG_HOME", "/xdg")

	if callerMinimums != nil {
		caller, err := LookUpPlugin(context.Background(), cfg, fs, "appB")
		require.NoError(t, err)
		caller.MinPluginVersions = callerMinimums
		require.NoError(t, writeLocalPluginMetadata(&cfg.Config, fs, caller))
	}

	peer, err := LookUpPlugin(context.Background(), cfg, fs, "appA")
	require.NoError(t, err)
	require.NoError(t, writeLocalPluginMetadata(&cfg.Config, fs, peer))

	if peerInstalledVersion != "" {
		placeFakeBinary(t, fs, getPluginsDir(&cfg.Config), "appA", "stripe-cli-app-a", peerInstalledVersion)
	}

	helper := NewCoreCLIHelperForPlugin(context.Background(), &cfg.Config, fs, "appB", "", "", "")
	return helper, cfg, fs, stubs
}

func TestRunPeerPluginUpgradesPeerBelowDeclaredMinimum(t *testing.T) {
	helper, _, _, stubs := setUpPeerEnforcement(t, map[string]string{"appA": "2.0.0"}, "1.0.1")

	runErr := helper.RunPeerPlugin("appA", nil, "")

	require.Equal(t, []string{"appA"}, stubs.resolveCalls)
	require.Equal(t, []string{"appA@3.0.0"}, stubs.installCalls)

	// The run itself fails against the fake binary, but it got past enforcement:
	// the error is a launch failure, not the minimum-version refusal.
	require.Error(t, runErr)
	require.NotContains(t, runErr.Error(), "needs the appA plugin")
}

func TestRunPeerPluginInstallsMissingDeclaredPeer(t *testing.T) {
	helper, cfg, fs, stubs := setUpPeerEnforcement(t, map[string]string{"appA": "2.0.0"}, "")

	// Make the stubbed install land on disk, as the real one would, so the run
	// that follows finds the version instead of starting a network auto-install.
	stubs.installWrites = func(resolved *ResolvedPluginVersion) error {
		placeFakeBinary(t, fs, getPluginsDir(&cfg.Config), "appA", "stripe-cli-app-a", resolved.Version)
		return nil
	}

	runErr := helper.RunPeerPlugin("appA", nil, "")

	require.Equal(t, []string{"appA"}, stubs.resolveCalls)
	require.Equal(t, []string{"appA@3.0.0"}, stubs.installCalls)
	require.Error(t, runErr)
	require.NotContains(t, runErr.Error(), "needs the appA plugin")
	require.NotContains(t, runErr.Error(), "not found")
}

func TestRunPeerPluginReportsPeerItCouldNotInstall(t *testing.T) {
	helper, _, _, stubs := setUpPeerEnforcement(t, map[string]string{"appA": "2.0.0"}, "1.0.1")
	stubs.installErr = errors.New("download exploded")

	runErr := helper.RunPeerPlugin("appA", nil, "")

	require.Error(t, runErr)
	require.Contains(t, runErr.Error(), "the appB plugin needs the appA plugin v2.0.0 or newer (installed: 1.0.1). Run `stripe plugin install appA`")
	require.Contains(t, runErr.Error(), "download exploded")

	category, categorized := errorcategory.Get(runErr)
	require.True(t, categorized)
	require.Equal(t, errorcategory.UserInput, category)
}

func TestRunPeerPluginReportsPeerItCouldNotResolve(t *testing.T) {
	helper, _, _, stubs := setUpPeerEnforcement(t, map[string]string{"appA": "2.0.0"}, "")
	stubs.resolveErr = errors.New("endpoint unreachable")

	runErr := helper.RunPeerPlugin("appA", nil, "")

	require.Error(t, runErr)
	require.Contains(t, runErr.Error(), "the appB plugin needs the appA plugin v2.0.0 or newer (installed: none). Run `stripe plugin install appA`")
	require.Empty(t, stubs.installCalls)
}

func TestRunPeerPluginRefusesWhenNewestReleaseIsBelowMinimum(t *testing.T) {
	helper, _, _, stubs := setUpPeerEnforcement(t, map[string]string{"appA": "4.0.0"}, "1.0.1")

	runErr := helper.RunPeerPlugin("appA", nil, "")

	require.Error(t, runErr)
	require.Contains(t, runErr.Error(), "the appB plugin needs the appA plugin v4.0.0 or newer")
	require.Contains(t, runErr.Error(), "newest release available to this CLI is v3.0.0")
	require.Empty(t, stubs.installCalls, "a release below the floor must not be installed as if it satisfied it")
}

func TestRunPeerPluginSkipsEnforcementWhenMinimumIsSatisfied(t *testing.T) {
	helper, _, _, stubs := setUpPeerEnforcement(t, map[string]string{"appA": "1.0.0"}, "1.0.1")

	runErr := helper.RunPeerPlugin("appA", nil, "")

	require.Empty(t, stubs.resolveCalls)
	require.Empty(t, stubs.installCalls)
	require.Error(t, runErr)
	require.NotContains(t, runErr.Error(), "needs the appA plugin")
}

func TestRunPeerPluginIgnoresUndeclaredPeer(t *testing.T) {
	// The caller declares a minimum, but for some other plugin: appA runs exactly
	// as it does today, even though it is far behind that version.
	helper, _, _, stubs := setUpPeerEnforcement(t, map[string]string{"somethingElse": "9.0.0"}, "1.0.1")

	runErr := helper.RunPeerPlugin("appA", nil, "")

	require.Empty(t, stubs.resolveCalls)
	require.Empty(t, stubs.installCalls)
	require.Error(t, runErr)
	require.NotContains(t, runErr.Error(), "needs the appA plugin")
}

func TestRunPeerPluginNeverReplacesALocalPeerBuild(t *testing.T) {
	helper, _, _, stubs := setUpPeerEnforcement(t, map[string]string{"appA": "99.0.0"}, localDevelopmentVersion)

	runErr := helper.RunPeerPlugin("appA", nil, "")

	require.Empty(t, stubs.resolveCalls)
	require.Empty(t, stubs.installCalls)
	require.Error(t, runErr)
	require.NotContains(t, runErr.Error(), "needs the appA plugin")
}

func TestRunPeerPluginWithoutRecordedCallerMetadataEnforcesNothing(t *testing.T) {
	// The peer exists, but the caller has no local metadata at all — a plugin
	// installed before requirements were recorded. Nothing is enforced.
	helper, _, _, stubs := setUpPeerEnforcement(t, nil, "1.0.1")

	runErr := helper.RunPeerPlugin("appA", nil, "")

	require.Empty(t, stubs.resolveCalls)
	require.Empty(t, stubs.installCalls)
	require.Error(t, runErr)
	require.NotContains(t, runErr.Error(), "needs the appA plugin")
}

func TestPluginMetadataResponseMinimumsSurviveTheLocalMetadataRoundTrip(t *testing.T) {
	var decoded requests.PluginMetadata
	require.NoError(t, json.Unmarshal([]byte(`{
		"binary_url": "https://artifacts.example/generate/1.0.0",
		"plugin_manifest": "",
		"min_plugin_versions": {"apps": "1.17.0"}
	}`), &decoded))
	require.Equal(t, map[string]string{"apps": "1.17.0"}, decoded.MinPluginVersions)

	// And a response without the field means no requirements.
	var withoutField requests.PluginMetadata
	require.NoError(t, json.Unmarshal([]byte(`{"binary_url": "x", "plugin_manifest": ""}`), &withoutField))
	require.Nil(t, withoutField.MinPluginVersions)

	// The map survives the TOML round trip the local plugin metadata uses.
	plugin := Plugin{
		Shortname:         "generate",
		Binary:            "stripe-cli-generate",
		MinPluginVersions: decoded.MinPluginVersions,
		Releases:          []Release{{Arch: runtime.GOARCH, OS: runtime.GOOS, Version: "1.0.0"}},
	}
	fs := afero.NewMemMapFs()
	cfg := &TestConfig{}
	require.NoError(t, writeLocalPluginMetadata(cfg, fs, plugin))

	roundTripped, err := readLocalPluginMetadata(cfg, fs, "generate")
	require.NoError(t, err)
	require.Equal(t, map[string]string{"apps": "1.17.0"}, roundTripped.MinPluginVersions)
}
