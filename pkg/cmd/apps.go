package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/otiai10/copy"
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
				repoPath, err := recipe.Download(app)
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
						return err
					}
				}
				ansi.StopSpinner(spinner, "Finished downloading.", os.Stdout)

				integration, language, err := recipe.BuildPrompts(repoPath)
				if err != nil {
					return err
				}

				targetPath, err := recipe.MakeFolder(app)
				if err != nil {
					return err
				}

				// Possible approach one:
				// Copy everything over and then delete what I don't want
				// Might be a little weird because I'll be deleting most things and
				// needing to restructure folders after-the-fact
				//
				// err = copy.Copy(repoPath, targetPath)
				// if err != nil {
				//	return err
				// }

				var serverPath string
				var clientPath string

				// Approach two:
				// Copy over the specific things I care about
				// A lot of conditional logic depending on flags the users select
				// TODO -- clean all of this up
				if len(integration) > 1 {
					for _, i := range integration {
						serverPath = filepath.Join(repoPath, i, "server", language)
						clientPath = filepath.Join(repoPath, i, "client")

						err = copy.Copy(serverPath, filepath.Join(targetPath, i, "server"))
						if err != nil {
							return err
						}
						err = copy.Copy(clientPath, filepath.Join(targetPath, i, "client"))
						if err != nil {
							return err
						}
					}

					return nil
				} else if len(integration) == 1 {
					serverPath = filepath.Join(repoPath, integration[0], "server", language)
					clientPath = filepath.Join(repoPath, integration[0], "client")
				} else {
					serverPath = filepath.Join(repoPath, "server", language)
					clientPath = filepath.Join(repoPath, "client")
				}
				err = copy.Copy(serverPath, filepath.Join(targetPath, "server"))
				if err != nil {
					return err
				}
				err = copy.Copy(clientPath, filepath.Join(targetPath, "client"))
				if err != nil {
					return err
				}

				// TODO: setup .env
				return nil
			},
		},
	}
}
