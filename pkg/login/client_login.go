package login

import (
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"

	"github.com/stripe/stripe-cli/pkg/ansi"
	"github.com/stripe/stripe-cli/pkg/config"
	"github.com/stripe/stripe-cli/pkg/open"
	"github.com/stripe/stripe-cli/pkg/stripe"
	"github.com/stripe/stripe-cli/pkg/validators"
)

var openBrowser = open.Browser

const stripeCLIAuthPath = "/stripecli/auth"

// Links provides the URLs for the CLI to continue the login flow
type Links struct {
	BrowserURL       string `json:"browser_url"`
	PollURL          string `json:"poll_url"`
	VerificationCode string `json:"verification_code"`
}

//TODO
/*
4. Observability and associated alerting? Business metrics (how many users use this flow)?
5. Rate limiting for each operation?
6. Audit trail for key generation
7. Move configuration changes to profile package
*/

// Login function is used to obtain credentials via stripe dashboard.
func Login(baseURL string, config *config.Config, input io.Reader) error {
	links, err := getLinks(baseURL, config.Profile.DeviceName)
	if err != nil {
		return err
	}

	color := ansi.Color(os.Stdout)
	fmt.Printf("Your pairing code is: %s\n", color.Bold(links.VerificationCode))

	fmt.Printf("Press Enter to open up the browser (^C to quit)")
	fmt.Fscanln(input)

	s := ansi.StartSpinner("Waiting for confirmation...", os.Stdout)

	urlErr := openBrowser(links.BrowserURL)
	if urlErr != nil {
		return urlErr
	}

	//Call poll function
	apiKey, account, err := PollForKey(links.PollURL, 0, 0)
	if err != nil {
		return err
	}

	validateErr := validators.APIKey(apiKey)
	if validateErr != nil {
		return validateErr
	}

	config.Profile.APIKey = apiKey
	profileErr := config.Profile.CreateProfile()
	if profileErr != nil {
		return profileErr
	}

	message, err := SuccessMessage(account, stripe.DefaultAPIBaseURL, apiKey)
	if err != nil {
		fmt.Println(fmt.Sprintf("> Error verifying the CLI was set up successfully: %s", err))
	} else {
		ansi.StopSpinner(s, message, os.Stdout)
		fmt.Println(ansi.Italic("Please note: this key will expire after 90 days, at which point you'll need to re-authenticate."))
	}

	return nil
}

func getLinks(baseURL string, deviceName string) (*Links, error) {
	parsedBaseURL, err := url.Parse(baseURL)
	if err != nil {
		return nil, err
	}

	client := &stripe.Client{
		BaseURL: parsedBaseURL,
	}

	data := url.Values{}
	data.Set("device_name", deviceName)

	res, err := client.PerformRequest(http.MethodPost, stripeCLIAuthPath, data.Encode(), nil)
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
