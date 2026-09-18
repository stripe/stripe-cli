package login

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/signal"
	"time"

	"github.com/spf13/afero"

	"github.com/stripe/stripe-cli/pkg/ansi"
	"github.com/stripe/stripe-cli/pkg/config"
	"github.com/stripe/stripe-cli/pkg/errorcategory"
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

// InitiateLogin prints JSON with browser_url, verification_code, and a
// next_step command, then returns. Intended for non-interactive (agent/script)
// use. For the OAuth device-code flow it saves pending state to disk and emits
// `stripe login --complete-device` as the next_step.
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

// OAuthLoginSession describes a non-interactive OAuth device-code login that's ready for the
// user to complete out-of-band, whether just minted or resumed from a still-valid pending one.
type OAuthLoginSession struct {
	BrowserURL       string
	VerificationCode string
	ExpiresIn        int // seconds remaining until the device code expires
}

// InitiateOAuthLogin starts (or resumes) a non-interactive OAuth device-code login for
// accessBaseURL, without printing anything. Unlike `stripe login --non-interactive` (see
// initiateOAuthDeviceLogin), which always mints a fresh device code, this resumes a
// still-valid pending one instead of minting a new one - see initiateOrResumeOAuthDeviceLogin.
// This is the resume behavior exposed to plugins via the OAuthInitiateLogin RPC.
func InitiateOAuthLogin(ctx context.Context, accessBaseURL string) (*OAuthLoginSession, error) {
	return initiateOrResumeOAuthDeviceLogin(ctx, accessBaseURL)
}

// FindPendingOAuthLogin looks for an OAuth device-code login already in progress for
// accessBaseURL (started by this process or another one, e.g. `stripe login
// --non-interactive`), without starting a new one. Returns nil, nil if there is no pending
// login attempt, it's for a different accessBaseURL, or it has expired.
func FindPendingOAuthLogin(accessBaseURL string) (*OAuthLoginSession, error) {
	return findPendingOAuthLoginSession(accessBaseURL), nil
}

func findPendingOAuthLoginSession(accessBaseURL string) *OAuthLoginSession {
	cont, err := loadPendingDeviceAuth()
	if err != nil || cont.AccessBaseURL != accessBaseURL {
		return nil
	}
	remaining := time.Until(cont.deadline())
	if remaining <= 0 {
		return nil
	}
	return &OAuthLoginSession{
		BrowserURL:       cont.VerificationURI,
		VerificationCode: cont.UserCode,
		ExpiresIn:        int(remaining.Seconds()),
	}
}

// initiateOrResumeOAuthDeviceLogin returns the still-valid pending device code for
// accessBaseURL if one exists, instead of minting a new one - so repeated calls to the
// OAuthInitiateLogin RPC (e.g. a retrying plugin) converge on one browser_url/
// verification_code rather than orphaning the previous one every retry. `stripe login
// --non-interactive` does not use this - see mintOAuthDeviceLogin.
func initiateOrResumeOAuthDeviceLogin(ctx context.Context, accessBaseURL string) (*OAuthLoginSession, error) {
	if session := findPendingOAuthLoginSession(accessBaseURL); session != nil {
		return session, nil
	}
	return mintOAuthDeviceLogin(ctx, accessBaseURL)
}

// mintOAuthDeviceLogin always requests a fresh device code from accessBaseURL and saves it as
// the new pending state, overwriting any still-valid pending login that may already exist.
func mintOAuthDeviceLogin(ctx context.Context, accessBaseURL string) (*OAuthLoginSession, error) {
	clientID := clientIDForAccessBaseURL(accessBaseURL)
	authResp, err := RequestDeviceCode(ctx, accessBaseURL, clientID)
	if err != nil {
		return nil, fmt.Errorf("failed to request device code: %w", err)
	}
	if err := validateBrowserURL(authResp.VerificationURI, accessBaseURL); err != nil {
		return nil, err
	}

	cont := &oauthContinuation{
		DeviceCode:      authResp.DeviceCode,
		Interval:        authResp.Interval,
		ExpiresIn:       authResp.ExpiresIn,
		AccessBaseURL:   accessBaseURL,
		VerificationURI: authResp.VerificationURI,
		UserCode:        authResp.UserCode,
		IssuedAt:        time.Now(),
	}
	if err := savePendingDeviceAuth(cont); err != nil {
		return nil, fmt.Errorf("failed to save pending auth state: %w", err)
	}

	return &OAuthLoginSession{
		BrowserURL:       authResp.VerificationURI,
		VerificationCode: authResp.UserCode,
		ExpiresIn:        authResp.ExpiresIn,
	}, nil
}

// initiateOAuthDeviceLogin always mints a fresh device code (see mintOAuthDeviceLogin - unlike
// the OAuthInitiateLogin RPC, this does not resume a still-valid pending login), saves it as
// the pending state, and prints the JSON session output.
func initiateOAuthDeviceLogin(ctx context.Context, accessBaseURL string) error {
	session, err := mintOAuthDeviceLogin(ctx, accessBaseURL)
	if err != nil {
		return err
	}

	out := loginSessionOutput{
		BrowserURL:       session.BrowserURL,
		VerificationCode: session.VerificationCode,
		NextStep:         "stripe login --complete-device",
	}
	b, err := json.MarshalIndent(out, "", "  ")
	if err != nil {
		return err
	}
	fmt.Println(string(b))
	return nil
}

