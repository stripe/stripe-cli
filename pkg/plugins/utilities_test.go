package plugins

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/BurntSushi/toml"
	"github.com/spf13/afero"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/require"

	"github.com/stripe/stripe-cli/pkg/requests"
)

// CustomTestConfig is a test config that allows overriding the config folder path
type CustomTestConfig struct {
	TestConfig
	customConfigPath string
}

// GetConfigFolder overrides the TestConfig method to return a custom path
func (c *CustomTestConfig) GetConfigFolder(xdgPath string) string {
	return c.customConfigPath
}

func TestGetPluginList(t *testing.T) {
	fs := setUpFS()
	config := &TestConfig{}
	config.InitConfig()

	pluginList, err := GetPluginList(context.Background(), config, fs)

	require.Nil(t, err)
	require.Equal(t, 3, len(pluginList.Plugins))
	plugin := pluginList.Plugins[0]
	require.Equal(t, "appA", plugin.Shortname)
	require.Equal(t, "stripe-cli-app-a", plugin.Binary)
	require.Equal(t, "0337A75A-C3C4-4DCF-A9EF-E7A144E5A291", plugin.MagicCookieValue)

	require.Equal(t, 12, len(plugin.Releases))
	release := plugin.Releases[0]
	require.Equal(t, "amd64", release.Arch)
	require.Equal(t, "darwin", release.OS)
	require.Equal(t, "0.0.1", release.Version)
	require.Equal(t, "125653c37803a51a048f6687f7f66d511be614f675f199cd6c71928b74875238", release.Sum)
}

func TestLookUpPlugin(t *testing.T) {
	fs := setUpFS()
	config := &TestConfig{}

	plugin, err := LookUpPlugin(context.Background(), config, fs, "appB")
	require.Nil(t, err)
	require.Equal(t, "appB", plugin.Shortname)
	require.Equal(t, "stripe-cli-app-b", plugin.Binary)
	require.Equal(t, "FDBE6FB9-A149-44BD-9639-4D33D8B594E8", plugin.MagicCookieValue)
	require.Equal(t, 4, len(plugin.Releases))
}

func TestLookUpPluginUsesLocalMetadataWhenManifestMissing(t *testing.T) {
	fs := afero.NewMemMapFs()
	config := &TestConfig{}

	localPlugin := Plugin{
		Shortname:        "local-plugin",
		Binary:           "stripe-cli-local-plugin",
		MagicCookieValue: "LOCAL-PLUGIN-COOKIE",
		Releases: []Release{
			{
				Arch:    runtime.GOARCH,
				OS:      runtime.GOOS,
				Version: "1.0.0",
				Sum:     "abc123",
			},
		},
	}

	require.NoError(t, writeLocalPluginMetadata(config, fs, localPlugin))

	plugin, err := LookUpPlugin(context.Background(), config, fs, "local-plugin")
	require.NoError(t, err)
	require.Equal(t, localPlugin, plugin)
}

func TestGetInstalledPluginNamesIncludesLocalMetadata(t *testing.T) {
	fs := afero.NewMemMapFs()
	config := &TestConfig{}
	config.InstalledPlugins = []string{"projects"}

	localPlugin := Plugin{
		Shortname:        "docs",
		Binary:           "stripe-cli-docs",
		MagicCookieValue: "DOCS-COOKIE",
		Releases: []Release{
			{
				Arch:    runtime.GOARCH,
				OS:      runtime.GOOS,
				Version: "1.0.0",
				Sum:     "abc123",
			},
		},
	}

	require.NoError(t, writeLocalPluginMetadata(config, fs, localPlugin))

	pluginNames, err := GetInstalledPluginNames(config, fs)
	require.NoError(t, err)
	require.Equal(t, []string{"projects", "docs"}, pluginNames)
}

func TestRecordInstalledPlugin(t *testing.T) {
	config := &TestConfig{}

	require.NoError(t, RecordInstalledPlugin(config, "docs"))
	require.Equal(t, []string{"docs"}, config.GetInstalledPlugins())

	require.NoError(t, RecordInstalledPlugin(config, "docs"))
	require.Equal(t, []string{"docs"}, config.GetInstalledPlugins())
}

