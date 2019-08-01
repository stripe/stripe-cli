package recipes

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/manifoldco/promptui"
	"github.com/otiai10/copy"
	"github.com/stripe/stripe-cli/pkg/ansi"
	"github.com/stripe/stripe-cli/pkg/config"
)

// Recipes does stuff
// TODO
type Recipes struct {
	Config config.Config

	integrations []string
	languages    []string

	integration []string
	language    string
	repo        string
}

func (r *Recipes) Initialize(app string) error {
	appPath, err := r.appCacheFolder(app)
	if err != nil {
		return err
	}

	// We still set the repo path here. There are some failure cases
	// that we can still work with (like no updates or repo already exists)
	r.repo = appPath

	if _, err := os.Stat(appPath); os.IsNotExist(err) {
		err = r.clone(appPath, app)
		if err != nil {
			return err
		}
	} else {
		err := r.pull(appPath, app)
		if err != nil {
			return err
		}
	}

	err = r.checkForIntegrations()
	if err != nil {
		return err
	}

	err = r.loadLanguages()
	if err != nil {
		return err
	}

	return nil
}

func (r *Recipes) checkForIntegrations() error {
	folders, err := r.GetFolders(r.repo)
	if err != nil {
		return err
	}

	if !folderSearch(folders, "server") {
		r.integrations = folders
	}

	return nil
}

func (r *Recipes) loadLanguages() error {
	var err error

	if len(r.integrations) > 0 {
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

func (r *Recipes) SelectOptions() error {
	if len(r.integrations) > 0 {
		r.integration = integrationSelectPrompt(r.integrations)
	}

	r.language = languageSelectPrompt(r.languages)

	if len(r.integrations) > 0 {
		fmt.Println("Setting up", ansi.Bold(r.language), "for", ansi.Bold(strings.Join(r.integration, ",")))
	} else {
		fmt.Println("Setting up", ansi.Bold(r.language))
	}

	return nil
}

func (r *Recipes) Copy(target string) error {
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
	if len(r.integration) > 0 && len(r.integrations) > 0 {
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