// PollForLogin polls the given legacy poll URL until browser auth completes,
// then saves credentials. Intended as the second step of a non-interactive
// legacy login flow. For OAuth, use PollPendingDeviceAuth instead.
func PollForLogin(ctx context.Context, pollURL string, cfg *config.Config) error {
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

// loadValidatedPendingDeviceAuth loads the pending device auth state and checks that its
// AccessBaseURL is one ValidateAccessBaseURL accepts, since requests built from it carry OAuth
// bearer/refresh tokens.
func loadValidatedPendingDeviceAuth() (*oauthContinuation, error) {
	cont, err := loadPendingDeviceAuth()
	if err != nil {
		return nil, err
	}
	if err := ValidateAccessBaseURL(cont.AccessBaseURL); err != nil {
		return nil, err
	}
	return cont, nil
}

// PollPendingOAuthLogin waits for the login started by InitiateOAuthLogin (or InitiateLogin's
// OAuth path) to complete. Returns (nil, nil) if ctx is done before the user completes
// authentication and the device code's own expiry hasn't been reached yet - callers should
// call this again to keep waiting. Returns an error and clears the pending state if the
// device code has actually expired or the server returned a terminal OAuth error (e.g.
// access_denied); on success, also clears the pending state.
func PollPendingOAuthLogin(ctx context.Context, cfg *config.Config) (*DeviceCodeLoginResult, error) {
	cont, err := loadValidatedPendingDeviceAuth()
	if err != nil {
		return nil, err
	}

	clientID := clientIDForAccessBaseURL(cont.AccessBaseURL)
	interval := max(time.Duration(cont.Interval)*time.Second, 5*time.Second)
	deadline := cont.deadline()

	pollCtx, cancel := context.WithDeadline(ctx, deadline)
	defer cancel()

	result, err := PollAndSaveDeviceCredentials(pollCtx, cont.AccessBaseURL, clientID, cont.DeviceCode, interval, cfg)
	if err != nil {
		if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
			if time.Now().After(deadline) {
				clearPendingDeviceAuth()
				return nil, errorcategory.Errorf(errorcategory.Auth, "device code expired; please run 'stripe login --non-interactive' again")
			}
			// The caller's own ctx stopped waiting this time (e.g. ^C or its own timeout);
			// the device code is still good, so leave the pending state for a later retry.
			return nil, nil
		}
		// Terminal OAuth error (access_denied, expired_token, ...): the device code is dead.
		clearPendingDeviceAuth()
		return nil, err
	}

	clearPendingDeviceAuth()
	return result, nil
}

// CheckPendingOAuthLogin makes a single, non-blocking check on the login started by
// InitiateOAuthLogin (or InitiateLogin's OAuth path). Unlike PollPendingOAuthLogin, it never
// waits for the user - it makes one request and returns immediately, so callers that want to
// wait (e.g. a plugin driving its own retry loop) should call this repeatedly on their own
// schedule instead. Returns (nil, nil) if the device code is still valid but the user hasn't
// completed authentication yet. Returns an error and clears the pending state if the device
// code has actually expired or the server returned a terminal OAuth error (e.g.
// access_denied); on success, also clears the pending state.
func CheckPendingOAuthLogin(ctx context.Context, cfg *config.Config) (*DeviceCodeLoginResult, error) {
	cont, err := loadValidatedPendingDeviceAuth()
	if err != nil {
		return nil, err
	}

	if time.Now().After(cont.deadline()) {
		clearPendingDeviceAuth()
		return nil, errorcategory.Errorf(errorcategory.Auth, "device code expired; please run 'stripe login --non-interactive' again")
	}

	clientID := clientIDForAccessBaseURL(cont.AccessBaseURL)
	tokenResp, err := CheckDeviceToken(ctx, cont.AccessBaseURL, clientID, cont.DeviceCode)
	if err != nil {
		var oauthErr *OAuthError
		if errors.As(err, &oauthErr) && (oauthErr.Code == "authorization_pending" || oauthErr.Code == "slow_down") {
			// The user hasn't completed authentication yet; the device code is still good.
			return nil, nil
		}
		// Terminal OAuth error (access_denied, expired_token, ...): the device code is dead.
		clearPendingDeviceAuth()
		return nil, err
	}

	result, err := saveDeviceCredentials(context.WithoutCancel(ctx), cont.AccessBaseURL, tokenResp, cfg)
	if err != nil {
		return nil, err
	}
	clearPendingDeviceAuth()
	return result, nil
}

// PollPendingDeviceAuth loads the OAuth device auth state saved by InitiateLogin and polls
// the token endpoint until the user approves.
func PollPendingDeviceAuth(ctx context.Context, cfg *config.Config) error {
	waitCtx, stop := signal.NotifyContext(ctx, os.Interrupt)
	defer stop()

	s := ansi.StartNewSpinner("Waiting for confirmation...", os.Stdout)
	result, err := PollPendingOAuthLogin(waitCtx, cfg)
	ansi.StopSpinner(s, "", os.Stdout)
	if err != nil {
		return err
	}
	if result == nil {
		ansi.ClearLine(os.Stdout)
		fmt.Println("Canceled. Run 'stripe login --non-interactive' again to try again.")
		return nil
	}

	printAuthorizedSummary(result.Accounts, result.ActiveAccountID, result.ActiveLivemode)
	warnIfInsecureStorage()
	return nil
}
