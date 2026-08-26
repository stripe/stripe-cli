package plugins

import (
	"fmt"

	goversion "github.com/hashicorp/go-version"
	"github.com/spf13/afero"

	"github.com/stripe/stripe-cli/pkg/config"
	"github.com/stripe/stripe-cli/pkg/errorcategory"
)

// configV2MinimumVersions maps a plugin shortname to the lowest released version
// of that plugin which reads the v2 config layout.
//
// Plugins parse config.toml themselves rather than asking the CLI for it, and
// plugin auto-update is off by default, so an installed plugin binary can be
// arbitrarily old. Migrating the config file out from under one that only knows
// the flat layout would leave it unable to find any profile — hence the gate.
//
// The map is empty until those plugin releases exist. While it is empty the CLI
// cannot tell a compatible plugin from an incompatible one, so ConfigV2Ready
// reports false and the migration does not run at all.
var configV2MinimumVersions = map[string]string{}

// ConfigV2Ready reports whether the CLI knows which plugin releases can read a
// migrated config file. Until it does, migrating would break every installed
// plugin with no way to detect it, so callers must not migrate.
func ConfigV2Ready() bool {
	return len(configV2MinimumVersions) > 0
}

// ConfigV2Incompatibility is one installed plugin that cannot read a config file
// in the v2 layout.
type ConfigV2Incompatibility struct {
	// Plugin is the plugin's shortname, e.g. "apps".
	Plugin string

	// InstalledVersion is the version installed locally.
	InstalledVersion string

	// MinimumVersion is the lowest version that reads the v2 layout, or the empty
	// string when the CLI knows of no such version for this plugin.
	MinimumVersion string
}

// Error describes the incompatibility and how to resolve it.
func (i ConfigV2Incompatibility) Error() string {
	if i.MinimumVersion == "" {
		return fmt.Sprintf("the installed %s plugin (%s) cannot read the new config file format", i.Plugin, i.InstalledVersion)
	}

	return fmt.Sprintf("the installed %s plugin (%s) cannot read the new config file format; %s or later can", i.Plugin, i.InstalledVersion, i.MinimumVersion)
}

// UpgradeCommand returns the command that installs a version of this plugin able
// to read the v2 layout.
func (i ConfigV2Incompatibility) UpgradeCommand() string {
	return fmt.Sprintf("stripe plugin upgrade %s", i.Plugin)
}

// ConfigV2Incompatibilities lists the installed plugins that cannot read a config
// file in the v2 layout. An empty result means migrating is safe as far as
// plugins are concerned.
func ConfigV2Incompatibilities(cfg config.IConfig, fs afero.Fs) ([]ConfigV2Incompatibility, error) {
	names, err := GetInstalledPluginNames(cfg, fs)
	if err != nil {
		return nil, err
	}

	incompatibilities := make([]ConfigV2Incompatibility, 0)

	for _, name := range names {
		plugin := Plugin{Shortname: name}

		installedVersion, err := plugin.lookUpInstalledVersion(cfg, fs)
		if err != nil {
			return nil, err
		}

		if installedVersion == "" {
			// Recorded in config but no binary on disk, so there is nothing that
			// could read the config file. The next run installs a current version.
			continue
		}

		if readsConfigV2(name, installedVersion) {
			continue
		}

		incompatibilities = append(incompatibilities, ConfigV2Incompatibility{
			Plugin:           name,
			InstalledVersion: installedVersion,
			MinimumVersion:   configV2MinimumVersions[name],
		})
	}

	return incompatibilities, nil
}

// ReadsConfigV2 reports whether a specific plugin version can read the v2 config
// layout. Returns false when we could not determine.
func ReadsConfigV2(pluginName, installedVersion string) bool {
	return readsConfigV2(pluginName, installedVersion)
}

func readsConfigV2(pluginName, installedVersion string) bool {
	if isLocalDevelopmentVersion(installedVersion) {
		// A locally built plugin belongs to whoever built it, and its version says
		// nothing about what it supports.
		return true
	}

	minimum, known := configV2MinimumVersions[pluginName]
	if !known || minimum == "" {
		return false
	}

	installed, err := goversion.NewVersion(installedVersion)
	if err != nil {
		return false
	}

	required, err := goversion.NewVersion(minimum)
	if err != nil {
		return false
	}

	return !installed.LessThan(required)
}

// refuseIfConfigTooNew stops a plugin from running against a config file it
// cannot read. The migration gate keeps that from happening in the first place,
// but a config file can still arrive already migrated: a downgraded CLI, a
// dotfiles sync between machines, or a plugin binary restored from a backup.
func (p *Plugin) refuseIfConfigTooNew(installedVersion string) error {
	if !ConfigV2Ready() || !config.IsMigrated() {
		return nil
	}

	if readsConfigV2(p.Shortname, installedVersion) {
		return nil
	}

	incompatibility := ConfigV2Incompatibility{
		Plugin:           p.Shortname,
		InstalledVersion: installedVersion,
		MinimumVersion:   configV2MinimumVersions[p.Shortname],
	}

	return errorcategory.Errorf(errorcategory.UserInput, "%s. Run `%s` to update it", incompatibility.Error(), incompatibility.UpgradeCommand())
}
