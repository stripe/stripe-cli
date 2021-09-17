package login

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"golang.org/x/term"

	"github.com/stripe/stripe-cli/pkg/ansi"
	"github.com/stripe/stripe-cli/pkg/config"
	"github.com/stripe/stripe-cli/pkg/stripe"
	"github.com/stripe/stripe-cli/pkg/validators"
)

// InteractiveLogin lets the user set configuration on the command line
func InteractiveLogin(ctx context.Context, config *config.Config) error {
	apiKey, err := getConfigureAPIKey(os.Stdin)
	if err != nil {
		return err
	}

	config.Profile.DeviceName = getConfigureDeviceName(os.Stdin)
	config.Profile.TestModeAPIKey = apiKey
	displayName, _ := getDisplayName(ctx, nil, stripe.DefaultAPIBaseURL, apiKey)

	config.Profile.DisplayName = displayName

	profileErr := config.Profile.CreateProfile()
	if profileErr != nil {
		return profileErr
	}

	// The '>' character is automatically included at the end of client login
	// due to ansi spinner. Since no spinner is used with interactive login,
	// we need to include it manually to maintain consistency in outputs.
	message, err := SuccessMessage(ctx, nil, stripe.DefaultAPIBaseURL, apiKey)
	if err != nil {
		fmt.Printf("> Error verifying the CLI was setup successfully: %s\n", err)
	} else {
		fmt.Printf("> %s\n", message)
	}

	return nil
}

// getDisplayName returns the display name for a successfully authenticated user
func getDisplayName(ctx context.Context, account *Account, baseURL string, apiKey string) (string, error) {
	// Account will be nil if user did interactive login
	if account == nil {
		acc, err := GetUserAccount(ctx, baseURL, apiKey)
		if err != nil {
			return "", err
		}

		account = acc
	}
	displayName := account.Settings.Dashboard.DisplayName

	return displayName, nil
}

func getConfigureAPIKey(input io.Reader) (string, error) {
	fmt.Print("Enter your API key: ")

	apiKey, err := securePrompt(input)
	if err != nil {
		return "", err
	}

	apiKey = strings.TrimSpace(apiKey)
	if apiKey == "" {
		return "", errors.New("API key is required, please provide your API key")
	}

	err = validators.APIKey(apiKey)
	if err != nil {
		return "", err
	}

	fmt.Printf("Your API key is: %s\n", redactAPIKey(apiKey))

	return apiKey, nil
}

func getConfigureDeviceName(input io.Reader) string {
	hostName, _ := os.Hostname()
	reader := bufio.NewReader(input)

	color := ansi.Color(os.Stdout)
	fmt.Printf("How would you like to identify this device in the Stripe Dashboard? [default: %s] ", color.Bold(color.Cyan(hostName)))

	deviceName, _ := reader.ReadString('\n')
	if strings.TrimSpace(deviceName) == "" {
		deviceName = hostName
	}

	return deviceName
}

// redactAPIKey returns a redacted version of API keys. The first 8 and last 4
// characters are not redacted, everything else is replaced by "*" characters.
//
// It panics if the provided string has less than 12 characters.
func redactAPIKey(apiKey string) string {
	var b strings.Builder

	b.WriteString(apiKey[0:8])                         // #nosec G104 (gosec bug: https://github.com/securego/gosec/issues/267)
	b.WriteString(strings.Repeat("*", len(apiKey)-12)) // #nosec G104 (gosec bug: https://github.com/securego/gosec/issues/267)
	b.WriteString(apiKey[len(apiKey)-4:])              // #nosec G104 (gosec bug: https://github.com/securego/gosec/issues/267)

	return b.String()
}

func securePrompt(input io.Reader) (string, error) {
	if input == os.Stdin {
		// terminal.ReadPassword does not reset terminal state on ctrl-c interrupts,
		// this results in the terminal input staying hidden after program exit.
		// We need to manually catch the interrupt and restore terminal state before exiting.
		signalChan, err := protectTerminalState()
		if err != nil {
			return "", err
		}

		buf, err := term.ReadPassword(int(syscall.Stdin)) //nolint:unconvert
		if err != nil {
			return "", err
		}

		signal.Stop(signalChan)

		fmt.Print("\n")

		return string(buf), nil
	}

	reader := bufio.NewReader(input)

	return reader.ReadString('\n')
}

func protectTerminalState() (chan os.Signal, error) {
	originalTerminalState, err := term.GetState(int(syscall.Stdin)) //nolint:unconvert
	if err != nil {
		return nil, err
	}

	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, os.Interrupt)

	go func() {
		<-signalChan
		term.Restore(int(syscall.Stdin), originalTerminalState) //nolint:unconvert
		os.Exit(1)
	}()

	return signalChan, nil
}
