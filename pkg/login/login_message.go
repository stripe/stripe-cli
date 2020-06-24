package login

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"os"

	"github.com/stripe/stripe-cli/pkg/ansi"
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

// SuccessMessage returns the display message for a successfully authenticated user
func SuccessMessage(account *Account, baseURL string, apiKey string) (string, error) {
	// Account will be nil if user did interactive login
	if account == nil {
		acc, err := getUserAccount(baseURL, apiKey)
		if err != nil {
			return "", err
		}

		account = acc
	}

	color := ansi.Color(os.Stdout)

	displayName := account.Settings.Dashboard.DisplayName
	accountID := account.ID

	if displayName != "" && accountID != "" {
		return fmt.Sprintf(
			"Done! The Stripe CLI is configured for %s with account id %s\n",
			color.Bold(displayName),
			color.Bold(accountID),
		), nil
	}

	if accountID != "" {
		return fmt.Sprintf(
			"Done! The Stripe CLI is configured for your account with account id %s\n",
			color.Bold(accountID),
		), nil
	}

	return "Done! The Stripe CLI is configured\n", nil
}

func getUserAccount(baseURL string, apiKey string) (*Account, error) {
	parsedBaseURL, err := url.Parse(baseURL)
	if err != nil {
		return nil, err
	}

	client := &stripe.Client{
		BaseURL: parsedBaseURL,
		APIKey:  apiKey,
	}

	resp, err := client.PerformRequest(context.TODO(), "GET", "/v1/account", "", nil)

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
