package plugins

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
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
// This is the whole budget for deciding, not just for the first request: an
// auto-upgrade proceeds only on a resolution that already carries a binary URL, so
// install never spends a second, unbounded lookup on the way to the download.
//
// The download itself is not bounded. A slow endpoint should not hold up the user's
// command, but a large binary on a slow connection is not a failure to cut short.
var autoUpgradeResolveTimeout = 3 * time.Second

// autoUpgradeCheckInterval is how long one upgrade check stands before another is
// worth making.
//
// Without a floor on how often it runs, an opted-in plugin spends a request on every
// single command -- and, on a machine that cannot reach the endpoint, the whole
// timeout above on every single command. What that buys is a slightly sooner upgrade,
// which is not something the user is waiting for; they are waiting for the command
// they typed.
//
// A few hours rather than a day: this should still land an upgrade the same working
// session it ships, and one check per plugin per morning is already close to free.
//
// Best-effort, not a guarantee. Two CLIs launched close enough together can both find
// the stamp expired before either has written it, and both check. Bounding it properly
// would mean a lock file and a policy for when to steal one from a process that died
// holding it -- machinery whose failure mode is "never checks again", to save at most
// one duplicate request per concurrent invocation. The measure that matters is
// requests per command, and that is already one per interval for anyone not running
// two plugins at the same instant.
//
// What does deserve locking is the install itself, which is a separate and older
// problem: cleanUpPluginPath deletes every version directory but the one it just
// wrote, so any two processes installing at once can pull a binary out from under a
// third, whether or not an auto-upgrade check is what set them off.
var autoUpgradeCheckInterval = 4 * time.Hour

// Swappable for test injection. These are every effect maybeAutoUpgrade has outside
// its own package: the setting (global config state), the lookup (network), the
// download (network and disk), the plugin's own PostInstall hook (a subprocess), and
// the clock the check interval is measured against.
var (
	pluginUpdatesEnabled   = config.PluginUpdatesEnabled
	autoUpgradeResolver    = ResolvePluginForUpgrade
	autoUpgradePostInstall = runPostInstallHook
	autoUpgradeNow         = time.Now
	autoUpgradeInstaller   = func(ctx context.Context, resolved *ResolvedPluginVersion, cfg config.IConfig, fs afero.Fs, apiBaseURL, dashboardBaseURL string) error {
		return resolved.Install(ctx, cfg, fs, apiBaseURL, dashboardBaseURL)
	}
)

// autoUpgradeCheckStampPath returns the file recording when this plugin was last
// checked for an upgrade.
//
// It sits beside the plugin's local metadata, which is already the directory for
// per-plugin state the CLI keeps for itself, and is per-plugin because the setting is
// too -- one plugin's check should not silence another's. Deliberately not a config
// field: this is written on plugin commands the user runs all day, and viper rewrites
// the entire config file per field.
//
// The extension keeps it out of getLocalPluginMetadataNames, which reads that
// directory to enumerate installed plugins and counts only `.toml` entries.
func autoUpgradeCheckStampPath(cfg config.IConfig, pluginName string) (string, error) {
	if err := ValidatePluginShortname(pluginName); err != nil {
		return "", err
	}

	return filepath.Join(getLocalPluginMetadataDir(cfg), pluginName+".last-upgrade-check"), nil
}

// removeAutoUpgradeCheckStamp deletes a plugin's check stamp, for an uninstall that
// should not leave anything of the plugin behind.
//
// A missing stamp is not an error: most uninstalls are of plugins that never had one,
// because the setting is off by default.
func removeAutoUpgradeCheckStamp(cfg config.IConfig, fs afero.Fs, pluginName string) error {
	path, err := autoUpgradeCheckStampPath(cfg, pluginName)
	if err != nil {
		return err
	}

	if err := fs.Remove(path); err != nil && !os.IsNotExist(err) {
		return err
	}

	return nil
}

