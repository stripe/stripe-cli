package plugins

import (
	"errors"
	"fmt"
	"regexp"
	"strings"

	goversion "github.com/hashicorp/go-version"

	"github.com/stripe/stripe-cli/pkg/errorcategory"
	"github.com/stripe/stripe-cli/pkg/version"
)

// developmentCoreVersion is the version a CLI built from source reports. The
// plugin metadata API treats it as newer than every published release, so the
// same allowance is made here: otherwise a local build could not install a
// restricted release the API happily hands it.
const developmentCoreVersion = "master"

// coreVersionSemver bounds what counts as a comparable core CLI version. It
// matches the API's own VALID_SEMVER so that a release the API considers
// reachable is never rejected here, and vice versa.
var coreVersionSemver = regexp.MustCompile(`^\d+\.\d+\.\d+$`)

// ErrPluginRequiresNewerCLI is returned when a plugin release declares a minimum
// core CLI version that the running CLI does not meet.
type ErrPluginRequiresNewerCLI struct {
	Name           string
	Version        string
	MinCoreVersion string
	CoreVersion    string
}

func (e *ErrPluginRequiresNewerCLI) Error() string {
	// Both versions are named whenever they are known, and the ones that come from
	// the metadata endpoint are only as complete as its response. Degrading the
	// wording beats rendering "the appA plugin v requires Stripe CLI v or newer".
	subject := fmt.Sprintf("the %s plugin", e.Name)
	if e.Version != "" {
		subject = fmt.Sprintf("the %s plugin v%s", e.Name, e.Version)
	}

	requirement := "a newer Stripe CLI"
	if e.MinCoreVersion != "" {
		requirement = fmt.Sprintf("Stripe CLI v%s or newer", e.MinCoreVersion)
	}

	return fmt.Sprintf(
		"%s requires %s, but this is Stripe CLI %s. Upgrade the Stripe CLI to install it: https://docs.stripe.com/stripe-cli/upgrade",
		subject, requirement, e.CoreVersion,
	)
}

// newErrPluginRequiresNewerCLI builds the error every requires-a-newer-CLI report
// goes through, so that the constraint the metadata endpoint states and the one
// read out of a manifest are reported and routed identically.
//
// It is categorized as user input because the actionable part belongs to the
// caller: install a version this CLI supports, or upgrade the CLI.
func newErrPluginRequiresNewerCLI(name, pluginVersion, minCoreVersion string) error {
	return errorcategory.With(&ErrPluginRequiresNewerCLI{
		Name:           name,
		Version:        pluginVersion,
		MinCoreVersion: minCoreVersion,
		CoreVersion:    version.Version,
	}, errorcategory.UserInput)
}

// requiresNewerCLI reports whether err is the requires-a-newer-CLI answer.
//
// The resolve paths check it to stop before their cached fallback. That fallback is
// there to survive a source that could not answer, not to overrule one that did,
// and cached metadata can predate the constraint entirely -- so letting it win here
// would drop the constraint on exactly the machines it exists to stop.
func requiresNewerCLI(err error) bool {
	var requiresNewer *ErrPluginRequiresNewerCLI
	return errors.As(err, &requiresNewer)
}

// coreVersionSupports reports whether the running core CLI satisfies a release's
// MinCoreVersion constraint. An empty constraint means the release places no
// requirement on the core CLI, which is every release that predates the field.
//
// This deliberately mirrors the plugin metadata API's own compatibility check
// (PluginMetadata.core_version_supports_release?). The API already hides
// incompatible releases from live responses, so the two must agree or the same
// release would be installable from one source and not the other.
//
// It is enforced again on this side because cached plugin metadata outlives the
// CLI that wrote it. A CLI that was downgraded, or that cannot reach the API,
// resolves installs from `plugin-metadata/*.toml` and `plugins.toml` instead --
// caches a newer CLI populated -- and nothing in that path consults the API.
// Being hidden from the API is itself what sends an older CLI down it, since a
// hidden release makes the metadata endpoint 404.
//
// Anything unparseable counts as unsupported rather than unconstrained: a
// constraint that cannot be evaluated is precisely the case where installing
// anyway risks a binary that will not run.
func coreVersionSupports(minCoreVersion string) bool {
	if minCoreVersion == "" {
		return true
	}

	current := normalizeCoreVersion(version.Version)
	if current == developmentCoreVersion {
		return true
	}
	if !coreVersionSemver.MatchString(current) {
		return false
	}

	currentVersion, err := goversion.NewVersion(current)
	if err != nil {
		return false
	}

	minimumVersion, err := goversion.NewVersion(normalizeCoreVersion(minCoreVersion))
	if err != nil {
		return false
	}

	return currentVersion.GreaterThanOrEqual(minimumVersion)
}

// normalizeCoreVersion tolerates a leading "v" on either side of the comparison.
// Published versions carry no prefix, but pkg/version already strips one when
// comparing CLI versions, so accepting it here keeps the two consistent.
func normalizeCoreVersion(rawVersion string) string {
	return strings.TrimPrefix(strings.TrimSpace(rawVersion), "v")
}

// checkCoreVersionForRelease returns an *ErrPluginRequiresNewerCLI when the
// release for pluginVersion on this platform declares a MinCoreVersion the
// running CLI does not meet.
//
// A version with no matching release returns nil. Whether a release exists is
// not this check's question, and every caller already reports that case itself
// with more context than this could.
func (p *Plugin) checkCoreVersionForRelease(pluginVersion string) error {
	if pluginVersion == "" || isLocalDevelopmentVersion(pluginVersion) {
		return nil
	}

	release := p.getReleaseForVersion(pluginVersion)
	if release == nil || coreVersionSupports(release.MinCoreVersion) {
		return nil
	}

	return newErrPluginRequiresNewerCLI(p.Shortname, pluginVersion, release.MinCoreVersion)
}

// checkCoreVersionForLatestRelease reports the requires-a-newer-CLI case behind
// an empty LookUpLatestVersion result. It exists so that "every release for this
// platform needs a newer CLI" is not reported as "this plugin has no releases",
// which reads as though the plugin were unavailable altogether.
func (p *Plugin) checkCoreVersionForLatestRelease() error {
	return p.checkCoreVersionForRelease(p.lookUpLatestVersion(false))
}
