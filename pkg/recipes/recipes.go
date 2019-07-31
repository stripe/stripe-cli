package recipes

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/manifoldco/promptui"
	"github.com/stripe/stripe-cli/pkg/ansi"
	"github.com/stripe/stripe-cli/pkg/config"
)

// Recipes does stuff
// TODO
type Recipes struct {
	Config config.Config
}

func (r *Recipes) Download(app string) (string, error) {
	appPath, err := r.appCacheFolder(app)
	if err != nil {
		return "", err
	}

	if _, err := os.Stat(appPath); os.IsNotExist(err) {
		err = r.clone(appPath, app)
		if err != nil {
			return appPath, err
		}
	} else {
		err := r.pull(appPath, app)
		if err != nil {
			return appPath, err
		}
	}

	return appPath, nil
}

func (r *Recipes) BuildPrompts(repoPath string) ([]string, string, error) {
	var language string
	var integration []string

	topLevelFolders, err := r.GetFolders(repoPath)
	if err != nil {
		return []string{}, "", err
	}

	if folderSearch(topLevelFolders, "server") {
		languages, err := r.GetFolders(filepath.Join(repoPath, "server"))
		if err != nil {
			return []string{}, "", err
		}

		language = languageSelectPrompt(languages)
	} else {
		integrations, err := r.GetFolders(repoPath)
		if err != nil {
			return []string{}, "", err
		}
		integrations = append(integrations, "all")

		integration = integrationSelectPrompt(integrations)

		// All integrations will have the same language support so we can just pull the langauges
		// for the first integration type
		languages, err := r.GetFolders(filepath.Join(repoPath, integrations[0], "server"))
		if err != nil {
			return []string{}, "", err
		}

		language = languageSelectPrompt(languages)
	}

	if len(integration) > 0 {
		fmt.Println("Setting up", ansi.Bold(language), "for", ansi.Bold(strings.Join(integration, ",")))
	} else {
		fmt.Println("Setting up", ansi.Bold(language))
	}

	return integration, language, nil
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
	selected := selectOptions("What type of integration would you like to use?", integrations)
	if selected == "all" {
		return integrations
	}

	return []string{selected}
}
