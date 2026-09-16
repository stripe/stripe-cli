package login

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/stripe/stripe-cli/pkg/ansi"
	"github.com/stripe/stripe-cli/pkg/config"
	"github.com/stripe/stripe-cli/pkg/errorcategory"
)

const (
	// DefaultAccessBaseURL is the default (production) base URL for access-srv.
	DefaultAccessBaseURL = "https://access.stripe.com"
	// QAAccessBaseURL is the QA base URL for access-srv, used via --access-base.
	QAAccessBaseURL = "https://qa-access.stripe.com"
	accessAPNPath   = "/stripecli/oauth2"

	// StripeCLIClientIDProd is the registered OAuth client ID for production.
	StripeCLIClientIDProd = "oacli_V18aOD6v0hs9CU"
	// StripeCLIClientIDQA is the registered OAuth client ID for the QA environment.
	StripeCLIClientIDQA = "oacli_UjA5npk5UKXd9u"
)

// OAuthError is a structured OAuth error returned by the access-srv token or
// device-authorization endpoint.
type OAuthError struct {
	Code        string
	Description string
	HTTPStatus  int
}

func (e *OAuthError) Error() string {
	if e.Description != "" {
		return fmt.Sprintf("%s: %s", e.Code, e.Description)
	}
	return e.Code
}

// DeviceAuthResponse holds the device authorization endpoint response.
type DeviceAuthResponse struct {
	DeviceCode      string `json:"device_code"`
	UserCode        string `json:"user_code"`
	VerificationURI string `json:"verification_uri"`
	ExpiresIn       int    `json:"expires_in"`
	Interval        int    `json:"interval"`
}

// OAuthTokenResponse holds the token endpoint response.
type OAuthTokenResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	TokenType    string `json:"token_type"`
	ExpiresIn    int    `json:"expires_in"`
	Scope        string `json:"scope"`
}

type tokenErrorResponse struct {
	Error            string `json:"error"`
	ErrorDescription string `json:"error_description"`
}

// RequestDeviceCode calls the device authorization endpoint and returns the response.
func RequestDeviceCode(ctx context.Context, accessBaseURL, clientID string) (*DeviceAuthResponse, error) {
	data := url.Values{}
	data.Set("client_id", clientID)
	data.Set("scope", "stripecli")

	resp, err := doPostForm(ctx, accessBaseURL+accessAPNPath+"/device/authorization", data)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		return nil, errorcategory.Errorf(errorcategory.Auth, "device authorization request failed (status %d): %s", resp.StatusCode, string(body))
	}

	var authResp DeviceAuthResponse
	if err := json.Unmarshal(body, &authResp); err != nil {
		return nil, fmt.Errorf("failed to parse device authorization response: %w", err)
	}
	return &authResp, nil
}

// PollDeviceToken polls the token endpoint until the user approves, ctx is
// canceled or times out, or a terminal error is returned.
//
// Callers should create ctx with a deadline matching DeviceAuthResponse.ExpiresIn
// to automatically stop polling when the device code expires.
func PollDeviceToken(ctx context.Context, accessBaseURL, clientID, deviceCode string, interval time.Duration) (*OAuthTokenResponse, error) {
	for {
		token, err := pollDeviceTokenOnce(ctx, accessBaseURL, clientID, deviceCode)
		if err == nil {
			return token, nil
		}
		var oauthErr *OAuthError
		if !errors.As(err, &oauthErr) {
			return nil, err
		}
		wait := interval
		switch oauthErr.Code {
		case "authorization_pending":
		case "slow_down":
			wait = interval * 2
		default:
			return nil, err
		}
		t := time.NewTimer(wait)
		select {
		case <-ctx.Done():
			t.Stop()
			return nil, ctx.Err()
		case <-t.C:
		}
	}
}

