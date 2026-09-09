package plugin

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/spf13/afero"
	"github.com/stretchr/testify/require"

	"github.com/stripe/stripe-cli/pkg/plugins"
	"github.com/stripe/stripe-cli/pkg/requests"
)

// installablePluginBinaryContent is served as the plugin "binary" for tests in this file. It is
// not a real executable, so any lifecycle hook that tries to launch it as a plugin process is
// expected to fail; these tests exist to prove that failure is swallowed rather than propagated.
var installablePluginBinaryContent = []byte("not a real plugin binary")

func newInstallablePluginRegistryServers(t *testing.T, binaryContent []byte) *pluginRegistryServers {
	t.Helper()

	sum := sha256.Sum256(binaryContent)
	manifest := []byte(fmt.Sprintf(`[[Plugin]]
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
    Sum = "%s"
`, runtime.GOARCH, runtime.GOOS, hex.EncodeToString(sum[:])))

	artifactoryServer := httptest.NewServer(http.HandlerFunc(func(res http.ResponseWriter, req *http.Request) {
		switch req.URL.Path {
		case "/plugins.toml":
			_, _ = res.Write(manifest)
		default:
			_, _ = res.Write(binaryContent)
		}
	}))

	stripeServer := httptest.NewServer(http.HandlerFunc(func(res http.ResponseWriter, req *http.Request) {
		switch req.URL.Path {
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

	return &pluginRegistryServers{artifactory: artifactoryServer, stripe: stripeServer}
}

func TestRunInstallCmdSucceedsWhenPostInstallHookFails(t *testing.T) {
	cfg, fs, cleanup := setupPluginCommandTest(t)
	defer cleanup()

	servers := newInstallablePluginRegistryServers(t, installablePluginBinaryContent)
	defer servers.Close()

	ic := NewInstallCmd(cfg)
	ic.fs = fs
	ic.apiBaseURL = servers.stripe.URL
	ic.Cmd.SetContext(context.Background())

	require.NoError(t, ic.runInstallCmd(ic.Cmd, []string{"appA@2.0.1"}))

	configPath := cfg.GetConfigFolder(os.Getenv("XDG_CONFIG_HOME"))
	pluginBinaryPath := filepath.Join(configPath, "plugins", "appA", "2.0.1", "stripe-cli-app-a"+plugins.GetBinaryExtension())
	exists, err := afero.Exists(fs, pluginBinaryPath)
	require.NoError(t, err)
	require.True(t, exists)
}

func TestRunUpgradeCmdSucceedsWhenPostInstallHookFails(t *testing.T) {
	cfg, fs, cleanup := setupPluginCommandTest(t)
	defer cleanup()

	servers := newInstallablePluginRegistryServers(t, installablePluginBinaryContent)
	defer servers.Close()

	configPath := cfg.GetConfigFolder(os.Getenv("XDG_CONFIG_HOME"))
	oldBinaryPath := filepath.Join(configPath, "plugins", "appA", "1.0.1", "stripe-cli-app-a"+plugins.GetBinaryExtension())
	require.NoError(t, fs.MkdirAll(filepath.Dir(oldBinaryPath), 0755))
	require.NoError(t, afero.WriteFile(fs, oldBinaryPath, []byte("old version"), 0755))

	uc := NewUpgradeCmd(cfg)
	uc.fs = fs
	uc.apiBaseURL = servers.stripe.URL
	uc.Cmd.SetContext(context.Background())

	require.NoError(t, uc.runUpgradeCmd(uc.Cmd, []string{"appA"}))

	newBinaryPath := filepath.Join(configPath, "plugins", "appA", "2.0.1", "stripe-cli-app-a"+plugins.GetBinaryExtension())
	exists, err := afero.Exists(fs, newBinaryPath)
	require.NoError(t, err)
	require.True(t, exists)
}

func TestRunUninstallCmdSucceedsWhenPreUninstallHookFails(t *testing.T) {
	cfg, fs, cleanup := setupPluginCommandTest(t)
	defer cleanup()

	servers := newInstallablePluginRegistryServers(t, installablePluginBinaryContent)
	defer servers.Close()

	ic := NewInstallCmd(cfg)
	ic.fs = fs
	ic.apiBaseURL = servers.stripe.URL
	ic.Cmd.SetContext(context.Background())
	require.NoError(t, ic.runInstallCmd(ic.Cmd, []string{"appA@2.0.1"}))

	uc := NewUninstallCmd(cfg)
	uc.fs = fs
	uc.Cmd.SetContext(context.Background())

	require.NoError(t, uc.runUninstallCmd(uc.Cmd, []string{"appA"}))

	configPath := cfg.GetConfigFolder(os.Getenv("XDG_CONFIG_HOME"))
	pluginDir := filepath.Join(configPath, "plugins", "appA")
	exists, err := afero.Exists(fs, pluginDir)
	require.NoError(t, err)
	require.False(t, exists)
	require.Empty(t, cfg.GetInstalledPlugins())
}
