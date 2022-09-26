package login

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"time"

	"github.com/briandowns/spinner"

	"github.com/stripe/stripe-cli/pkg/ansi"
	configPkg "github.com/stripe/stripe-cli/pkg/config"
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
func Login(ctx context.Context, baseURL string, config *configPkg.Config, asyncInput AsyncInputReader) error {
	links, err := GetLinks(ctx, baseURL, config.Profile.DeviceName)
	if err != nil {
		return err
	}

	color := ansi.Color(os.Stdout)
	fmt.Printf("Your pairing code is: %s\n", color.Bold(links.VerificationCode))
	fmt.Println(ansi.Faint("This pairing code verifies your authentication with Stripe."))

	var s *spinner.Spinner

	pollResultCh := make(chan pollResult)
	inputCh := make(chan int)

	if isSSH() || !canOpenBrowser() {
		fmt.Printf("To authenticate with Stripe, please go to: %s\n", links.BrowserURL)
		s = ansi.StartNewSpinner("Waiting for confirmation...", os.Stdout)
		go asyncPollKey(ctx, links.PollURL, 0, 0, pollResultCh)
	} else {
		fmt.Printf("Press Enter to open the browser or visit %s (^C to quit)", links.BrowserURL)
		go asyncInput.scanln(inputCh)
		go asyncPollKey(ctx, links.PollURL, 0, 0, pollResultCh)
	}

	for {
		select {
		case <-inputCh:
			s = ansi.StartNewSpinner("Waiting for confirmation...", os.Stdout)

			err = openBrowser(links.BrowserURL)
			if err != nil {
				msg := fmt.Sprintf("Failed to open browser, please go to %s manually.", links.BrowserURL)
				ansi.StopSpinner(s, msg, os.Stdout)
				s = ansi.StartNewSpinner("Waiting for confirmation...", os.Stdout)
			}
		case res := <-pollResultCh:
			if res.err != nil {
				return res.err
			}

			err = ConfigureProfile(config, res.response)
			if err != nil {
				return err
			}

			message, err := SuccessMessage(ctx, res.account, stripe.DefaultAPIBaseURL, res.response.TestModeAPIKey)
			if err != nil {
				fmt.Printf("> Error verifying the CLI was set up successfully: %s\n", err)
				return err
			}

			if s == nil {
				fmt.Printf("\n> %s\n", message)
			} else {
				ansi.StopSpinner(s, message, os.Stdout)
			}
			fmt.Println(ansi.Italic("Please note: this key will expire after 90 days, at which point you'll need to re-authenticate."))
			return nil
		}
	}
}

// ConfigureProfile function sets config for this profile.
func ConfigureProfile(config *configPkg.Config, response *PollAPIKeyResponse) error {
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

	bodyBytes, err := io.ReadAll(res.Body)
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

type pollResult struct {
	response *PollAPIKeyResponse
	account  *Account
	err      error
}

func asyncPollKey(ctx context.Context, pollURL string, interval time.Duration, maxAttempts int, ch chan pollResult) {
	response, account, err := PollForKey(ctx, pollURL, interval, maxAttempts)
	ch <- pollResult{
		response: response,
		account:  account,
		err:      err,
	}
	close(ch)
}

// AsyncInputReader is an interface that has an async version of scanln
type AsyncInputReader interface {
	scanln(ch chan int)
}

// AsyncStdinReader implements scanln(ch chan int), an async version of scanln
type AsyncStdinReader struct {
}

func (r AsyncStdinReader) scanln(ch chan int) {
	n, _ := fmt.Fscanln(os.Stdin)
	ch <- n
}
