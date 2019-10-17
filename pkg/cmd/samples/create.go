package samples

import (
	"fmt"
	"os"
	"os/signal"

	"github.com/spf13/afero"
	"github.com/spf13/cobra"

	"github.com/stripe/stripe-cli/pkg/ansi"
	"github.com/stripe/stripe-cli/pkg/config"
	gitpkg "github.com/stripe/stripe-cli/pkg/git"
	"github.com/stripe/stripe-cli/pkg/samples"
	"github.com/stripe/stripe-cli/pkg/validators"
	"github.com/stripe/stripe-cli/pkg/version"

	"gopkg.in/src-d/go-git.v4"
)

// CreateCmd wraps the `create` command for samples which generates a new
// project
type CreateCmd struct {
	cfg *config.Config
	Cmd *cobra.Command
}

// NewCreateCmd creates and returns a create command for samples
func NewCreateCmd(config *config.Config) *CreateCmd {
	createCmd := &CreateCmd{
		cfg: config,
	}
	createCmd.Cmd = &cobra.Command{
		Use:       "create",
		ValidArgs: samples.Names(),
		Args:      validators.MaximumNArgs(2),
		Short:     "create a Stripe sample",
		RunE:      createCmd.runCreateCmd,
	}

	return createCmd
}

func (cc *CreateCmd) runCreateCmd(cmd *cobra.Command, args []string) error {
	version.CheckLatestVersion()

	if len(args) == 0 {
		return fmt.Errorf("Creating a sample requires at least 1 argument, received 0")
	}

	if _, ok := samples.List[args[0]]; !ok {
		errorMessage := fmt.Sprintf(`The sample provided is not currently supported by the CLI: %s
To see supported samples, run 'stripe samples list'`, args[0])
		return fmt.Errorf(errorMessage)
	}

	sample := samples.Samples{
		Config: cc.cfg,
		Fs:     afero.NewOsFs(),
		Git:    gitpkg.Operations{},
	}
	selectedSample := args[0]
	color := ansi.Color(os.Stdout)

	destination := selectedSample
	if len(args) > 1 {
		destination = args[1]
	}

	exists, _ := afero.DirExists(sample.Fs, destination)
	if exists {
		return fmt.Errorf("Path already exists for: %s", destination)
	}

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

	ansi.StopSpinner(spinner, "", os.Stdout)
	fmt.Println(fmt.Sprintf("%s %s", color.Green("✔"), ansi.Faint("Finished downloading")))

	// Once we've initialized the sample in the local cache
	// directory, the user needs to select which integration they
	// want to work with (if selectedSamplelicable) and which language they
	// want to copy
	err = sample.SelectOptions()
	if err != nil {
		return err
	}

	// Setup to intercept ctrl+c
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)

	go func() {
		<-c
		sample.Cleanup(selectedSample)
		os.Exit(1)
	}()

	spinner = ansi.StartSpinner(fmt.Sprintf("Copying files over... %s", selectedSample), os.Stdout)
	// Create the target folder to copy the sample in to. We do
	// this here in case any of the steps above fail, minimizing
	// the change that we create a dangling empty folder
	targetPath, err := sample.MakeFolder(destination)
	if err != nil {
		return err
	}

	// Perform the copy of the sample given the selected options
	// from the selections above
	err = sample.Copy(targetPath)
	if err != nil {
		return err
	}

	ansi.StopSpinner(spinner, "", os.Stdout)
	fmt.Println(fmt.Sprintf("%s %s", color.Green("✔"), ansi.Faint("Files copied")))

	spinner = ansi.StartSpinner(fmt.Sprintf("Configuring your code... %s", selectedSample), os.Stdout)

	err = sample.ConfigureDotEnv(targetPath)
	if err != nil {
		return err
	}

	ansi.StopSpinner(spinner, "", os.Stdout)
	fmt.Println(fmt.Sprintf("%s %s", color.Green("✔"), ansi.Faint("Project configured")))
	fmt.Println("You're all set. To get started: cd", selectedSample)
	fmt.Println(sample.PostInstall())

	return nil
}
