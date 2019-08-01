package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/stripe/stripe-cli/pkg/ansi"
	"github.com/stripe/stripe-cli/pkg/recipes"
	"gopkg.in/src-d/go-git.v4"
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
			RunE: func(cmd *cobra.Command, args []string) error {
				recipe := recipes.Recipes{
					Config: Config,
				}
				app := args[0]

				spinner := ansi.StartSpinner(fmt.Sprintf("Downloading %s", app), os.Stdout)
				err := recipe.Initialize(app)
				if err != nil {
					switch e := err.Error(); e {
					case git.NoErrAlreadyUpToDate.Error():
						// Repo is already up to date. This isn't a program
						// error to continue as normal
						break
					case git.ErrRepositoryAlreadyExists.Error():
						// If the repository already exists and we don't pull
						// for some reason, that's fine as we can use the existing
						// repository
						break
					default:
						ansi.StopSpinner(spinner, "An error occured.", os.Stdout)
						return err
					}
				}
				ansi.StopSpinner(spinner, "Finished downloading.", os.Stdout)

				err = recipe.SelectOptions()
				if err != nil {
					return err
				}

				targetPath, err := recipe.MakeFolder(app)
				if err != nil {
					return err
				}

				err = recipe.Copy(targetPath)
				if err != nil {
					return err
				}

				// TODO: setup .env
				return nil
			},
		},
	}
}
