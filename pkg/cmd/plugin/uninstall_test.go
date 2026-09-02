package plugin

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/spf13/afero"
	"github.com/stretchr/testify/require"

	"github.com/stripe/stripe-cli/pkg/config"
	"github.com/stripe/stripe-cli/pkg/errorcategory"
	"github.com/stripe/stripe-cli/pkg/plugins"
)

func TestRunUninstallCmdAllUninstallsEveryInstalledPlugin(t *testing.T) {
	cfg, fs, cleanup := setupPluginCommandTest(t)
	defer cleanup()

	appsBinary := installTestPlugin(t, cfg, fs, "apps")
	generateBinary := installTestPlugin(t, cfg, fs, "generate")

	// `generate` is only discoverable through its local metadata, like an
	// installation whose installed_plugins entry did not survive a config
	// migration from an older CLI version.
	require.NoError(t, plugins.RemoveInstalledPlugin(cfg, "generate"))
	require.Equal(t, []string{"apps"}, cfg.GetInstalledPlugins())

	output, err := executeUninstallCmd(t, cfg, fs, "--all")
	require.NoError(t, err)

	require.Contains(t, output, "✔ apps has been uninstalled.")
	require.Contains(t, output, "✔ generate has been uninstalled.")

	requirePluginUninstalled(t, cfg, fs, "apps", appsBinary)
	requirePluginUninstalled(t, cfg, fs, "generate", generateBinary)
	require.Empty(t, cfg.GetInstalledPlugins())
}

func TestRunUninstallCmdAllUninstallsPluginRecordedOnlyInConfig(t *testing.T) {
	cfg, fs, cleanup := setupPluginCommandTest(t)
	defer cleanup()

	appsBinary := installTestPlugin(t, cfg, fs, "apps")

	// Older CLI versions recorded installed_plugins without local metadata.
	require.NoError(t, fs.Remove(testPluginMetadataPath(cfg, "apps")))

	output, err := executeUninstallCmd(t, cfg, fs, "--all")
	require.NoError(t, err)

	require.Contains(t, output, "✔ apps has been uninstalled.")
	requirePluginUninstalled(t, cfg, fs, "apps", appsBinary)
	require.Empty(t, cfg.GetInstalledPlugins())
}

func TestRunUninstallCmdAllReportsNoOpWhenNoPluginsAreInstalled(t *testing.T) {
	cfg, fs, cleanup := setupPluginCommandTest(t)
	defer cleanup()

	output, err := executeUninstallCmd(t, cfg, fs, "--all")
	require.NoError(t, err)

	require.Contains(t, output, "No Stripe CLI plugins are installed, nothing to uninstall.")
	require.NotContains(t, output, "has been uninstalled")
}

func TestRunUninstallCmdAllStopsWhenAPluginCannotBeRemoved(t *testing.T) {
	cfg, fs, cleanup := setupPluginCommandTest(t)
	defer cleanup()

	appsBinary := installTestPlugin(t, cfg, fs, "apps")
	generateBinary := installTestPlugin(t, cfg, fs, "generate")

	output, err := executeUninstallCmd(t, cfg, afero.NewReadOnlyFs(fs), "--all")
	require.Error(t, err)

	require.NotContains(t, output, "has been uninstalled")
	requirePluginInstalled(t, cfg, fs, "apps", appsBinary)
	requirePluginInstalled(t, cfg, fs, "generate", generateBinary)
	require.Equal(t, []string{"apps", "generate"}, cfg.GetInstalledPlugins())
}

func TestRunUninstallCmdUninstallsOnlyTheNamedPlugin(t *testing.T) {
	cfg, fs, cleanup := setupPluginCommandTest(t)
	defer cleanup()

	appsBinary := installTestPlugin(t, cfg, fs, "apps")
	generateBinary := installTestPlugin(t, cfg, fs, "generate")

	output, err := executeUninstallCmd(t, cfg, fs, "apps")
	require.NoError(t, err)

	require.Contains(t, output, "✔ apps has been uninstalled.")
	require.NotContains(t, output, "generate")

	requirePluginUninstalled(t, cfg, fs, "apps", appsBinary)
	requirePluginInstalled(t, cfg, fs, "generate", generateBinary)
	require.Equal(t, []string{"generate"}, cfg.GetInstalledPlugins())
}

