// Package logout handles clearing stored Stripe credentials.
package logout

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/stripe/stripe-cli/pkg/ansi"
	"github.com/stripe/stripe-cli/pkg/config"
	"github.com/stripe/stripe-cli/pkg/login"
)

// Logout clears credentials for the current profile. For OAuth sessions it
// also revokes the token before clearing.
func Logout(ctx context.Context, accessBaseURL string, cfg *config.Config) error {
	liveKey, _ := cfg.Profile.GetAPIKey(true)
	testKey, _ := cfg.Profile.GetAPIKey(false)
	uat, _ := cfg.Profile.GetUAT()

	if liveKey == "" && testKey == "" && uat == "" && !hasStoredOAuthData() {
		fmt.Println("You are already logged out.")
		return nil
	}

	if strings.HasPrefix(uat, "oak_") {
		if err := login.RevokeToken(ctx, accessBaseURL); err != nil {
			// Log but don't block — credentials should still be cleared.
			fmt.Fprintf(os.Stderr, "Warning: token revocation failed: %s\n", err)
		}
		if err := cfg.RemoveAuthFields(cfg.Profile.ProfileName); err != nil {
			return err
		}
		color := ansi.Color(os.Stdout)
		fmt.Printf("%s Logged out of all contexts and revoked session.\n", color.Green("✓"))
		return nil
	}

	fmt.Println("Logging out...")
	if err := cfg.RemoveAuthFields(cfg.Profile.ProfileName); err != nil {
		return err
	}
	profileName := cfg.Profile.ProfileName
	if profileName == "default" {
		fmt.Println("Credentials have been cleared for the default project.")
	} else {
		fmt.Printf("Credentials have been cleared for %s.\n", profileName)
	}
	return nil
}

func hasStoredOAuthData() bool {
	if config.KeyRing == nil {
		return false
	}
	for _, key := range []string{config.OAuthRefreshTokenKeychainKey, config.OAuthActiveContextKeychainKey} {
		if _, err := config.KeyRing.Get(key); err == nil {
			return true
		}
	}
	return false
}

// All clears credentials for all profiles.
func All(ctx context.Context, accessBaseURL string, cfg *config.Config) error {
	uat, _ := cfg.Profile.GetUAT()
	if strings.HasPrefix(uat, "oak_") {
		if err := login.RevokeToken(ctx, accessBaseURL); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: token revocation failed: %s\n", err)
		}
	}

	fmt.Println("Logging out...")
	if err := cfg.RemoveAllAuthFields(); err != nil {
		return err
	}
	fmt.Println("Credentials have been cleared for all projects.")
	return nil
}
