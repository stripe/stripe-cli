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
	"sync"
	"testing"

	"github.com/spf13/afero"
	"github.com/stretchr/testify/require"

	"github.com/stripe/stripe-cli/pkg/config"
	"github.com/stripe/stripe-cli/pkg/errorcategory"
	"github.com/stripe/stripe-cli/pkg/plugins"
	"github.com/stripe/stripe-cli/pkg/requests"
)

// multiPluginRegistry serves version 2.0.1 of appA and appB, and answers every
// other plugin as not found. It records which plugins had metadata requested.
type multiPluginRegistry struct {
	*pluginRegistryServers

	mu        sync.Mutex
	requested []string
}

func (r *multiPluginRegistry) requestedPlugins() []string {
	r.mu.Lock()
	defer r.mu.Unlock()
	return append([]string(nil), r.requested...)
}

func multiPluginManifest(shortname string) string {
	sum := sha256.Sum256(installablePluginBinaryContent)
	return fmt.Sprintf(`[[Plugin]]
  Shortname = "%[1]s"
  Shortdesc = "%[1]s"
  Binary = "stripe-cli-%[1]s"
  MagicCookieValue = "%[1]s-COOKIE"

  [[Plugin.Release]]
    Arch = "%[2]s"
    OS = "%[3]s"
    Version = "2.0.1"
    Sum = "%[4]s"
`, shortname, runtime.GOARCH, runtime.GOOS, hex.EncodeToString(sum[:]))
}

func newMultiPluginRegistry(t *testing.T) *multiPluginRegistry {
	t.Helper()

	registry := &multiPluginRegistry{}
	artifactoryServer := httptest.NewServer(http.HandlerFunc(func(res http.ResponseWriter, req *http.Request) {
		_, _ = res.Write(installablePluginBinaryContent)
	}))

	stripeServer := httptest.NewServer(http.HandlerFunc(func(res http.ResponseWriter, req *http.Request) {
		require.Equal(t, "/v1/stripecli/get-plugin-metadata", req.URL.Path)

		pluginName := req.URL.Query().Get("plugin")
		registry.mu.Lock()
		registry.requested = append(registry.requested, pluginName)
		registry.mu.Unlock()

		if pluginName != "appA" && pluginName != "appB" {
			res.WriteHeader(http.StatusNotFound)
			_, _ = res.Write([]byte(`{"error":{"message":"not found"}}`))
			return
		}

		body, err := json.Marshal(requests.PluginMetadata{
			BinaryURL:      fmt.Sprintf("%s/%s/2.0.1/%s/%s/stripe-cli-%s", artifactoryServer.URL, pluginName, runtime.GOOS, runtime.GOARCH, pluginName),
			PluginManifest: multiPluginManifest(pluginName),
		})
		require.NoError(t, err)
		_, _ = res.Write(body)
	}))

	registry.pluginRegistryServers = &pluginRegistryServers{artifactory: artifactoryServer, stripe: stripeServer}
	return registry
}

func multiPluginBinaryPath(cfg *config.Config, shortname, version string) string {
	return filepath.Join(cfg.GetConfigFolder(os.Getenv("XDG_CONFIG_HOME")), "plugins", shortname, version, "stripe-cli-"+shortname+plugins.GetBinaryExtension())
}

func requireFileExists(t *testing.T, fs afero.Fs, path string, expected bool) {
	t.Helper()
	exists, err := afero.Exists(fs, path)
	require.NoError(t, err)
	require.Equalf(t, expected, exists, "existence of %s", path)
}

func newTestInstallCmd(cfg *config.Config, fs afero.Fs, apiBaseURL string) *InstallCmd {
	ic := NewInstallCmd(cfg)
	ic.fs = fs
	ic.apiBaseURL = apiBaseURL
	ic.Cmd.SetContext(context.Background())
	return ic
}

func newTestUpgradeCmd(cfg *config.Config, fs afero.Fs, apiBaseURL string) *UpgradeCmd {
	uc := NewUpgradeCmd(cfg)
	uc.fs = fs
	uc.apiBaseURL = apiBaseURL
	uc.Cmd.SetContext(context.Background())
	return uc
}

func writeOldPluginBinary(t *testing.T, cfg *config.Config, fs afero.Fs, shortname string) {
	t.Helper()
	path := multiPluginBinaryPath(cfg, shortname, "1.0.1")
	require.NoError(t, fs.MkdirAll(filepath.Dir(path), 0755))
	require.NoError(t, afero.WriteFile(fs, path, []byte("old version"), 0755))
}

