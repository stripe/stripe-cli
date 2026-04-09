package plugins

import (
	"context"
	"fmt"
	"os"
	"testing"

	"github.com/BurntSushi/toml"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/require"

	"github.com/stripe/stripe-cli/pkg/requests"
)

func TestLookUpLatestVersion(t *testing.T) {
	fs := setUpFS()
	config := &TestConfig{}

	plugin, _ := LookUpPlugin(context.Background(), config, fs, "appA")
	version := plugin.LookUpLatestVersion()
	require.Equal(t, "2.0.1", version)
}

func TestLookupLatestVersionForMajor(t *testing.T) {
	fs := setUpFS()
	config := &TestConfig{}

	plugin, _ := LookUpPlugin(context.Background(), config, fs, "appA")
	version := plugin.LookupLatestVersionForMajor(1)
	require.Equal(t, "1.0.1", version)

	version = plugin.LookupLatestVersionForMajor(2)
	require.Equal(t, "2.0.1", version)

	version = plugin.LookupLatestVersionForMajor(3)
	require.Equal(t, "", version)
}

func TestInstall(t *testing.T) {
	fs := setUpFS()
	config := &TestConfig{}
	config.InitConfig()
	manifestContent, _ := os.ReadFile("./test_artifacts/plugins.toml")
	testServers := setUpServers(t, manifestContent, nil)

	plugin, _ := LookUpPlugin(context.Background(), config, fs, "appA")
	err := plugin.Install(context.Background(), config, fs, "2.0.1", testServers.StripeServer.URL)
	require.Nil(t, err)
	file := fmt.Sprintf("/plugins/appA/2.0.1/stripe-cli-app-a%s", GetBinaryExtension())
	fileExists, err := afero.Exists(fs, file)
	require.Nil(t, err)
	require.True(t, fileExists)

	require.Equal(t, []string{"appA"}, config.GetInstalledPlugins())
}

func TestInstallSucceedsIfNoAPIKey(t *testing.T) {
	fs := setUpFS()
	config := &TestConfig{}
	config.InitConfig()
	config.Profile.APIKey = ""
	manifestContent, _ := os.ReadFile("./test_artifacts/plugins.toml")
	testServers := setUpServers(t, manifestContent, nil)

	originalPluginBaseURL := requests.DefaultPluginData.PluginBaseURL
	requests.DefaultPluginData.PluginBaseURL = testServers.ArtifactoryServer.URL
	defer func() {
		requests.DefaultPluginData.PluginBaseURL = originalPluginBaseURL
	}()

	plugin, _ := LookUpPlugin(context.Background(), config, fs, "appA")
	err := plugin.Install(context.Background(), config, fs, "2.0.1", testServers.StripeServer.URL)
	require.Nil(t, err)
	file := fmt.Sprintf("/plugins/appA/2.0.1/stripe-cli-app-a%s", GetBinaryExtension())
	fileExists, err := afero.Exists(fs, file)
	require.Nil(t, err)
	require.True(t, fileExists)

	require.Equal(t, []string{"appA"}, config.GetInstalledPlugins())
}

func TestInstallFailsIfChecksumCouldNotBeFound(t *testing.T) {
	fs := setUpFS()
	config := &TestConfig{}
	config.InitConfig()
	manifestContent, _ := os.ReadFile("./test_artifacts/plugins.toml")
	testServers := setUpServers(t, manifestContent, nil)

	plugin, _ := LookUpPlugin(context.Background(), config, fs, "appA")
	err := plugin.Install(context.Background(), config, fs, "0.0.0", testServers.StripeServer.URL)
	require.EqualError(t, err, "could not locate a valid checksum for appA version 0.0.0")

	// Require that we don't save the binary if checkum does not match
	file := fmt.Sprintf("/plugins/appA/0.0.0/stripe-cli-app-a%s", GetBinaryExtension())
	fileExists, err := afero.Exists(fs, file)
	require.Nil(t, err)
	require.False(t, fileExists)

	require.Equal(t, 0, len(config.GetInstalledPlugins()))
}

func TestInstallationFailsIfChecksumDoesNotMatch(t *testing.T) {
	fs := setUpFS()
	config := &TestConfig{}
	config.InitConfig()
	manifestContent, _ := os.ReadFile("./test_artifacts/plugins.toml")
	testServers := setUpServers(t, manifestContent, nil)

	plugin, _ := LookUpPlugin(context.Background(), config, fs, "appB")
	err := plugin.Install(context.Background(), config, fs, "1.2.1", testServers.StripeServer.URL)
	require.EqualError(t, err, "installed plugin 'appB' could not be verified, aborting installation")

	// Require that we don't save the binary if checkum does not match
	file := fmt.Sprintf("/plugins/appB/1.2.1/stripe-cli-app-b%s", GetBinaryExtension())
	fileExists, err := afero.Exists(fs, file)
	require.Nil(t, err)
	require.False(t, fileExists)

	require.Equal(t, 0, len(config.GetInstalledPlugins()))
}

