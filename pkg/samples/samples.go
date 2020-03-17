package samples

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/joho/godotenv"
	"github.com/manifoldco/promptui"
	"github.com/otiai10/copy"
	"github.com/spf13/afero"
	"gopkg.in/src-d/go-git.v4"

	"github.com/stripe/stripe-cli/pkg/ansi"
	"github.com/stripe/stripe-cli/pkg/config"
	g "github.com/stripe/stripe-cli/pkg/git"
	"github.com/stripe/stripe-cli/pkg/stripeauth"
)

type sampleConfig struct {
	Name            string                    `json:"name"`
	ConfigureDotEnv bool                      `json:"configureDotEnv"`
	PostInstall     map[string]string         `json:"postInstall"`
	Integrations    []sampleConfigIntegration `json:"integrations"`
}

func (sc *sampleConfig) hasIntegrations() bool {
	return len(sc.Integrations) > 1
}

func (sc *sampleConfig) integrationNames() []string {
	names := []string{}
	for _, integration := range sc.Integrations {
		names = append(names, integration.Name)
	}

	return names
}

func (sc *sampleConfig) integrationServers(name string) []string {
	for _, integration := range sc.Integrations {
		if integration.Name == name {
			return integration.Servers
		}
	}

	return []string{}
}

type sampleConfigIntegration struct {
	Name string `json:"name"`
	// Clients are the frontend clients built for each sample
	Clients []string `json:"clients"`
	// Servers are the backend server implementations available for a sample
	Servers []string `json:"servers"`
}

func (i *sampleConfigIntegration) hasClients() bool {
	return len(i.Clients) > 0
}

func (i *sampleConfigIntegration) hasServers() bool {
	return len(i.Servers) > 0
}

func (i *sampleConfigIntegration) hasMultipleClients() bool {
	return len(i.Clients) > 1
}

func (i *sampleConfigIntegration) hasMultipleServers() bool {
	return len(i.Servers) > 1
}

func (i *sampleConfigIntegration) name() string {
	if i.Name == "main" {
		return ""
	}

	return i.Name
}

// Samples stores the information for the selected sample in addition to the
// selected configuration option to copy over
type Samples struct {
	Config *config.Config
	Fs     afero.Fs
	Git    g.Interface

	name string

	// source repository to clone from
	repo string

	sampleConfig sampleConfig

	integration *sampleConfigIntegration

	client string
	server string
}

// Initialize get the sample ready for the user to copy. It:
// 1. creates the sample cache folder if it doesn't exist
// 2. store the path of the locale cache folder for later use
// 3. if the selected app does not exist in the local cache folder, clone it
// 4. if the selected app does exist in the local cache folder, pull changes
// 5. parse the sample cli config file
func (s *Samples) Initialize(app string) error {
	s.name = app

	appPath, err := s.appCacheFolder(app)
	if err != nil {
		return err
	}

	// We still set the repo path here. There are some failure cases
	// that we can still work with (like no updates or repo already exists)
	s.repo = appPath
	list := s.GetSamples("create")
	if _, err := s.Fs.Stat(appPath); os.IsNotExist(err) {
		err = s.Git.Clone(appPath, list[app].GitRepo())
		if err != nil {
			return err
		}
	} else {
		err := s.Git.Pull(appPath)
		if err != nil {
			if err != nil {
				switch e := err.Error(); e {
				case git.NoErrAlreadyUpToDate.Error():
					// Repo is already up to date. This isn't a program
					// error to continue as normal
					break
				default:
					return err
				}
			}
		}
	}

	configFile, err := afero.ReadFile(s.Fs, filepath.Join(appPath, ".cli.json"))
	if err != nil {
		return err
	}

	err = json.Unmarshal(configFile, &s.sampleConfig)
	if err != nil {
		return err
	}

	return nil
}

// SelectOptions prompts the user to select the integration they want to use
// (if available) and the language they want the integration to be.
func (s *Samples) SelectOptions() error {
	var err error

	if s.sampleConfig.hasIntegrations() {
		s.integration, err = integrationSelectPrompt(&s.sampleConfig)
		if err != nil {
			return err
		}
	} else {
		s.integration = &s.sampleConfig.Integrations[0]
	}

	if s.integration.hasMultipleClients() {
		s.client, err = clientSelectPrompt(s.integration.Clients)
		if err != nil {
			return nil
		}
	} else {
		s.client = ""
	}

	if s.integration.hasMultipleServers() {
		s.server, err = serverSelectPrompt(s.integration.Servers)
		if err != nil {
			return err
		}
	} else {
		s.server = ""
	}

	return nil
}

