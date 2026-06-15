package login

import (
	"context"
	"fmt"
	"os"

	"github.com/stripe/stripe-cli/pkg/ansi"
	"github.com/stripe/stripe-cli/pkg/i18n"
	"github.com/stripe/stripe-cli/pkg/login/acct"
)

// SuccessMessage returns the display message for a successfully authenticated user
func SuccessMessage(ctx context.Context, account *acct.Account, baseURL string, apiKey string) (string, error) {
	// Account will be nil if user did interactive login
	if account == nil {
		acc, err := acct.GetUserAccount(ctx, baseURL, apiKey)
		if err != nil {
			return "", err
		}

		account = acc
	}

	color := ansi.Color(os.Stdout)

	displayName := account.Settings.Dashboard.DisplayName
	accountID := account.ID

	if displayName != "" && accountID != "" {
		return i18n.Tf("login_flow.success.with_display_and_account",
			i18n.Args{
				"display_name": fmt.Sprint(color.Bold(displayName)),
				"account_id":   fmt.Sprint(color.Bold(accountID)),
			},
		), nil
	}

	if accountID != "" {
		return i18n.Tf("login_flow.success.with_account",
			i18n.Args{"account_id": fmt.Sprint(color.Bold(accountID))},
		), nil
	}

	return i18n.T("login_flow.success.basic"), nil
}
