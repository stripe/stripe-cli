package plugin

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/99designs/keyring"
	"github.com/spf13/afero"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/require"

	"github.com/stripe/stripe-cli/pkg/config"
	"github.com/stripe/stripe-cli/pkg/plugins"
	"github.com/stripe/stripe-cli/pkg/requests"
)

type pluginRegistryServers struct {
	artifactory *httptest.Server
	stripe      *httptest.Server
}

func (s *pluginRegistryServers) Close() {
	s.artifactory.Close()
	s.stripe.Close()
}

func TestRunInstallCmdBackfillsLocalMetadataWhenVersionAlreadyInstalled(t *testing.T) {
	cfg, fs, cleanup := setupPluginCommandTest(t)
	defer cleanup()

	servers := newPluginRegistryServers(t, testPluginManifest())
	defer servers.Close()

	configPath := cfg.GetConfigFolder(os.Getenv("XDG_CONFIG_HOME"))
	pluginBinaryPath := filepath.Join(configPath, "plugins", "appA", "2.0.1", "stripe-cli-app-a"+plugins.GetBinaryExtension())
	require.NoError(t, fs.MkdirAll(filepath.Dir(pluginBinaryPath), 0755))
	require.NoError(t, afero.WriteFile(fs, pluginBinaryPath, []byte("already installed"), 0755))

	ic := NewInstallCmd(cfg)
	ic.fs = fs
	ic.apiBaseURL = servers.stripe.URL
	ic.Cmd.SetContext(context.Background())

	require.NoError(t, ic.runInstallCmd(ic.Cmd, []string{"appA@2.0.1"}))
	assertLocalMetadataBackfilled(t, cfg, fs, configPath)
}

func TestRunUpgradeCmdBackfillsLocalMetadataWhenAlreadyInstalled(t *testing.T) {
	cfg, fs, cleanup := setupPluginCommandTest(t)
	defer cleanup()

	servers := newPluginRegistryServers(t, testPluginManifest())
	defer servers.Close()

	configPath := cfg.GetConfigFolder(os.Getenv("XDG_CONFIG_HOME"))
	pluginBinaryPath := filepath.Join(configPath, "plugins", "appA", "2.0.1", "stripe-cli-app-a"+plugins.GetBinaryExtension())
	require.NoError(t, fs.MkdirAll(filepath.Dir(pluginBinaryPath), 0755))
	require.NoError(t, afero.WriteFile(fs, pluginBinaryPath, []byte("already installed"), 0755))

	uc := NewUpgradeCmd(cfg)
	uc.fs = fs
	uc.apiBaseURL = servers.stripe.URL
	uc.Cmd.SetContext(context.Background())

	require.NoError(t, uc.runUpgradeCmd(uc.Cmd, []string{"appA"}))
	assertLocalMetadataBackfilled(t, cfg, fs, configPath)
}

func setupPluginCommandTest(t *testing.T) (*config.Config, afero.Fs, func()) {
	t.Helper()

	xdgConfigHome := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", xdgConfigHome)

	profilesFile := filepath.Join(t.TempDir(), "config.toml")
	cfg := &config.Config{
		Color:        "auto",
		LogLevel:     "info",
		ProfilesFile: profilesFile,
		Profile:      config.Profile{ProfileName: "default"},
	}
	cfg.InitConfig()
	cfg.Profile.APIKey = "rk_test_11111111111111111111111111"
	config.KeyRing = keyring.NewArrayKeyring(nil)

	return cfg, afero.NewMemMapFs(), func() {
		viper.Reset()
	}
}

func newPluginRegistryServers(t *testing.T, manifest []byte) *pluginRegistryServers {
	t.Helper()

	artifactoryServer := httptest.NewServer(http.HandlerFunc(func(res http.ResponseWriter, req *http.Request) {
		switch req.URL.Path {
		case "/plugins.toml":
			_, _ = res.Write(manifest)
		default:
			t.Fatalf("unexpected artifactory request: %s", req.URL.String())
		}
	}))

	stripeServer := httptest.NewServer(http.HandlerFunc(func(res http.ResponseWriter, req *http.Request) {
		switch req.URL.Path {
		case "/v1/stripecli/get-plugin-url":
			body, err := json.Marshal(requests.PluginData{
				PluginBaseURL:       artifactoryServer.URL,
				AdditionalManifests: nil,
			})
			require.NoError(t, err)
			_, _ = res.Write(body)
		case "/v1/stripecli/get-plugin-metadata":
			body, err := json.Marshal(requests.PluginMetadata{
				BinaryURL:      artifactoryServer.URL + "/appA/2.0.1/" + runtime.GOOS + "/" + runtime.GOARCH + "/stripe-cli-app-a",
				PluginManifest: string(manifest),
			})
			require.NoError(t, err)
			_, _ = res.Write(body)
		default:
			t.Fatalf("unexpected stripe request: %s", req.URL.String())
		}
	}))

	return &pluginRegistryServers{
		artifactory: artifactoryServer,
		stripe:      stripeServer,
	}
}

func testPluginManifest() []byte {
	return []byte(fmt.Sprintf(`[[Plugin]]
  Shortname = "appA"
  Shortdesc = "App A"
  Binary = "stripe-cli-app-a"
  MagicCookieValue = "APP-A-COOKIE"

  [[Plugin.Command]]
    Name = "serve"
    Desc = "Serve app A"

  [[Plugin.Release]]
    Arch = "%s"
    OS = "%s"
    Version = "2.0.1"
    Sum = "abc123"
`, runtime.GOARCH, runtime.GOOS))
}

func assertLocalMetadataBackfilled(t *testing.T, cfg *config.Config, fs afero.Fs, configPath string) {
	t.Helper()

	metadataPath := filepath.Join(configPath, "plugin-metadata", "appA.toml")
	metadataExists, err := afero.Exists(fs, metadataPath)
	require.NoError(t, err)
	require.True(t, metadataExists)

	require.Equal(t, []string{"appA"}, cfg.GetInstalledPlugins())

	manifestPath := filepath.Join(configPath, "plugins.toml")
	manifestExists, err := afero.Exists(fs, manifestPath)
	require.NoError(t, err)
	if manifestExists {
		require.NoError(t, fs.Remove(manifestPath))
	}

	plugin, err := plugins.LookUpPlugin(context.Background(), cfg, fs, "appA")
	require.NoError(t, err)
	require.Equal(t, "stripe-cli-app-a", plugin.Binary)
	require.Len(t, plugin.Commands, 1)
	require.Equal(t, "serve", plugin.Commands[0].Name)
}
