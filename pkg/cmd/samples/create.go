package samples

import (
	"fmt"
	"os"

	"github.com/spf13/afero"
	"github.com/spf13/cobra"
	"github.com/stripe/stripe-cli/pkg/ansi"
	"github.com/stripe/stripe-cli/pkg/config"
	gitpkg "github.com/stripe/stripe-cli/pkg/git"
	"github.com/stripe/stripe-cli/pkg/samples"
	"github.com/stripe/stripe-cli/pkg/validators"
	"gopkg.in/src-d/go-git.v4"
)

type CreateCmd struct {
	cfg *config.Config
	Cmd *cobra.Command
}

func NewCreateCmd(config *config.Config) *CreateCmd {
	createCmd := &CreateCmd{
		cfg: config,
	}
	createCmd.Cmd = &cobra.Command{
		Use:       "create",
		Args:      validators.ExactArgs(1),
		ValidArgs: samples.Names(),
		Short:     "create a Stripe sample",
		RunE:      createCmd.runCreateCmd,
	}

	return createCmd
}

func (cc *CreateCmd) runCreateCmd(cmd *cobra.Command, args []string) error {
	sample := samples.Samples{
		Config: cc.cfg,
		Fs:     afero.NewOsFs(),
		Git:    gitpkg.Operations{},
	}
	selectedSample := args[0]

	spinner := ansi.StartSpinner(fmt.Sprintf("Downloading %s", selectedSample), os.Stdout)
	// Initialize the selected sample in the local cache directory.
	// This will either clone or update the specified sample,
	// depending on whether or not it's. Additionally, this
	// identifies if the sample has multiple integrations and what
	// languages it supports.
	err := sample.Initialize(selectedSample)
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

	// Once we've initialized the sample in the local cache
	// directory, the user needs to select which integration they
	// want to work with (if selectedSamplelicable) and which language they
	// want to copy
	err = sample.SelectOptions()
	if err != nil {
		return err
	}

	// Create the target folder to copy the sample in to. We do
	/// this here in case any of the steps above fail, minimizing
	// the change that we create a dangling empty folder
	targetPath, err := sample.MakeFolder(selectedSample)
	if err != nil {
		return err
	}

	// Perform the copy of the sample given the selected options
	// from the selections above
	err = sample.Copy(targetPath)
	if err != nil {
		return err
	}

	// TODO: setup .env
	return nil
}
