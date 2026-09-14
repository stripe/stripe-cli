package p400

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/manifoldco/promptui"

	"github.com/stripe/stripe-cli/pkg/ansi"
	"github.com/stripe/stripe-cli/pkg/validators"
)

// ReaderRegistrationCodePrompt prompts the user to generate a new registration code on their P400 and asks them to enter it at the prompt
// it returns the code that the user typed in
func ReaderRegistrationCodePrompt() (string, error) {
	fmt.Println("On your reader, enter the key sequence 0-7-1-3-9 to display a unique registration code.\nNow enter the registration code:")

	result, err := textPrompt("Code", nil)

	if err != nil {
		return "", err
	}

	return result, nil
}

// ReaderChargeAmountPrompt prompts the user to enter a monetary amount to charge as a test payment
// it converts the string the user entered into an integer and returns that
func ReaderChargeAmountPrompt() (int, error) {
	fmt.Println("Enter the amount you’d like to charge.")

	result, err := textPrompt("Amount", validators.OneDollar)

	if err != nil {
		return 0, err
	}

	amountInt, err := strconv.Atoi(result)

	return amountInt, err
}

// ReaderChargeCurrencyPrompt prompts the user for the currency they want to take a test payment in
// it returns the currency code that the user entered
func ReaderChargeCurrencyPrompt() (string, error) {
	fmt.Println("Now let’s take a test payment. Enter the currency code for the payment.")

	currency, err := textPrompt("Currency", nil)

	if err != nil {
		return "", err
	}

	// support any casing the user types but Stripe call needs lowercase
	currency = strings.ToLower(currency)

	return currency, nil
}

// ReaderNewOrExistingPrompt prompts the user to choose to set up a new reader, or continue with an already registered reader
// it returns their choice
func ReaderNewOrExistingPrompt() (string, error) {
	options := ActivationTypeLabels
	templates := &promptui.SelectTemplates{
		Label:    "{{ . }} ",
		Selected: ansi.Faint(fmt.Sprintf("✔ Selected %s: {{ . | bold }} ", "setup type")),
	}

	_, selected, err := selectOptions(templates, "Is this reader new or already registered?", options)

	if err != nil {
		return "", err
	}

	return selected, nil
}

// RegisteredReaderChoicePrompt takes a list of registered p400 readers and prompts the reader to choose one to use
// it returns the IP address of the chosen reader
func RegisteredReaderChoicePrompt(readerList []Reader, tsCtx TerminalSessionContext) (Reader, error) {
	var reader Reader

	templates := &promptui.SelectTemplates{
		Label:    "{{ .Label }} ({{ .Status }}) ",
		Active:   "▸ {{ .Label | underline }} ({{ .Status }})",
		Inactive: "{{ .Label }} ({{ .Status }})",
		Selected: ansi.Faint(fmt.Sprintf("✔ Selected %s: {{ .Label | bold }} ", "reader")),
	}

	index, _, err := selectOptions(templates, "Select a reader:", readerList)

	if err != nil {
		return reader, err
	}

	reader = readerList[index]

	return reader, nil
}

func textPrompt(label string, validator promptui.ValidateFunc) (string, error) {
	templates := &promptui.PromptTemplates{
		Prompt:  "▸ {{ . }}: ",
		Valid:   "▸ {{ . }}: ",
		Invalid: "▸ {{ . }}: ",
		Success: "▸ {{ . }}: ",
	}

	prompt := promptui.Prompt{
		Label:     label,
		Templates: templates,
		Validate:  validator,
	}

	result, err := prompt.Run()

	if err != nil {
		return "", err
	}

	return result, nil
}

func selectOptions(templates *promptui.SelectTemplates, label string, options interface{}) (int, string, error) {
	prompt := promptui.Select{
		Label:     label,
		Items:     options,
		Templates: templates,
	}

	index, result, err := prompt.Run()

	if err != nil {
		return 0, "", err
	}

	return index, result, nil
}
