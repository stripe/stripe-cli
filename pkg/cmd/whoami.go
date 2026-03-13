package cmd

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/stripe/stripe-cli/pkg/ansi"
	"github.com/stripe/stripe-cli/pkg/validators"
)

type whoamiCmd struct {
	cmd    *cobra.Command
	format string
}

func newWhoamiCmd() *whoamiCmd {
	wc := &whoamiCmd{}

	wc.cmd = &cobra.Command{
		Use:   "whoami",
		Args:  validators.NoArgs,
		Short: "Show the current Stripe account details",
		Long: `Display information about the currently authenticated Stripe account.

This command shows the account ID, display name, and other details
from your current CLI configuration. Useful for verifying which account
is active before running commands.`,
		Example: `stripe whoami
  stripe whoami --format json`,
		RunE: wc.runWhoamiCmd,
	}

	wc.cmd.Flags().StringVar(&wc.format, "format", "default", "Output format (default or json)")

	return wc
}

func (wc *whoamiCmd) runWhoamiCmd(cmd *cobra.Command, args []string) error {
	profile := &Config.Profile

	// Gather account information
	accountID, accountIDErr := profile.GetAccountID()
	displayName := profile.GetDisplayName()
	deviceName, _ := profile.GetDeviceName()
	profileName := profile.ProfileName

	// Check API key status
	testKeyConfigured := false
	liveKeyConfigured := false
	var testKeyExpiry time.Time
	var liveKeyExpiry time.Time

	if _, err := profile.GetAPIKey(false); err == nil {
		testKeyConfigured = true
		testKeyExpiry, _ = profile.GetExpiresAt(false)
	}

	if _, err := profile.GetAPIKey(true); err == nil {
		liveKeyConfigured = true
		liveKeyExpiry, _ = profile.GetExpiresAt(true)
	}

	// Check if logged in
	if accountIDErr != nil {
		return fmt.Errorf("not logged in. Run 'stripe login' to authenticate")
	}

	if wc.format == "json" {
		return wc.printJSON(accountID, displayName, deviceName, profileName,
			testKeyConfigured, liveKeyConfigured, testKeyExpiry, liveKeyExpiry)
	}

	return wc.printDefault(accountID, displayName, deviceName, profileName,
		testKeyConfigured, liveKeyConfigured, testKeyExpiry, liveKeyExpiry)
}

func (wc *whoamiCmd) printDefault(accountID, displayName, deviceName, profileName string,
	testKeyConfigured, liveKeyConfigured bool, testKeyExpiry, liveKeyExpiry time.Time) error {

	color := ansi.Color(os.Stdout)

	fmt.Println()
	fmt.Printf("%s\n", color.Bold("Stripe CLI Account Information"))
	fmt.Println(strings.Repeat("-", 40))

	fmt.Printf("%-18s %s\n", "Account ID:", color.Cyan(accountID))

	if displayName != "" {
		fmt.Printf("%-18s %s\n", "Display Name:", displayName)
	}

	fmt.Printf("%-18s %s\n", "Project:", profileName)

	if deviceName != "" {
		fmt.Printf("%-18s %s\n", "Device:", deviceName)
	}

	fmt.Println()
	fmt.Printf("%s\n", color.Bold("API Keys"))
	fmt.Println(strings.Repeat("-", 40))

	// Test mode key status
	if testKeyConfigured {
		status := color.Green("✓ Configured")
		if !testKeyExpiry.IsZero() {
			if testKeyExpiry.Before(time.Now()) {
				status = color.Red("✗ Expired")
			} else {
				status = fmt.Sprintf("%s (expires %s)", color.Green("✓ Configured"), testKeyExpiry.Format("Jan 02, 2006"))
			}
		}
		fmt.Printf("%-18s %s\n", "Test mode:", status)
	} else {
		fmt.Printf("%-18s %s\n", "Test mode:", color.Yellow("○ Not configured"))
	}

	// Live mode key status
	if liveKeyConfigured {
		status := color.Green("✓ Configured")
		if !liveKeyExpiry.IsZero() {
			if liveKeyExpiry.Before(time.Now()) {
				status = color.Red("✗ Expired")
			} else {
				status = fmt.Sprintf("%s (expires %s)", color.Green("✓ Configured"), liveKeyExpiry.Format("Jan 02, 2006"))
			}
		}
		fmt.Printf("%-18s %s\n", "Live mode:", status)
	} else {
		fmt.Printf("%-18s %s\n", "Live mode:", color.Yellow("○ Not configured"))
	}

	fmt.Println()
	return nil
}

func (wc *whoamiCmd) printJSON(accountID, displayName, deviceName, profileName string,
	testKeyConfigured, liveKeyConfigured bool, testKeyExpiry, liveKeyExpiry time.Time) error {

	testExpiry := ""
	if !testKeyExpiry.IsZero() {
		testExpiry = testKeyExpiry.Format(time.RFC3339)
	}

	liveExpiry := ""
	if !liveKeyExpiry.IsZero() {
		liveExpiry = liveKeyExpiry.Format(time.RFC3339)
	}

	fmt.Printf(`{
  "account_id": "%s",
  "display_name": "%s",
  "project": "%s",
  "device_name": "%s",
  "test_mode_key": {
    "configured": %t,
    "expires_at": "%s"
  },
  "live_mode_key": {
    "configured": %t,
    "expires_at": "%s"
  }
}
`, accountID, displayName, profileName, deviceName,
		testKeyConfigured, testExpiry,
		liveKeyConfigured, liveExpiry)

	return nil
}