// pollDeviceTokenOnce does not wait on authorization_pending. Both the blocking
// CLI and the resumable helper share the same OAuth request/response handling.
func pollDeviceTokenOnce(ctx context.Context, accessBaseURL, clientID, deviceCode string) (*OAuthTokenResponse, error) {
	data := url.Values{}
	data.Set("grant_type", "urn:ietf:params:oauth:grant-type:device_code")
	data.Set("client_id", clientID)
	data.Set("device_code", deviceCode)
	resp, err := doPostForm(ctx, accessBaseURL+accessAPNPath+"/token", data)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode == http.StatusOK {
		var token OAuthTokenResponse
		if err := json.Unmarshal(body, &token); err != nil {
			return nil, fmt.Errorf("failed to parse token response: %w", err)
		}
		return &token, nil
	}
	var response tokenErrorResponse
	if err := json.Unmarshal(body, &response); err != nil || response.Error == "" {
		return nil, errorcategory.Errorf(errorcategory.Auth, "token request failed (status %d)", resp.StatusCode)
	}
	return nil, &OAuthError{Code: response.Error, Description: response.ErrorDescription, HTTPStatus: resp.StatusCode}
}

// clientIDForAccessBaseURL returns the OAuth client ID registered for the given
// access base URL. QA and production each have a distinct client ID.
func clientIDForAccessBaseURL(accessBaseURL string) string {
	if accessBaseURL == QAAccessBaseURL {
		return StripeCLIClientIDQA
	}
	return StripeCLIClientIDProd
}

// RequestDeviceCodeForAccessBase requests a device code using the client ID registered for
// accessBaseURL, returning that client ID alongside the response so callers can later poll for
// the token with PollAndSaveDeviceCredentials.
func RequestDeviceCodeForAccessBase(ctx context.Context, accessBaseURL string) (authResp *DeviceAuthResponse, clientID string, err error) {
	clientID = clientIDForAccessBaseURL(accessBaseURL)
	authResp, err = RequestDeviceCode(ctx, accessBaseURL, clientID)
	return authResp, clientID, err
}

// DeviceCodeLoginResult holds the accounts and the active account/mode saved by a completed
// OAuth device-code login.
type DeviceCodeLoginResult struct {
	Accounts          []config.AuthorizedAccount
	ActiveAccountID   string
	ActiveDisplayName string
	ActiveLivemode    bool
}

// PollAndSaveDeviceCredentials polls the token endpoint until the user approves, ctx is
// canceled, or ctx's deadline is exceeded, then saves the resulting OAuth credentials and
// populates cfg's profile with the active account. Unlike LoginWithDeviceCode, it does not print
// progress to stdout, so callers with their own UX (e.g. the RPC service) can drive completion
// themselves.
func PollAndSaveDeviceCredentials(ctx context.Context, accessBaseURL, clientID, deviceCode string, interval time.Duration, cfg *config.Config) (*DeviceCodeLoginResult, error) {
	// Fence the legacy completion against a login/logout that occurs while
	// this process waits for browser approval. Never hold a lock during that wait.
	lockCtx, cancelLock := context.WithTimeout(ctx, handoffOperationTimeout)
	unlock, err := lockOAuthHandoff(lockCtx)
	cancelLock()
	if err != nil {
		return nil, err
	}
	snapshot, _, snapshotErr := handoffSnapshot(cfg)
	original, readErr := readOptionalPendingDeviceAuth()
	unlock()
	if snapshotErr != nil {
		return nil, snapshotErr
	}
	if readErr != nil {
		return nil, readErr
	}
	tokenResp, err := PollDeviceToken(ctx, accessBaseURL, clientID, deviceCode, interval)
	if err != nil {
		return nil, err
	}

	// The token has been issued, so from here on use a context detached from ctx's
	// cancellation/deadline: a caller-side timeout (or the natural device-code expiry) firing at
	// this exact moment shouldn't leave a valid token saved but the account list and active
	// context unpopulated.
	ctx = context.WithoutCancel(ctx)
	ctx, cancel := context.WithTimeout(ctx, handoffOperationTimeout)
	defer cancel()
	unlock, err = lockOAuthHandoff(ctx)
	if err != nil {
		return nil, err
	}
	defer unlock()
	current, _, err := handoffSnapshot(cfg)
	if err != nil {
		return nil, err
	}
	pending, err := readOptionalPendingDeviceAuth()
	if err != nil {
		return nil, err
	}
	sameAttempt := original == nil && pending == nil || original != nil && pending != nil &&
		original.ID == pending.ID && original.DeviceCode == pending.DeviceCode
	if current != snapshot || !sameAttempt {
		return nil, &HandoffError{Reason: "legacy_completion_superseded"}
	}
	if err := forgetPendingLoginLocked(); err != nil {
		return nil, err
	}

	// Clear all stale credentials before saving new ones, so this succeeds even if a
	// previously stored credential is expired or revoked.
	_ = cfg.RemoveAuthFields(cfg.Profile.ProfileName)

	if err := saveOAuthCredentials(cfg, tokenResp); err != nil {
		return nil, fmt.Errorf("failed to save credentials: %w", err)
	}

	accounts, err := ListAuthorizedAccounts(ctx, accessBaseURL, tokenResp.AccessToken)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch account info: %w", err)
	}
	activeID, activeLivemode := pickActiveContext(accounts)
	if err := populateProfileFromAccounts(cfg, accounts, activeID, activeLivemode); err != nil {
		return nil, fmt.Errorf("failed to save account info: %w", err)
	}

	return &DeviceCodeLoginResult{
		Accounts:          accounts,
		ActiveAccountID:   activeID,
		ActiveDisplayName: cfg.Profile.DisplayName,
		ActiveLivemode:    activeLivemode,
	}, nil
}