func TestRemoveInstalledPlugin(t *testing.T) {
	config := &TestConfig{}
	config.InstalledPlugins = []string{"projects", "docs"}

	require.NoError(t, RemoveInstalledPlugin(config, "docs"))
	require.Equal(t, []string{"projects"}, config.GetInstalledPlugins())

	require.NoError(t, RemoveInstalledPlugin(config, "docs"))
	require.Equal(t, []string{"projects"}, config.GetInstalledPlugins())
}

func TestPersistInstalledPluginState(t *testing.T) {
	fs := afero.NewMemMapFs()
	config := &TestConfig{}
	plugin := Plugin{
		Shortname:        "docs",
		Shortdesc:        "Docs plugin",
		Binary:           "stripe-cli-docs",
		MagicCookieValue: "DOCS-COOKIE",
		Commands: []CommandInfo{
			{
				Name: "search",
				Desc: "Search docs",
			},
		},
		Releases: []Release{
			{
				Arch:    runtime.GOARCH,
				OS:      runtime.GOOS,
				Version: "1.0.0",
				Sum:     "abc123",
			},
		},
	}

	require.NoError(t, PersistInstalledPluginState(config, fs, plugin))
	require.Equal(t, []string{"docs"}, config.GetInstalledPlugins())

	cachedPlugin, err := readLocalPluginMetadata(config, fs, "docs")
	require.NoError(t, err)
	require.Equal(t, plugin, cachedPlugin)
}

func TestPersistInstalledPluginStateRollsBackOnConfigWriteFailure(t *testing.T) {
	fs := afero.NewMemMapFs()
	config := &FailingWriteConfig{
		WriteErr:                 errors.New("boom"),
		MutateInstalledPluginsOn: true,
	}
	plugin := Plugin{
		Shortname:        "docs",
		Shortdesc:        "Docs plugin",
		Binary:           "stripe-cli-docs",
		MagicCookieValue: "DOCS-COOKIE",
		Releases: []Release{
			{
				Arch:    runtime.GOARCH,
				OS:      runtime.GOOS,
				Version: "1.0.0",
				Sum:     "abc123",
			},
		},
	}

	err := PersistInstalledPluginState(config, fs, plugin)
	require.ErrorIs(t, err, config.WriteErr)

	metadataExists, err := afero.Exists(fs, getLocalPluginMetadataPath(config, "docs"))
	require.NoError(t, err)
	require.False(t, metadataExists)
	require.Empty(t, config.GetInstalledPlugins())
}

func TestPersistInstalledPluginStateRestoresPreviousMetadataOnConfigWriteFailure(t *testing.T) {
	fs := afero.NewMemMapFs()
	config := &FailingWriteConfig{
		WriteErr:                 errors.New("boom"),
		MutateInstalledPluginsOn: true,
	}
	existingPlugin := Plugin{
		Shortname:        "docs",
		Shortdesc:        "Existing docs plugin",
		Binary:           "stripe-cli-docs",
		MagicCookieValue: "DOCS-COOKIE",
		Releases: []Release{
			{
				Arch:    runtime.GOARCH,
				OS:      runtime.GOOS,
				Version: "1.0.0",
				Sum:     "abc123",
			},
		},
	}
	updatedPlugin := Plugin{
		Shortname:        "docs",
		Shortdesc:        "Updated docs plugin",
		Binary:           "stripe-cli-docs",
		MagicCookieValue: "DOCS-COOKIE",
		Releases: []Release{
			{
				Arch:    runtime.GOARCH,
				OS:      runtime.GOOS,
				Version: "1.1.0",
				Sum:     "def456",
			},
		},
	}
	require.NoError(t, writeLocalPluginMetadata(config, fs, existingPlugin))

	err := PersistInstalledPluginState(config, fs, updatedPlugin)
	require.ErrorIs(t, err, config.WriteErr)

	cachedPlugin, err := readLocalPluginMetadata(config, fs, "docs")
	require.NoError(t, err)
	require.Equal(t, existingPlugin, cachedPlugin)
	require.Empty(t, config.GetInstalledPlugins())
}