func TestUninstallCmdRejectsMissingAndConflictingArguments(t *testing.T) {
	tests := []struct {
		name          string
		args          []string
		errorContains string
	}{
		{
			name:          "without a plugin name",
			args:          nil,
			errorContains: "requires exactly 1 positional argument",
		},
		{
			name:          "with more than one plugin name",
			args:          []string{"apps", "generate"},
			errorContains: "requires exactly 1 positional argument",
		},
		{
			name:          "with both --all and a plugin name",
			args:          []string{"--all", "apps"},
			errorContains: "does not take a plugin name",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			cfg, fs, cleanup := setupPluginCommandTest(t)
			defer cleanup()

			appsBinary := installTestPlugin(t, cfg, fs, "apps")

			output, err := executeUninstallCmd(t, cfg, fs, test.args...)
			require.ErrorContains(t, err, test.errorContains)

			category, ok := errorcategory.Get(err)
			require.True(t, ok)
			require.Equal(t, errorcategory.UserInput, category)

			require.NotContains(t, output, "has been uninstalled")
			requirePluginInstalled(t, cfg, fs, "apps", appsBinary)
		})
	}
}

func executeUninstallCmd(t *testing.T, cfg *config.Config, fs afero.Fs, args ...string) (string, error) {
	t.Helper()

	uc := NewUninstallCmd(cfg)
	uc.fs = fs

	var output bytes.Buffer
	uc.Cmd.SetOut(&output)
	uc.Cmd.SetErr(&output)
	uc.Cmd.SetContext(context.Background())
	uc.Cmd.SetArgs(args)

	err := uc.Cmd.Execute()

	return output.String(), err
}

// installTestPlugin records a plugin as installed, writes its local metadata,
// and creates its plugin binary on disk. It returns the binary's path.
func installTestPlugin(t *testing.T, cfg *config.Config, fs afero.Fs, shortname string) string {
	t.Helper()

	plugin := plugins.Plugin{
		Shortname:        shortname,
		Binary:           "stripe-cli-" + shortname,
		MagicCookieValue: shortname + "-cookie",
		Releases: []plugins.Release{
			{
				Arch:    runtime.GOARCH,
				OS:      runtime.GOOS,
				Version: "1.0.0",
				Sum:     "abc123",
			},
		},
	}
	require.NoError(t, plugins.PersistInstalledPluginState(cfg, fs, plugin))

	binaryPath := filepath.Join(
		testPluginDir(cfg, shortname),
		"1.0.0",
		plugin.Binary+plugins.GetBinaryExtension(),
	)
	require.NoError(t, fs.MkdirAll(filepath.Dir(binaryPath), 0755))
	require.NoError(t, afero.WriteFile(fs, binaryPath, []byte("installed"), 0755))

	return binaryPath
}

func requirePluginUninstalled(t *testing.T, cfg *config.Config, fs afero.Fs, shortname, binaryPath string) {
	t.Helper()

	paths := []string{binaryPath, testPluginDir(cfg, shortname), testPluginMetadataPath(cfg, shortname)}
	for _, path := range paths {
		exists, err := afero.Exists(fs, path)
		require.NoError(t, err)
		require.Falsef(t, exists, "expected %s to be removed", path)
	}

	require.NotContains(t, cfg.GetInstalledPlugins(), shortname)
}

func requirePluginInstalled(t *testing.T, cfg *config.Config, fs afero.Fs, shortname, binaryPath string) {
	t.Helper()

	for _, path := range []string{binaryPath, testPluginMetadataPath(cfg, shortname)} {
		exists, err := afero.Exists(fs, path)
		require.NoError(t, err)
		require.Truef(t, exists, "expected %s to still exist", path)
	}
}

func testPluginDir(cfg *config.Config, shortname string) string {
	return filepath.Join(testConfigFolder(cfg), "plugins", shortname)
}

func testPluginMetadataPath(cfg *config.Config, shortname string) string {
	return filepath.Join(testConfigFolder(cfg), "plugin-metadata", shortname+".toml")
}

func testConfigFolder(cfg *config.Config) string {
	return cfg.GetConfigFolder(os.Getenv("XDG_CONFIG_HOME"))
}
