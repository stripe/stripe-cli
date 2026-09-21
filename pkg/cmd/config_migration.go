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

	"github.com/stripe/stripe-cli/pkg/ansi"
	"github.com/stripe/stripe-cli/pkg/config"
	"github.com/stripe/stripe-cli/pkg/errorcategory"
	"github.com/stripe/stripe-cli/pkg/plugins"
	"github.com/stripe/stripe-cli/pkg/stripe"
)

// configMigration is the policy around config.MigrateConfigFile: whether the
// file should be rewritten in the v2 layout right now, and what the user is told
// about it. Every dependency is a field so the policy can be tested without a
// terminal or an installed plugin.
type configMigration struct {
	profilesFile      string
	needsMigration    func() bool
	pluginsReady      func() bool
	incompatibilities func() ([]plugins.ConfigV2Incompatibility, error)
	upgradePlugin     func(plugins.ConfigV2Incompatibility) (string, error)
	migrate           func(path string) (bool, error)
	stampNew          func(path string) error
	reload            func() error
	out               io.Writer
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
		upgradePlugin: func(incompatibility plugins.ConfigV2Incompatibility) (string, error) {
			return upgradePluginForConfigV2(ctx, cfg, fs, incompatibility)
		},
		migrate:  config.MigrateConfigFile,
		stampNew: config.StampNewConfigFile,
		reload:   config.ReloadConfigFile,
		out:      os.Stderr,
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

// migrationSafeCommand reports whether it is acceptable to rewrite the config
// file, and to say anything at all, while running this command. Help and shell
// completion output gets read by other programs, and a plugin upgrade notice in
// the middle of it would be worse than a config file left in the old layout.
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
		// No config file yet, so there is nothing to move -- but there is a layout
		// to choose for whatever gets written first.
		m.stampNewConfigFile(logger, err)
		return
	}

	if !m.pluginsReady() {
		logger.Debug("Skipping the config migration: no plugin release is known to read the new config format yet")
		return
	}

	if !m.ensurePluginsCompatible(logger) {
		return
	}

	m.migrateAndReload(logger)
}

// stampNewConfigFile records the v2 layout in a config file that does not exist
// yet, so that the first write into it -- usually a login -- lands in the new
// layout directly.
//
// Without this, the first command writes the flat layout and the *next* command
// migrates it. That shows a brand-new user a migration notice before they have
// logged in, and leaves a backup file holding a copy of the credentials the
// previous command just wrote.
//
// Gated on plugins for the same reason the migration is: once a profile is written
// under the profiles table, a plugin too old to look there cannot find it. Silent,
// though. There is nothing to migrate and nothing at risk, so an incompatible
// plugin simply means the file keeps the flat layout until that plugin is
// upgraded -- and a status line here would land in front of a user who has not
// run anything yet.
func (m configMigration) stampNewConfigFile(logger *log.Entry, statErr error) {
	logger.Debugf("No config file to migrate: %s", statErr)

	if m.stampNew == nil || !m.pluginsReady() {
		return
	}

	incompatibilities, err := m.incompatibilities()
	if err != nil {
		logger.Debugf("Not recording the new config format: could not check installed plugins: %s", err)
		return
	}

	if len(incompatibilities) > 0 {
		logger.Debugf("Not recording the new config format: %s", incompatibilities[0].Error())
		return
	}

	if err := m.stampNew(m.profilesFile); err != nil {
		logger.Debugf("Could not record the new config format in %s: %s", m.profilesFile, err)
		return
	}

	if err := m.reload(); err != nil {
		logger.Debugf("Recorded the new config format in %s but could not re-read it: %s", m.profilesFile, err)
	}
}

// ensurePluginsCompatible upgrades any installed plugin that cannot read the v2
// layout. It returns false when an upgrade fails, in which case the config file
// is left alone.
func (m configMigration) ensurePluginsCompatible(logger *log.Entry) bool {
	incompatibilities, err := m.incompatibilities()
	if err != nil {
		logger.Debugf("Skipping the config migration: could not check installed plugins: %s", err)
		return false
	}

	for _, incompatibility := range incompatibilities {
		color := ansi.Color(m.out)
		fmt.Fprintln(m.out, color.Faint(fmt.Sprintf(
			"Upgrading the %s plugin so it can read the updated config file.", incompatibility.Plugin,
		)).String())

		newVersion, err := m.upgradeOne(incompatibility)
		if err != nil {
			logger.Debugf("Skipping the config migration: could not upgrade %s (%s): %s", incompatibility.Plugin, incompatibility.Error(), err)
			return false
		}

		logger.Debugf("Upgraded %s from v%s to v%s", incompatibility.Plugin, incompatibility.InstalledVersion, newVersion)
	}

	return true
}

func (m configMigration) upgradeOne(incompatibility plugins.ConfigV2Incompatibility) (string, error) {
	if m.upgradePlugin == nil {
		return "", errorcategory.New(errorcategory.Internal, "plugin upgrade is not configured")
	}

	return m.upgradePlugin(incompatibility)
}

func (m configMigration) migrateAndReload(logger *log.Entry) {
	changed, err := m.migrate(m.profilesFile)
	if err != nil {
		// MigrateConfigFile leaves the original in place when it cannot finish, and
		// this CLI reads that layout, so the command the user typed is unaffected.
		logger.Debugf("Could not update %s to the new config format, leaving it as it was: %s", m.profilesFile, err)
		return
	}

	if !changed {
		return
	}

	if err := m.reload(); err != nil {
		logger.Debugf("Updated %s to the new config format but could not re-read it: %s", m.profilesFile, err)
		return
	}

	logger.Debugf("Updated %s to the new config format, backup saved to %s", m.profilesFile, filepath.Base(m.profilesFile+config.ConfigBackupSuffix))
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
		return "", errorcategory.Errorf(errorcategory.Internal, "latest %s version %s cannot read the v2 config format (need %s)", incompatibility.Plugin, resolved.Version, incompatibility.MinimumVersion)
	}

	if err := resolved.Install(ctx, cfg, fs, apiBaseURL, dashboardBaseURL); err != nil {
		return "", err
	}

	return resolved.Version, nil
}