func TestLookUpPluginInManifestIgnoresLocalMetadata(t *testing.T) {
	fs := setUpFS()
	config := &TestConfig{}

	localPlugin := Plugin{
		Shortname:        "appA",
		Binary:           "stripe-cli-local-app-a",
		MagicCookieValue: "LOCAL-APP-A-COOKIE",
		Releases: []Release{
			{
				Arch:    runtime.GOARCH,
				OS:      runtime.GOOS,
				Version: "9.9.9",
				Sum:     "abc123",
			},
		},
	}

	require.NoError(t, writeLocalPluginMetadata(config, fs, localPlugin))

	plugin, err := LookUpPluginInManifest(context.Background(), config, fs, "appA")
	require.NoError(t, err)
	require.Equal(t, "stripe-cli-app-a", plugin.Binary)
	require.Equal(t, "0337A75A-C3C4-4DCF-A9EF-E7A144E5A291", plugin.MagicCookieValue)
}

func TestResolvePluginForInstallUsesLocalMetadataAsMetadataBase(t *testing.T) {
	fs := afero.NewMemMapFs()
	config := &TestConfig{}
	config.InitConfig()

	localPlugin := Plugin{
		Shortname:        "generate",
		Binary:           "stripe-cli-generate",
		MagicCookieValue: "GENERATE-COOKIE",
		Commands: []CommandInfo{
			{
				Name: "create",
				Desc: "Create generated artifacts",
			},
		},
		Releases: []Release{
			{
				Arch:    runtime.GOARCH,
				OS:      runtime.GOOS,
				Version: "1.0.0",
				Sum:     "abc123",
				Runtime: map[string]string{"node": "20"},
			},
		},
	}
	require.NoError(t, writeLocalPluginMetadata(config, fs, localPlugin))

	metadataManifest := fmt.Sprintf(`[[Plugin]]
  Shortname = "generate"
  Shortdesc = "Generate things"
  Binary = "stripe-cli-generate"
  MagicCookieValue = "GENERATE-COOKIE"

  [[Plugin.Release]]
    Arch = "%s"
    OS = "%s"
    Version = "1.0.0"
    Sum = "abc123"
`, runtime.GOARCH, runtime.GOOS)

	stripeServer := httptest.NewServer(http.HandlerFunc(func(res http.ResponseWriter, req *http.Request) {
		switch req.URL.Path {
		case "/v1/stripecli/get-plugin-metadata":
			body, err := json.Marshal(requests.PluginMetadata{
				BinaryURL:      "https://example.test/generate/1.0.0",
				PluginManifest: metadataManifest,
			})
			require.NoError(t, err)
			res.Write(body)
		default:
			t.Errorf("Received an unexpected request URL: %s", req.URL.String())
		}
	}))
	defer stripeServer.Close()

	plugin, version, err := ResolvePluginForInstall(context.Background(), config, fs, "generate", "1.0.0", stripeServer.URL)
	require.NoError(t, err)
	require.Equal(t, "1.0.0", version)
	require.Len(t, plugin.Commands, 1)
	require.Equal(t, "create", plugin.Commands[0].Name)
	release := plugin.getReleaseForVersion("1.0.0")
	require.NotNil(t, release)
	require.Equal(t, "20", release.Runtime["node"])
}

