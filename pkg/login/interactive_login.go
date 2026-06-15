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
	"github.com/stripe/stripe-cli/pkg/i18n"
	"github.com/stripe/stripe-cli/pkg/login/acct"
	"github.com/stripe/stripe-cli/pkg/stripe"
	"github.com/stripe/stripe-cli/pkg/validators"
)

// InteractiveLogin lets the user set configuration on the command line
func InteractiveLogin(ctx context.Context, config *config.Config) error {
	return interactiveLoginWithParams(ctx, config, os.Stdin, stripe.DefaultAPIBaseURL)
}

func interactiveLoginWithParams(ctx context.Context, config *config.Config, input io.Reader, baseURL string) error {
	apiKey, err := getConfigureAPIKey(input)
	if err != nil {
		return err
	}

	config.Profile.DeviceName = getConfigureDeviceName(input)

	livemode := strings.HasPrefix(apiKey, "sk_live_") || strings.HasPrefix(apiKey, "rk_live_")
	if livemode {
		config.Profile.LiveModeAPIKey = apiKey
	} else {
		config.Profile.TestModeAPIKey = apiKey
	}

	account, err := acct.GetUserAccount(ctx, baseURL, apiKey)
	if err == nil {
		config.Profile.DisplayName = account.Settings.Dashboard.DisplayName
		config.Profile.AccountID = account.ID
	}

	profileErr := config.Profile.CreateProfile()
	if profileErr != nil {
		return profileErr
	}

	// The '>' character is automatically included at the end of client login
	// due to ansi spinner. Since no spinner is used with interactive login,
	// we need to include it manually to maintain consistency in outputs.
	message, err := SuccessMessage(ctx, nil, baseURL, apiKey)
	if err != nil {
		fmt.Print(i18n.Tf("login_flow.interactive.error_verifying", i18n.Args{"error": err.Error()}))
	} else {
		fmt.Print(i18n.Tf("login_flow.output.success", i18n.Args{"message": message}))
	}

	return nil
}

func getConfigureAPIKey(input io.Reader) (string, error) {
	fmt.Print(i18n.T("login_flow.interactive.prompt_api_key"))

	apiKey, err := securePrompt(input)
	if err != nil {
		return "", err
	}

	apiKey = strings.TrimSpace(apiKey)
	if apiKey == "" {
		return "", errors.New(i18n.T("login_flow.interactive.api_key_required"))
	}

	err = validators.APIKey(apiKey)
	if err != nil {
		return "", err
	}

	fmt.Print(i18n.Tf("login_flow.interactive.api_key_display", i18n.Args{"key": config.RedactAPIKey(apiKey)}))

	return apiKey, nil
}

func getConfigureDeviceName(input io.Reader) string {
	hostName, _ := os.Hostname()
	reader := bufio.NewReader(input)

	color := ansi.Color(os.Stdout)
	fmt.Print(i18n.Tf("login_flow.interactive.prompt_device_name", i18n.Args{"default": fmt.Sprint(color.Bold(color.Cyan(hostName)))}))

	deviceName, _ := reader.ReadString('\n')
	if strings.TrimSpace(deviceName) == "" {
		deviceName = hostName
	}

	return deviceName
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
