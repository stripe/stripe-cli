package version

import (
	"fmt"

	"github.com/stripe/stripe-cli/pkg/ansi"
)

// Version of the CLI -- currently in beta.
// This is set to the actual version by GoReleaser, identify by the
// git tag assigned to the release. Versions built from source will
// always show master.
var Version = "master"

// Template for the version string.
var Template = fmt.Sprintf("stripe version %s %s\n", Version, ansi.Bold("(beta)"))
