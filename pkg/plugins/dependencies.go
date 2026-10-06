package plugins

import (
	"context"
	"fmt"
	"sort"

	goversion "github.com/hashicorp/go-version"
	"github.com/spf13/afero"

	"github.com/stripe/stripe-cli/pkg/config"
	"github.com/stripe/stripe-cli/pkg/errorcategory"
	"github.com/stripe/stripe-cli/pkg/stripe"
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
	if isLocalDevelopmentVersion(installedVersion) {
		return true
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

// installPluginDependencies installs or upgrades the peer plugins a release
// declares minimum versions for, before the release itself is installed. For each
// declared peer that is missing or below its minimum, it installs the newest
// release of that peer — the minimum is a floor, not a pin.
//
// installing maps the plugins already being installed up this call chain,
// including the requester itself, to the versions going in. A dependency found
// there has its floor checked against that in-flight version rather than being
// recursed into, so a cycle ends without waving the requirement through. The
// recursive install resolves each dependency's own requirements the same way, so
// dependencies land transitively.
//
// A failure names the dependency and leaves the requester uninstalled, since the
// caller runs this before downloading the requester.
func installPluginDependencies(ctx context.Context, cfg config.IConfig, fs afero.Fs, requester *Plugin, apiBaseURL, dashboardBaseURL string, installing map[string]string) error {
	dependencyNames := make([]string, 0, len(requester.MinPluginVersions))
	for dependencyName := range requester.MinPluginVersions {
		dependencyNames = append(dependencyNames, dependencyName)
	}
	sort.Strings(dependencyNames)

	for _, dependencyName := range dependencyNames {
		minimum := requester.MinPluginVersions[dependencyName]

		// Release validation rejects a plugin depending on itself, so a self-entry
		// that still got here is bad data to skip, not something to recurse on.
		if dependencyName == requester.Shortname {
			continue
		}

		// Already being installed somewhere up this call chain. Recursing would
		// never end, but the version on its way in still has to clear this floor:
		// a cycle is a reason to stop recursing, not permission to skip the
		// requirement.
		if inFlightVersion, alreadyInstalling := installing[dependencyName]; alreadyInstalling {
			if pluginVersionSatisfiesMinimum(inFlightVersion, minimum) {
				continue
			}

			return dependencyInstallError(requester.Shortname, dependencyName, errorcategory.Errorf(
				errorcategory.API,
				"v%s or newer is required, but v%s is already being installed by this same operation",
				minimum, inFlightVersion,
			))
		}

		if err := ValidatePluginShortname(dependencyName); err != nil {
			return dependencyInstallError(requester.Shortname, dependencyName, err)
		}

		installedVersion, err := (&Plugin{Shortname: dependencyName}).lookUpInstalledVersion(cfg, fs)
		if err != nil {
			// Unreadable is handled as absent: the ways this install can be wrong
			// about that all end in installing a release that was already there.
			installedVersion = ""
		}
		if pluginVersionSatisfiesMinimum(installedVersion, minimum) {
			continue
		}

		resolved, err := ResolvePluginForUpgrade(ctx, cfg, fs, dependencyName, apiBaseURL, dashboardBaseURL)
		if err != nil {
			return dependencyInstallError(requester.Shortname, dependencyName, err)
		}

		// The newest release the endpoint offers can sit below the minimum when it
		// withholds releases this CLI is too old to run. Installing it anyway would
		// claim success while leaving the requirement unmet, to be rediscovered as
		// a failure at run time.
		if !pluginVersionSatisfiesMinimum(resolved.Version, minimum) {
			return dependencyInstallError(requester.Shortname, dependencyName, errorcategory.Errorf(
				errorcategory.API,
				"v%s or newer is required, but the newest release available to this CLI is v%s. A newer Stripe CLI may be required: https://docs.stripe.com/stripe-cli/upgrade",
				minimum, resolved.Version,
			))
		}

		if err := resolved.Plugin.install(ctx, cfg, fs, resolved.Version, apiBaseURL, dashboardBaseURL, resolved.BinaryURL, resolved.BinaryURL != "", installing); err != nil {
			return dependencyInstallError(requester.Shortname, dependencyName, err)
		}

		// Best-effort, like every other path that installs a plugin the user did
		// not name. The hook gets the user's overrides back out of the resolved
		// URLs install runs on; see hookBaseURLOverrides.
		hookAPIBaseURL, hookDashboardBaseURL := hookBaseURLOverrides(apiBaseURL, dashboardBaseURL)
		dependencyPostInstall(ctx, cfg, fs, resolved.Plugin, resolved.Version, installedVersion, hookAPIBaseURL, hookDashboardBaseURL, "")
	}

	return nil
}

// dependencyPostInstall runs a freshly installed dependency's PostInstall hook.
// Swappable for test injection. The access base URL is always empty: install
// never learns it, and a hook that needs it reads an empty value as its own
// default, which is all the CLI could offer here anyway.
var dependencyPostInstall = func(ctx context.Context, cfg config.IConfig, fs afero.Fs, p *Plugin, version, previousVersion, apiBaseURL, dashboardBaseURL, accessBaseURL string) {
	realConfig, isRealConfig := cfg.(*config.Config)
	if !isRealConfig {
		return
	}

	runPostInstallHook(ctx, realConfig, fs, p, version, previousVersion, apiBaseURL, dashboardBaseURL, accessBaseURL)
}

// hookBaseURLOverrides recovers the user's explicit base URL overrides from the
// resolved URLs install runs on, for forwarding to a plugin hook. A plugin reads
// an empty value as "use your own default", so a hook must only ever hear about a
// URL the user actually chose — not the default install resolved so its metadata
// request could name a real host.
//
// install is never told which values were explicit, but they are recoverable: the
// API URL is only ever non-default when the user set it, and a dashboard URL equal
// to the one derived from the API URL is exactly what resolution fills in when the
// user said nothing (a user who explicitly passed that same value loses nothing by
// having it elided — it is the value the plugin would derive too).
func hookBaseURLOverrides(apiBaseURL, dashboardBaseURL string) (hookAPIBaseURL, hookDashboardBaseURL string) {
	if apiBaseURL != stripe.DefaultAPIBaseURL {
		hookAPIBaseURL = apiBaseURL
	}
	if dashboardBaseURL != stripe.DashboardBaseURLForAPIBaseURL(apiBaseURL) {
		hookDashboardBaseURL = dashboardBaseURL
	}

	return hookAPIBaseURL, hookDashboardBaseURL
}

func dependencyInstallError(requesterName, dependencyName string, cause error) error {
	return fmt.Errorf("could not install the %s plugin, which the %s plugin depends on: %w", dependencyName, requesterName, cause)
}
