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
// of authorized contexts.
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

	fmt.Println()
	if !isSSH() && canOpenBrowser() {
		fmt.Printf("To authorize more contexts, visit %s\n\n", reauthURL)
		fmt.Println("Press enter to open the browser (^C to quit)")
		go func() {
			fmt.Scanln() //nolint:errcheck
			if err := openBrowser(reauthURL); err != nil {
				fmt.Fprintf(os.Stderr, "Failed to open browser: %s\n", err)
			}
		}()
	} else {
		fmt.Printf("Visit the following URL to re-authorize the CLI:\n  %s\n", reauthURL)
	}

	return waitForReauthCompletion(ctx, accessBaseURL, accessToken, before, nil)
}

type reauthSessionOutput struct {
	BrowserURL string `json:"browser_url"`
	NextStep   string `json:"next_step"`
}

// InitiateReauth fetches a reauthentication URL for the active OAuth session
// and prints it as JSON with a next_step command, then returns immediately.
// Intended for non-interactive (agent/script) use.
func InitiateReauth(ctx context.Context, accessBaseURL, accessToken string) error {
	// Best-effort: if this snapshot can't be fetched or saved, the later
	// --complete-reauth invocation (a separate process) just falls back to a
	// single check instead of polling for a change from it.
	if before, err := ListAuthorizedAccounts(ctx, accessBaseURL, accessToken); err == nil {
		_ = savePendingReauthAccounts(before)
	}

	reauthURL, err := fetchReauthURL(ctx, accessBaseURL, accessToken)
	if err != nil {
		return err
	}
	if err := validateBrowserURL(reauthURL, accessBaseURL); err != nil {
		return err
	}

	out := reauthSessionOutput{
		BrowserURL: reauthURL,
		NextStep:   "stripe login --complete-reauth",
	}
	b, err := json.MarshalIndent(out, "", "  ")
	if err != nil {
		return err
	}
	fmt.Println(string(b))
	return nil
}

// PollPendingReauth checks, immediately on invocation, whether the
// authorized-accounts list already differs from the snapshot InitiateReauth
// saved - covering both a user who finished in the browser before running
// this, and one who runs `--complete-reauth` without a snapshot at all (e.g.
// skipped `--non-interactive`). Only if there's a snapshot and nothing has
// changed yet does it fall back to waiting/polling for a change.
func PollPendingReauth(ctx context.Context, accessBaseURL, accessToken string) error {
	before, err := loadPendingReauthAccounts()
	if err != nil {
		return err
	}

	accounts, err := ListAuthorizedAccounts(ctx, accessBaseURL, accessToken)
	if err != nil {
		return err
	}

	if before == nil || accountsSignature(accounts) != accountsSignature(before) {
		if before != nil {
			clearPendingReauthAccounts()
		}
		ac, _ := config.GetActiveContext()
		activeID, activeLivemode := "", false
		if ac != nil {
			activeID = ac.AccountID
			activeLivemode = ac.Livemode
		}
		printAuthorizedSummary(accounts, activeID, activeLivemode)
		return nil
	}

	fmt.Println("Waiting for you to finish in the browser. Press ^C to cancel.")
	return waitForReauthCompletion(ctx, accessBaseURL, accessToken, before, func([]config.AuthorizedAccount) {
		clearPendingReauthAccounts()
	})
}

// waitForReauthCompletion waits for the authorized-accounts list to change
// from before, then prints the updated list of authorized contexts;
// access-srv has no dedicated signal for "the user finished reauthorizing,"
// so a change to the accounts/scopes returned for the token is used as a
// proxy for completion. onComplete, if non-nil, runs once a change is
// detected and before the summary is printed.
func waitForReauthCompletion(ctx context.Context, accessBaseURL, accessToken string, before []config.AuthorizedAccount, onComplete func([]config.AuthorizedAccount)) error {
	waitCtx, stop := signal.NotifyContext(ctx, os.Interrupt)
	defer stop()

	after, err := waitForAccountsChange(waitCtx, accessBaseURL, accessToken, before, reauthPollInterval, reauthPollTimeout)
	if err != nil {
		switch {
		case errors.Is(err, context.Canceled):
			fmt.Println("Canceled. Run 'stripe whoami' to see your authorized contexts or 'stripe login --new-session' to log in as a different user.")
			return nil
		case errors.Is(err, errReauthTimeout):
			fmt.Println("Still waiting on the re-authorization. Run 'stripe whoami' once you've finished.")
			return nil
		default:
			return err
		}
	}

	if onComplete != nil {
		onComplete(after)
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
