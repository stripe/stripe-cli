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

	"github.com/stripe/stripe-cli/pkg/ansi"
	"github.com/stripe/stripe-cli/pkg/config"
	"github.com/stripe/stripe-cli/pkg/git"
	"github.com/stripe/stripe-cli/pkg/stripeauth"
)

type sampleConfig struct {
	Name            string            `json:"name"`
	ConfigureDotEnv bool              `json:"configureDotEnv"`
	PostInstall     map[string]string `json:"postInstall"`
	Integrations    []integration     `json:"integrations"`
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

type integration struct {
	Name    string   `json:"name"`
	Clients []string `json:"clients"`
	Servers []string `json:"servers"`
}

var languageDisplayNames = map[string]string{
	"java":   "Java",
	"node":   "Node",
	"python": "Python",
	"php":    "PHP",
	"ruby":   "Ruby",
}

var displayNameLanguages = reverseStringMap(languageDisplayNames)

func reverseStringMap(m map[string]string) map[string]string {
	rm := make(map[string]string, len(m))

	for key, value := range m {
		rm[value] = key
	}

	return rm
}

// Samples stores the information for the selected sample in addition to the
// selected configuration option to copy over
type Samples struct {
	Config *config.Config
	Fs     afero.Fs
	Git    git.Interface

	name string

	// source repository to clone from
	repo string

	sampleConfig sampleConfig

	//selected integrations. We store selected integration as an array
	// in case they pick more than one
	integration []string

	// selected language
	language string
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

	if _, err := s.Fs.Stat(appPath); os.IsNotExist(err) {
		err = s.Git.Clone(appPath, List[app].GitRepo())
		if err != nil {
			return err
		}
	} else {
		err := s.Git.Pull(appPath)
		if err != nil {
			return err
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
		s.integration, err = integrationSelectPrompt(s.sampleConfig.integrationNames())
		if err != nil {
			return err
		}
	} else {
		s.integration = []string{s.sampleConfig.Integrations[0].Name}
	}

	s.language, err = languageSelectPrompt(s.sampleConfig.integrationServers(s.integration[0]))
	if err != nil {
		return err
	}

	fmt.Println("Setting up", ansi.Bold(s.name))

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
// * If the user selects 1 integration, mirror the structure above for the
//   selected integration (example above)
// * If they selected >1 integration, we want the same structure above but
//   replicated once per selected in integration.
func (s *Samples) Copy(target string) error {
	// The condition for the loop starts as true since we will always want to
	// process at least once.
	for i := 0; true; i++ {
		integration := s.destinationName(i)

		serverSource := filepath.Join(s.repo, integration, "server", s.language)
		clientSource := filepath.Join(s.repo, integration, "client")

		filesSource, err := s.GetFiles(filepath.Join(s.repo, integration))
		if err != nil {
			return err
		}

		serverDestination := s.destinationPath(target, integration, "server")
		clientDestination := s.destinationPath(target, integration, "client")

		err = copy.Copy(serverSource, serverDestination)
		if err != nil {
			return err
		}

		err = copy.Copy(clientSource, clientDestination)
		if err != nil {
			return err
		}

		// This copies all top-level files specific to integrations
		for _, file := range filesSource {
			err = copy.Copy(filepath.Join(s.repo, integration, file), filepath.Join(s.destinationPath(target, integration, ""), file))
			if err != nil {
				return err
			}
		}

		if i >= len(s.integration)-1 {
			break
		}
	}

	// This copies all top-level files specific to the entire sample repo
	filesSource, err := s.GetFiles(s.repo)
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

	envFile := filepath.Join(sampleLocation, ".env")

	err = godotenv.Write(dotenv, envFile)
	if err != nil {
		return err
	}

	return nil
}

// PostInstall returns any installation for post installation instructions
func (s *Samples) PostInstall() string {
	message := s.sampleConfig.PostInstall["message"]
	return message
}

func (s *Samples) destinationName(i int) string {
	if len(s.integration) > 0 && s.sampleConfig.hasIntegrations() {
		if s.integration[0] == "all" {
			return s.sampleConfig.Integrations[i].Name
		}

		return s.integration[i]
	}

	return ""
}

// Cleanup performs cleanup for the recently created sample
func (s *Samples) Cleanup(name string) error {
	fmt.Println("Cleaning up...")

	return s.delete(name)
}

func (s *Samples) destinationPath(target string, integration string, folder string) string {
	if len(s.integration) <= 1 {
		return filepath.Join(target, folder)
	}

	return filepath.Join(target, integration, folder)
}

func selectOptions(template, label string, options []string) (string, error) {
	templates := &promptui.SelectTemplates{
		Selected: ansi.Faint(fmt.Sprintf("Selected %s: {{ . | bold }} ", template)),
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

func languageSelectPrompt(languages []string) (string, error) {
	var displayLangs []string

	for _, lang := range languages {
		if val, ok := languageDisplayNames[lang]; ok {
			displayLangs = append(displayLangs, val)
		} else {
			displayLangs = append(displayLangs, lang)
		}
	}

	selected, err := selectOptions("language", "What language would you like to use", displayLangs)
	if err != nil {
		return "", err
	}

	if val, ok := displayNameLanguages[selected]; ok {
		return val, nil
	}

	return selected, nil
}

func integrationSelectPrompt(integrations []string) ([]string, error) {
	selected, err := selectOptions("integration", "What type of integration would you like to use", append(integrations, "all"))
	if err != nil {
		return []string{}, err
	}

	if selected == "all" {
		return integrations, nil
	}

	return []string{selected}, nil
}
