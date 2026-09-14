package terminal

import (
	"fmt"

	"github.com/manifoldco/promptui"

	"github.com/stripe/stripe-cli/pkg/ansi"
)

// ReaderTypeSelectPrompt prompts the user to choose which type of reader they want to set up
// currently the only supported choice is the Verifone P400
func ReaderTypeSelectPrompt(readers []string) (string, error) {
	selected, err := selectOptions("reader type", "Select which type of reader you’d like to set up", readers)

	if err != nil {
		return "", err
	}

	return selected, nil
}

func selectOptions(template string, label string, options []string) (string, error) {
	templates := &promptui.SelectTemplates{
		Selected: ansi.Faint(fmt.Sprintf("✔ Selected %s: {{ . | bold }} ", template)),
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
