package plugins

import (
	"context"
	"fmt"
	"sort"

	goversion "github.com/hashicorp/go-version"
	"github.com/spf13/afero"

	"github.com/stripe/stripe-cli/pkg/config"
	"github.com/stripe/stripe-cli/pkg/errorcategory"
)

// pluginVersionSatisfiesMinimum reports whether an installed plugin version meets
// a declared minimum.
//
// Compared with goversion.NewVersion and LessThan, deliberately never with
// goversion.NewConstraint: a constraint reads a bare "1.17.0" as "= 1.17.0",
// which would turn a minimum into an exact pin.
//
// A version that does not parse — local.build.dev, or anything else a developer
// put there by hand — satisfies every minimum. Such a build belongs to whoever
// built it, its version string says nothing about what it can do, and the one
// thing enforcement must never do is block them or replace their build.
func pluginVersionSatisfiesMinimum(installedVersion, minimum string) bool {
	if minimum == "" {
		return true
	}
	if installedVersion == "" {
		return false
	}

	installed, err := goversion.NewVersion(installedVersion)
	if err != nil {
		return true
	}

	required, err := goversion.NewVersion(minimum)
	if err != nil {
		// Release validation keeps minimums to bare x.y.z, so this is bad data.
		// There is nothing coherent to enforce, and refusing to run over it would
		// break the peer for everyone.
		return true
	}

	return !installed.LessThan(required)
}

// installChain is the state one top-level install threads through its recursive
// dependency installs: which plugins are already on their way in (the cycle
// guard), and the post-install hooks owed once the whole chain has finished.
//
// Hooks are deferred rather than run as each dependency lands because a hook is
// plugin code that can call back into the CLI: a hook run mid-chain that calls
// RunPeerPlugin on a plugin this chain is still carrying would see it as absent
// or stale and start a second install of it, outside the cycle guard — and
// possibly of a different version than the one on its way in.
type installChain struct {
	inFlight      map[string]string
	deferredHooks []func()
	// hookBaseURLs are the explicit overrides the chain's creator could vouch
	// for, forwarded to the hook of every dependency installed along the chain.
	hookBaseURLs HookBaseURLs
}

func newInstallChain(hookBaseURLs HookBaseURLs) *installChain {
	return &installChain{
		inFlight:     map[string]string{},
		hookBaseURLs: hookBaseURLs,
	}
}

// runDeferredHooks runs the hooks owed by this chain, in install order. The
// install that created the chain calls it once its own outcome is decided —
// on failure too, since a dependency that landed before the failure is
// installed and owed its hook either way.
func (c *installChain) runDeferredHooks() {
	for _, hook := range c.deferredHooks {
		hook()
	}
}

// Swappable for test injection, shared by every path that brings a plugin up to
// a declared floor; see ensurePluginAtLeast.
var (
	minimumVersionResolver  = ResolvePluginForUpgrade
	minimumVersionInstaller func(ctx context.Context, resolved *ResolvedPluginVersion, cfg config.IConfig, fs afero.Fs, apiBaseURL, dashboardBaseURL string, chain *installChain) error
)

// Assigned here rather than at the declaration: install resolves its own
// dependencies through ensurePluginAtLeast, so a declaration-time closure over
// install is an initialization cycle.
func init() {
	minimumVersionInstaller = func(ctx context.Context, resolved *ResolvedPluginVersion, cfg config.IConfig, fs afero.Fs, apiBaseURL, dashboardBaseURL string, chain *installChain) error {
		return resolved.install(ctx, cfg, fs, apiBaseURL, dashboardBaseURL, chain)
	}
}

// ensurePluginAtLeast brings the named plugin up to a declared minimum version:
// when the installed version is missing or below the floor, it installs the
// newest release — the minimum is a floor, not a pin — and returns that
// resolution along with the previously installed version. A floor the installed
// version already satisfies returns nil with no work done, so the caller knows
// whether a post-install hook is owed.
//
// announce, when non-nil, runs once the floor is known to be unmet and before
// any network or download work, so a caller can say why the pause is about to
// happen without making the satisfied-check a second time itself.
//
// Errors are bare causes: the caller attributes them to whoever declared the
// floor, which is the one thing the two callers legitimately say differently.
func ensurePluginAtLeast(ctx context.Context, cfg config.IConfig, fs afero.Fs, pluginName, minimum, apiBaseURL, dashboardBaseURL string, announce func(), chain *installChain) (*ResolvedPluginVersion, string, error) {
	if err := ValidatePluginShortname(pluginName); err != nil {
		return nil, "", err
	}
	// Implicit installs must not replace builds in a developer's plugin directory.
	if pluginsDirOverride() != "" {
		return nil, "", nil
	}

	// Every error path inside the lookup already reports "", and unreadable is
	// handled as absent: the ways that can be wrong all end in installing a
	// release that was already there.
	installedVersion, _ := (&Plugin{Shortname: pluginName}).lookUpInstalledVersion(cfg, fs)
	if pluginVersionSatisfiesMinimum(installedVersion, minimum) {
		return nil, installedVersion, nil
	}

	if announce != nil {
		announce()
	}

	resolved, err := minimumVersionResolver(ctx, cfg, fs, pluginName, apiBaseURL, dashboardBaseURL)
	if err != nil {
		return nil, installedVersion, err
	}
	if resolved == nil || resolved.Plugin == nil || resolved.Version == "" {
		return nil, installedVersion, errorcategory.Errorf(errorcategory.API, "could not determine the newest release of the %s plugin", pluginName)
	}

	// The newest release the endpoint offers can sit below the minimum when it
	// withholds releases this CLI is too old to run. Installing it anyway would
	// claim success while leaving the requirement unmet, to be rediscovered as
	// a failure at run time.
	if !pluginVersionSatisfiesMinimum(resolved.Version, minimum) {
		return nil, installedVersion, errorcategory.Errorf(
			errorcategory.API,
			"v%s or newer is required, but the newest release available to this CLI is v%s. A newer Stripe CLI may be required: https://docs.stripe.com/stripe-cli/upgrade",
			minimum, resolved.Version,
		)
	}

	if err := minimumVersionInstaller(ctx, resolved, cfg, fs, apiBaseURL, dashboardBaseURL, chain); err != nil {
		return nil, installedVersion, err
	}

	return resolved, installedVersion, nil
}

