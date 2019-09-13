package cmd

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/stripe/stripe-cli/pkg/validators"
	"github.com/stripe/stripe-cli/pkg/version"
)

type versionCmd struct {
	cmd *cobra.Command
}

func newVersionCmd() *versionCmd {
	return &versionCmd{
		cmd: &cobra.Command{
			Use:   "version",
			Args:  validators.NoArgs,
			Short: "Get the version of the Stripe CLI",
			Run: func(cmd *cobra.Command, args []string) {
				fmt.Print(version.Template)

				version.CheckLatestVersion()
			},
		},
	}
}
