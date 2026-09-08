package plugins

import (
	"testing"

	"github.com/spf13/afero"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/require"

	"github.com/stripe/stripe-cli/pkg/config"
)

func withConfigV2MinimumVersions(t *testing.T, versions map[string]string) {
	t.Helper()

	original := configV2MinimumVersions
	configV2MinimumVersions = versions
	t.Cleanup(func() {
		configV2MinimumVersions = original
	})
}

// Until plugin releases that read the v2 layout exist, the CLI cannot tell a
// compatible plugin from an incompatible one, so it must not migrate at all.
func TestConfigV2NotReadyWithoutKnownVersions(t *testing.T) {
	require.False(t, ConfigV2Ready())

	withConfigV2MinimumVersions(t, map[string]string{"apps": "1.5.0"})
	require.True(t, ConfigV2Ready())
}

func TestReadsConfigV2(t *testing.T) {
	withConfigV2MinimumVersions(t, map[string]string{
		"apps":     "1.5.0",
		"projects": "",
	})

	tests := []struct {
		name             string
		plugin           string
		installedVersion string
		want             bool
	}{
		{name: "above the minimum", plugin: "apps", installedVersion: "1.6.0", want: true},
		{name: "at the minimum", plugin: "apps", installedVersion: "1.5.0", want: true},
		{name: "below the minimum", plugin: "apps", installedVersion: "1.4.9", want: false},
		{name: "no compatible release yet", plugin: "projects", installedVersion: "9.9.9", want: false},
		{name: "plugin the CLI does not know", plugin: "somethingelse", installedVersion: "9.9.9", want: false},
		{name: "unparseable version", plugin: "apps", installedVersion: "not-a-version", want: false},
		// A locally built plugin belongs to whoever built it.
		{name: "local dev build", plugin: "apps", installedVersion: localDevelopmentVersion, want: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require.Equal(t, tt.want, readsConfigV2(tt.plugin, tt.installedVersion))
		})
	}
}

func TestConfigV2IncompatibilitiesReportsOldPlugin(t *testing.T) {
	withConfigV2MinimumVersions(t, map[string]string{"apps": "1.5.0", "projects": "2.0.0"})

	fs := afero.NewMemMapFs()
	require.NoError(t, fs.MkdirAll("/plugins/apps/1.4.0", 0755))
	require.NoError(t, fs.MkdirAll("/plugins/projects/2.1.0", 0755))

	cfg := &TestConfig{}
	cfg.InstalledPlugins = []string{"apps", "projects"}

	incompatibilities, err := ConfigV2Incompatibilities(cfg, fs)
	require.NoError(t, err)
	require.Equal(t, []ConfigV2Incompatibility{{
		Plugin:           "apps",
		InstalledVersion: "1.4.0",
		MinimumVersion:   "1.5.0",
	}}, incompatibilities)

	require.Contains(t, incompatibilities[0].Error(), "1.5.0 or later can")
	require.Equal(t, "stripe plugin upgrade apps", incompatibilities[0].UpgradeCommand())
}

// A plugin recorded in the config with no binary on disk cannot read anything.
// The next run installs a current version.
func TestConfigV2IncompatibilitiesIgnoresUninstalledPlugin(t *testing.T) {
	withConfigV2MinimumVersions(t, map[string]string{"apps": "1.5.0"})

	cfg := &TestConfig{}
	cfg.InstalledPlugins = []string{"apps"}

	incompatibilities, err := ConfigV2Incompatibilities(cfg, afero.NewMemMapFs())
	require.NoError(t, err)
	require.Empty(t, incompatibilities)
}

func TestConfigV2IncompatibilitiesEmptyWhenEverythingIsCurrent(t *testing.T) {
	withConfigV2MinimumVersions(t, map[string]string{"apps": "1.5.0"})

	fs := afero.NewMemMapFs()
	require.NoError(t, fs.MkdirAll("/plugins/apps/1.5.0", 0755))

	cfg := &TestConfig{}
	cfg.InstalledPlugins = []string{"apps"}

	incompatibilities, err := ConfigV2Incompatibilities(cfg, fs)
	require.NoError(t, err)
	require.Empty(t, incompatibilities)
}

// A migrated config file can reach a machine whose plugin is too old to read it:
// a downgraded CLI, synced dotfiles, or a restored binary.
func TestRefuseIfConfigTooNew(t *testing.T) {
	withConfigV2MinimumVersions(t, map[string]string{"apps": "1.5.0"})
	t.Cleanup(viper.Reset)

	plugin := Plugin{Shortname: "apps"}

	viper.Set(config.ConfigVersionName, config.ConfigVersionV2)
	err := plugin.refuseIfConfigTooNew("1.4.0")
	require.ErrorContains(t, err, "cannot read the new config file format")
	require.ErrorContains(t, err, "stripe plugin upgrade apps")

	require.NoError(t, plugin.refuseIfConfigTooNew("1.5.0"))

	// An unmigrated config file is readable by every version.
	viper.Set(config.ConfigVersionName, nil)
	require.NoError(t, plugin.refuseIfConfigTooNew("1.4.0"))
}

// With no known plugin versions the CLI never migrates, so it has no grounds to
// refuse a plugin either.
func TestRefuseIfConfigTooNewIsInertUntilVersionsAreKnown(t *testing.T) {
	t.Cleanup(viper.Reset)
	viper.Set(config.ConfigVersionName, config.ConfigVersionV2)

	plugin := Plugin{Shortname: "apps"}
	require.NoError(t, plugin.refuseIfConfigTooNew("1.4.0"))
}
