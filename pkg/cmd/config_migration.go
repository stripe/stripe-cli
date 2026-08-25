package cmd

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/afero"
	"github.com/spf13/cobra"

	"github.com/stripe/stripe-cli/pkg/config"
	"github.com/stripe/stripe-cli/pkg/plugins"
	"github.com/stripe/stripe-cli/pkg/stripe"
)

// configMigration is the policy around config.MigrateConfigFile: whether the
// file should be rewritten in the v2 layout right now, and what the user is told
// about it. Every dependency is a field so the policy can be tested without a
// terminal or an installed plugin.
type configMigration struct {
	profilesFile         string
	needsMigration       func() bool
	pluginsReady         func() bool
	incompatibilities    func() ([]plugins.ConfigV2Incompatibility, error)
	installedPluginCount func() int
	upgradePlugin        func(plugins.ConfigV2Incompatibility) (string, error)
	migrate              func(path string) (bool, error)
	reload               func() error
	out                  io.Writer
}

func newConfigMigration(cfg *config.Config, ctx context.Context) configMigration {
	if ctx == nil {
		ctx = context.Background()
	}

	return configMigration{
		profilesFile:   cfg.ProfilesFile,
		needsMigration: config.NeedsMigration,
		pluginsReady:   plugins.ConfigV2Ready,
		incompatibilities: func() ([]plugins.ConfigV2Incompatibility, error) {
			return plugins.ConfigV2Incompatibilities(cfg, fs)
		},
		installedPluginCount: func() int {
			names, err := plugins.GetInstalledPluginNames(cfg, fs)
			if err != nil {
				return 0
			}

			return len(names)
		},
		upgradePlugin: func(incompatibility plugins.ConfigV2Incompatibility) (string, error) {
			return upgradePluginForConfigV2(ctx, cfg, fs, incompatibility)
		},
		migrate: config.MigrateConfigFile,
		reload:  config.ReloadConfigFile,
		out:     os.Stderr,
	}
}

// migrateConfigIfNeeded rewrites the config file to the v2 layout before the
// command the user asked for runs. It does not ask: plugins that are too old
// are upgraded, then the file is migrated.
func migrateConfigIfNeeded(cmd *cobra.Command) {
	if !migrationSafeCommand(cmd) {
		return
	}

	newConfigMigration(&Config, cmd.Context()).run()
}

// migrationSafeCommand reports whether it is acceptable to write status lines
// and to rewrite the config file while running this command. Help and shell
// completion output gets read by other programs, and a status line in the
// middle of it would be worse than a config file left in the old layout.
func migrationSafeCommand(cmd *cobra.Command) bool {
	for c := cmd; c != nil; c = c.Parent() {
		switch c.Name() {
		case "help", "completion":
			return false
		}
	}

	return true
}

// run migrates the config file when doing so is both needed and safe.
//
// Nothing here fails the command the user actually asked for. Every reason to
// stop is a reason the old layout stays, and the old layout keeps working: this
// version of the CLI reads both.
func (m configMigration) run() {
	logger := log.WithFields(log.Fields{
		"prefix": "cmd.configMigration.run",
	})

	if !m.needsMigration() {
		return
	}

	if _, err := os.Stat(m.profilesFile); err != nil {
		// No config file yet, so there is nothing to move. The first write picks
		// the layout.
		logger.Debugf("Skipping the config migration: %s", err)
		return
	}

	if !m.pluginsReady() {
		logger.Debug("Skipping the config migration: no plugin release is known to read the new config format yet")
		return
	}

	if !m.ensurePluginsCompatible(logger) {
		return
	}

	m.migrateAndReload()
}

