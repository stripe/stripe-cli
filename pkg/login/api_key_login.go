package login

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/stripe/stripe-cli/pkg/config"
	"github.com/stripe/stripe-cli/pkg/validators"
)

// LoginWithAPIKey configures the CLI using a user-provided API key.
//
// This path intentionally avoids the browser/pairing-code flow so that it can
// be used in headless environments (e.g., Docker/CI).
func LoginWithAPIKey(ctx context.Context, apiBaseURL string, cfg *config.Config, apiKey string) error {
	apiKey = strings.TrimSpace(apiKey)
	if err := validators.APIKey(apiKey); err != nil {
		return err
	}

	// Ensure we have a device name even if InitConfig hasn't run (e.g., in tests).
	if strings.TrimSpace(cfg.Profile.DeviceName) == "" {
		hostName, err := os.Hostname()
		if err != nil {
			hostName = "unknown"
		}
		cfg.Profile.DeviceName = hostName
	}

	// Treat the provided key as the configured test mode key, mirroring the
	// interactive login flow.
	cfg.Profile.TestModeAPIKey = apiKey

	displayName, _ := getDisplayName(ctx, nil, apiBaseURL, apiKey)
	cfg.Profile.DisplayName = displayName

	if err := cfg.Profile.CreateProfile(); err != nil {
		return err
	}

	message, err := SuccessMessage(ctx, nil, apiBaseURL, apiKey)
	if err != nil {
		fmt.Printf("> Error verifying the CLI was setup successfully: %s\n",
			err,
		)
		return nil
	}

	fmt.Printf("> %s\n", message)
	return nil
}
