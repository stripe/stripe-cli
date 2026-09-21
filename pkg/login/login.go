package login

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/signal"
	"time"

	"github.com/kballard/go-shellquote"
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
	if cont, err := readOptionalPendingDeviceAuth(); err != nil {
		return err
	} else if cont != nil && cont.Version == 1 {
		return LoginWithDeviceCode(ctx, accessBaseURL, cfg)
	}
	links, useOAuth, err := GetLinks(ctx, dashboardBaseURL, cfg.Profile.DeviceName, cfg.GetMachineUUID())
	if err != nil {
		return err
	}

	if useOAuth {
		return LoginWithDeviceCode(ctx, accessBaseURL, cfg)
	}
	// The legacy flow retains its existing replacement behavior.
	_ = cfg.RemoveAuthFields(cfg.Profile.ProfileName)

	configurer := keys.NewRAKConfigurer(cfg, afero.NewOsFs())
	rt := keys.NewRAKTransfer(configurer)
	auth := NewAuthenticator(rt)
	return auth.Login(ctx, links)
}

type loginSessionOutput struct {
	BrowserURL       string     `json:"browser_url"`
	VerificationCode string     `json:"verification_code"`
	NextStep         string     `json:"next_step"`
	HandoffID        string     `json:"handoff_id,omitempty"`
	ExpiresAt        *time.Time `json:"expires_at,omitempty"`
	Reused           bool       `json:"reused"`
}

// InitiateLogin prints JSON with browser_url, verification_code, and a
// next_step command, then returns. Intended for non-interactive (agent/script)
// use. For the OAuth device-code flow it saves pending state to disk and emits
// `stripe login --complete-device` as the next_step.
func InitiateLogin(ctx context.Context, baseURL, accessBaseURL string, cfg *config.Config) error {
	if cont, err := readOptionalPendingDeviceAuth(); err != nil {
		return err
	} else if cont != nil {
		return initiateOAuthDeviceLogin(ctx, accessBaseURL, cfg)
	}
	deviceName, err := cfg.Profile.GetDeviceName()
	if err != nil {
		return err
	}

	links, useOAuth, err := GetLinks(ctx, baseURL, deviceName, cfg.GetMachineUUID())
	if err != nil {
		return err
	}

	if useOAuth {
		return initiateOAuthDeviceLogin(ctx, accessBaseURL, cfg)
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

// initiateOAuthDeviceLogin calls the device authorization endpoint, saves the
// pending state to disk, and prints the JSON session output.
func initiateOAuthDeviceLogin(ctx context.Context, accessBaseURL string, cfg *config.Config) error {
	result, err := BeginOrResumeLogin(ctx, accessBaseURL, cfg)
	if err != nil {
		return err
	}
	if result.State != LoginHandoffPending && result.State != LoginHandoffCompleting {
		return handoffStateError(result)
	}
	out := loginSessionOutput{
		BrowserURL: result.BrowserURL, VerificationCode: result.VerificationCode,
		NextStep: pendingLoginCommand(cfg), HandoffID: result.ID,
		ExpiresAt: &result.ExpiresAt, Reused: result.Reused,
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

// PollPendingDeviceAuth loads the OAuth device auth state saved by
// InitiateLogin and polls the token endpoint until the user approves.
func PollPendingDeviceAuth(ctx context.Context, cfg *config.Config) error {
	cont, err := loadPendingDeviceAuth()
	if err != nil {
		return err
	}
	if cont.Version == 1 {
		return waitForLoginHandoff(ctx, cont.AccessBaseURL, cfg, cont.ID)
	}
	if cont.Version != 0 {
		return &HandoffError{Reason: "unsupported_continuation"}
	}

	if err := ValidateAccessBaseURL(cont.AccessBaseURL); err != nil {
		return err
	}

	clientID := clientIDForAccessBaseURL(cont.AccessBaseURL)
	interval := max(time.Duration(cont.Interval)*time.Second, 5*time.Second)
	expiresIn := max(time.Duration(cont.ExpiresIn)*time.Second, 10*time.Minute)

	pollCtx, cancel := context.WithTimeout(ctx, expiresIn)
	defer cancel()
	waitCtx, stop := signal.NotifyContext(pollCtx, os.Interrupt)
	defer stop()

	s := ansi.StartNewSpinner("Waiting for confirmation...", os.Stdout)
	result, err := PollAndSaveDeviceCredentials(waitCtx, cont.AccessBaseURL, clientID, cont.DeviceCode, interval, cfg)
	ansi.StopSpinner(s, "", os.Stdout)
	if err != nil {
		switch {
		case errors.Is(err, context.Canceled):
			ansi.ClearLine(os.Stdout)
			fmt.Println("Canceled. Run 'stripe login --non-interactive' again to try again.")
			return nil
		case pollCtx.Err() != nil:
			return errorcategory.Errorf(errorcategory.Auth, "device code expired; please run 'stripe login --non-interactive' again")
		default:
			return err
		}
	}

	printAuthorizedSummary(result.Accounts, result.ActiveAccountID, result.ActiveLivemode)
	warnIfInsecureStorage()
	return nil
}

func pendingLoginCommand(cfg *config.Config) string {
	args := []string{"stripe", "login", "--complete-device"}
	if cfg.Profile.ProfileName != "" && cfg.Profile.ProfileName != "default" {
		args = append(args, "--project-name", cfg.Profile.ProfileName)
	}
	if cfg.ProfilesFile != "" {
		args = append(args, "--config", cfg.ProfilesFile)
	}
	return shellquote.Join(args...)
}