// ensurePluginsCompatible upgrades any installed plugin that cannot read the v2
// layout. It returns false when an upgrade fails, in which case the config file
// is left alone.
func (m configMigration) ensurePluginsCompatible(logger *log.Entry) bool {
	fmt.Fprint(m.out, "checking installed plugins...")

	incompatibilities, err := m.incompatibilities()
	if err != nil {
		fmt.Fprintln(m.out)
		logger.Debugf("Skipping the config migration: could not check installed plugins: %s", err)
		return false
	}

	if len(incompatibilities) == 0 {
		switch n := m.pluginCount(); n {
		case 0:
			fmt.Fprintln(m.out, " none installed.")
		case 1:
			fmt.Fprintln(m.out, " 1 is compatible.")
		default:
			fmt.Fprintf(m.out, " all %d are compatible.\n", n)
		}

		return true
	}

	fmt.Fprintln(m.out)

	for _, incompatibility := range incompatibilities {
		newVersion, err := m.upgradeOne(incompatibility)
		if err != nil {
			logger.Debugf("could not upgrade %s: %s", incompatibility.Plugin, err)
			m.reportUpgradeFailure(incompatibility)
			return false
		}

		fmt.Fprintf(m.out, "✔ upgraded %s from v%s to v%s.\n", incompatibility.Plugin, incompatibility.InstalledVersion, newVersion)
	}

	return true
}

func (m configMigration) pluginCount() int {
	if m.installedPluginCount == nil {
		return 0
	}

	return m.installedPluginCount()
}

func (m configMigration) upgradeOne(incompatibility plugins.ConfigV2Incompatibility) (string, error) {
	if m.upgradePlugin == nil {
		return "", fmt.Errorf("plugin upgrade is not configured")
	}

	return m.upgradePlugin(incompatibility)
}

func (m configMigration) reportUpgradeFailure(incompatibility plugins.ConfigV2Incompatibility) {
	if incompatibility.MinimumVersion != "" {
		fmt.Fprintf(m.out, "! could not upgrade %s to the minimum required version (%s).\n", incompatibility.Plugin, incompatibility.MinimumVersion)
	} else {
		fmt.Fprintf(m.out, "! could not upgrade %s to a version that reads the new config format.\n", incompatibility.Plugin)
	}

	fmt.Fprintf(m.out, "  run `%s`, then try again.\n", incompatibility.UpgradeCommand())
	fmt.Fprintln(m.out, "your config file was not changed.")
}

func (m configMigration) migrateAndReload() {
	changed, err := m.migrate(m.profilesFile)
	if err != nil {
		fmt.Fprintf(m.out, "Could not update %s to the new config format: %s\n", m.profilesFile, err)
		fmt.Fprintln(m.out, "The file was left as it was, and the CLI still reads it.")

		return
	}

	if !changed {
		return
	}

	if err := m.reload(); err != nil {
		fmt.Fprintf(m.out, "Updated %s to the new config format, but could not re-read it: %s\n", m.profilesFile, err)
		return
	}

	fmt.Fprintf(m.out, "✔ updated %s to the new config format (backup saved to %s)\n", m.profilesFile, filepath.Base(m.profilesFile+config.ConfigBackupSuffix))
}

// upgradePluginForConfigV2 installs the latest release of a plugin that is too
// old to read the v2 layout. If that latest release is still below the minimum,
// it does not install anything.
func upgradePluginForConfigV2(ctx context.Context, cfg *config.Config, fs afero.Fs, incompatibility plugins.ConfigV2Incompatibility) (string, error) {
	apiBaseURL := stripe.DefaultAPIBaseURL
	dashboardBaseURL := stripe.DashboardBaseURLForAPIBaseURL(apiBaseURL)

	resolved, err := plugins.ResolvePluginForUpgrade(ctx, cfg, fs, incompatibility.Plugin, apiBaseURL, dashboardBaseURL)
	if err != nil {
		return "", err
	}

	if !plugins.ReadsConfigV2(incompatibility.Plugin, resolved.Version) {
		return "", fmt.Errorf("latest %s version %s cannot read the v2 config format (need %s)", incompatibility.Plugin, resolved.Version, incompatibility.MinimumVersion)
	}

	if err := resolved.Install(ctx, cfg, fs, apiBaseURL, dashboardBaseURL); err != nil {
		return "", err
	}

	return resolved.Version, nil
}
