package login

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/spf13/afero"

	"github.com/stripe/stripe-cli/pkg/ansi"
	"github.com/stripe/stripe-cli/pkg/config"
	"github.com/stripe/stripe-cli/pkg/keyring"
	"github.com/stripe/stripe-cli/pkg/login/keys"
	"github.com/stripe/stripe-cli/pkg/stripe"
)

func warnIfInsecureStorage() {
	if keyring.IsUsingInsecureStorage(config.KeyRing) {
		color := ansi.Color(os.Stdout)
		path := keyring.FallbackStoragePath(config.KeyRing)
		fmt.Println(color.Yellow(fmt.Sprintf("Warning: the system keyring is unavailable. Your credentials have been stored unencrypted in %s", path)))
	}
}

// Login is the main entrypoint for logging in to the CLI.
//
// When the /stripecli/auth server responds with a 3xx, the machine UUID is
// enrolled in the OAuth feature flag and the OAuth device-code flow is used
// instead of the legacy RAK flow. accessBaseURL controls which access-srv
// environment is used (production by default; QA via --access-base).
func Login(ctx context.Context, dashboardBaseURL, accessBaseURL string, cfg *config.Config) error {
	links, useOAuth, err := GetLinks(ctx, dashboardBaseURL, cfg.Profile.DeviceName, cfg.GetMachineUUID())
	if err != nil {
		return err
	}

	// Clear all stale credentials before saving new ones, regardless of flow.
	_ = cfg.RemoveAuthFields(cfg.Profile.ProfileName)

	if useOAuth {
		return LoginWithDeviceCode(ctx, accessBaseURL, cfg)
	}

	configurer := keys.NewRAKConfigurer(cfg, afero.NewOsFs())
	rt := keys.NewRAKTransfer(configurer)
	auth := NewAuthenticator(rt)
	return auth.Login(ctx, links)
}

type loginSessionOutput struct {
	BrowserURL       string `json:"browser_url"`
	VerificationCode string `json:"verification_code"`
	NextStep         string `json:"next_step"`
}

// oauthContinuation holds the data needed to poll for an OAuth device token.
// It is base64-encoded and passed to `stripe login --complete` as `oauth:<token>`.
type oauthContinuation struct {
	DeviceCode    string `json:"device_code"`
	Interval      int    `json:"interval"`
	ExpiresIn     int    `json:"expires_in"`
	AccessBaseURL string `json:"access_base"`
}

// InitiateLogin prints JSON with browser_url, verification_code, and a
// next_step command, then returns. Intended for non-interactive (agent/script)
// use. For the OAuth device-code flow it calls the device authorization
// endpoint and encodes the continuation in the next_step token.
func InitiateLogin(ctx context.Context, baseURL, accessBaseURL string, cfg *config.Config) error {
	deviceName, err := cfg.Profile.GetDeviceName()
	if err != nil {
		return err
	}

	links, useOAuth, err := GetLinks(ctx, baseURL, deviceName, cfg.GetMachineUUID())
	if err != nil {
		return err
	}

	if useOAuth {
		return initiateOAuthDeviceLogin(ctx, accessBaseURL)
	}

	out := loginSessionOutput{
		BrowserURL:       links.BrowserURL,
		VerificationCode: links.VerificationCode,
		NextStep:         fmt.Sprintf("stripe login --complete '%s'", links.PollURL),
	}
	b, err := json.MarshalIndent(out, "", "  ")
	if err != nil {
		return err
	}
	fmt.Println(string(b))
	return nil
}

