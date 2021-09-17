package samples

import (
	"errors"
	"fmt"
	"os"

	"github.com/manifoldco/promptui"
	"github.com/spf13/cobra"

	"github.com/stripe/stripe-cli/pkg/ansi"
	"github.com/stripe/stripe-cli/pkg/config"
	"github.com/stripe/stripe-cli/pkg/samples"
	"github.com/stripe/stripe-cli/pkg/validators"
	"github.com/stripe/stripe-cli/pkg/version"
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
		Example: `stripe samples create accept-a-payment
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

	selectedSample := args[0]
	destination := selectedSample
	if len(args) > 1 {
		destination = args[1]
	}

	color := ansi.Color(os.Stdout)
	spinner := ansi.StartNewSpinner(fmt.Sprintf("Downloading %s", selectedSample), os.Stdout)

	sampleConfig, err := samples.GetSampleConfig(selectedSample, cc.forceRefresh)
	if err != nil {
		ansi.StopSpinner(spinner, "", os.Stdout)
		return err
	}
	ansi.StopSpinner(spinner, "", os.Stdout)
	fmt.Printf("%s %s\n", color.Green("✔"), ansi.Faint("Finished downloading"))

	// Once we've initialized the sample in the local cache
	// directory, the user needs to select which integration they
	// want to work with (if selectedSamplelicable) and which language they
	// want to copy
	selectedConfig, err := promptSampleConfig(sampleConfig)
	if err != nil {
		return err
	}

	resultChan := make(chan samples.CreationResult)

	go samples.Create(
		cmd.Context(),
		cc.cfg,
		selectedSample,
		selectedConfig,
		destination,
		cc.forceRefresh,
		resultChan,
	)

	for res := range resultChan {
		if res.Err != nil {
			ansi.StopSpinner(spinner, "", os.Stdout)
			return res.Err
		}

		switch res.State {
		case samples.WillInitialize:
		case samples.DidInitialize:
		case samples.WillCopy:
			spinner = ansi.StartNewSpinner(fmt.Sprintf("Copying files over... %s", destination), os.Stdout)
		case samples.DidCopy:
			ansi.StopSpinner(spinner, "", os.Stdout)
			fmt.Printf("%s %s\n", color.Green("✔"), ansi.Faint("Files copied"))
		case samples.WillConfigure:
			spinner = ansi.StartNewSpinner(fmt.Sprintf("Configuring your code... %s", selectedSample), os.Stdout)
		case samples.DidConfigure:
			ansi.StopSpinner(spinner, "", os.Stdout)
			fmt.Printf("%s %s\n", color.Green("✔"), ansi.Faint("Project configured"))
		case samples.Done:
			fmt.Println("You're all set. To get started: cd", destination)
			if res.PostInstall != "" {
				fmt.Println(res.PostInstall)
			}
		default:
			return errors.New("an unknown error occurred during sample creation")
		}
	}

	return nil
}

// promptSampleConfig prompts the user to select the integration they want to use
// (if available) and the language they want the integration to be.
func promptSampleConfig(sampleConfig *samples.SampleConfig) (*samples.SelectedConfig, error) {
	var selectedConfig samples.SelectedConfig

	if sampleConfig.HasIntegrations() {
		integration, err := integrationSelectPrompt(sampleConfig)
		if err != nil {
			return nil, err
		}
		selectedConfig.Integration = integration
	} else {
		selectedConfig.Integration = &sampleConfig.Integrations[0]
	}

	if selectedConfig.Integration.HasMultipleClients() {
		client, err := clientSelectPrompt(selectedConfig.Integration.Clients)
		if err != nil {
			return nil, err
		}
		selectedConfig.Client = client
	} else {
		selectedConfig.Client = ""
	}

	if selectedConfig.Integration.HasMultipleServers() {
		server, err := serverSelectPrompt(selectedConfig.Integration.Servers)
		if err != nil {
			return nil, err
		}
		selectedConfig.Server = server
	} else {
		selectedConfig.Server = ""
	}

	return &selectedConfig, nil
}

func selectOptions(template, label string, options []string) (string, error) {
	color := ansi.Color(os.Stdout)

	templates := &promptui.SelectTemplates{
		Selected: color.Green("✔").String() + ansi.Faint(fmt.Sprintf(" Selected %s: {{ . | bold }} ", template)),
	}
	prompt := promptui.Select{
		Label:     label,
		Items:     options,
		Templates: templates,
	}

	_, result, err := prompt.Run()

	if err != nil {
		return "", err
	}

	return result, nil
}

func clientSelectPrompt(clients []string) (string, error) {
	selected, err := selectOptions("client", "Which client would you like to use", clients)
	if err != nil {
		return "", err
	}

	return selected, nil
}

func integrationSelectPrompt(sc *samples.SampleConfig) (*samples.SampleConfigIntegration, error) {
	selected, err := selectOptions("integration", "What type of integration would you like to use", sc.IntegrationNames())
	if err != nil {
		return nil, err
	}

	var selectedIntegration *samples.SampleConfigIntegration

	for i, integration := range sc.Integrations {
		if integration.Name == selected {
			selectedIntegration = &sc.Integrations[i]
		}
	}

	return selectedIntegration, nil
}

func serverSelectPrompt(servers []string) (string, error) {
	selected, err := selectOptions("server", "What server would you like to use", servers)
	if err != nil {
		return "", err
	}

	return selected, nil
}
