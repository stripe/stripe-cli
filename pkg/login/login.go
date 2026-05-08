package login

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/spf13/afero"

	"github.com/stripe/stripe-cli/pkg/ansi"
	"github.com/stripe/stripe-cli/pkg/config"
	"github.com/stripe/stripe-cli/pkg/login/keys"
	"github.com/stripe/stripe-cli/pkg/stripe"
)

// Login is the main entrypoint for logging in to the CLI.
func Login(ctx context.Context, baseURL string, config *config.Config) error {
	links, err := GetLinks(ctx, baseURL, config.Profile.DeviceName)
	if err != nil {
		return err
	}

	configurer := keys.NewRAKConfigurer(config, afero.NewOsFs())
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

	links, err := GetLinks(ctx, baseURL, deviceName)
	if err != nil {
		return err
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
	return nil
}