func TestInstallCleansOtherVersionsOfPlugin(t *testing.T) {
	fs := setUpFS()
	config := &TestConfig{}
	config.InitConfig()
	manifestContent, _ := os.ReadFile("./test_artifacts/plugins.toml")
	testServers := setUpServers(t, manifestContent, nil)

	// Download plugin version 0.0.1
	plugin, _ := LookUpPlugin(context.Background(), config, fs, "appA")
	err := plugin.Install(context.Background(), config, fs, "0.0.1", testServers.StripeServer.URL)
	require.Nil(t, err)
	file := fmt.Sprintf("/plugins/appA/0.0.1/stripe-cli-app-a%s", GetBinaryExtension())
	fileExists, _ := afero.Exists(fs, file)
	require.True(t, fileExists, "Test setup failed -- did not download plugin version 0.0.1")

	// Download valid plugin
	err = plugin.Install(context.Background(), config, fs, "2.0.1", testServers.StripeServer.URL)
	require.Nil(t, err)
	newFile := fmt.Sprintf("/plugins/appA/2.0.1/stripe-cli-app-a%s", GetBinaryExtension())
	fileExists, _ = afero.Exists(fs, newFile)
	require.True(t, fileExists, "Test setup failed -- did not download plugin version 2.0.1")

	// Require that the older version got removed from the fs
	fileExists, _ = afero.Exists(fs, file)
	require.False(t, fileExists, "Expected the original version of the plugin to be deleted.")

	require.Equal(t, []string{"appA"}, config.GetInstalledPlugins())
}

func TestInstallDoesNotCleanIfInstallFails(t *testing.T) {
	fs := setUpFS()
	config := &TestConfig{}
	config.InitConfig()
	manifestContent, _ := os.ReadFile("./test_artifacts/plugins.toml")
	testServers := setUpServers(t, manifestContent, nil)

	// Download valid plugin
	plugin, _ := LookUpPlugin(context.Background(), config, fs, "appA")
	err := plugin.Install(context.Background(), config, fs, "2.0.1", testServers.StripeServer.URL)
	require.Nil(t, err)
	file := fmt.Sprintf("/plugins/appA/2.0.1/stripe-cli-app-a%s", GetBinaryExtension())
	fileExists, _ := afero.Exists(fs, file)
	require.True(t, fileExists, "Test setup failed -- did not download valid plugin")

	// Install fails for the same plugin because the checksum could not be found in manifest
	err = plugin.Install(context.Background(), config, fs, "0.0.0", testServers.StripeServer.URL)
	require.EqualError(t, err, "could not locate a valid checksum for appA version 0.0.0")
	failedFile := fmt.Sprintf("/plugins/appA/0.0.0/stripe-cli-app-a%s", GetBinaryExtension())
	fileExists, _ = afero.Exists(fs, failedFile)
	require.False(t, fileExists, "Test setup failed -- did not expect plugin to be downloaded")

	// Require that we did not delete the initial version of the plugin
	fileExists, _ = afero.Exists(fs, file)
	require.True(t, fileExists, "Did not expect the original version of the plugin to be deleted.")
}

func TestCommandInfoParsedFromManifest(t *testing.T) {
	manifestContent, _ := os.ReadFile("./test_artifacts/plugins.toml")
	var pluginList PluginList
	_, err := toml.Decode(string(manifestContent), &pluginList)
	require.Nil(t, err)

	// appC should have Commands metadata
	var appC *Plugin
	for i, p := range pluginList.Plugins {
		if p.Shortname == "appC" {
			appC = &pluginList.Plugins[i]
			break
		}
	}
	require.NotNil(t, appC, "appC should be present in manifest")
	require.Equal(t, 2, len(appC.Commands))

	require.Equal(t, "create", appC.Commands[0].Name)
	require.Equal(t, "Create a resource", appC.Commands[0].Desc)
	require.Equal(t, 0, len(appC.Commands[0].Commands))

	require.Equal(t, "logs", appC.Commands[1].Name)
	require.Equal(t, "View logs", appC.Commands[1].Desc)
	require.Equal(t, 1, len(appC.Commands[1].Commands))
	require.Equal(t, "tail", appC.Commands[1].Commands[0].Name)
	require.Equal(t, "Tail logs in real-time", appC.Commands[1].Commands[0].Desc)
}

func TestCommandInfoNilWhenAbsent(t *testing.T) {
	manifestContent, _ := os.ReadFile("./test_artifacts/plugins.toml")
	var pluginList PluginList
	_, err := toml.Decode(string(manifestContent), &pluginList)
	require.Nil(t, err)

	// appA has no Commands metadata — field should be nil
	var appA *Plugin
	for i, p := range pluginList.Plugins {
		if p.Shortname == "appA" {
			appA = &pluginList.Plugins[i]
			break
		}
	}
	require.NotNil(t, appA)
	require.Nil(t, appA.Commands)
}

func TestUninstall(t *testing.T) {
	fs := setUpFS()
	config := &TestConfig{}
	config.InitConfig()
	manifestContent, _ := os.ReadFile("./test_artifacts/plugins.toml")
	testServers := setUpServers(t, manifestContent, nil)

	// install a plugin to be uninstalled
	plugin, _ := LookUpPlugin(context.Background(), config, fs, "appA")
	err := plugin.Install(context.Background(), config, fs, "2.0.1", testServers.StripeServer.URL)
	require.Nil(t, err)

	pluginDir := "/plugins/appA"
	err = plugin.Uninstall(context.Background(), config, fs)
	require.Nil(t, err)
	dirExists, _ := afero.Exists(fs, pluginDir)
	require.False(t, dirExists)

	require.Equal(t, 0, len(config.GetInstalledPlugins()))
}
