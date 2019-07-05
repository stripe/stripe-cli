package login

import (
	"bufio"
	"errors"
	"fmt"
	"github.com/stripe/stripe-cli/ansi"
	"github.com/stripe/stripe-cli/profile"
	"github.com/stripe/stripe-cli/validators"
	"golang.org/x/crypto/ssh/terminal"
	"io"
	"os"
	"strings"
	"syscall"
)

// InteractiveLogin lets the user set configuration on the command line
func InteractiveLogin(profile profile.Profile) error {

	apiKey, err := getConfigureAPIKey(os.Stdin)
	if err != nil {
		return err
	}

	profile.DeviceName = getConfigureDeviceName(os.Stdin)

	configErr := profile.ConfigureProfile(apiKey)
	if configErr != nil {
		return configErr
	}

	return nil
}

func getConfigureAPIKey(input io.Reader) (string, error) {
	apiKey, err := securePrompt(input)
	if err != nil {
		return "", err
	}
	apiKey = strings.TrimSpace(apiKey)
	if apiKey == "" {
		return "", errors.New("API key is required, please provide your test mode secret API key")
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

func  securePrompt(input io.Reader) (string, error) {
	if input == os.Stdin {
		buf, err := terminal.ReadPassword(int(syscall.Stdin))
		if err != nil {
			return "", err
		}
		fmt.Print("\n")
		return string(buf), nil
	}

	reader := bufio.NewReader(input)
	return reader.ReadString('\n')
}
