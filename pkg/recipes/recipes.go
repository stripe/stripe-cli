package recipes

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

// Recipes stores the information for the selected recipe in addition to the
// selected configuration option to copy over
type Recipes struct {
	Config config.Config
	Fs     afero.Fs
	git    git.Interface

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

// Initialize get the recipe ready for the user to copy. It:
// 1. creates the recipe cache folder if it doesn't exist
// 2. store the path of the locale cache folder for later use
// 3. if the selected app does not exist in the local cache folder, clone it
// 4. if the selected app does exist in the local cache folder, pull changes
// 5. see if there are different integrations available for the recipe
// 6. see what languages the recipe is available in
func (r *Recipes) Initialize(app string) error {
	appPath, err := r.appCacheFolder(app)
	if err != nil {
		return err
	}

	// We still set the repo path here. There are some failure cases
	// that we can still work with (like no updates or repo already exists)
	r.repo = appPath

	if _, err := r.Fs.Stat(appPath); os.IsNotExist(err) {
		err = r.git.Clone(appPath, recipesList[app])
		if err != nil {
			return err
		}
	} else {
		err := r.git.Pull(appPath)
		if err != nil {
			return err
		}
	}

	// Recipes can have multiple integration types, each of which will have its
	// own client/server implementation. For example, the adding sales tax
	// sample, has a manual confirmation and automatic confirmation integration.
	// These integrations are stored as folders in the top-level of the recipe.
	// Since much of the recipe setup logic is going to be dependent on the
	// structure of the recipe, we want to check for whether there are
	// integrations upfront.
	err = r.checkForIntegrations()
	if err != nil {
		return err
	}

	// Once we've pulled the integration, we want to check what languages are
	// supported so that we can ask the user which language they want to copy.
	err = r.loadLanguages()
	if err != nil {
		return err
	}

	return nil
}

// checkForIntegrations scans the recipe to see if there are different
// integration option available. Integratios are the different ways to build
// the specific recipe, for example if it uses charges or payment intents
// would be two separate integrations.
//
// A recipe's folder structure will either contain "client" and "server"
// folders in its top-level or it'll have folders that each contain a different
// integration. This function scans to see if there is a "server" folder in the
// top level and uses that to determine if there are integrations.
func (r *Recipes) checkForIntegrations() error {
	folders, err := r.GetFolders(r.repo)
	if err != nil {
		return err
	}

	if !folderSearch(folders, "server") {
		r.integrations = folders
		r.isIntegration = true
	}

	r.isIntegration = false
	return nil
}

// Each recipe will have specific languages that it supports. Right now, there
// is a goal to support java, node, php, python, and ruby for all our recipes.
// We did not hard code those to avoid having to release a CLI update if we
// ever add new language support.
//
// Recipes do not release until all integrations have all supported languages
// built out. With that, we can simply check the languages supported in any
// folder and assume that all will have the same languages.
func (r *Recipes) loadLanguages() error {
	var err error

	if r.isIntegration {
		// The same languages will be supported by all integrations in a repo so we can
		// rely on only checking the first
		r.languages, err = r.GetFolders(filepath.Join(r.repo, r.integrations[0], "server"))
	} else {
		r.languages, err = r.GetFolders(filepath.Join(r.repo, "server"))
	}

	if err != nil {
		return err
	}

	return nil
}

// SelectOptions prompts the user to select the integration they want to use
// (if available) and the language they want the integration to be.
func (r *Recipes) SelectOptions() error {
	if r.isIntegration {
		r.integration = integrationSelectPrompt(r.integrations)
	}

	r.language = languageSelectPrompt(r.languages)

	if r.isIntegration {
		fmt.Println("Setting up", ansi.Bold(r.language), "for", ansi.Bold(strings.Join(r.integration, ",")))
	} else {
		fmt.Println("Setting up", ansi.Bold(r.language))
	}

	return nil
}

// Copy will copy all of the files from the selected configuration above over.
// This has a few different behaviors, depending on the configuration.
// Ultimately, we want the user to do as minimal of folder traversing as
// possible. What we want to end up with is:
//
// |- example-recipe/
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
func (r *Recipes) Copy(target string) error {
	// The condition for the loop starts as true since we will always want to
	// process at least once.
	for i := 0; true; i++ {
		integration := r.destinationName(i)

		serverSource := filepath.Join(r.repo, integration, "server", r.language)
		clientSource := filepath.Join(r.repo, integration, "client")
		filesSource, err := r.GetFiles(filepath.Join(r.repo, integration))
		if err != nil {
			return err
		}

		serverDestination := r.destinationPath(target, integration, "server")
		clientDestination := r.destinationPath(target, integration, "client")

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
			err = copy.Copy(filepath.Join(r.repo, integration, file), filepath.Join(target, integration, file))
			if err != nil {
				return err
			}
		}

		if i >= len(r.integration)-1 {
			break
		}
	}

	// This copies all top-level files specific to the entire recipe repo
	filesSource, err := r.GetFiles(r.repo)
	if err != nil {
		return err
	}
	for _, file := range filesSource {
		err = copy.Copy(filepath.Join(r.repo, file), filepath.Join(target, file))
		if err != nil {
			return err
		}
	}

	return nil
}

func (r *Recipes) destinationName(i int) string {
	if len(r.integration) > 0 && r.isIntegration {
		if r.integration[0] == "all" {
			return r.integrations[i]
		}

		return r.integration[i]
	}

	return ""
}

func (r *Recipes) destinationPath(target string, integration string, folder string) string {
	if len(r.integration) <= 1 {
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
