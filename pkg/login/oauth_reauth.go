package login

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/signal"
	"sort"
	"strings"
	"time"

	"github.com/stripe/stripe-cli/pkg/config"
	"github.com/stripe/stripe-cli/pkg/errorcategory"
)

const reauthPath = accessAPNPath + "/token/reauth"

const (
	reauthPollInterval = 3 * time.Second
	reauthPollTimeout  = 10 * time.Minute
)

// errReauthTimeout is returned by waitForAccountsChange when reauthPollTimeout
// elapses with no detected change.
var errReauthTimeout = errorcategory.New(errorcategory.Auth, "timed out waiting for the reauthorization to finish")

type reauthResponse struct {
	ReauthURL string `json:"reauth_url"`
}

// Reauth fetches a reauthentication URL for the active OAuth session and
// directs the user to it. If a browser is available it is opened automatically;
// otherwise the URL is printed for the user to visit manually. It then waits
// for the authorized-accounts list to change before printing the updated list
// of authorized contexts; access-srv has no dedicated signal for "the user
// finished reauthorizing," so a change to the accounts/scopes returned for the
// token is used as a proxy for completion.
func Reauth(ctx context.Context, accessBaseURL, accessToken string) error {
	before, err := ListAuthorizedAccounts(ctx, accessBaseURL, accessToken)
	if err != nil {
		return err
	}

	reauthURL, err := fetchReauthURL(ctx, accessBaseURL, accessToken)
	if err != nil {
		return err
	}
	if err := validateBrowserURL(reauthURL, accessBaseURL); err != nil {
		return err
	}

	if !isSSH() && canOpenBrowser() {
		fmt.Println("Opening the Stripe Dashboard to re-authorize the CLI...")
		fmt.Printf("If the browser does not open automatically, visit:\n  %s\n", reauthURL)
		if err := openBrowser(reauthURL); err != nil {
			fmt.Fprintf(os.Stderr, "Failed to open browser: %s\n", err)
		}
	} else {
		fmt.Printf("Visit the following URL to re-authorize the CLI:\n  %s\n", reauthURL)
	}
	fmt.Println("Waiting for you to finish in the browser. Press ^C to cancel.")

	waitCtx, stop := signal.NotifyContext(ctx, os.Interrupt)
	defer stop()

	after, err := waitForAccountsChange(waitCtx, accessBaseURL, accessToken, before, reauthPollInterval, reauthPollTimeout)
	if err != nil {
		switch {
		case errors.Is(err, context.Canceled):
			fmt.Println("Canceled. Run 'stripe whoami' to check your authorized contexts.")
			return nil
		case errors.Is(err, errReauthTimeout):
			fmt.Println("Still waiting on the re-authorization. Run 'stripe whoami' once you've finished.")
			return nil
		default:
			return err
		}
	}

	ac, _ := config.GetActiveContext()
	activeID, activeLivemode := "", false
	if ac != nil {
		activeID = ac.AccountID
		activeLivemode = ac.Livemode
	}
	printAuthorizedSummary(after, activeID, activeLivemode)
	return nil
}

// waitForAccountsChange polls the authorized-accounts list on interval until
// it differs from before, ctx is canceled, or timeout elapses.
func waitForAccountsChange(ctx context.Context, accessBaseURL, accessToken string, before []config.AuthorizedAccount, interval, timeout time.Duration) ([]config.AuthorizedAccount, error) {
	beforeSig := accountsSignature(before)

	deadline := time.NewTimer(timeout)
	defer deadline.Stop()
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-deadline.C:
			return nil, errReauthTimeout
		case <-ticker.C:
			accounts, err := ListAuthorizedAccounts(ctx, accessBaseURL, accessToken)
			if err != nil {
				return nil, err
			}
			if accountsSignature(accounts) != beforeSig {
				return accounts, nil
			}
		}
	}
}

// accountsSignature returns a stable string representation of accounts,
// used to detect whether the authorized-accounts list has changed.
func accountsSignature(accounts []config.AuthorizedAccount) string {
	sorted := make([]config.AuthorizedAccount, len(accounts))
	copy(sorted, accounts)
	sort.Slice(sorted, func(i, j int) bool { return sorted[i].ID < sorted[j].ID })

	var sb strings.Builder
	for _, a := range sorted {
		modes := append([]string{}, a.Modes...)
		sort.Strings(modes)
		fmt.Fprintf(&sb, "%s|%s|%s;", a.ID, a.Name, strings.Join(modes, ","))
	}
	return sb.String()
}

func fetchReauthURL(ctx context.Context, accessBaseURL, accessToken string) (string, error) {
	endpoint := accessBaseURL + reauthPath
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("Authorization", "Bearer "+accessToken)

	resp, err := accessSrvHTTPClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	switch resp.StatusCode {
	case http.StatusOK:
		var r reauthResponse
		if err := json.Unmarshal(body, &r); err != nil {
			return "", fmt.Errorf("failed to parse reauth response: %w", err)
		}
		if r.ReauthURL == "" {
			return "", errorcategory.Errorf(errorcategory.Auth, "reauth response missing reauth_url")
		}
		return r.ReauthURL, nil
	case http.StatusUnauthorized:
		return "", errorcategory.Errorf(errorcategory.Auth, "session expired or revoked; run 'stripe login' to authenticate again")
	case http.StatusBadRequest:
		return "", errorcategory.Errorf(errorcategory.Auth, "invalid reauth request (invalid_request); check your configuration")
	default:
		return "", errorcategory.Errorf(errorcategory.Auth, "reauth request failed (status %d): %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}
}
