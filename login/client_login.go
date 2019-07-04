package login

import (
	"encoding/json"
	"fmt"
	"github.com/stripe/stripe-cli/stripeauth"
	"io/ioutil"
	"net/http"
	"net/url"
)

const stripeCLIAuthURL = "https://dashboard.stripe.com/stripecli/auth"

// Links provides the URLs for the CLI to continue the login flow
type Links struct {
	BrowserURL       string `json:"browser_url"`
	PollURL          string `json:"poll_url"`
	VerificationCode string `json:"verification_code"`
}

func getLinks(authURL string, deviceName string) (Links, error) {
	client := stripeauth.NewHTTPClient("")

	if authURL == "" {
		authURL = stripeCLIAuthURL
	}

	data := url.Values{}
	data.Set("device_name", deviceName)

	res, err := client.PostForm(authURL, data)
	if err != nil {
		return Links{}, err
	}

	defer res.Body.Close()

	bodyBytes, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return Links{}, err
	}

	if res.StatusCode != http.StatusOK {
		return Links{}, fmt.Errorf("unexpected http status code: %d %s", res.StatusCode, string(bodyBytes))
	}

	var links Links
	err = json.Unmarshal(bodyBytes, &links)
	if err != nil {
		return Links{}, err
	}

	return links, nil
}
