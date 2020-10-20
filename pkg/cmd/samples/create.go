package samples

import (
	"fmt"
	"os"
	"os/signal"

	log "github.com/sirupsen/logrus"
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

	forceRefresh bool
}

// NewCreateCmd creates and returns a create command for samples
func NewCreateCmd(config *config.Config) *CreateCmd {
	createCmd := &CreateCmd{
		cfg:          config,
		forceRefresh: false,
	}
	createCmd.Cmd = &cobra.Command{
		Use:   "create <sample> [destination]",
		Args:  validators.MaximumNArgs(2),
		Short: "Setup and bootstrap a Stripe Sample",
		Long: `The create command will locally clone a sample, let you select which integration,
client, and server you want to run. It then automatically bootstraps the
local configuration to let you get started faster.`,
		Example: `stripe samples create adding-sales-tax
  stripe samples create react-elements-card-payment my-payments-form`,
		RunE: createCmd.runCreateCmd,
	}

	createCmd.Cmd.Flags().BoolVar(&createCmd.forceRefresh, "force-refresh", false, "Forcefully refresh the local samples cache")

	return createCmd
}

func (cc *CreateCmd) runCreateCmd(cmd *cobra.Command, args []string) error {
	version.CheckLatestVersion()

	if len(args) == 0 {
		cmd.Help()
		return nil
	}

	sample := samples.Samples{
		Config: cc.cfg,
		Fs:     afero.NewOsFs(),
		Git:    gitpkg.Operations{},
	}

	if _, ok := sample.GetSamples("create")[args[0]]; !ok {
		errorMessage := fmt.Sprintf(`The sample provided is not currently supported by the CLI: %s
To see supported samples, run 'stripe samples list'`, args[0])
		return fmt.Errorf(errorMessage)
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

	spinner := ansi.StartNewSpinner(fmt.Sprintf("Downloading %s", selectedSample), os.Stdout)

	if cc.forceRefresh {
		err := sample.DeleteCache(selectedSample)
		if err != nil {
			logger := log.Logger{
				Out: os.Stdout,
			}

			logger.WithFields(log.Fields{
				"prefix": "samples.create.forceRefresh",
				"error":  err,
			}).Debug("Could not clear cache")
		}
	}

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
	fmt.Printf("%s %s\n", color.Green("✔"), ansi.Faint("Finished downloading"))

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

	spinner = ansi.StartNewSpinner(fmt.Sprintf("Copying files over... %s", destination), os.Stdout)
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
	fmt.Printf("%s %s\n", color.Green("✔"), ansi.Faint("Files copied"))

	spinner = ansi.StartNewSpinner(fmt.Sprintf("Configuring your code... %s", selectedSample), os.Stdout)

	err = sample.ConfigureDotEnv(targetPath)
	if err != nil {
		return err
	}

	ansi.StopSpinner(spinner, "", os.Stdout)
	fmt.Printf("%s %s\n", color.Green("✔"), ansi.Faint("Project configured"))
	fmt.Println("You're all set. To get started: cd", destination)

	if sample.PostInstall() != "" {
		fmt.Println(sample.PostInstall())
	}

	return nil
}
