package login

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"

	"github.com/stripe/stripe-cli/pkg/ansi"
	"github.com/stripe/stripe-cli/pkg/config"
	"github.com/stripe/stripe-cli/pkg/errorcategory"
)

type listAccountsResponse struct {
	Accounts []config.AuthorizedAccount `json:"accounts"`
}

// ListAuthorizedAccounts returns the Stripe accounts accessible to accessToken.
func ListAuthorizedAccounts(ctx context.Context, accessBaseURL, accessToken string) ([]config.AuthorizedAccount, error) {
	return fetchAuthorizedAccounts(ctx, accessBaseURL, accessToken)
}

func fetchAuthorizedAccounts(ctx context.Context, accessBaseURL, accessToken string) ([]config.AuthorizedAccount, error) {
	endpoint := accessBaseURL + accessAPNPath + "/token/accounts"
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+accessToken)

	resp, err := accessSrvHTTPClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		return nil, errorcategory.Errorf(errorcategory.Auth, "accounts request failed (status %d): %s", resp.StatusCode, string(body))
	}

	var result listAccountsResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("failed to parse accounts response: %w", err)
	}
	return result.Accounts, nil
}

// PrintAuthorizedContexts fetches the authorized accounts for accessToken and
// prints them as a formatted list, marking the active context.
func PrintAuthorizedContexts(ctx context.Context, accessBaseURL, accessToken string) error {
	accounts, err := ListAuthorizedAccounts(ctx, accessBaseURL, accessToken)
	if err != nil {
		return fmt.Errorf("failed to fetch authorized accounts: %w", err)
	}

	ac, _ := config.GetActiveContext()
	activeID, activeMode := "", "test"
	if ac != nil {
		activeID = ac.AccountID
		if ac.Livemode {
			activeMode = "live"
		}
	}

	type row struct {
		name   string
		id     string
		mode   string
		active bool
	}
	var rows []row
	nameW, idW := 0, 0
	for _, a := range accounts {
		modes := a.Modes
		if len(modes) == 0 {
			modes = []string{"test"}
		}
		for _, m := range modes {
			rows = append(rows, row{
				name:   a.Name,
				id:     a.ID,
				mode:   displayMode(m),
				active: a.ID == activeID && m == activeMode,
			})
			if len(a.Name) > nameW {
				nameW = len(a.Name)
			}
			if len(a.ID) > idW {
				idW = len(a.ID)
			}
		}
	}

	color := ansi.Color(os.Stdout)
	fmt.Printf("Authorized contexts (%d):\n", len(rows))
	for _, r := range rows {
		if r.active {
			fmt.Printf("  %-*s  %-*s  %-7s  %s\n", nameW, r.name, idW, r.id, r.mode, color.Green("● active"))
		} else {
			fmt.Printf("  %-*s  %-*s  %s\n", nameW, r.name, idW, r.id, r.mode)
		}
	}
	return nil
}

// displayMode maps the API's "test" mode value to the CLI's "sandbox" terminology.
func displayMode(m string) string {
	if m == "test" {
		return "sandbox"
	}
	return m
}

// pickActiveContext selects the active account and livemode from the list of
// authorized accounts. It prefers the first test-mode account; if none is
// found it falls back to the first account's first mode.
func pickActiveContext(accounts []config.AuthorizedAccount) (accountID string, livemode bool) {
	for _, a := range accounts {
		for _, m := range a.Modes {
			if m == "test" {
				return a.ID, false
			}
		}
	}
	if len(accounts) > 0 {
		a := accounts[0]
		if len(a.Modes) > 0 {
			return a.ID, a.Modes[0] == "live"
		}
		return a.ID, false
	}
	return "", false
}

// populateProfileFromAccounts saves the active OAuth context and updates the
// profile with the active account's display info.
func populateProfileFromAccounts(cfg *config.Config, accounts []config.AuthorizedAccount, activeID string, activeLivemode bool) error {
	if len(accounts) == 0 {
		return errorcategory.Errorf(errorcategory.Auth, "no authorized accounts returned")
	}

	if err := config.SaveActiveContext(activeID, activeLivemode); err != nil {
		return err
	}

	cfg.Profile.AccountID = activeID
	for _, a := range accounts {
		if a.ID == activeID {
			cfg.Profile.DisplayName = a.Name
			break
		}
	}
	return cfg.Profile.CreateProfile()
}