// autoUpgradeCheckDue reports whether enough time has passed since the last upgrade
// check to be worth spending another request on. See autoUpgradeCheckInterval.
//
// Anything unreadable counts as due. A missing stamp is a first run, and a corrupt one
// is not worth refusing to upgrade over: being wrong in this direction costs the one
// request the stamp exists to save, while being wrong in the other direction means
// never upgrading again.
func autoUpgradeCheckDue(cfg config.IConfig, fs afero.Fs, pluginName string) bool {
	path, err := autoUpgradeCheckStampPath(cfg, pluginName)
	if err != nil {
		return true
	}

	body, err := afero.ReadFile(fs, path)
	if err != nil {
		return true
	}

	lastCheck, err := time.Parse(time.RFC3339, strings.TrimSpace(string(body)))
	if err != nil {
		return true
	}

	// A stamp in the future is a clock that has moved backwards, most often a machine
	// correcting its time. Waiting for the future to arrive could park the check for
	// years, so treat it as due and let the next check overwrite it.
	if now := autoUpgradeNow(); lastCheck.After(now) {
		return true
	} else if now.Sub(lastCheck) < autoUpgradeCheckInterval {
		return false
	}

	return true
}

// recordAutoUpgradeCheck claims the current interval for a check about to be made.
//
// Stored as text rather than leaned on the file's mtime, so that `cat`-ing it while
// working out why a plugin did or did not upgrade answers the question.
func recordAutoUpgradeCheck(cfg config.IConfig, fs afero.Fs, pluginName string) error {
	path, err := autoUpgradeCheckStampPath(cfg, pluginName)
	if err != nil {
		return err
	}

	if err := fs.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}

	return afero.WriteFile(fs, path, []byte(autoUpgradeNow().UTC().Format(time.RFC3339)+"\n"), 0644)
}