func TestResolvePluginForUpgradeUsesLocalMetadataWhenManifestMissing(t *testing.T) {
	fs := afero.NewMemMapFs()
	config := &TestConfig{}

	localPlugin := Plugin{
		Shortname:        "docs",
		Shortdesc:        "Docs plugin",
		Binary:           "stripe-cli-docs",
		MagicCookieValue: "DOCS-COOKIE",
		Commands: []CommandInfo{
			{
				Name: "search",
				Desc: "Search docs",
			},
		},
		Releases: []Release{
			{
				Arch:    runtime.GOARCH,
				OS:      runtime.GOOS,
				Version: "0.1.25",
				Sum:     "abc123",
			},
		},
	}
	require.NoError(t, writeLocalPluginMetadata(config, fs, localPlugin))

	plugin, err := ResolvePluginForUpgrade(config, fs, "docs")
	require.NoError(t, err)
	require.Equal(t, localPlugin, *plugin)
}

func TestResolvePluginForUpgradePrefersNewerManifestVersion(t *testing.T) {
	fs := afero.NewMemMapFs()
	config := &TestConfig{}

	localPlugin := Plugin{
		Shortname:        "docs",
		Shortdesc:        "Docs plugin",
		Binary:           "stripe-cli-docs",
		MagicCookieValue: "DOCS-COOKIE",
		Commands: []CommandInfo{
			{
				Name: "search",
				Desc: "Search docs",
			},
		},
		Releases: []Release{
			{
				Arch:    runtime.GOARCH,
				OS:      runtime.GOOS,
				Version: "0.1.25",
				Sum:     "abc123",
			},
		},
	}
	require.NoError(t, writeLocalPluginMetadata(config, fs, localPlugin))

	manifestContent := fmt.Sprintf(`[[Plugin]]
  Shortname = "docs"
  Shortdesc = "Docs plugin"
  Binary = "stripe-cli-docs"
  MagicCookieValue = "DOCS-COOKIE"

  [[Plugin.Release]]
    Arch = "%s"
    OS = "%s"
    Version = "0.1.26"
    Sum = "def456"
`, runtime.GOARCH, runtime.GOOS)
	require.NoError(t, afero.WriteFile(fs, "/plugins.toml", []byte(manifestContent), os.ModePerm))

	plugin, err := ResolvePluginForUpgrade(config, fs, "docs")
	require.NoError(t, err)
	require.Equal(t, "0.1.26", plugin.LookUpLatestVersion())
	require.Len(t, plugin.Commands, 1)
	require.Equal(t, "search", plugin.Commands[0].Name)
}

func TestRefreshPluginManifest(t *testing.T) {
	fs := setUpFS()
	config := &TestConfig{}
	config.InitConfig()
	updatedManifestContent, _ := os.ReadFile("./test_artifacts/plugins_updated.toml")
	testServers := setUpServers(t, updatedManifestContent, nil)
	defer func() { testServers.CloseAll() }()

	err := RefreshPluginManifest(context.Background(), config, fs, testServers.StripeServer.URL)
	require.Nil(t, err)

	// We expect the /plugins.toml file in the test fs is updated
	pluginManifestContent, err := afero.ReadFile(fs, "/plugins.toml")
	require.Nil(t, err)

	actualPluginList := PluginList{}
	err = toml.Unmarshal(pluginManifestContent, &actualPluginList)
	require.Nil(t, err)

	expectedPluginList := PluginList{}
	err = toml.Unmarshal(updatedManifestContent, &expectedPluginList)
	require.Nil(t, err)

	require.Equal(t, expectedPluginList, actualPluginList)
}

func TestRefreshPluginManifestMergesAdditionalManifest(t *testing.T) {
	fs := setUpFS()
	config := &TestConfig{}
	config.InitConfig()
	manifestContent, _ := os.ReadFile("./test_artifacts/plugins.toml")
	mergedManifestContent, _ := os.ReadFile("./test_artifacts/plugins-merged-foo-1.toml")

	fooManifest, _ := os.ReadFile("./test_artifacts/plugins-foo-1.toml")
	additionalManifests := map[string][]byte{
		"plugins-foo-1.toml": fooManifest,
	}

	testServers := setUpServers(t, manifestContent, additionalManifests)
	defer func() { testServers.CloseAll() }()

	err := RefreshPluginManifest(context.Background(), config, fs, testServers.StripeServer.URL)
	require.Nil(t, err)

	// We expect the /plugins.toml file in the test fs is updated
	pluginManifestContent, err := afero.ReadFile(fs, "/plugins.toml")
	require.Nil(t, err)

	actualPluginList := PluginList{}
	err = toml.Unmarshal(pluginManifestContent, &actualPluginList)
	require.Nil(t, err)

	expectedPluginList := PluginList{}
	err = toml.Unmarshal(mergedManifestContent, &expectedPluginList)
	require.Nil(t, err)

	require.Equal(t, expectedPluginList, actualPluginList)
}

