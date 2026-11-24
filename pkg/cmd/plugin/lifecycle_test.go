package plugin

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
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

type cleanupFailureFs struct {
	*afero.OsFs
	path string
	err  error
}

func (fs cleanupFailureFs) RemoveAll(path string) error {
	if path == fs.path {
		return fs.err
	}
	return fs.OsFs.RemoveAll(path)
}

func TestRunInstallCmdCleanupFailureDoesNotReportSuccessOrRunHook(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("uses a harmless POSIX executable marker")
	}
	cfg, _, cleanup := setupPluginCommandTest(t)
	defer cleanup()
	marker := filepath.Join(t.TempDir(), "hook-executed")
	payload := []byte(fmt.Sprintf("#!/bin/sh\nprintf unused > %q\n", marker))
	servers := newInstallablePluginRegistryServers(t, payload)
	defer servers.Close()
	base := &afero.OsFs{}
	old := filepath.Join(cfg.GetConfigFolder(os.Getenv("XDG_CONFIG_HOME")), "plugins", "appA", "local.build.dev")
	require.NoError(t, base.MkdirAll(old, 0755))
	cleanupErr := errors.New("cleanup removal failed")
	ic := NewInstallCmd(cfg)
	ic.fs = cleanupFailureFs{OsFs: base, path: old, err: cleanupErr}
	ic.apiBaseURL = servers.stripe.URL
	ic.Cmd.SetContext(context.Background())

	reader, writer, err := os.Pipe()
	require.NoError(t, err)
	previousStdout := os.Stdout
	os.Stdout = writer
	defer func() { os.Stdout = previousStdout }()
	runErr := ic.runInstallCmd(ic.Cmd, []string{"appA@2.0.1"})
	os.Stdout = previousStdout
	require.NoError(t, writer.Close())
	output, err := io.ReadAll(reader)
	require.NoError(t, reader.Close())
	require.NoError(t, err)
	require.ErrorIs(t, runErr, cleanupErr)
	require.NotContains(t, string(output), "✔")
	_, err = os.Stat(marker)
	require.ErrorIs(t, err, os.ErrNotExist)
	_, err = base.Stat(filepath.Join(filepath.Dir(old), "2.0.1", "stripe-cli-app-a"))
	require.NoError(t, err)
}

func TestRunInstallCmdAlreadyInstalledRemovesStaleDevelopmentVersion(t *testing.T) {
	cfg, fs, cleanup := setupPluginCommandTest(t)
	defer cleanup()
	servers := newInstallablePluginRegistryServers(t, installablePluginBinaryContent)
	defer servers.Close()
	root := filepath.Join(cfg.GetConfigFolder(os.Getenv("XDG_CONFIG_HOME")), "plugins", "appA")
	for _, version := range []string{"local.build.dev", "2.0.1"} {
		dir := filepath.Join(root, version)
		require.NoError(t, fs.MkdirAll(dir, 0755))
		require.NoError(t, afero.WriteFile(fs, filepath.Join(dir, "stripe-cli-app-a")+plugins.GetBinaryExtension(), installablePluginBinaryContent, 0755))
	}
	ic := NewInstallCmd(cfg)
	ic.fs = fs
	ic.apiBaseURL = servers.stripe.URL
	ic.Cmd.SetContext(context.Background())
	require.NoError(t, ic.runInstallCmd(ic.Cmd, []string{"appA@2.0.1"}))
	_, err := fs.Stat(filepath.Join(root, "local.build.dev"))
	require.True(t, os.IsNotExist(err))
	_, err = fs.Stat(filepath.Join(root, "2.0.1"))
	require.NoError(t, err)
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