// LoginWithDeviceCode runs the full OAuth 2.1 device-code flow and saves credentials.
func LoginWithDeviceCode(ctx context.Context, accessBaseURL string, cfg *config.Config) error {
	handoff, err := BeginOrResumeLogin(ctx, accessBaseURL, cfg)
	if err != nil {
		return err
	}
	if handoff.State == LoginHandoffAuthenticated {
		return printCompletedHandoff(handoff)
	}
	if handoff.State != LoginHandoffPending && handoff.State != LoginHandoffCompleting {
		return handoffStateError(handoff)
	}
	if handoff.State == LoginHandoffPending {
		fmt.Printf("To authorize, visit %s\n\n", handoff.BrowserURL)
		fmt.Println("When prompted, enter your verification code:")
		fmt.Println(ansi.Purple(handoff.VerificationCode))
		fmt.Println("This login survives an interrupted wait. Re-run 'stripe login' to resume it.")
		if !isSSH() && canOpenBrowser() {
			fmt.Println("Press enter to open the browser (^C to stop waiting)")
			go func() { fmt.Scanln(); _ = openBrowser(handoff.BrowserURL) }() //nolint:errcheck
		}
	}
	return waitForLoginHandoff(ctx, accessBaseURL, cfg, handoff.ID)
}

func waitForLoginHandoff(ctx context.Context, accessBaseURL string, cfg *config.Config, id string) error {
	waitCtx, stop := signal.NotifyContext(ctx, os.Interrupt, syscall.SIGTERM)
	defer stop()
	for {
		result, err := CheckLogin(waitCtx, accessBaseURL, cfg, id)
		if waitCtx.Err() != nil {
			fmt.Println("Stopped waiting. Complete the original browser link, then run 'stripe login --complete-device' to resume.")
			return nil
		}
		if err != nil {
			return err
		}
		switch result.State {
		case LoginHandoffAuthenticated:
			return printCompletedHandoff(result)
		case LoginHandoffPending, LoginHandoffCompleting:
		default:
			return handoffStateError(result)
		}
		timer := time.NewTimer(time.Duration(max(result.CheckAfterSeconds, 1)) * time.Second)
		select {
		case <-waitCtx.Done():
			timer.Stop()
		case <-timer.C:
		}
	}
}

func printCompletedHandoff(result *LoginHandoff) error {
	fmt.Printf("Done! Authenticated for %s (%s).\n", result.AccountID, displayMode(map[bool]string{true: "live", false: "test"}[result.Livemode]))
	warnIfInsecureStorage()
	return nil
}

type contextRow struct {
	name   string
	mode   string
	id     string
	active bool
}