// Copy will copy all of the files from the selected configuration above oves.
// This has a few different behaviors, depending on the configuration.
// Ultimately, we want the user to do as minimal of folder traversing as
// possible. What we want to end up with is:
//
// |- example-sample/
// +-- client/
// +-- server/
// +-- readme.md
// +-- ...
// `-- .env.example
//
// The behavior here is:
// * If there are no integrations available, copy the top-level files, the
//   client folder, and the selected language inside of the server folder to
//   the server top-level (example above)
// * If the user selects an integration, mirror the structure above for the
//   selected integration (example above)
func (s *Samples) Copy(target string) error {
	integration := s.integration.name()

	if s.integration.hasServers() {
		serverSource := filepath.Join(s.repo, integration, "server", s.server)
		serverDestination := filepath.Join(target, "server")

		err := copy.Copy(serverSource, serverDestination)
		if err != nil {
			return err
		}
	}

	if s.integration.hasClients() {
		clientSource := filepath.Join(s.repo, integration, "client", s.client)
		clientDestination := filepath.Join(target, "client")

		err := copy.Copy(clientSource, clientDestination)
		if err != nil {
			return err
		}
	}

	filesSource, err := s.GetFiles(filepath.Join(s.repo, integration))
	if err != nil {
		return err
	}

	for _, file := range filesSource {
		err = copy.Copy(filepath.Join(s.repo, integration, file), filepath.Join(target, file))
		if err != nil {
			return err
		}
	}

	// This copies all top-level files specific to the entire sample repo
	filesSource, err = s.GetFiles(s.repo)
	if err != nil {
		return err
	}

	for _, file := range filesSource {
		err = copy.Copy(filepath.Join(s.repo, file), filepath.Join(target, file))
		if err != nil {
			return err
		}
	}

	return nil
}

// ConfigureDotEnv takes the .env.example from the provided location and
// modifies it to automatically configure it for the users settings
func (s *Samples) ConfigureDotEnv(sampleLocation string) error {
	if s.integration.hasServers() {
		if !s.sampleConfig.ConfigureDotEnv {
			return nil
		}

		// .env.example file will always be at the project root
		exFile := filepath.Join(sampleLocation, ".env.example")

		file, err := s.Fs.Open(exFile)
		if err != nil {
			return err
		}

		dotenv, err := godotenv.Parse(file)
		if err != nil {
			return err
		}

		publishableKey := s.Config.Profile.GetPublishableKey()
		if publishableKey == "" {
			return fmt.Errorf("we could not set the publishable key in the .env file; please set this manually or login again to set it automatically next time")
		}

		apiKey, err := s.Config.Profile.GetAPIKey(false)
		if err != nil {
			return err
		}

		deviceName, err := s.Config.Profile.GetDeviceName()
		if err != nil {
			return err
		}

		authClient := stripeauth.NewClient(apiKey, nil)

		authSession, err := authClient.Authorize(context.TODO(), deviceName, "webhooks", nil)
		if err != nil {
			return err
		}

		dotenv["STRIPE_PUBLISHABLE_KEY"] = publishableKey
		dotenv["STRIPE_SECRET_KEY"] = apiKey
		dotenv["STRIPE_WEBHOOK_SECRET"] = authSession.Secret
		dotenv["STATIC_DIR"] = "../client"

		envFile := filepath.Join(sampleLocation, "server", ".env")

		err = godotenv.Write(dotenv, envFile)
		if err != nil {
			return err
		}
	}

	return nil
}

// PostInstall returns any installation for post installation instructions
func (s *Samples) PostInstall() string {
	message := s.sampleConfig.PostInstall["message"]
	return message
}

// Cleanup performs cleanup for the recently created sample
func (s *Samples) Cleanup(name string) error {
	fmt.Println("Cleaning up...")

	return s.delete(name)
}

// DeleteCache forces the local sample cache to refresh in case something
// goes awry during the initial clone or to clean out stale samples
func (s *Samples) DeleteCache(sample string) error {
	appPath, err := s.appCacheFolder(sample)
	if err != nil {
		return err
	}

	err = s.Fs.RemoveAll(appPath)
	if err != nil {
		return err
	}

	return nil
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

func integrationSelectPrompt(sc *sampleConfig) (*sampleConfigIntegration, error) {
	selected, err := selectOptions("integration", "What type of integration would you like to use", sc.integrationNames())
	if err != nil {
		return nil, err
	}

	var selectedIntegration *sampleConfigIntegration

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
