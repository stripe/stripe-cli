package samples

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"os"
	"path/filepath"

	"github.com/joho/godotenv"
	"github.com/otiai10/copy"
	"github.com/spf13/afero"
	"gopkg.in/src-d/go-git.v4"

	log "github.com/sirupsen/logrus"

	"github.com/stripe/stripe-cli/pkg/config"
	g "github.com/stripe/stripe-cli/pkg/git"
	gitpkg "github.com/stripe/stripe-cli/pkg/git"
	"github.com/stripe/stripe-cli/pkg/stripe"
	"github.com/stripe/stripe-cli/pkg/stripeauth"
)

// SampleConfig contains all the configuration options for a sample
type SampleConfig struct {
	Name            string                    `json:"name"`
	ConfigureDotEnv bool                      `json:"configureDotEnv"`
	PostInstall     map[string]string         `json:"postInstall"`
	Integrations    []SampleConfigIntegration `json:"integrations"`
}

// HasIntegrations returns true if the sample has multiple integrations
func (sc *SampleConfig) HasIntegrations() bool {
	return len(sc.Integrations) > 1
}

// IntegrationNames returns the names of the available integrations for the sample
func (sc *SampleConfig) IntegrationNames() []string {
	names := []string{}
	for _, integration := range sc.Integrations {
		names = append(names, integration.Name)
	}

	return names
}

func (sc *SampleConfig) integrationServers(name string) []string {
	for _, integration := range sc.Integrations {
		if integration.Name == name {
			return integration.Servers
		}
	}

	return []string{}
}

// SampleConfigIntegration is a particular integration for a sample
type SampleConfigIntegration struct {
	Name string `json:"name"`
	// Clients are the frontend clients built for each sample
	Clients []string `json:"clients"`
	// Servers are the backend server implementations available for a sample
	Servers []string `json:"servers"`
}

func (i *SampleConfigIntegration) hasClients() bool {
	return len(i.Clients) > 0
}

func (i *SampleConfigIntegration) hasServers() bool {
	return len(i.Servers) > 0
}

// HasMultipleClients returns true if this integration has multiple options for the client language
func (i *SampleConfigIntegration) HasMultipleClients() bool {
	return len(i.Clients) > 1
}

// HasMultipleServers returns true if this integration has multiple options for the server language
func (i *SampleConfigIntegration) HasMultipleServers() bool {
	return len(i.Servers) > 1
}

func (i *SampleConfigIntegration) name() string {
	if i.Name == "main" {
		return ""
	}

	return i.Name
}

// SelectedConfig is the sample config that the user has selected to create
type SelectedConfig struct {
	Integration *SampleConfigIntegration
	Client      string
	Server      string
}

// SampleManager supports operations related to listing, cloning, and setup of
// Stripe samples. It contains configurable dependencies like `SampleLister`
// and `Fs` so that the behavior can be customized in tests or plugins, and it
// contains stateful properties like `repoPath`, `selectedConfig` that keep
// track of the operation as it goes along.
type SampleManager struct {
	// Holds functionality for getting the list of stripe sample repositories.
	SampleLister SampleLister
	// Function to populating the .env file in a created sample
	// (using information from the user's stripe-cli config)
	ConfigureDotEnv func(ctx context.Context, config *config.Config) (map[string]string, error)
	// Filesystem operations
	Fs afero.Fs
	// Git operations.
	Git g.Interface

	Config *config.Config

	// Filesystem path where the sample repository is cloned
	repoPath string

	// Information (defined by the sample) about what integrations it supports
	// and how it ought to be set up by the CLI.
	SampleConfig SampleConfig

	// Represents the user selection about which part of the sample to copy over during
	// setup.
	SelectedConfig SelectedConfig
}

// NewSampleManager creates a SampleManager with default behavior. I.e. it uses
// go-git for git operations, it uses the standard filesystem implementation,
// it lists samples by fetching samples.json from the standard samples-list
// repository, and it copies .env files in the standard way
func NewSampleManager(config *config.Config) (*SampleManager, error) {
	sampleManager := &SampleManager{
		Fs:              afero.NewOsFs(),
		Git:             gitpkg.Operations{},
		ConfigureDotEnv: ConfigureDotEnv,
		Config:          config,
	}

	cacheFolder, err := sampleManager.appCacheFolder("samples-list")
	if err != nil {
		return nil, err
	}
	sampleManager.SampleLister = newCachedGithubSampleLister(sampleManager, sampleListGithubURL, cacheFolder)
	return sampleManager, nil
}