func TestRefreshPluginManifestMergesExisitingPlugin(t *testing.T) {
	fs := setUpFS()
	config := &TestConfig{}
	config.InitConfig()
	manifestContent, _ := os.ReadFile("./test_artifacts/plugins-2.toml")
	mergedManifestContent, _ := os.ReadFile("./test_artifacts/plugins-2-merged-foo-1.toml")

	fooManifest, _ := os.ReadFile("./test_artifacts/plugins-foo-1.toml")
	additionalManifests := map[string][]byte{
		"plugins-foo-1.toml": fooManifest,
	}

	testServers := setUpServers(t, manifestContent, additionalManifests)
	defer func() { testServers.CloseAll() }()

	err := RefreshPluginManifest(context.Background(), config, fs, testServers.StripeServer.URL)
	require.Nil(t, err)

	// We expect the /plugins.toml file in the test fs is updated
	pluginManifestContent, err := afero.ReadFile(fs, "/plugins.toml")
	require.Nil(t, err)

	actualPluginList := PluginList{}
	err = toml.Unmarshal(pluginManifestContent, &actualPluginList)
	require.Nil(t, err)

	expectedPluginList := PluginList{}
	err = toml.Unmarshal(mergedManifestContent, &expectedPluginList)
	require.Nil(t, err)

	require.Equal(t, expectedPluginList, actualPluginList)
}

func TestRefreshPluginManifestSortsPluginReleases(t *testing.T) {
	fs := setUpFS()
	config := &TestConfig{}
	config.InitConfig()
	manifestContent, _ := os.ReadFile("./test_artifacts/plugins-3.toml")
	mergedManifestContent, _ := os.ReadFile("./test_artifacts/plugins-3-merged-foo-1.toml")

	fooManifest, _ := os.ReadFile("./test_artifacts/plugins-foo-1.toml")
	additionalManifests := map[string][]byte{
		"plugins-foo-1.toml": fooManifest,
	}

	testServers := setUpServers(t, manifestContent, additionalManifests)
	defer func() { testServers.CloseAll() }()

	err := RefreshPluginManifest(context.Background(), config, fs, testServers.StripeServer.URL)
	require.Nil(t, err)

	// We expect the /plugins.toml file in the test fs is updated
	pluginManifestContent, err := afero.ReadFile(fs, "/plugins.toml")
	require.Nil(t, err)

	actualPluginList := PluginList{}
	err = toml.Unmarshal(pluginManifestContent, &actualPluginList)
	require.Nil(t, err)

	expectedPluginList := PluginList{}
	err = toml.Unmarshal(mergedManifestContent, &expectedPluginList)
	require.Nil(t, err)

	require.Equal(t, expectedPluginList, actualPluginList)
}

func TestRefreshPluginManifestFailsInvalidManifest(t *testing.T) {
	fs := setUpFS()
	config := &TestConfig{}
	config.InitConfig()
	emptyManifestContent := []byte{}
	testServers := setUpServers(t, emptyManifestContent, nil)
	defer func() { testServers.CloseAll() }()

	err := RefreshPluginManifest(context.Background(), config, fs, testServers.StripeServer.URL)
	require.NotNil(t, err)
	require.ErrorContains(t, err, "received an empty plugin manifest")
	// We expect the /plugins.toml file in the test fs has NOT been updated
	pluginManifestContent, err := afero.ReadFile(fs, "/plugins.toml")
	require.Nil(t, err)
	require.NotEqual(t, emptyManifestContent, pluginManifestContent)
}

