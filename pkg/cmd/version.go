package cmd

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/stripe/stripe-cli/pkg/validators"
	"github.com/stripe/stripe-cli/pkg/version"
)

type versionCmd struct {
	cmd   *cobra.Command
	notes bool
}

func newVersionCmd() *versionCmd {
	vc := &versionCmd{}
	vc.cmd = &cobra.Command{
		Use:   "version",
		Args:  validators.NoArgs,
		Short: "Get the version of the Stripe CLI",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Fprint(cmd.OutOrStdout(), version.Template)

			if vc.notes {
				vc.printReleaseNotes(cmd)
				return
			}

			version.CheckLatestVersion()
		},
	}
	vc.cmd.Flags().BoolVar(&vc.notes, "notes", false, "Show the release notes for the current version")

	return vc
}

func (vc *versionCmd) printReleaseNotes(cmd *cobra.Command) {
	out := cmd.OutOrStdout()

	if version.Version == "master" {
		fmt.Fprintln(out, "Release notes aren't available for development builds.")
		return
	}

	notes, err := version.GetReleaseNotesFn(version.Version)
	if err != nil {
		fmt.Fprintf(cmd.ErrOrStderr(), "Could not fetch release notes: %v\n", err)
		return
	}

	if notes == "" {
		fmt.Fprintln(out, "No release notes found for this version.")
		return
	}

	fmt.Fprintln(out)
	fmt.Fprintln(out, notes)
}