func TestRunInstallCmdInstallsEveryNamedPlugin(t *testing.T) {
	cfg, fs, cleanup := setupPluginCommandTest(t)
	defer cleanup()

	registry := newMultiPluginRegistry(t)
	defer registry.Close()

	ic := newTestInstallCmd(cfg, fs, registry.stripe.URL)
	require.NoError(t, ic.runInstallCmd(ic.Cmd, []string{"appA@2.0.1", "appB"}))

	requireFileExists(t, fs, multiPluginBinaryPath(cfg, "appA", "2.0.1"), true)
	requireFileExists(t, fs, multiPluginBinaryPath(cfg, "appB", "2.0.1"), true)
	require.ElementsMatch(t, []string{"appA", "appB"}, cfg.GetInstalledPlugins())
}

func TestRunInstallCmdInstallsNothingWhenAnyPluginCannotBeResolved(t *testing.T) {
	cfg, fs, cleanup := setupPluginCommandTest(t)
	defer cleanup()
	cfg.Profile.AccountID = "acct_123"

	registry := newMultiPluginRegistry(t)
	defer registry.Close()

	ic := newTestInstallCmd(cfg, fs, registry.stripe.URL)
	err := ic.runInstallCmd(ic.Cmd, []string{"appA", "nonexistent-plugin", "appB"})
	require.ErrorContains(t, err, "nonexistent-plugin")

	require.Equal(t, []string{"appA", "nonexistent-plugin"}, registry.requestedPlugins())
	requireFileExists(t, fs, multiPluginBinaryPath(cfg, "appA", "2.0.1"), false)
	require.Empty(t, cfg.GetInstalledPlugins())
}

func TestRunInstallCmdRejectsRepeatedPluginBeforeResolvingAny(t *testing.T) {
	cfg, fs, cleanup := setupPluginCommandTest(t)
	defer cleanup()

	registry := newMultiPluginRegistry(t)
	defer registry.Close()

	ic := newTestInstallCmd(cfg, fs, registry.stripe.URL)
	err := ic.runInstallCmd(ic.Cmd, []string{"appA", "appB", "appA@2.0.1"})
	require.EqualError(t, err, `plugin "appA" was specified more than once`)
	category, ok := errorcategory.Get(err)
	require.True(t, ok)
	require.Equal(t, errorcategory.UserInput, category)

	require.Empty(t, registry.requestedPlugins())
	require.Empty(t, cfg.GetInstalledPlugins())
}

func TestRunUpgradeCmdUpgradesEveryNamedPlugin(t *testing.T) {
	cfg, fs, cleanup := setupPluginCommandTest(t)
	defer cleanup()

	registry := newMultiPluginRegistry(t)
	defer registry.Close()

	writeOldPluginBinary(t, cfg, fs, "appA")
	writeOldPluginBinary(t, cfg, fs, "appB")

	uc := newTestUpgradeCmd(cfg, fs, registry.stripe.URL)
	require.NoError(t, uc.runUpgradeCmd(uc.Cmd, []string{"appA", "appB"}))

	requireFileExists(t, fs, multiPluginBinaryPath(cfg, "appA", "2.0.1"), true)
	requireFileExists(t, fs, multiPluginBinaryPath(cfg, "appB", "2.0.1"), true)
}

func TestRunUpgradeCmdUpgradesNothingWhenAnyPluginCannotBeResolved(t *testing.T) {
	cfg, fs, cleanup := setupPluginCommandTest(t)
	defer cleanup()
	cfg.Profile.AccountID = "acct_123"

	registry := newMultiPluginRegistry(t)
	defer registry.Close()

	writeOldPluginBinary(t, cfg, fs, "appA")

	uc := newTestUpgradeCmd(cfg, fs, registry.stripe.URL)
	err := uc.runUpgradeCmd(uc.Cmd, []string{"appA", "nonexistent-plugin"})
	require.ErrorContains(t, err, "nonexistent-plugin")

	requireFileExists(t, fs, multiPluginBinaryPath(cfg, "appA", "2.0.1"), false)
	requireFileExists(t, fs, multiPluginBinaryPath(cfg, "appA", "1.0.1"), true)
}

func TestRunUpgradeCmdRejectsRepeatedPluginBeforeResolvingAny(t *testing.T) {
	cfg, fs, cleanup := setupPluginCommandTest(t)
	defer cleanup()

	registry := newMultiPluginRegistry(t)
	defer registry.Close()

	uc := newTestUpgradeCmd(cfg, fs, registry.stripe.URL)
	err := uc.runUpgradeCmd(uc.Cmd, []string{"appA", "appA"})
	require.EqualError(t, err, `plugin "appA" was specified more than once`)

	require.Empty(t, registry.requestedPlugins())
}
