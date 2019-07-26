package cmd

import (
	"github.com/spf13/cobra"
	"github.com/stripe/stripe-cli/pkg/recipes"
)

type appsCmd struct {
	cmd *cobra.Command
}

func newAppsCmd() *appsCmd {
	return &appsCmd{
		cmd: &cobra.Command{
			// TODO: create subcommand
			// TODO: list subcommand
			// TODO: fixtures subcommand
			Use: "apps",
			Run: func(cmd *cobra.Command, args []string) {
				recipe := recipes.Recipes{
					Config: Config,
				}
				// TODO: display waiting prompt
				recipe.Download(args[0])
				// TODO: display interactive prompt to select folders
				// TODO: copy select config to specified directory
				// TODO: setup .env
			},
		},
	}
}