// maybeAutoUpgrade upgrades a plugin to the newest release available to this CLI
// before it runs, when the user turned `stripe plugin auto-update` on for it. It
// returns the plugin and version to run: the newly installed pair when it upgraded,
// and the pair it was given every other time.
//
// Most calls return without looking anything up. It runs on every invocation of an
// opted-in plugin, but only actually checks once per autoUpgradeCheckInterval.
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
	case pluginsDirOverride() != "":
		// A plugin loaded from a directory the user pointed the CLI at is not something
		// the metadata endpoint knows about, and installing over it would throw away
		// whatever was built there -- not just overwrite the one version, since the
		// install then deletes every other version directory beside it.
		//
		// Asked of both ways to point the CLI somewhere else, not just the compiled-in
		// one. A plugin developer working under STRIPE_PLUGINS_PATH is the likeliest
		// person here, and their build is the likeliest thing to lose.
		//
		// This is broader than it strictly has to be: someone using that variable to
		// relocate ordinary installs, rather than to develop a plugin, gives up
		// automatic upgrades for them. That is the safe direction -- they still get the
		// upgrade hint, and `stripe plugin install` still upgrades on request, whereas
		// guessing wrong the other way destroys work with no way to get it back.
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
	case !autoUpgradeCheckDue(cfg, fs, p.Shortname):
		// Checked recently enough. Ordered after the setting because that read is free
		// and this one touches the disk. See autoUpgradeCheckInterval.
		logger.Debug("skipping auto-upgrade, checked for one recently")
		return p, installedVersion
	}

	// Filled in here because the base URLs handed to Run carry only what the user
	// explicitly passed, and a metadata request has to name a real host.
	installAPIBaseURL, installDashboardBaseURL := resolveInstallBaseURLs(apiBaseURL, dashboardBaseURL)

	// Stamped before the lookup rather than after it, and regardless of how it turns
	// out. The request is the cost being rationed, so a lookup that fails or finds
	// nothing has to count; claiming the interval up front is what keeps two CLIs
	// started at once from both reading an expired stamp and both making the request,
	// which stamping afterwards would leave a whole lookup's worth of room for.
	//
	// It narrows that window rather than closing it -- read and write are still two
	// operations. See autoUpgradeCheckInterval for why that is where this stops.
	//
	// The cost of claiming first is that a process killed between here and the answer
	// defers the check by an interval, having learned nothing. For an upgrade the user
	// did not ask for, and can wait a few hours for, that is the safe direction to err
	// in -- interrupting a plugin download is a supported thing to do.
	if stampErr := recordAutoUpgradeCheck(cfg, fs, p.Shortname); stampErr != nil {
		// Nothing to do about it beyond checking again next time, which is what the
		// feature did before there was a stamp at all.
		logger.Debugf("could not record the upgrade check: %s", stampErr)
	}

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

	// A resolution carrying no binary URL is one install has to look up all over
	// again, and it would do that on ctx rather than the budget above -- so an
	// endpoint that just failed to answer in time would get a second turn with no
	// deadline at all, immediately before the command the user typed. Requiring the
	// URL here is what makes the budget cover the whole decision instead of only its
	// first request.
	//
	// The cached-metadata fallback is what usually lands here: it can name a version
	// but never a binary URL. Skipping it defers the upgrade by an
	// autoUpgradeCheckInterval, since the check it just declined is the one that got
	// stamped -- acceptable for something the user did not ask for, and the price of
	// not letting a machine that cannot reach the endpoint retry on every command. It
	// also means an auto-upgrade only ever installs a release a live metadata response
	// just offered, which is the guarantee ErrPluginRequiresNewerCLI's doc relies on.
	//
	// Nothing tells the user about the version named here, because the upgrade hint is
	// suppressed for a plugin that auto-updates; see CheckLatestPluginVersion. A plugin
	// that keeps landing here therefore stays quietly behind, which is worth reporting
	// from this side one day rather than by putting the hint's request back.
	if resolved.BinaryURL == "" {
		logger.Debugf("skipping auto-upgrade to v%s, the lookup returned no binary URL", resolved.Version)
		return p, installedVersion
	}

	// Handed ctx, not resolveCtx: deciding whether to upgrade is on a timer, the
	// download is not.
	if err := autoUpgradeInstaller(ctx, resolved, cfg, fs, installAPIBaseURL, installDashboardBaseURL); err != nil {
		// install already told the user it could not install the plugin, and the
		// version they have still works, so the command can carry on with it.
		logger.Debugf("auto-upgrade to v%s failed, continuing with v%s: %s", resolved.Version, installedVersion, err)
		return p, installedVersion
	}

	// Said out loud because the user did not ask for this: something they did not
	// type changed the version of the plugin they are about to run, and the install
	// spinner erases itself on the way out, leaving no trace that it happened.
	//
	// Reported after the install rather than announced before it, so it only ever
	// claims an upgrade that actually landed -- an install that breaks says so
	// itself, and until then the spinner is what covers the wait.
	//
	// On stderr, like every other thing the CLI says about itself: a plugin command
	// whose output is being piped somewhere should not have this turn up in it.
	color := ansi.Color(os.Stderr)
	fmt.Fprintln(os.Stderr, color.Faint(fmt.Sprintf(
		"Updated the %s plugin to v%s. Run `stripe plugin auto-update %s --disable` to stop updating it automatically.",
		p.Shortname, resolved.Version, p.Shortname,
	)).String())

	// Given the caller's base URLs rather than the resolved ones: a plugin reads an
	// empty value as "use your own default", and it should not inherit this CLI's just
	// because the CLI needed one to reach the metadata endpoint.
	autoUpgradePostInstall(ctx, cfg, fs, resolved.Plugin, resolved.Version, installedVersion, apiBaseURL, dashboardBaseURL, accessBaseURL)
	SendPluginLifecycleEvent(ctx, PluginUpgradedEvent, resolved.Version)

	return resolved.Plugin, resolved.Version
}
