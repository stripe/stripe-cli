package login

import (
	"context"
	"fmt"
	"os"

	"github.com/stripe/stripe-cli/pkg/ansi"
)

// SuccessMessage returns the display message for a successfully authenticated user
func SuccessMessage(ctx context.Context, account *Account, baseURL string, apiKey string) (string, error) {
	// Account will be nil if user did interactive login
	if account == nil {
		acc, err := GetUserAccount(ctx, baseURL, apiKey)
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
