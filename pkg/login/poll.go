package login

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"time"

	"github.com/stripe/stripe-cli/pkg/stripe"
)

const maxAttemptsDefault = 2 * 60
const intervalDefault = 1 * time.Second

// PollAPIKeyResponse returns the data of the polling client login
type PollAPIKeyResponse struct {
	Redeemed               bool   `json:"redeemed"`
	AccountID              string `json:"account_id"`
	AccountDisplayName     string `json:"account_display_name"`
	TestModeAPIKey         string `json:"testmode_key_secret"`
	TestModePublishableKey string `json:"testmode_key_publishable"`
}

// PollForKey polls Stripe at the specified interval until either the API key is available or we've reached the max attempts.
func PollForKey(pollURL string, interval time.Duration, maxAttempts int) (PollAPIKeyResponse, *Account, error) {
	var response PollAPIKeyResponse

	if maxAttempts == 0 {
		maxAttempts = maxAttemptsDefault
	}

	if interval == 0 {
		interval = intervalDefault
	}

	parsedURL, err := url.Parse(pollURL)
	if err != nil {
		return response, nil, err
	}

	baseURL := &url.URL{Scheme: parsedURL.Scheme, Host: parsedURL.Host}

	client := &stripe.Client{
		BaseURL: baseURL,
	}

	var count = 0
	for count < maxAttempts {
		res, err := client.PerformRequest(http.MethodGet, parsedURL.Path, parsedURL.Query().Encode(), nil)
		if err != nil {
			return response, nil, err
		}
		defer res.Body.Close()

		bodyBytes, err := ioutil.ReadAll(res.Body)
		if err != nil {
			return response, nil, err
		}

		if res.StatusCode != http.StatusOK {
			return response, nil, fmt.Errorf("unexpected http status code: %d %s", res.StatusCode, string(bodyBytes))
		}

		jsonErr := json.Unmarshal(bodyBytes, &response)
		if jsonErr != nil {
			return response, nil, jsonErr
		}

		if response.Redeemed {
			account := &Account{
				ID: response.AccountID,
			}

			account.Settings.Dashboard.DisplayName = response.AccountDisplayName

			return response, account, nil
		}

		count++
		time.Sleep(interval)

	}

	return response, nil, errors.New("exceeded max attempts")
}
