package plugin

import (
	"context"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/afero"

	"github.com/stripe/stripe-cli/pkg/config"
	"github.com/stripe/stripe-cli/pkg/plugins"
)

// runPostInstallHook calls the plugin's PostInstall RPC, if it implements one. This is
// best-effort: a failure here must never block `install`/`upgrade` from succeeding.
func runPostInstallHook(ctx context.Context, cfg *config.Config, fs afero.Fs, plugin *plugins.Plugin, version, previousVersion string) {
	defer plugins.CleanupAllClients()

	if err := plugin.PostInstall(ctx, cfg, fs, version, previousVersion); err != nil {
		log.WithFields(log.Fields{
			"prefix": "cmd.plugin.runPostInstallHook",
			"plugin": plugin.Shortname,
		}).Debugf("plugin PostInstall hook failed: %s", err)
	}
}

// runPreUninstallHook calls the plugin's PreUninstall RPC, if it implements one. This is
// best-effort: a failure here must never block `uninstall` from succeeding.
func runPreUninstallHook(ctx context.Context, cfg *config.Config, fs afero.Fs, plugin *plugins.Plugin, version string) {
	defer plugins.CleanupAllClients()

	if err := plugin.PreUninstall(ctx, cfg, fs, version); err != nil {
		log.WithFields(log.Fields{
			"prefix": "cmd.plugin.runPreUninstallHook",
			"plugin": plugin.Shortname,
		}).Debugf("plugin PreUninstall hook failed: %s", err)
	}
}
