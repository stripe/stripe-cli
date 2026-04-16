package plugins

import (
	"context"
	"fmt"
	"io"
	"os"
	"testing"

	"github.com/spf13/afero"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/stripe/stripe-cli/pkg/config"
)

func TestUpdatesEnabled(t *testing.T) {
	tests := []struct {
		name       string
		pluginVal  string
		globalVal  string
		pluginName string
		want       bool
	}{
		{
			name: "no config set defaults to off",
			want: false,
		},
		{
			name:      "global on enables updates",
			globalVal: "on",
			want:      true,
		},
		{
			name:      "global off disables updates",
			globalVal: "off",
			want:      false,
		},
		{
			name:       "plugin-specific on overrides global off",
			pluginVal:  "on",
			globalVal:  "off",
			pluginName: "apps",
			want:       true,
		},
		{
			name:       "plugin-specific off overrides global on",
			pluginVal:  "off",
			globalVal:  "on",
			pluginName: "apps",
			want:       false,
		},
		{
			name:       "plugin-specific on with no global",
			pluginVal:  "on",
			pluginName: "apps",
			want:       true,
		},
		{
			name:       "plugin-specific off with no global",
			pluginVal:  "off",
			pluginName: "apps",
			want:       false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			viper.Reset()
			t.Cleanup(viper.Reset)

			pluginName := tt.pluginName
			if pluginName == "" {
				pluginName = "apps"
			}

			if tt.pluginVal != "" {
				viper.Set(config.PluginConfigKey(pluginName, config.PluginConfigUpdatesField), tt.pluginVal)
			}
			if tt.globalVal != "" {
				viper.Set(config.PluginConfigKey(config.PluginConfigGlobalScope, config.PluginConfigUpdatesField), tt.globalVal)
			}

			assert.Equal(t, tt.want, updatesEnabled(pluginName))
		})
	}
}

// setUpFSWithInstalledVersion creates a memmap FS with the standard manifest
// and an already-installed binary at the given version (empty = none installed).
func setUpFSWithInstalledVersion(installedVersion string) afero.Fs {
	manifestContent, _ := os.ReadFile("./test_artifacts/plugins.toml")
	fs := afero.NewMemMapFs()
	afero.WriteFile(fs, "/plugins.toml", manifestContent, os.ModePerm)
	if installedVersion != "" {
		binaryPath := fmt.Sprintf("/plugins/appA/%s/stripe-cli-app-a%s", installedVersion, GetBinaryExtension())
		afero.WriteFile(fs, binaryPath, []byte("fake binary"), os.ModePerm)
	}
	return fs
}

func setViperUpdates(t *testing.T, pluginName, value string) {
	t.Helper()
	viper.Reset()
	t.Cleanup(viper.Reset)
	viper.Set(config.PluginConfigKey(pluginName, config.PluginConfigUpdatesField), value)
}

func setViperUpdatesGlobal(t *testing.T, value string) {
	t.Helper()
	viper.Reset()
	t.Cleanup(viper.Reset)
	viper.Set(config.PluginConfigKey(config.PluginConfigGlobalScope, config.PluginConfigUpdatesField), value)
}

func TestCheckAndUpdateSkipsWhenUpdatesDisabled(t *testing.T) {
	cfg := &TestConfig{}
	cfg.InitConfig()
	manifestContent, _ := os.ReadFile("./test_artifacts/plugins.toml")
	testServers := setUpServers(t, manifestContent, nil)
	defer testServers.CloseAll()

	// updates off (default) — no viper set
	viper.Reset()
	t.Cleanup(viper.Reset)

	fs := setUpFSWithInstalledVersion("1.0.1")
	plugin, _ := LookUpPlugin(context.Background(), cfg, fs, "appA")
	cleanupReady := make(chan struct{})
	close(cleanupReady)
	checkAndUpdate(context.Background(), cfg, fs, &plugin, io.Discard, testServers.StripeServer.URL, cleanupReady)

	file := fmt.Sprintf("/plugins/appA/2.0.1/stripe-cli-app-a%s", GetBinaryExtension())
	exists, _ := afero.Exists(fs, file)
	require.False(t, exists)
}

func TestCheckAndUpdateSkipsWhenNoInstalledVersion(t *testing.T) {
	cfg := &TestConfig{}
	cfg.InitConfig()
	manifestContent, _ := os.ReadFile("./test_artifacts/plugins.toml")
	testServers := setUpServers(t, manifestContent, nil)
	defer testServers.CloseAll()

	setViperUpdates(t, "appA", "on")

	fs := setUpFSWithInstalledVersion("") // nothing installed
	plugin, _ := LookUpPlugin(context.Background(), cfg, fs, "appA")
	cleanupReady := make(chan struct{})
	close(cleanupReady)
	checkAndUpdate(context.Background(), cfg, fs, &plugin, io.Discard, testServers.StripeServer.URL, cleanupReady)

	file := fmt.Sprintf("/plugins/appA/2.0.1/stripe-cli-app-a%s", GetBinaryExtension())
	exists, _ := afero.Exists(fs, file)
	require.False(t, exists)
}

