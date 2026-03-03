package cmd

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/stripe/stripe-cli/pkg/validators"
	"github.com/stripe/stripe-cli/pkg/version"
)

type versionCmd struct {
	cmd   *cobra.Command
	short bool
}

func newVersionCmd() *versionCmd {
	vc := &versionCmd{}

	vc.cmd = &cobra.Command{
		Use:   "version",
		Args:  validators.NoArgs,
		Short: "Get the version of the Stripe CLI",
		Long: `Get the version of the Stripe CLI along with build information.

By default, displays version, Go version, and OS/architecture.
Use --short to display only the version number.`,
		Example: `stripe version
  stripe version --short`,
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Print(version.GetVersionInfo(vc.short))

			if !vc.short {
				version.CheckLatestVersion()
			}
		},
	}

	vc.cmd.Flags().BoolVarP(&vc.short, "short", "s", false, "Print only the version number")

	return vc
}
