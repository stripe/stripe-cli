package cmd

import (
	"fmt"

	"github.com/otiai10/copy"
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
				repoPath, err := recipe.Download(args[0])
				if err != nil {
					fmt.Println(err)
				}

				targetPath, err := recipe.MakeFolder(args[0])
				if err != nil {
					fmt.Println(err)
				}

				err = copy.Copy(repoPath, targetPath)
				if err != nil {
					fmt.Println(err)
				}
				// TODO: display interactive prompt to select folders
				// TODO: copy select config to specified directory
				// TODO: setup .env
			},
		},
	}
}
