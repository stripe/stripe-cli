package plugins

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
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
	require.Equal(t, 2, len(pluginList.Plugins))
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
	require.ErrorContains(t, err, "Received an empty plugin manifest")
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
	require.ErrorContains(t, err, "Invalid Node.js version '19'")
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
	require.ErrorContains(t, err, "Invalid Node.js version '10'")
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
	require.ErrorContains(t, err, "Invalid Node.js version '17'")
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
		switch url := req.URL.String(); {
		case url == "/plugins.toml":
			res.Write(manifestContent)
		case url == "/plugins-nonexistent.toml":
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