// initiateOAuthDeviceLogin calls the device authorization endpoint, encodes
// the continuation as a base64 token, and prints the JSON session output.
func initiateOAuthDeviceLogin(ctx context.Context, accessBaseURL string) error {
	clientID := clientIDForAccessBaseURL(accessBaseURL)
	authResp, err := RequestDeviceCode(ctx, accessBaseURL, clientID)
	if err != nil {
		return fmt.Errorf("failed to request device code: %w", err)
	}

	cont := oauthContinuation{
		DeviceCode:    authResp.DeviceCode,
		Interval:      authResp.Interval,
		ExpiresIn:     authResp.ExpiresIn,
		AccessBaseURL: accessBaseURL,
	}
	contJSON, err := json.Marshal(cont)
	if err != nil {
		return err
	}
	contToken := "oauth:" + base64.RawURLEncoding.EncodeToString(contJSON)

	out := loginSessionOutput{
		BrowserURL:       authResp.VerificationURI,
		VerificationCode: authResp.UserCode,
		NextStep:         fmt.Sprintf("stripe login --complete '%s'", contToken),
	}
	b, err := json.MarshalIndent(out, "", "  ")
	if err != nil {
		return err
	}
	fmt.Println(string(b))
	return nil
}

// PollForLogin polls until browser auth completes, then saves credentials.
// Intended as the second step of a non-interactive login flow.
//
// For OAuth, pollURL must be of the form "oauth:<base64>" as produced by
// InitiateLogin; for legacy it is the plain HTTPS poll URL from the dashboard.
func PollForLogin(ctx context.Context, pollURL string, cfg *config.Config) error {
	if encoded, ok := strings.CutPrefix(pollURL, "oauth:"); ok {
		return pollOAuthDeviceLogin(ctx, encoded, cfg)
	}

	response, account, err := keys.PollForKey(ctx, pollURL, 0, 0)
	if err != nil {
		return err
	}

	// Clear all stale credentials before saving new ones.
	_ = cfg.RemoveAuthFields(cfg.Profile.ProfileName)

	configurer := keys.NewRAKConfigurer(cfg, afero.NewOsFs())
	if err := configurer.SaveLoginDetails(response); err != nil {
		return err
	}

	msg, err := SuccessMessage(ctx, account, stripe.DefaultAPIBaseURL, response.TestModeAPIKey)
	if err != nil {
		fmt.Printf("> Error verifying setup: %s\n", err)
		return err
	}
	fmt.Printf("> %s\n", msg)
	fmt.Println(ansi.Italic("Please note: this key will expire after 90 days, at which point you'll need to re-authenticate."))
	warnIfInsecureStorage()
	return nil
}

// pollOAuthDeviceLogin decodes the base64 continuation token and polls the
// OAuth token endpoint until the user approves in the browser.
func pollOAuthDeviceLogin(ctx context.Context, encoded string, cfg *config.Config) error {
	data, err := base64.RawURLEncoding.DecodeString(encoded)
	if err != nil {
		return fmt.Errorf("invalid OAuth continuation token: %w", err)
	}
	var cont oauthContinuation
	if err := json.Unmarshal(data, &cont); err != nil {
		return fmt.Errorf("invalid OAuth continuation token: %w", err)
	}

	clientID := clientIDForAccessBaseURL(cont.AccessBaseURL)
	interval := max(time.Duration(cont.Interval)*time.Second, 5*time.Second)
	expiresIn := max(time.Duration(cont.ExpiresIn)*time.Second, 10*time.Minute)

	pollCtx, cancel := context.WithTimeout(ctx, expiresIn)
	defer cancel()

	tokenResp, err := PollDeviceToken(pollCtx, cont.AccessBaseURL, clientID, cont.DeviceCode, interval)
	if err != nil {
		if pollCtx.Err() != nil {
			return fmt.Errorf("device code expired; please run 'stripe login' again")
		}
		return err
	}

	// Clear all stale credentials before saving new ones.
	_ = cfg.RemoveAuthFields(cfg.Profile.ProfileName)

	if err := saveOAuthCredentials(cfg, tokenResp); err != nil {
		return fmt.Errorf("failed to save credentials: %w", err)
	}

	accounts, err := ListAuthorizedAccounts(ctx, cont.AccessBaseURL, tokenResp.AccessToken)
	if err != nil {
		return fmt.Errorf("failed to fetch account info: %w", err)
	}
	activeID, activeLivemode := pickActiveContext(accounts)
	if err := populateProfileFromAccounts(cfg, accounts, activeID, activeLivemode); err != nil {
		return fmt.Errorf("failed to save account info: %w", err)
	}

	printLoginSuccess(accounts, activeID, activeLivemode)
	warnIfInsecureStorage()
	return nil
}
