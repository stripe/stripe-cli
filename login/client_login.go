package login

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const stripeCLIAuthURL = "https://dashboard.stripe.com/stripecli/auth"

// Links provides the URLs for the CLI to continue the login flow
type Links struct {
	BrowserURL       string `json:"browser_url"`
	PollURL          string `json:"poll_url"`
	VerificationCode string `json:"verification_code"`
}

func getLinks(authURL string, deviceName string) (Links, error) {
	var netClient = &http.Client{
		Timeout: time.Second * 30,
	}

	if authURL == "" {
		authURL = stripeCLIAuthURL
	}

	data := url.Values{}
	data.Set("device_name", deviceName)

	res, err := netClient.Post(authURL, "application/x-www-form-urlencoded", strings.NewReader(data.Encode()))
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
