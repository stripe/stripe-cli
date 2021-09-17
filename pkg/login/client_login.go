package login

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"

	"github.com/briandowns/spinner"

	"github.com/stripe/stripe-cli/pkg/ansi"
	"github.com/stripe/stripe-cli/pkg/config"
	"github.com/stripe/stripe-cli/pkg/open"
	"github.com/stripe/stripe-cli/pkg/stripe"
	"github.com/stripe/stripe-cli/pkg/validators"
)

var openBrowser = open.Browser
var canOpenBrowser = open.CanOpenBrowser

const stripeCLIAuthPath = "/stripecli/auth"

// Links provides the URLs for the CLI to continue the login flow
type Links struct {
	BrowserURL       string `json:"browser_url"`
	PollURL          string `json:"poll_url"`
	VerificationCode string `json:"verification_code"`
}

// TODO
/*
4. Observability and associated alerting? Business metrics (how many users use this flow)?
5. Rate limiting for each operation?
6. Audit trail for key generation
7. Move configuration changes to profile package
*/

// Login function is used to obtain credentials via stripe dashboard.
func Login(ctx context.Context, baseURL string, config *config.Config, input io.Reader) error {
	links, err := GetLinks(ctx, baseURL, config.Profile.DeviceName)
	if err != nil {
		return err
	}

	color := ansi.Color(os.Stdout)
	fmt.Printf("Your pairing code is: %s\n", color.Bold(links.VerificationCode))
	fmt.Println(ansi.Faint("This pairing code verifies your authentication with Stripe."))

	var s *spinner.Spinner

	if isSSH() || !canOpenBrowser() {
		fmt.Printf("To authenticate with Stripe, please go to: %s\n", links.BrowserURL)

		s = ansi.StartNewSpinner("Waiting for confirmation...", os.Stdout)
	} else {
		fmt.Printf("Press Enter to open the browser or visit %s (^C to quit)", links.BrowserURL)
		fmt.Fscanln(input)

		s = ansi.StartNewSpinner("Waiting for confirmation...", os.Stdout)

		err = openBrowser(links.BrowserURL)
		if err != nil {
			msg := fmt.Sprintf("Failed to open browser, please go to %s manually.", links.BrowserURL)
			ansi.StopSpinner(s, msg, os.Stdout)
			s = ansi.StartNewSpinner("Waiting for confirmation...", os.Stdout)
		}
	}

	response, account, err := PollForKey(ctx, links.PollURL, 0, 0)
	if err != nil {
		return err
	}

	err = ConfigureProfile(config, response)
	if err != nil {
		return err
	}

	message, err := SuccessMessage(ctx, account, stripe.DefaultAPIBaseURL, response.TestModeAPIKey)
	if err != nil {
		fmt.Printf("> Error verifying the CLI was set up successfully: %s\n", err)
		return err
	}

	ansi.StopSpinner(s, message, os.Stdout)
	fmt.Println(ansi.Italic("Please note: this key will expire after 90 days, at which point you'll need to re-authenticate."))
	return nil
}

// ConfigureProfile function sets config for this profile.
func ConfigureProfile(config *config.Config, response *PollAPIKeyResponse) error {
	validateErr := validators.APIKey(response.TestModeAPIKey)
	if validateErr != nil {
		return validateErr
	}

	config.Profile.LiveModeAPIKey = response.LiveModeAPIKey
	config.Profile.LiveModePublishableKey = response.LiveModePublishableKey
	config.Profile.TestModeAPIKey = response.TestModeAPIKey
	config.Profile.TestModePublishableKey = response.TestModePublishableKey
	config.Profile.DisplayName = response.AccountDisplayName
	config.Profile.AccountID = response.AccountID

	profileErr := config.Profile.CreateProfile()
	if profileErr != nil {
		return profileErr
	}

	return nil
}

// GetLinks provides the URLs for the CLI to continue the login flow
func GetLinks(ctx context.Context, baseURL string, deviceName string) (*Links, error) {
	parsedBaseURL, err := url.Parse(baseURL)
	if err != nil {
		return nil, err
	}

	client := &stripe.Client{
		BaseURL: parsedBaseURL,
	}

	data := url.Values{}
	data.Set("device_name", deviceName)

	res, err := client.PerformRequest(ctx, http.MethodPost, stripeCLIAuthPath, data.Encode(), nil)
	if err != nil {
		return nil, err
	}

	defer res.Body.Close()

	bodyBytes, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return nil, err
	}

	if res.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected http status code: %d %s", res.StatusCode, string(bodyBytes))
	}

	var links Links

	err = json.Unmarshal(bodyBytes, &links)
	if err != nil {
		return nil, err
	}

	return &links, nil
}

func isSSH() bool {
	if os.Getenv("SSH_TTY") != "" || os.Getenv("SSH_CONNECTION") != "" || os.Getenv("SSH_CLIENT") != "" {
		return true
	}

	return false
}