// installPluginDependencies installs or upgrades the peer plugins a release
// declares minimum versions for, before the release itself is installed.
//
// chain carries the plugins already being installed up this call chain,
// including the requester itself, mapped to the versions going in. A dependency
// found there has its floor checked against that in-flight version rather than
// being recursed into, so a cycle ends without waving the requirement through.
// The recursive install resolves each dependency's own requirements the same
// way, so dependencies land transitively. Each installed dependency's
// post-install hook is deferred onto the chain; see installChain for why.
//
// A failure names the dependency and leaves the requester uninstalled, since the
// caller runs this before downloading the requester.
func installPluginDependencies(ctx context.Context, cfg config.IConfig, fs afero.Fs, requester *Plugin, apiBaseURL, dashboardBaseURL string, chain *installChain) error {
	dependencyNames := make([]string, 0, len(requester.MinPluginVersions))
	for dependencyName := range requester.MinPluginVersions {
		dependencyNames = append(dependencyNames, dependencyName)
	}
	sort.Strings(dependencyNames)

	for _, dependencyName := range dependencyNames {
		minimum := requester.MinPluginVersions[dependencyName]

		// Already being installed somewhere up this call chain. Recursing would
		// never end, but the version on its way in still has to clear this floor:
		// a cycle is a reason to stop recursing, not permission to skip the
		// requirement. The requester itself is always in flight, so a self-entry
		// — bad data that release validation rejects — ends here too.
		if inFlightVersion, alreadyInstalling := chain.inFlight[dependencyName]; alreadyInstalling {
			if pluginVersionSatisfiesMinimum(inFlightVersion, minimum) {
				continue
			}

			return dependencyInstallError(requester.Shortname, dependencyName, errorcategory.Errorf(
				errorcategory.API,
				"v%s or newer is required, but v%s is already being installed by this same operation",
				minimum, inFlightVersion,
			))
		}

		resolved, previousVersion, err := ensurePluginAtLeast(ctx, cfg, fs, dependencyName, minimum, apiBaseURL, dashboardBaseURL, nil, chain)
		if err != nil {
			return dependencyInstallError(requester.Shortname, dependencyName, err)
		}
		if resolved == nil {
			continue
		}

		// Best-effort, like every other path that installs a plugin the user did
		// not name — but deferred onto the chain rather than run here, so hook
		// code never sees the chain's other plugins half-installed.
		chain.deferredHooks = append(chain.deferredHooks, func() {
			dependencyPostInstall(ctx, cfg, fs, resolved.Plugin, resolved.Version, previousVersion,
				chain.hookBaseURLs.APIBaseURL, chain.hookBaseURLs.DashboardBaseURL, chain.hookBaseURLs.AccessBaseURL)
		})
	}

	return nil
}

// dependencyPostInstall runs the PostInstall hook of a plugin that was installed
// to satisfy a declared floor rather than by name. Swappable for test injection.
// The base URLs are the explicit overrides the install's initiator vouched for
// (see HookBaseURLs); a hook reads an empty value as its own default.
var dependencyPostInstall = func(ctx context.Context, cfg config.IConfig, fs afero.Fs, p *Plugin, version, previousVersion, apiBaseURL, dashboardBaseURL, accessBaseURL string) {
	realConfig, isRealConfig := cfg.(*config.Config)
	if !isRealConfig {
		return
	}

	runPostInstallHook(ctx, realConfig, fs, p, version, previousVersion, apiBaseURL, dashboardBaseURL, accessBaseURL)
}

func dependencyInstallError(requesterName, dependencyName string, cause error) error {
	return fmt.Errorf("could not install the %s plugin, which the %s plugin depends on: %w", dependencyName, requesterName, cause)
}
