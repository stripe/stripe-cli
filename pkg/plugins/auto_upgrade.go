package plugins

import (
	"context"
	"fmt"
	"os"
	"time"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/afero"

	"github.com/stripe/stripe-cli/pkg/ansi"
	"github.com/stripe/stripe-cli/pkg/config"
)

// autoUpgradeResolveTimeout bounds the lookup that decides whether a newer release
// exists. It is deliberately short because this runs before the command the user
// actually typed, where waiting is felt; it is a little longer than the upgrade
// hint's budget because giving up here means the upgrade silently never happens,
// while giving up on the hint only costs a message.
//
// Only the lookup is bounded. The download that may follow is not: a slow endpoint
// should not hold up the user's command, but a large binary on a slow connection is
// not a failure to cut short.
var autoUpgradeResolveTimeout = 3 * time.Second

// Swappable for test injection. These are every effect maybeAutoUpgrade has outside
// its own package: the setting (global config state), the lookup (network), the
// download (network and disk), and the plugin's own PostInstall hook (a subprocess).
var (
	pluginUpdatesEnabled   = config.PluginUpdatesEnabled
	autoUpgradeResolver    = ResolvePluginForUpgrade
	autoUpgradePostInstall = runPostInstallHook
	autoUpgradeInstaller   = func(ctx context.Context, resolved *ResolvedPluginVersion, cfg config.IConfig, fs afero.Fs, apiBaseURL, dashboardBaseURL string) error {
		return resolved.Install(ctx, cfg, fs, apiBaseURL, dashboardBaseURL)
	}
)

// maybeAutoUpgrade upgrades a plugin to the newest release available to this CLI
// before it runs, when the user turned `stripe plugin auto-update` on for it. It
// returns the plugin and version to run: the newly installed pair when it upgraded,
// and the pair it was given every other time.
//
// It returns no error, by design. The user asked to run a plugin, not to upgrade
// one, so every way this can come up short -- a setting that is off, an endpoint
// that will not answer, a download that breaks -- is handled the same way: leave
// the installed version alone and let the command run.
//
// The returned plugin matters as much as the version. Dispensing the new binary
// needs that release's checksum, which only the resolved metadata carries, so a
// caller that kept its own copy of the plugin would fail the integrity check.
func maybeAutoUpgrade(ctx context.Context, cfg *config.Config, fs afero.Fs, p *Plugin, installedVersion, apiBaseURL, dashboardBaseURL, accessBaseURL string) (*Plugin, string) {
	logger := log.WithFields(log.Fields{
		"prefix": "plugins.maybeAutoUpgrade",
		"plugin": p.Shortname,
	})

	// Normalized once here rather than at each use. Not every path that reaches a
	// plugin threads a context down, and the telemetry and install calls below read
	// from it without checking.
	if ctx == nil {
		ctx = context.Background()
	}

	switch {
	case PluginsPath != "":
		// A plugin loaded from a local path is not something the metadata endpoint
		// knows about, and overwriting it would throw away what was built there.
		return p, installedVersion
	case installedVersion == "":
		// Nothing to upgrade. Run's auto-install handles a missing binary before
		// reaching here, and it resolves the newest release on its own.
		return p, installedVersion
	case isLocalDevelopmentVersion(installedVersion):
		return p, installedVersion
	case !pluginUpdatesEnabled(p.Shortname):
		// Read before the lookup below so a user who left this off pays nothing for
		// the feature, not even one request per plugin command.
		return p, installedVersion
	}

	// Filled in here because the base URLs handed to Run carry only what the user
	// explicitly passed, and a metadata request has to name a real host.
	installAPIBaseURL, installDashboardBaseURL := resolveInstallBaseURLs(apiBaseURL, dashboardBaseURL)

	resolveCtx, cancel := context.WithTimeout(ctx, autoUpgradeResolveTimeout)
	defer cancel()

	resolved, err := autoUpgradeResolver(resolveCtx, cfg, fs, p.Shortname, installAPIBaseURL, installDashboardBaseURL)
	if err != nil {
		// A requires-a-newer-CLI answer lands here too, and is right to: the endpoint
		// withheld every release because this CLI is too old, so there is nothing to
		// upgrade to. Reporting it would interrupt a command that is about to run
		// perfectly well on the installed version. See ErrPluginRequiresNewerCLI.
		logger.Debugf("skipping auto-upgrade, could not resolve the latest version: %s", err)
		return p, installedVersion
	}
	if resolved == nil || resolved.Plugin == nil || resolved.Version == "" {
		logger.Debug("skipping auto-upgrade, the latest version could not be determined")
		return p, installedVersion
	}

	// Strictly newer, which is load-bearing rather than defensive. The endpoint
	// withholds releases this CLI is too old to run, so the newest one it offers can
	// be older than what is installed -- most plainly after the CLI itself is
	// downgraded. Accepting anything but ">" would roll the plugin back, and would do
	// it again on every command.
	if comparePluginVersions(installedVersion, resolved.Version) >= 0 {
		return p, installedVersion
	}

	// Said out loud because the user did not ask for this and is waiting on it. The
	// install spinner alone would look like the CLI hanging for no reason.
	color := ansi.Color(os.Stderr)
	fmt.Fprintln(os.Stderr, color.Faint(fmt.Sprintf(
		"Upgrading the %s plugin to v%s. Run `stripe plugin auto-update %s --disable` to stop upgrading it automatically.",
		p.Shortname, resolved.Version, p.Shortname,
	)).String())

	// Handed ctx, not resolveCtx: deciding whether to upgrade is on a timer, the
	// download is not.
	if err := autoUpgradeInstaller(ctx, resolved, cfg, fs, installAPIBaseURL, installDashboardBaseURL); err != nil {
		// install already told the user it could not install the plugin, and the
		// version they have still works, so the command can carry on with it.
		logger.Debugf("auto-upgrade to v%s failed, continuing with v%s: %s", resolved.Version, installedVersion, err)
		return p, installedVersion
	}

	// Given the caller's base URLs rather than the resolved ones: a plugin reads an
	// empty value as "use your own default", and it should not inherit this CLI's just
	// because the CLI needed one to reach the metadata endpoint.
	autoUpgradePostInstall(ctx, cfg, fs, resolved.Plugin, resolved.Version, installedVersion, apiBaseURL, dashboardBaseURL, accessBaseURL)
	SendPluginLifecycleEvent(ctx, PluginUpgradedEvent, resolved.Version)

	return resolved.Plugin, resolved.Version
}
