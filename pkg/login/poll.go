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

type pollAPIKeyResponse struct {
	Redeemed  bool   `json:"redeemed"`
	AccountID string `json:"account_id"`
	APIKey    string `json:"testmode_key_secret"`
}

// PollForKey polls Stripe at the specified interval until either the API key is available or we've reached the max attempts.
func PollForKey(pollURL string, interval time.Duration, maxAttempts int) (string, string, error) {
	if maxAttempts == 0 {
		maxAttempts = maxAttemptsDefault
	}

	if interval == 0 {
		interval = intervalDefault
	}

	parsedURL, err := url.Parse(pollURL)
	if err != nil {
		return "", "", err
	}

	baseURL := &url.URL{Scheme: parsedURL.Scheme, Host: parsedURL.Host}

	client := &stripe.Client{
		BaseURL: baseURL,
	}

	var count = 0
	for count < maxAttempts {
		res, err := client.PerformRequest("GET", parsedURL.Path, parsedURL.Query(), nil)
		if err != nil {
			return "", "", err
		}

		bodyBytes, err := ioutil.ReadAll(res.Body)
		if err != nil {
			return "", "", err
		}

		if res.StatusCode != http.StatusOK {
			return "", "", fmt.Errorf("unexpected http status code: %d %s", res.StatusCode, string(bodyBytes))
		}

		var response pollAPIKeyResponse
		jsonErr := json.Unmarshal(bodyBytes, &response)
		if jsonErr != nil {
			return "", "", jsonErr
		}

		if response.Redeemed {
			return response.APIKey, response.AccountID, nil
		}

		count++
		time.Sleep(interval)

	}

	return "", "", errors.New("exceeded max attempts")
}
