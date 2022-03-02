package plugins

import (
	"context"
	"os"
	"testing"

	"github.com/spf13/afero"
	"github.com/stretchr/testify/require"
)

func TestLookUpLatestVersion(t *testing.T) {
	fs := setUpFS()
	config := &TestConfig{}

	plugin, _ := LookUpPlugin(context.Background(), config, fs, "appA")
	version := plugin.LookUpLatestVersion()
	require.Equal(t, "2.0.1", version)
}

func TestInstall(t *testing.T) {
	fs := setUpFS()
	config := &TestConfig{}
	manifestContent, _ := os.ReadFile("./test_artifacts/plugins.toml")
	testServers := setUpServers(t, manifestContent)

	plugin, _ := LookUpPlugin(context.Background(), config, fs, "appA")
	err := plugin.Install(context.Background(), config, fs, "2.0.1", testServers.StripeServer.URL)
	require.Nil(t, err)
	fileExists, err := afero.Exists(fs, "/plugins/appA/2.0.1/stripe-cli-app-a")
	require.Nil(t, err)
	require.True(t, fileExists)
}

func TestInstallFailsIfChecksumCouldNotBeFound(t *testing.T) {
	fs := setUpFS()
	config := &TestConfig{}
	manifestContent, _ := os.ReadFile("./test_artifacts/plugins.toml")
	testServers := setUpServers(t, manifestContent)

	plugin, _ := LookUpPlugin(context.Background(), config, fs, "appA")
	err := plugin.Install(context.Background(), config, fs, "0.0.0", testServers.StripeServer.URL)
	require.EqualError(t, err, "could not locate a valid checksum for appA version 0.0.0")

	// Require that we don't save the binary if checkum does not match
	fileExists, err := afero.Exists(fs, "/plugins/appA/0.0.1/stripe-cli-app-a")
	require.Nil(t, err)
	require.False(t, fileExists)
}

func TestInstallationFailsIfChecksumDoesNotMatch(t *testing.T) {
	fs := setUpFS()
	config := &TestConfig{}
	manifestContent, _ := os.ReadFile("./test_artifacts/plugins.toml")
	testServers := setUpServers(t, manifestContent)

	plugin, _ := LookUpPlugin(context.Background(), config, fs, "appB")
	err := plugin.Install(context.Background(), config, fs, "1.2.1", testServers.StripeServer.URL)
	require.EqualError(t, err, "installed plugin 'appB' could not be verified, aborting installation")

	// Require that we don't save the binary if checkum does not match
	fileExists, err := afero.Exists(fs, "/plugins/appB/1.2.1/stripe-cli-app-b")
	require.Nil(t, err)
	require.False(t, fileExists)
}

func TestInstallCleansOtherVersionsOfPlugin(t *testing.T) {
	fs := setUpFS()
	config := &TestConfig{}
	manifestContent, _ := os.ReadFile("./test_artifacts/plugins.toml")
	testServers := setUpServers(t, manifestContent)

	// Download plugin version 0.0.1
	plugin, _ := LookUpPlugin(context.Background(), config, fs, "appA")
	err := plugin.Install(context.Background(), config, fs, "0.0.1", testServers.StripeServer.URL)
	require.Nil(t, err)
	fileExists, _ := afero.Exists(fs, "/plugins/appA/0.0.1/stripe-cli-app-a")
	require.True(t, fileExists, "Test setup failed -- did not download plugin version 0.0.1")

	// Download valid plugin
	err = plugin.Install(context.Background(), config, fs, "2.0.1", testServers.StripeServer.URL)
	require.Nil(t, err)
	fileExists, _ = afero.Exists(fs, "/plugins/appA/2.0.1/stripe-cli-app-a")
	require.True(t, fileExists, "Test setup failed -- did not download plugin version 2.0.1")

	// Require that the older version got removed from the fs
	fileExists, _ = afero.Exists(fs, "/plugins/appA/0.0.1/stripe-cli-app-a")
	require.False(t, fileExists, "Expected the original version of the plugin to be deleted.")
}

func TestInstallDoesNotCleanIfInstallFails(t *testing.T) {
	fs := setUpFS()
	config := &TestConfig{}
	manifestContent, _ := os.ReadFile("./test_artifacts/plugins.toml")
	testServers := setUpServers(t, manifestContent)

	// Download valid plugin
	plugin, _ := LookUpPlugin(context.Background(), config, fs, "appA")
	err := plugin.Install(context.Background(), config, fs, "2.0.1", testServers.StripeServer.URL)
	require.Nil(t, err)
	fileExists, _ := afero.Exists(fs, "/plugins/appA/2.0.1/stripe-cli-app-a")
	require.True(t, fileExists, "Test setup failed -- did not download valid plugin")

	// Install fails for the same plugin because the checksum could not be found in manifest
	err = plugin.Install(context.Background(), config, fs, "0.0.0", testServers.StripeServer.URL)
	require.EqualError(t, err, "could not locate a valid checksum for appA version 0.0.0")
	fileExists, _ = afero.Exists(fs, "/plugins/appA/0.0.1/stripe-cli-app-a")
	require.False(t, fileExists, "Test setup failed -- did not expect plugin to be downloaded")

	// Require that we did not delete the initial version of the plugin
	fileExists, _ = afero.Exists(fs, "/plugins/appA/2.0.1/stripe-cli-app-a")
	require.True(t, fileExists, "Did not expect the original version of the plugin to be deleted.")
}