func TestCheckAndUpdateSkipsLocalDevBuild(t *testing.T) {
	cfg := &TestConfig{}
	cfg.InitConfig()
	manifestContent, _ := os.ReadFile("./test_artifacts/plugins.toml")
	testServers := setUpServers(t, manifestContent, nil)
	defer testServers.CloseAll()

	setViperUpdates(t, "appA", "on")

	fs := setUpFSWithInstalledVersion("local.build.dev")
	plugin, _ := LookUpPlugin(context.Background(), cfg, fs, "appA")
	cleanupReady := make(chan struct{})
	close(cleanupReady)
	checkAndUpdate(context.Background(), cfg, fs, &plugin, io.Discard, testServers.StripeServer.URL, cleanupReady)

	file := fmt.Sprintf("/plugins/appA/2.0.1/stripe-cli-app-a%s", GetBinaryExtension())
	exists, _ := afero.Exists(fs, file)
	require.False(t, exists)
}

func TestCheckAndUpdateInstallsNewerMinorVersion(t *testing.T) {
	cfg := &TestConfig{}
	cfg.InitConfig()
	manifestContent, _ := os.ReadFile("./test_artifacts/plugins.toml")
	testServers := setUpServers(t, manifestContent, nil)
	defer testServers.CloseAll()

	setViperUpdates(t, "appA", "on")

	// Installed 2.0.0 (not in manifest, but on disk); latest same-major is 2.0.1.
	fs := setUpFSWithInstalledVersion("2.0.0")
	cfg.InstalledPlugins = []string{"appA"}

	plugin, _ := LookUpPlugin(context.Background(), cfg, fs, "appA")
	cleanupReady := make(chan struct{})
	close(cleanupReady)
	checkAndUpdate(context.Background(), cfg, fs, &plugin, io.Discard, testServers.StripeServer.URL, cleanupReady)

	newFile := fmt.Sprintf("/plugins/appA/2.0.1/stripe-cli-app-a%s", GetBinaryExtension())
	exists, err := afero.Exists(fs, newFile)
	require.Nil(t, err)
	require.True(t, exists, "expected 2.0.1 to be installed after update")
}

func TestCheckAndUpdateSkipsWhenAlreadyLatest(t *testing.T) {
	cfg := &TestConfig{}
	cfg.InitConfig()
	manifestContent, _ := os.ReadFile("./test_artifacts/plugins.toml")
	testServers := setUpServers(t, manifestContent, nil)
	defer testServers.CloseAll()

	setViperUpdates(t, "appA", "on")

	fs := setUpFSWithInstalledVersion("2.0.1")
	plugin, _ := LookUpPlugin(context.Background(), cfg, fs, "appA")
	cleanupReady := make(chan struct{})
	close(cleanupReady)
	checkAndUpdate(context.Background(), cfg, fs, &plugin, io.Discard, testServers.StripeServer.URL, cleanupReady)

	// Only 2.0.1 should exist — no spurious re-install or removal.
	file := fmt.Sprintf("/plugins/appA/2.0.1/stripe-cli-app-a%s", GetBinaryExtension())
	exists, _ := afero.Exists(fs, file)
	require.True(t, exists)
}

func TestCheckAndUpdateSkipsDifferentMajorVersion(t *testing.T) {
	cfg := &TestConfig{}
	cfg.InitConfig()
	manifestContent, _ := os.ReadFile("./test_artifacts/plugins.toml")
	testServers := setUpServers(t, manifestContent, nil)
	defer testServers.CloseAll()

	setViperUpdates(t, "appA", "on")

	// Installed 1.0.1; LookupLatestVersionForMajor(1) returns 1.0.1 (already latest),
	// so no update should happen and the 2.x binary must not be installed.
	fs := setUpFSWithInstalledVersion("1.0.1")
	plugin, _ := LookUpPlugin(context.Background(), cfg, fs, "appA")
	cleanupReady := make(chan struct{})
	close(cleanupReady)
	checkAndUpdate(context.Background(), cfg, fs, &plugin, io.Discard, testServers.StripeServer.URL, cleanupReady)

	v2File := fmt.Sprintf("/plugins/appA/2.0.1/stripe-cli-app-a%s", GetBinaryExtension())
	exists, _ := afero.Exists(fs, v2File)
	require.False(t, exists, "expected no cross-major update")
}

func TestCheckAndUpdateUsesGlobalEnableSetting(t *testing.T) {
	cfg := &TestConfig{}
	cfg.InitConfig()
	manifestContent, _ := os.ReadFile("./test_artifacts/plugins.toml")
	testServers := setUpServers(t, manifestContent, nil)
	defer testServers.CloseAll()

	// Enable via global config (no per-plugin key set).
	setViperUpdatesGlobal(t, "on")

	fs := setUpFSWithInstalledVersion("2.0.0")
	cfg.InstalledPlugins = []string{"appA"}

	plugin, _ := LookUpPlugin(context.Background(), cfg, fs, "appA")

	cleanupReady := make(chan struct{})
	close(cleanupReady)
	checkAndUpdate(context.Background(), cfg, fs, &plugin, io.Discard, testServers.StripeServer.URL, cleanupReady)

	newFile := fmt.Sprintf("/plugins/appA/2.0.1/stripe-cli-app-a%s", GetBinaryExtension())
	exists, err := afero.Exists(fs, newFile)
	require.Nil(t, err)
	require.True(t, exists, "expected update via global enable setting")
}