func buildContextRows(accounts []config.AuthorizedAccount, activeID string, activeLivemode bool) []contextRow {
	activeMode := "test"
	if activeLivemode {
		activeMode = "live"
	}
	var rows []contextRow
	for _, a := range accounts {
		modes := a.Modes
		if len(modes) == 0 {
			modes = []string{"test"}
		}
		for _, m := range modes {
			rows = append(rows, contextRow{
				name:   a.Name,
				mode:   m,
				id:     a.ID,
				active: a.ID == activeID && m == activeMode,
			})
		}
	}
	return rows
}

// printAuthorizedSummary prints the "Done! The Stripe CLI is authorized for
// ..." banner and context table shared by login and reauth.
func printAuthorizedSummary(accounts []config.AuthorizedAccount, activeID string, activeLivemode bool) {
	color := ansi.Color(os.Stdout)
	rows := buildContextRows(accounts, activeID, activeLivemode)

	if len(rows) == 0 {
		fmt.Printf("%s Done! The Stripe CLI is configured with your credentials.\n", color.Green("✓"))
		return
	}

	if len(rows) == 1 {
		r := rows[0]
		ctx := fmt.Sprintf("%s · %s", r.name, displayMode(r.mode))
		fmt.Printf("%s Done! The Stripe CLI is authorized for %s (%s)\n\n", color.Green("✓"), ctx, r.id)
		fmt.Println("Run 'stripe login' to change permissions or authorize access to additional accounts or sandboxes.")
		return
	}

	fmt.Printf("%s Done! The Stripe CLI is authorized for:\n\n", color.Green("✓"))

	nameW, modeW, idW := 0, 0, 0
	for _, r := range rows {
		if len(r.name) > nameW {
			nameW = len(r.name)
		}
		if dl := len(displayMode(r.mode)); dl > modeW {
			modeW = dl
		}
		if len(r.id) > idW {
			idW = len(r.id)
		}
	}

	var active contextRow
	for _, r := range rows {
		mode := displayMode(r.mode)
		if r.active {
			active = r
			fmt.Printf("  %-*s  %-*s  %-*s  %s active\n", nameW, r.name, modeW, mode, idW, r.id, color.Green("●"))
		} else {
			fmt.Printf("  %-*s  %-*s  %s\n", nameW, r.name, modeW, mode, r.id)
		}
	}

	fmt.Println()
	fmt.Printf("Currently active: %s · %s (%s)\n\n", active.name, displayMode(active.mode), active.id)
	fmt.Println("Run 'stripe switch' to switch to a different account, or between live mode and a sandbox.")
	fmt.Println("Run 'stripe login' to change permissions or authorize access to additional accounts or sandboxes.")
}

// RefreshAccessToken exchanges a refresh token for a new access token.
// On success, callers must persist the returned OAuthTokenResponse.RefreshToken
// (replacing the previously stored value) before discarding the old token.
// If the response does not include a refresh token, the caller must require a
// new interactive login when the access token next expires.
//
// On invalid_grant, callers should clear stored credentials and start a new
// device authorization flow.
func RefreshAccessToken(ctx context.Context, accessBaseURL, clientID, refreshToken string) (*OAuthTokenResponse, error) {
	data := url.Values{}
	data.Set("grant_type", "refresh_token")
	data.Set("refresh_token", refreshToken)
	data.Set("client_id", clientID)

	resp, err := doPostForm(ctx, accessBaseURL+accessAPNPath+"/token", data)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode == http.StatusOK {
		var tokenResp OAuthTokenResponse
		if err := json.Unmarshal(body, &tokenResp); err != nil {
			return nil, fmt.Errorf("failed to parse refresh token response: %w", err)
		}
		return &tokenResp, nil
	}

	var errResp tokenErrorResponse
	if jsonErr := json.Unmarshal(body, &errResp); jsonErr == nil && errResp.Error != "" {
		return nil, &OAuthError{Code: errResp.Error, Description: errResp.ErrorDescription, HTTPStatus: resp.StatusCode}
	}

	return nil, errorcategory.Errorf(errorcategory.Auth, "refresh token request failed (status %d): %s", resp.StatusCode, string(body))
}

func doPostForm(ctx context.Context, endpoint string, data url.Values) (*http.Response, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, strings.NewReader(data.Encode()))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	return accessSrvHTTPClient.Do(req)
}
