package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/afero"
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
					Fs:     afero.NewOsFs(),
				}
				app := args[0]

				spinner := ansi.StartSpinner(fmt.Sprintf("Downloading %s", app), os.Stdout)
				// Initialize the selected recipe in the local cache directory.
				// This will either clone or update the specified recipe,
				// depending on whether or not it's. Additionally, this
				// identifies if the recipe has multiple integrations and what
				// languages it supports.
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
						ansi.StopSpinner(spinner, "An error occurred.", os.Stdout)
						return err
					}
				}
				ansi.StopSpinner(spinner, "Finished downloading.", os.Stdout)

				// Once we've initialized the recipe in the local cache
				// directory, the user needs to select which integration they
				// want to work with (if applicable) and which language they
				// want to copy
				err = recipe.SelectOptions()
				if err != nil {
					return err
				}

				// Create the target folder to copy the recipe in to. We do
				/// this here in case any of the steps above fail, minimizing
				// the change that we create a dangling empty folder
				targetPath, err := recipe.MakeFolder(app)
				if err != nil {
					return err
				}

				// Perform the copy of the recipe given the selected options
				// from the selections above
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
