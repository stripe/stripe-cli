package login

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
)

const reauthPath = accessAPNPath + "/token/reauth"

type reauthResponse struct {
	ReauthURL string `json:"reauth_url"`
}

// Reauth fetches a reauthentication URL for the active OAuth session and
// directs the user to it. If a browser is available it is opened automatically;
// otherwise the URL is printed for the user to visit manually.
func Reauth(ctx context.Context, accessBaseURL, accessToken string) error {
	reauthURL, err := fetchReauthURL(ctx, accessBaseURL, accessToken)
	if err != nil {
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
	return nil
}

func fetchReauthURL(ctx context.Context, accessBaseURL, accessToken string) (string, error) {
	endpoint := accessBaseURL + reauthPath
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("Authorization", "Bearer "+accessToken)

	resp, err := http.DefaultClient.Do(req)
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
			return "", fmt.Errorf("reauth response missing reauth_url")
		}
		return r.ReauthURL, nil
	case http.StatusUnauthorized:
		return "", fmt.Errorf("session expired or revoked; run 'stripe login' to authenticate again")
	case http.StatusBadRequest:
		return "", fmt.Errorf("invalid reauth request (invalid_request); check your configuration")
	default:
		return "", fmt.Errorf("reauth request failed (status %d): %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}
}
