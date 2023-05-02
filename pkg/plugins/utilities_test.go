package plugins

import (
	"context"
	"os"
	"testing"

	"github.com/BurntSushi/toml"
	"github.com/spf13/afero"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/require"
)

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
