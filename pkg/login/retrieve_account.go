package login

import (
	"context"
	"encoding/json"
	"net/url"

	"github.com/stripe/stripe-cli/pkg/stripe"
)

// Account is the most outer layer of the json response from Stripe
type Account struct {
	ID       string   `json:"id"`
	Settings Settings `json:"settings"`
}

// Settings is within the Account json response from Stripe
type Settings struct {
	Dashboard Dashboard `json:"dashboard"`
}

// Dashboard is within the Settings json response from Stripe
type Dashboard struct {
	DisplayName string `json:"display_name"`
}

// GetUserAccount retrieves the account information
func GetUserAccount(ctx context.Context, baseURL string, apiKey string) (*Account, error) {
	parsedBaseURL, err := url.Parse(baseURL)
	if err != nil {
		return nil, err
	}

	client := &stripe.Client{
		BaseURL: parsedBaseURL,
		APIKey:  apiKey,
	}

	resp, err := client.PerformRequest(ctx, "GET", "/v1/account", "", nil)

	if err != nil {
		return nil, err
	}

	defer resp.Body.Close()

	account := &Account{}

	err = json.NewDecoder(resp.Body).Decode(account)
	if err != nil {
		return nil, err
	}

	return account, nil
}