func TestIsPluginCommand(t *testing.T) {
	pluginCmd := &cobra.Command{
		Annotations: map[string]string{"scope": "plugin"},
	}

	notPluginCmd := &cobra.Command{}

	require.True(t, IsPluginCommand(pluginCmd))
	require.False(t, IsPluginCommand(notPluginCmd))
}

func TestRefreshPluginManifestSucceedsIfNoAPIKey(t *testing.T) {
	fs := setUpFS()
	config := &TestConfig{}
	config.InitConfig()
	config.Profile.APIKey = ""
	testServers := setUpServers(t, nil, nil)
	defer func() { testServers.CloseAll() }()

	err := RefreshPluginManifest(context.Background(), config, fs, testServers.StripeServer.URL)
	require.Nil(t, err)
}

func TestIsValidNodeLTSVersion(t *testing.T) {
	validVersions := []string{"18", "20", "22", "24", "26"}
	for _, version := range validVersions {
		require.True(t, isValidNodeLTSVersion(version), "Expected %s to be valid LTS version", version)
	}

	invalidVersions := []string{"10", "11", "12", "13", "14", "15", "16", "17", "19", "21", "23", "25"}
	for _, version := range invalidVersions {
		require.False(t, isValidNodeLTSVersion(version), "Expected %s to be invalid LTS version", version)
	}

	invalidFormats := []string{"", "abc", "20.0", "v20", "20.0.0", "node20"}
	for _, version := range invalidFormats {
		require.False(t, isValidNodeLTSVersion(version), "Expected %s to be invalid format", version)
	}
}

func TestValidateRuntimeVersionsValid(t *testing.T) {
	pluginList := &PluginList{
		Plugins: []Plugin{
			{
				Shortname: "test-plugin",
				Releases: []Release{
					{
						Version: "1.0.0",
						Runtime: map[string]string{"node": "18"},
					},
					{
						Version: "1.1.0",
						Runtime: map[string]string{"node": "20"},
					},
					{
						Version: "2.0.0",
						Runtime: map[string]string{"node": "24"},
					},
				},
			},
		},
	}

	err := validateRuntimeVersions(pluginList)
	require.Nil(t, err)
}

func TestValidateRuntimeVersionsInvalidNonLTS(t *testing.T) {
	pluginList := &PluginList{
		Plugins: []Plugin{
			{
				Shortname: "test-plugin",
				Releases: []Release{
					{
						Version: "1.0.0",
						Runtime: map[string]string{"node": "19"},
					},
				},
			},
		},
	}

	err := validateRuntimeVersions(pluginList)
	require.NotNil(t, err)
	require.ErrorContains(t, err, "invalid Node.js version '19'")
	require.ErrorContains(t, err, "test-plugin")
	require.ErrorContains(t, err, "Only LTS major versions are allowed")
}

func TestValidateRuntimeVersionsInvalidOldVersion(t *testing.T) {
	pluginList := &PluginList{
		Plugins: []Plugin{
			{
				Shortname: "test-plugin",
				Releases: []Release{
					{
						Version: "1.0.0",
						Runtime: map[string]string{"node": "10"},
					},
				},
			},
		},
	}

	err := validateRuntimeVersions(pluginList)
	require.NotNil(t, err)
	require.ErrorContains(t, err, "invalid Node.js version '10'")
}

func TestValidateRuntimeVersionsNoRuntime(t *testing.T) {
	pluginList := &PluginList{
		Plugins: []Plugin{
			{
				Shortname: "test-plugin",
				Releases: []Release{
					{
						Version: "1.0.0",
					},
				},
			},
		},
	}

	err := validateRuntimeVersions(pluginList)
	require.Nil(t, err)
}