// Initialize gets the sample ready for the user to copy. It:
// 1. creates the sample cache folder if it doesn't exist
// 2. store the path of the local cache folder for later use
// 3. if the selected app does not exist in the local cache folder, clone it
// 4. if the selected app does exist in the local cache folder, pull changes
// 5. parse the sample cli config file
func (s *SampleManager) Initialize(app string) error {
	if app == "" {
		return errors.New("Sample name is empty")
	}

	appPath, err := s.appCacheFolder(app)
	if err != nil {
		return err
	}

	// We still set the repo path here. There are some failure cases
	// that we can still work with (like no updates or repo already exists)
	s.repoPath = appPath

	list, err := s.SampleLister.ListSamples("create")
	if err != nil {
		return err
	}

	if _, err := s.Fs.Stat(appPath); os.IsNotExist(err) {
		sampleData, ok := list[app]
		if !ok {
			return fmt.Errorf("Sample %s does not exist", app)
		}
		err = s.Git.Clone(appPath, sampleData.GitRepo())
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

	err = json.Unmarshal(configFile, &s.SampleConfig)
	if err != nil {
		return err
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
//   - If there are no integrations available, copy the top-level files, the
//     client folder, and the selected language inside of the server folder to
//     the server top-level (example above)
//   - If the user selects an integration, mirror the structure above for the
//     selected integration (example above)
func (s *SampleManager) Copy(target string) error {
	integration := s.SelectedConfig.Integration.name()

	if s.SelectedConfig.Integration.hasServers() {
		// empty string is a valid option
		if s.SelectedConfig.Server != "" && !contains(s.SelectedConfig.Integration.Servers, s.SelectedConfig.Server) {
			return fmt.Errorf(
				"Server %s doesn't exist for sample integration %s. Available servers: %v",
				s.SelectedConfig.Server,
				integration,
				s.SelectedConfig.Integration.Servers,
			)
		}

		serverSource := filepath.Join(s.repoPath, integration, "server", s.SelectedConfig.Server)
		serverDestination := filepath.Join(target, "server")

		err := copy.Copy(serverSource, serverDestination)
		if err != nil {
			return err
		}
	}

	if s.SelectedConfig.Integration.hasClients() {
		// empty string is a valid option
		if s.SelectedConfig.Client != "" && !contains(s.SelectedConfig.Integration.Clients, s.SelectedConfig.Client) {
			return fmt.Errorf(
				"Client %s doesn't exist for sample integration %s. Available clients: %v",
				s.SelectedConfig.Client,
				integration,
				s.SelectedConfig.Integration.Clients,
			)
		}

		clientSource := filepath.Join(s.repoPath, integration, "client", s.SelectedConfig.Client)
		clientDestination := filepath.Join(target, "client")

		err := copy.Copy(clientSource, clientDestination)
		if err != nil {
			return err
		}
	}

	filesSource, err := s.GetFiles(filepath.Join(s.repoPath, integration))
	if err != nil {
		return err
	}

	for _, file := range filesSource {
		err = copy.Copy(filepath.Join(s.repoPath, integration, file), filepath.Join(target, file))
		if err != nil {
			return err
		}
	}

	// This copies all top-level files specific to the entire sample repo
	filesSource, err = s.GetFiles(s.repoPath)
	if err != nil {
		return err
	}

	for _, file := range filesSource {
		err = copy.Copy(filepath.Join(s.repoPath, file), filepath.Join(target, file))
		if err != nil {
			return err
		}
	}

	return nil
}

// ConfigureDotEnv returns a map of environment variables to copy into the
// .env file of the target project, based on the user's stripe-cli profile.
func ConfigureDotEnv(ctx context.Context, config *config.Config) (map[string]string, error) {
	publishableKey, _ := config.Profile.GetPublishableKey(false)
	if publishableKey == "" {
		return nil, fmt.Errorf("we could not set the publishable key in the .env file; please set this manually or login again to set it automatically next time")
	}

	apiKey, err := config.Profile.GetAPIKey(false)
	if err != nil {
		return nil, err
	}

	deviceName, err := config.Profile.GetDeviceName()
	if err != nil {
		return nil, err
	}

	apiBase, _ := url.Parse(stripe.DefaultAPIBaseURL)

	stripeClient := &stripe.Client{
		APIKey:  apiKey,
		BaseURL: apiBase,
	}
	authClient := stripeauth.NewClient(stripeClient, nil)

	authSession, err := authClient.Authorize(ctx, deviceName, "webhooks", nil, nil)
	if err != nil {
		return nil, err
	}

	return map[string]string{
		"STRIPE_PUBLISHABLE_KEY": publishableKey,
		"STRIPE_SECRET_KEY":      apiKey,
		"STRIPE_WEBHOOK_SECRET":  authSession.Secret,
		"STATIC_DIR":             "../client",
	}, nil
}

// WriteDotEnv takes the .env.example from the provided location and
// modifies it to automatically configure it for the users settings
func (s *SampleManager) WriteDotEnv(ctx context.Context, sampleLocation string) error {
	if s.SelectedConfig.Integration.hasServers() {
		if !s.SampleConfig.ConfigureDotEnv {
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

		envVars, err := s.ConfigureDotEnv(ctx, s.Config)
		if err != nil {
			return err
		}
		for k, v := range envVars {
			dotenv[k] = v
		}

		envFile := filepath.Join(sampleLocation, "server", ".env")
		err = godotenv.Write(dotenv, envFile)
		if err != nil {
			return err
		}
	}

	return nil
}

// PostInstall returns any installation for post installation instructions
func (s *SampleManager) PostInstall() string {
	message := s.SampleConfig.PostInstall["message"]
	return message
}

// Cleanup performs cleanup for the recently created sample
func (s *SampleManager) Cleanup(name string) error {
	fmt.Println("Cleaning up...")

	return s.delete(name)
}

// DeleteCache forces the local sample cache to refresh in case something
// goes awry during the initial clone or to clean out stale samples
func (s *SampleManager) DeleteCache(sample string) error {
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

// contains returns true if s contains e.
func contains(s []string, e string) bool {
	for _, a := range s {
		if a == e {
			return true
		}
	}
	return false
}

// GetSampleConfig returns the available config for this sample
func (s *SampleManager) GetSampleConfig(sampleName string, forceRefresh bool) (*SampleConfig, error) {
	if forceRefresh {
		err := s.DeleteCache(sampleName)
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

	samplesList, err := s.SampleLister.ListSamples("create")
	if err != nil {
		return nil, err
	}
	if _, ok := samplesList[sampleName]; !ok {
		errorMessage := fmt.Sprintf(`The sample provided is not currently supported by the CLI: %s
To see supported samples, run 'stripe samples list'`, sampleName)
		return nil, fmt.Errorf(errorMessage)
	}

	err = s.Initialize(sampleName)
	if err != nil {
		return nil, err
	}

	return &s.SampleConfig, nil
}
