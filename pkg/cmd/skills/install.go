// Package skills provides CLI commands for managing Stripe AI skills.
package skills

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/stripe/stripe-cli/pkg/ansi"
	"github.com/stripe/stripe-cli/pkg/skills"
	"github.com/stripe/stripe-cli/pkg/validators"
)

// InstallCmd wraps the `install` subcommand for skills.
type InstallCmd struct {
	Cmd *cobra.Command
}

// NewInstallCmd creates and returns the install command for skills.
func NewInstallCmd() *InstallCmd {
	ic := &InstallCmd{}
	ic.Cmd = &cobra.Command{
		Use:   "install",
		Args:  validators.NoArgs,
		Short: "Install Stripe AI skills into .skills in the current directory",
		Long: `Downloads all Stripe AI skills from docs.stripe.com into a .skills/
directory in your current working directory. These skills can be used
with AI coding assistants such as Claude Code.`,
		Example: `stripe skills install`,
		RunE:    ic.runInstallCmd,
	}
	return ic
}

func (ic *InstallCmd) runInstallCmd(cmd *cobra.Command, args []string) error {
	color := ansi.Color(os.Stdout)
	spinner := ansi.StartNewSpinner("Fetching skills index...", os.Stdout)

	cwd, err := os.Getwd()
	if err != nil {
		ansi.StopSpinner(spinner, "", os.Stdout)
		return fmt.Errorf("could not determine current directory: %w", err)
	}

	installed, err := skills.Install(cwd)
	if err != nil {
		ansi.StopSpinner(spinner, "", os.Stdout)
		return err
	}

	ansi.StopSpinner(spinner, "", os.Stdout)

	if len(installed) == 0 {
		fmt.Println("No skills found in the index.")
		return nil
	}

	fmt.Printf("%s Installed %d skill(s) into .skills/\n", color.Green("✔"), len(installed))
	for _, path := range installed {
		fmt.Printf("  %s\n", ansi.Faint(path))
	}
	return nil
}
