package plugins

import (
	"testing"

	goversion "github.com/hashicorp/go-version"

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

// With no known versions the CLI cannot tell a compatible plugin from an
// incompatible one, so it must not migrate at all.
func TestConfigV2NotReadyWithoutKnownVersions(t *testing.T) {
	withConfigV2MinimumVersions(t, map[string]string{})
	require.False(t, ConfigV2Ready())

	withConfigV2MinimumVersions(t, map[string]string{"apps": "1.5.0"})
	require.True(t, ConfigV2Ready())
}

// A blank or unparseable minimum is indistinguishable from an unmapped plugin:
// readsConfigV2 reports false either way, so the plugin is treated as
// incompatible and its owner is told to upgrade to a version that cannot be
// resolved. Guard the shipped map against that.
func TestConfigV2MinimumVersionsAreResolvable(t *testing.T) {
	for plugin, minimum := range configV2MinimumVersions {
		require.NotEmpty(t, minimum, "plugin %q has a blank minimum version", plugin)

		_, err := goversion.NewVersion(minimum)
		require.NoError(t, err, "plugin %q has an unparseable minimum version %q", plugin, minimum)
	}
}

// Keep this inventory in sync with the CLI hints and current Plugin Hub catalog,
// including conditional plugins. A plugin installed after migration needs a
// compatible release just as much as one already installed when migration runs.
// Legacy spec is not in the current Hub inventory; its rejection is tested below.
var supportedConfigV2Plugins = []string{
	"apps", "atlas", "directory", "generate", "health", "pay", "projects", "tools",
}

func missingConfigV2MinimumVersions(versions map[string]string) []string {
	if len(versions) == 0 {
		// An intentionally empty map disables the rollout.
		return nil
	}

	var missing []string
	for _, name := range supportedConfigV2Plugins {
		if versions[name] == "" {
			missing = append(missing, name)
		}
	}
	return missing
}

func TestConfigV2MinimumVersionsAreComplete(t *testing.T) {
	require.Empty(t, missingConfigV2MinimumVersions(configV2MinimumVersions),
		"migration must stay disabled until every supported plugin has a verified compatible release")
}

func TestConfigV2MinimumVersionsCompleteness(t *testing.T) {
	require.Empty(t, missingConfigV2MinimumVersions(map[string]string{}))

	complete := make(map[string]string, len(supportedConfigV2Plugins))
	for _, name := range supportedConfigV2Plugins {
		complete[name] = "1.0.0"
	}
	require.Empty(t, missingConfigV2MinimumVersions(complete))

	for _, name := range supportedConfigV2Plugins {
		t.Run(name, func(t *testing.T) {
			delete(complete, name)
			require.Equal(t, []string{name}, missingConfigV2MinimumVersions(complete))
			complete[name] = ""
			require.Equal(t, []string{name}, missingConfigV2MinimumVersions(complete))
			complete[name] = "1.0.0"
		})
	}
}

func TestConfigV2OmittedPluginBeforeAndAfterMigration(t *testing.T) {
	withConfigV2MinimumVersions(t, map[string]string{"known": "1.0.0"})
	t.Cleanup(viper.Reset)
	viper.Set(config.ConfigVersionName, config.ConfigVersionV2)

	for _, name := range supportedConfigV2Plugins {
		t.Run(name, func(t *testing.T) {
			fs := afero.NewMemMapFs()
			require.NoError(t, fs.MkdirAll("/plugins/"+name+"/9.9.9", 0755))
			cfg := &TestConfig{}
			cfg.InstalledPlugins = []string{name}

			// Installed before migration: even a high version cannot satisfy an
			// omitted threshold, so migration must wait.
			incompatibilities, err := ConfigV2Incompatibilities(cfg, fs)
			require.NoError(t, err)
			require.Equal(t, []ConfigV2Incompatibility{{
				Plugin: name, InstalledVersion: "9.9.9",
			}}, incompatibilities)

			// Installed after migration: execution remains guarded too.
			plugin := Plugin{Shortname: name}
			require.ErrorContains(t, plugin.refuseIfConfigTooNew("9.9.9"),
				"stripe plugin upgrade "+name)
		})
	}
}

func TestConfigV2LegacySpecRemainsIncompatible(t *testing.T) {
	// Use the shipped map: excluding spec from the active inventory must not
	// accidentally admit its old, flat-layout binary.
	require.NotContains(t, configV2MinimumVersions, "spec")
	require.False(t, ReadsConfigV2("spec", "0.2.1"))

	// Exercise an enabled rollout even when the shipped map is intentionally
	// empty. The explicit disabled state must remain valid for the whole suite.
	if !ConfigV2Ready() {
		withConfigV2MinimumVersions(t, map[string]string{"apps": "1.24.0"})
	}

	fs := afero.NewMemMapFs()
	require.NoError(t, fs.MkdirAll("/plugins/spec/0.2.1", 0755))
	cfg := &TestConfig{}
	cfg.InstalledPlugins = []string{"spec"}
	incompatibilities, err := ConfigV2Incompatibilities(cfg, fs)
	require.NoError(t, err)
	require.Equal(t, []ConfigV2Incompatibility{{
		Plugin: "spec", InstalledVersion: "0.2.1",
	}}, incompatibilities)

	t.Cleanup(viper.Reset)
	viper.Set(config.ConfigVersionName, config.ConfigVersionV2)
	plugin := Plugin{Shortname: "spec"}
	require.ErrorContains(t, plugin.refuseIfConfigTooNew("0.2.1"),
		"cannot read the new config file format")

	// An old installation can still run while migration is deferred.
	viper.Set(config.ConfigVersionName, nil)
	require.NoError(t, plugin.refuseIfConfigTooNew("0.2.1"))
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
	withConfigV2MinimumVersions(t, map[string]string{})

	plugin := Plugin{Shortname: "apps"}
	require.NoError(t, plugin.refuseIfConfigTooNew("1.4.0"))
}
