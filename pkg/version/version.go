package version

import (
	"fmt"

	"github.com/stripe/stripe-cli/pkg/ansi"
)

// Version number of the CLI
var Version = "master"

var Template = fmt.Sprintf("stripe version %s %s\n", Version, ansi.Bold("(beta)"))
