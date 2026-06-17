package cmd

import (
	"github.com/spf13/cobra"

	"github.com/stripe/stripe-cli/pkg/cmd/skills"
)

type skillsCmd struct {
	cmd *cobra.Command
}

func newSkillsCmd() *skillsCmd {
	sc := &skillsCmd{
		cmd: &cobra.Command{
			Use:   "skills",
			Short: "Manage Stripe AI skills",
			Long:  `Commands for managing Stripe AI skills for use with AI coding assistants.`,
		},
	}

	sc.cmd.AddCommand(skills.NewInstallCmd().Cmd)

	return sc
}