func TestValidatePluginManifestWithInvalidRuntime(t *testing.T) {
	invalidManifest := `
[[Plugin]]
  Shortname = "test-app"
  Binary = "stripe-cli-test-app"
  MagicCookieValue = "TEST-COOKIE"

  [[Plugin.Release]]
    Arch = "amd64"
    OS = "darwin"
    Version = "1.0.0"
    Sum = "abcdef1234567890"
    Runtime = {node = "17"}
`

	_, err := validatePluginManifest([]byte(invalidManifest))
	require.NotNil(t, err)
	require.ErrorContains(t, err, "invalid Node.js version '17'")
}

func TestValidatePluginManifestWithValidRuntime(t *testing.T) {
	validManifest := `
[[Plugin]]
  Shortname = "test-app"
  Binary = "stripe-cli-test-app"
  MagicCookieValue = "TEST-COOKIE"

  [[Plugin.Release]]
    Arch = "amd64"
    OS = "darwin"
    Version = "1.0.0"
    Sum = "abcdef1234567890"
    Runtime = {node = "24"}
`

	pluginList, err := validatePluginManifest([]byte(validManifest))
	require.Nil(t, err)
	require.NotNil(t, pluginList)
	require.Equal(t, 1, len(pluginList.Plugins))
	require.Equal(t, "24", pluginList.Plugins[0].Releases[0].Runtime["node"])
}

func TestRefreshPluginSucceedsIfAdditionalManifestNotFound(t *testing.T) {
	fs := setUpFS()
	config := &TestConfig{}
	config.InitConfig()
	manifestContent, _ := os.ReadFile("./test_artifacts/plugins.toml")

	artifactoryServer := httptest.NewServer(http.HandlerFunc(func(res http.ResponseWriter, req *http.Request) {
		switch req.URL.String() {
		case "/plugins.toml":
			res.Write(manifestContent)
		case "/plugins-nonexistent.toml":
			res.WriteHeader(http.StatusNotFound)
		default:
			t.Errorf("Received an unexpected request URL: %s", req.URL.String())
		}
	}))
	defer func() { artifactoryServer.Close() }()

	stripeServer := httptest.NewServer(http.HandlerFunc(func(res http.ResponseWriter, req *http.Request) {
		switch url := req.URL.String(); url {
		case "/v1/stripecli/get-plugin-url":
			pd := requests.PluginData{
				PluginBaseURL:       artifactoryServer.URL,
				AdditionalManifests: []string{"plugins-nonexistent.toml"},
			}
			body, err := json.Marshal(pd)
			if err != nil {
				t.Error(err)
			}
			res.Write(body)
		default:
			t.Errorf("Received an unexpected request URL: %s", req.URL.String())
		}
	}))

	err := RefreshPluginManifest(context.Background(), config, fs, stripeServer.URL)
	require.Nil(t, err)
}

func TestAddPluginToListSortsBySemver(t *testing.T) {
	pluginList := &PluginList{
		Plugins: []Plugin{
			{
				Shortname:        "test-plugin",
				MagicCookieValue: "TEST-COOKIE-123",
				Releases: []Release{
					{Version: "1.0.0", OS: "darwin", Arch: "amd64"},
					{Version: "1.2.0", OS: "darwin", Arch: "amd64"},
				},
			},
		},
	}

	// Add a plugin with versions that would be sorted incorrectly by string comparison
	newPlugin := Plugin{
		Shortname:        "test-plugin",
		MagicCookieValue: "TEST-COOKIE-123",
		Releases: []Release{
			{Version: "1.10.0", OS: "darwin", Arch: "amd64"}, // String comparison would put this before 1.2.0
			{Version: "1.9.0", OS: "darwin", Arch: "amd64"},  // Should come after 1.2.0 but before 1.10.0
			{Version: "2.0.0", OS: "darwin", Arch: "amd64"},
			{Version: "1.0.1", OS: "darwin", Arch: "amd64"}, // Should come after 1.0.0 but before 1.2.0
		},
	}

	addPluginToList(pluginList, newPlugin)

	// Verify the plugin was merged (not added as a new one)
	require.Equal(t, 1, len(pluginList.Plugins))

	// Verify all releases are present
	require.Equal(t, 6, len(pluginList.Plugins[0].Releases))

	// Verify they are sorted by semver (not by string comparison)
	expectedOrder := []string{"1.0.0", "1.0.1", "1.2.0", "1.9.0", "1.10.0", "2.0.0"}
	for i, release := range pluginList.Plugins[0].Releases {
		require.Equal(t, expectedOrder[i], release.Version,
			"Expected release %d to be version %s, but got %s", i, expectedOrder[i], release.Version)
	}
}

