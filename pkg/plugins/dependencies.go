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
// installing is the set of plugins already being installed up this call chain,
// including the requester itself; a dependency found there is skipped, so a cycle
// ends instead of recursing. The recursive install resolves each dependency's own
// requirements the same way, so dependencies land transitively.
//
// A failure names the dependency and leaves the requester uninstalled, since the
// caller runs this before downloading the requester.
func installPluginDependencies(ctx context.Context, cfg config.IConfig, fs afero.Fs, requester *Plugin, apiBaseURL, dashboardBaseURL string, installing map[string]struct{}) error {
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

		if _, alreadyInstalling := installing[dependencyName]; alreadyInstalling {
			continue
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
		// not name. Base URL overrides are deliberately not forwarded: a plugin
		// reads an empty value as "use your own default", and the resolved URLs
		// here exist only because the metadata request needed a real host.
		if realConfig, isRealConfig := cfg.(*config.Config); isRealConfig {
			runPostInstallHook(ctx, realConfig, fs, resolved.Plugin, resolved.Version, installedVersion, "", "", "")
		}
	}

	return nil
}

func dependencyInstallError(requesterName, dependencyName string, cause error) error {
	return fmt.Errorf("could not install the %s plugin, which the %s plugin depends on: %w", dependencyName, requesterName, cause)
}
