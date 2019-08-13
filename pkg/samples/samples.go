package samples

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/manifoldco/promptui"
	"github.com/otiai10/copy"
	"github.com/spf13/afero"

	"github.com/stripe/stripe-cli/pkg/ansi"
	"github.com/stripe/stripe-cli/pkg/config"
	"github.com/stripe/stripe-cli/pkg/git"
)

// Samples stores the information for the selected sample in addition to the
// selected configuration option to copy over
type Samples struct {
	Config config.Config
	Fs     afero.Fs
	Git    git.Interface

	// source repository to clone from
	repo string

	// Available integrations
	integrations  []string
	isIntegration bool

	// Available languages
	languages []string

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
// 5. see if there are different integrations available for the sample
// 6. see what languages the sample is available in
func (s *Samples) Initialize(app string) error {
	appPath, err := s.appCacheFolder(app)
	if err != nil {
		return err
	}

	// We still set the repo path here. There are some failure cases
	// that we can still work with (like no updates or repo already exists)
	s.repo = appPath

	if _, err := s.Fs.Stat(appPath); os.IsNotExist(err) {
		err = s.Git.Clone(appPath, samplesList[app])
		if err != nil {
			return err
		}
	} else {
		err := s.Git.Pull(appPath)
		if err != nil {
			return err
		}
	}

	// Samples can have multiple integration types, each of which will have its
	// own client/server implementation. For example, the adding sales tax
	// sample, has a manual confirmation and automatic confirmation integration.
	// These integrations are stored as folders in the top-level of the sample.
	// Since much of the sample setup logic is going to be dependent on the
	// structure of the sample, we want to check for whether there are
	// integrations upfront.
	err = s.checkForIntegrations()
	if err != nil {
		return err
	}

	// Once we've pulled the integration, we want to check what languages are
	// supported so that we can ask the user which language they want to copy.
	err = s.loadLanguages()
	if err != nil {
		return err
	}

	return nil
}

// checkForIntegrations scans the sample to see if there are different
// integration option available. Integratios are the different ways to build
// the specific sample, for example if it uses charges or payment intents
// would be two separate integrations.
//
// A sample's folder structure will either contain "client" and "server"
// folders in its top-level or it'll have folders that each contain a different
// integration. This function scans to see if there is a "server" folder in the
// top level and uses that to determine if there are integrations.
func (s *Samples) checkForIntegrations() error {
	folders, err := s.GetFolders(s.repo)
	if err != nil {
		return err
	}

	if !folderSearch(folders, "server") {
		s.integrations = folders
		s.isIntegration = true
		return nil
	}

	s.isIntegration = false
	return nil
}

// Each sample will have specific languages that it supports. Right now, there
// is a goal to support java, node, php, python, and ruby for all our samples.
// We did not hard code those to avoid having to release a CLI update if we
// ever add new language support.
//
// Samples do not release until all integrations have all supported languages
// built out. With that, we can simply check the languages supported in any
// folder and assume that all will have the same languages.
func (s *Samples) loadLanguages() error {
	var err error

	if s.isIntegration {
		// The same languages will be supported by all integrations in a repo so we can
		// rely on only checking the first
		s.languages, err = s.GetFolders(filepath.Join(s.repo, s.integrations[0], "server"))
	} else {
		s.languages, err = s.GetFolders(filepath.Join(s.repo, "server"))
	}

	if err != nil {
		return err
	}

	return nil
}

// SelectOptions prompts the user to select the integration they want to use
// (if available) and the language they want the integration to be.
func (s *Samples) SelectOptions() error {
	if s.isIntegration {
		s.integration = integrationSelectPrompt(s.integrations)
	}

	s.language = languageSelectPrompt(s.languages)

	if s.isIntegration {
		fmt.Println("Setting up", ansi.Bold(s.language), "for", ansi.Bold(strings.Join(s.integration, ",")))
	} else {
		fmt.Println("Setting up", ansi.Bold(s.language))
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
			err = copy.Copy(filepath.Join(s.repo, integration, file), filepath.Join(target, integration, file))
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

func (s *Samples) destinationName(i int) string {
	if len(s.integration) > 0 && s.isIntegration {
		if s.integration[0] == "all" {
			return s.integrations[i]
		}

		return s.integration[i]
	}

	return ""
}

func (s *Samples) destinationPath(target string, integration string, folder string) string {
	if len(s.integration) <= 1 {
		return filepath.Join(target, folder)
	}

	return filepath.Join(target, integration, folder)
}

func selectOptions(label string, options []string) string {
	prompt := promptui.Select{
		Label: label,
		Items: options,
	}

	_, result, err := prompt.Run()

	if err != nil {
		fmt.Printf("Prompt failed %v\n", err)
		return ""
	}

	return result
}

func languageSelectPrompt(languages []string) string {
	return selectOptions("What language would you like to use?", languages)
}

func integrationSelectPrompt(integrations []string) []string {
	selected := selectOptions("What type of integration would you like to use?", append(integrations, "all"))
	if selected == "all" {
		return integrations
	}

	return []string{selected}
}
