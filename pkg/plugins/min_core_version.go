package plugins

import (
	"errors"
	"fmt"

	"github.com/stripe/stripe-cli/pkg/errorcategory"
	"github.com/stripe/stripe-cli/pkg/version"
)

// ErrPluginRequiresNewerCLI is returned when the plugin metadata endpoint answers
// that the plugin needs a newer core CLI than this one. Version is the release the
// caller asked for, and is empty when the caller asked for no particular one.
//
// This is the one place min_core_version is explained. The rest of the package
// points back here instead of restating it, so there is a single copy to keep true.
//
// The API owns the judgment: it knows the constraint for every release and withholds
// the ones this CLI cannot run, sending this answer when that leaves nothing to hand
// back. See requests.PluginRequiresNewerCLI for when it does so. Two things follow
// for this package:
//
//   - Nothing re-derives the constraint. Releases carry MinCoreVersion in the
//     manifest and it is deliberately left undecoded, and a manifest that came from
//     the endpoint has already been filtered, so the newest release it lists is the
//     newest one this CLI can install.
//   - Nothing has to. An install cannot complete without a binary URL from a live
//     metadata response, so no path reaches a download the API did not agree to.
//     Reading this answer buys the reason, not the refusal.
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

// newErrPluginRequiresNewerCLI builds the error both metadata lookups report, so
// that the endpoint's answer is routed identically whichever one received it.
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
// and the cache holds no constraint of its own to weigh against this -- so letting
// it win here would trade a precise answer for "no such version".
func requiresNewerCLI(err error) bool {
	var requiresNewer *ErrPluginRequiresNewerCLI
	return errors.As(err, &requiresNewer)
}
