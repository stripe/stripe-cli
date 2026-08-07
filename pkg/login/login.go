package login

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

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

// InitiateLogin calls GetLinks, prints JSON with browser_url, verification_code,
// and a next_step command to complete login, then exits. Intended for non-interactive
// (agent/script) use.
func InitiateLogin(ctx context.Context, baseURL string, cfg *config.Config) error {
	deviceName, err := cfg.Profile.GetDeviceName()
	if err != nil {
		return err
	}

	links, useOAuth, err := GetLinks(ctx, baseURL, deviceName, cfg.GetMachineUUID())
	if err != nil {
		return err
	}

	if useOAuth {
		return fmt.Errorf("OAuth login required for this account; run 'stripe login' (without --non-interactive) to complete browser authorization")
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

// PollForLogin polls the given poll URL until browser auth completes, then saves
// credentials. Intended as the second step of a non-interactive login flow.
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