func TestRefreshPluginManifestCreatesConfigDirectory(t *testing.T) {
	// Create a test config that uses a non-root directory
	testConfigPath := "/test-config-dir"
	customConfig := &CustomTestConfig{
		customConfigPath: testConfigPath,
	}
	customConfig.InitConfig()

	// Create a fresh filesystem without the config directory
	fs := afero.NewMemMapFs()

	manifestContent, _ := os.ReadFile("./test_artifacts/plugins.toml")
	testServers := setUpServers(t, manifestContent, nil)
	defer func() { testServers.CloseAll() }()

	// Verify the config directory doesn't exist yet
	exists, err := afero.DirExists(fs, testConfigPath)
	require.Nil(t, err)
	require.False(t, exists)

	// Refresh the manifest which should create the config directory
	err = RefreshPluginManifest(context.Background(), customConfig, fs, testServers.StripeServer.URL)
	require.Nil(t, err)

	// Verify the config directory now exists
	exists, err = afero.DirExists(fs, testConfigPath)
	require.Nil(t, err)
	require.True(t, exists)

	// Verify the plugins.toml file was created
	pluginManifestPath := testConfigPath + "/plugins.toml"
	exists, err = afero.Exists(fs, pluginManifestPath)
	require.Nil(t, err)
	require.True(t, exists)
}

func TestRefreshPluginManifestRefusesSymlink(t *testing.T) {
	tempDir := t.TempDir()
	customConfig := &CustomTestConfig{
		customConfigPath: tempDir,
	}
	customConfig.InitConfig()

	fs := afero.NewOsFs()
	manifestContent, err := os.ReadFile("./test_artifacts/plugins_updated.toml")
	require.NoError(t, err)

	testServers := setUpServers(t, manifestContent, nil)
	defer func() { testServers.CloseAll() }()

	victimFile := filepath.Join(tempDir, "victim.toml")
	require.NoError(t, os.WriteFile(victimFile, []byte("original = true\n"), 0o644))

	pluginManifestPath := filepath.Join(tempDir, "plugins.toml")
	require.NoError(t, os.Symlink(victimFile, pluginManifestPath))

	err = RefreshPluginManifest(context.Background(), customConfig, fs, testServers.StripeServer.URL)
	require.ErrorContains(t, err, "symlink")

	victimContents, err := os.ReadFile(victimFile)
	require.NoError(t, err)
	require.Equal(t, "original = true\n", string(victimContents))
}

func TestRefreshPluginManifestRefusesSymlinkedParent(t *testing.T) {
	tempDir := t.TempDir()
	victimDir := filepath.Join(tempDir, "victim-config")
	require.NoError(t, os.MkdirAll(victimDir, 0o755))

	configPath := filepath.Join(tempDir, "config-link")
	require.NoError(t, os.Symlink(victimDir, configPath))

	customConfig := &CustomTestConfig{
		customConfigPath: configPath,
	}
	customConfig.InitConfig()

	fs := afero.NewOsFs()
	manifestContent, err := os.ReadFile("./test_artifacts/plugins_updated.toml")
	require.NoError(t, err)

	testServers := setUpServers(t, manifestContent, nil)
	defer func() { testServers.CloseAll() }()

	err = RefreshPluginManifest(context.Background(), customConfig, fs, testServers.StripeServer.URL)
	require.ErrorContains(t, err, "symlink")

	_, err = os.Stat(filepath.Join(victimDir, "plugins.toml"))
	require.ErrorIs(t, err, os.ErrNotExist)
}
