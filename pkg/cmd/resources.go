package cmd

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/stripe/stripe-cli/pkg/ansi"
	"github.com/stripe/stripe-cli/pkg/validators"
)

type resourcesCmd struct {
	cmd *cobra.Command
}

func newResourcesCmd() *resourcesCmd {
	rc := &resourcesCmd{}

	rc.cmd = &cobra.Command{
		Use:   "resources",
		Args:  validators.NoArgs,
		Short: "List resource commands",
		RunE:  rc.run,
	}
	rc.cmd.SetHelpFunc(func(cmd *cobra.Command, args []string) {
		_ = rc.run(cmd, args)
	})

	return rc
}

func (rc *resourcesCmd) run(cmd *cobra.Command, _ []string) error {
	parent := cmd.Parent()
	if parent == nil {
		return nil
	}

	out := cmd.OutOrStdout()
	fmt.Fprintln(out, ansi.Bold("Available commands:"))
	for _, child := range parent.Commands() {
		if !showInResources(parent, child) {
			continue
		}

		fmt.Fprintf(out, "  %-*s %s\n", child.NamePadding(), child.Name(), child.Short)
	}
	fmt.Fprintln(out)
	fmt.Fprintln(out, `Use "stripe [command] --help" for more information about a command.`)
	return nil
}

func showInResources(parent *cobra.Command, cmd *cobra.Command) bool {
	kind := parent.Annotations[cmd.Name()]
	if kind != "resource" && kind != "namespace" {
		return false
	}

	return !cmd.Hidden
}
