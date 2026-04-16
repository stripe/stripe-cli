package plugins

import (
	"context"
	"fmt"
	"io"
	"path/filepath"

	"github.com/hashicorp/go-version"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/afero"
	"github.com/spf13/viper"

	"github.com/stripe/stripe-cli/pkg/ansi"
	"github.com/stripe/stripe-cli/pkg/config"
	"github.com/stripe/stripe-cli/pkg/gatedwriter"
)

// WithBackgroundUpdate runs fn (typically plugin.Run) while concurrently
// checking for and applying a plugin update. Update output is buffered until
// fn returns, then flushed to stderr so it never interleaves with plugin output.
func WithBackgroundUpdate(ctx context.Context, cfg config.IConfig, fs afero.Fs, baseURL string, plugin *Plugin, out io.Writer, fn func() error) error {
	w := gatedwriter.NewGatedWriter(out, 0)
	cleanupReady := make(chan struct{})
	updateDone := make(chan struct{})

	go func() {
		checkAndUpdate(ctx, cfg, fs, plugin, w, baseURL, cleanupReady)
		close(updateDone)
	}()

	err := fn()

	w.Open()
	CleanupAllClients()
	close(cleanupReady)
	<-updateDone

	return err
}

// checkAndUpdate refreshes the plugin manifest and, if a newer version is
// available, downloads and installs it.
func checkAndUpdate(ctx context.Context, cfg config.IConfig, fs afero.Fs, plugin *Plugin, out io.Writer, baseURL string, cleanupReady chan struct{}) {
	logger := log.WithFields(log.Fields{"prefix": "plugins.updater"})

	if !updatesEnabled(plugin.Shortname) {
		logger.Debugf("Automatic updates disabled")
		return
	}
	logger.Debugf("Automatic updates enabled")

	currentVersion, err := installedPluginVersion(cfg, fs, plugin)
	if err != nil {
		logger.Debugf("Error getting installed plugin version: %s", err)
		return
	}
	if currentVersion == "" {
		logger.Debugf("No installed plugin version found")
		return
	}
	if currentVersion == "local.build.dev" {
		logger.Debugf("Local build dev version found, skipping update check")
		return
	}

	spinner := ansi.StartNewSpinner(ansi.Faint(fmt.Sprintf("checking for '%s' updates...", plugin.Shortname)), out)

	if err := RefreshPluginManifest(ctx, cfg, fs, baseURL); err != nil {
		ansi.StopSpinner(spinner, ansi.Faint(fmt.Sprintf("'%s' update failed: couldn't find the plugins list", plugin.Shortname)), out)
		return
	}

	// Re-look up the plugin so we have the freshest release list.
	fresh, err := LookUpPlugin(ctx, cfg, fs, plugin.Shortname)
	if err != nil {
		ansi.StopSpinner(spinner, ansi.Faint(fmt.Sprintf("'%s' update failed: couldn't find the plugin", plugin.Shortname)), out)
		return
	}

	current, err := version.NewVersion(currentVersion)
	if err != nil {
		ansi.StopSpinner(spinner, ansi.Faint(fmt.Sprintf("'%s' update failed: couldn't parse current version", plugin.Shortname)), out)
		return
	}

	latestVersion := fresh.LookupLatestVersionForMajor(current.Segments()[0])
	if latestVersion == "" {
		ansi.StopSpinner(spinner, ansi.Faint(fmt.Sprintf("'%s' update failed: couldn't find latest release for major version %d", plugin.Shortname, current.Segments()[0])), out)
		return
	}

	latest, err := version.NewVersion(latestVersion)
	if err != nil {
		ansi.StopSpinner(spinner, ansi.Faint(fmt.Sprintf("'%s' update failed: couldn't parse latest version", plugin.Shortname)), out)
		return
	}

	if latest.Segments()[0] != current.Segments()[0] {
		ansi.StopSpinner(spinner, ansi.Faint(fmt.Sprintf("Skipping update for '%s' because you're on the latest major version", plugin.Shortname)), out)
		return
	}

	if !latest.GreaterThan(current) {
		ansi.StopSpinner(spinner, ansi.Faint(fmt.Sprintf("'%s' is already up to date", plugin.Shortname)), out)
		return
	}

	ansi.StartSpinner(spinner, ansi.Faint(fmt.Sprintf("updating '%s' to v%s...", plugin.Shortname, latestVersion)), out)

	if err := fresh.install(ctx, cfg, fs, latestVersion, baseURL, io.Discard, false); err != nil {
		ansi.StopSpinner(spinner, ansi.Faint(fmt.Sprintf("'%s' update to v%s failed", plugin.Shortname, latestVersion)), out)
		return
	}

	// Wait for cleanup to be ready because you can't delete a running executable on Windows.
	<-cleanupReady
	fresh.cleanUpPluginPath(cfg, fs, latestVersion)
	ansi.StopSpinner(spinner, ansi.Faint(fmt.Sprintf("'%s' updated to v%s — this version will be used next time", plugin.Shortname, latestVersion)), out)
}

// installedPluginVersion returns the version string of the locally installed
// plugin binary, or an empty string if no installation is found.
func installedPluginVersion(cfg config.IConfig, fs afero.Fs, p *Plugin) (string, error) {
	pluginDir := filepath.Join(getPluginsDir(cfg), p.Shortname)
	entries, err := afero.ReadDir(fs, pluginDir)
	if err != nil {
		// Directory doesn't exist means nothing is installed.
		return "", nil
	}
	for _, e := range entries {
		if e.IsDir() {
			return e.Name(), nil
		}
	}
	return "", nil
}

// updatesEnabled reports whether automatic updates are enabled for the given
// plugin. It checks the plugin-specific config first, then the global config.
// The default when neither is set is false (updates off).
func updatesEnabled(pluginName string) bool {
	logger := log.WithFields(log.Fields{"prefix": "plugins.updater"})
	pluginVal := viper.GetString(config.PluginConfigKey(pluginName, config.PluginConfigUpdatesField))
	if pluginVal != "" {
		logger.Debugf("Automatic updates for plugin '%s' enabled: %t", pluginName, pluginVal == "on")
		return pluginVal == "on"
	}
	globalVal := viper.GetString(config.PluginConfigKey(config.PluginConfigGlobalScope, config.PluginConfigUpdatesField))
	logger.Debugf("Automatic updates for plugins globally enabled: %t", globalVal == "on")
	return globalVal == "on"
}
