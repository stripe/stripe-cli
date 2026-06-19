package cmd

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"
	"text/tabwriter"

	"github.com/spf13/cobra"

	"github.com/stripe/stripe-cli/pkg/config"
	"github.com/stripe/stripe-cli/pkg/requests"
	"github.com/stripe/stripe-cli/pkg/validators"
)

// errNotAuthenticated is returned by whoami when no credentials are found.
// root.go recognizes this sentinel to suppress duplicate error output while
// still exiting non-zero.
var errNotAuthenticated = errors.New("not authenticated")

type whoamiCmd struct {
	cmd     *cobra.Command
	profile *config.Profile
	format  string
}

type whoamiKeyInfo struct {
	Available bool    `json:"available"`
	ExpiresAt *string `json:"expires_at"`
}

type whoamiOutput struct {
	Authenticated     bool          `json:"authenticated"`
	ProfileName       string        `json:"profile_name"`
	DisplayName       string        `json:"display_name,omitempty"`
	AccountID         string        `json:"account_id,omitempty"`
	DeviceName        string        `json:"device_name,omitempty"`
	TestModeKey       whoamiKeyInfo `json:"test_mode_key"`
	LiveModeKey       whoamiKeyInfo `json:"live_mode_key"`
	APIVersion        string        `json:"api_version"`
	PreviewAPIVersion string        `json:"preview_api_version"`
}

func newWhoamiCmd() *whoamiCmd {
	wc := &whoamiCmd{
		profile: &Config.Profile,
	}

	wc.cmd = &cobra.Command{
		Use:   "whoami",
		Args:  validators.NoArgs,
		Short: "Show the current Stripe auth state",
		Long: `Display the current authentication state for the Stripe CLI.

Reads credentials from the config file and keychain — no API calls are made.

Use --format json for output suitable for scripting or agent consumption. The
schema is stable: test_mode_key and live_mode_key are always present regardless
of auth state, and authenticated: false indicates no usable credentials exist.

Exit codes:
  0  Authenticated (at least one key is available)
  1  Not authenticated, or an error occurred`,
		Example: `stripe whoami
  stripe whoami --format json
  stripe whoami --project-name myproject --format json`,
		RunE: wc.runWhoamiCmd,
	}

	wc.cmd.Flags().StringVar(&wc.format, "format", "", "Output format: 'json' for a stable JSON schema (suitable for scripting)")

	return wc
}

func (wc *whoamiCmd) runWhoamiCmd(cmd *cobra.Command, args []string) error {
	profile := wc.profile

	testKey := resolveKeyInfo(profile, false)
	liveKey := resolveKeyInfo(profile, true)

	displayName := profile.GetDisplayName()
	accountID, _ := profile.GetAccountID()
	deviceName, _ := profile.GetDeviceName()

	out := whoamiOutput{
		Authenticated:     testKey.Available || liveKey.Available,
		ProfileName:       profile.ProfileName,
		DisplayName:       displayName,
		AccountID:         accountID,
		DeviceName:        deviceName,
		TestModeKey:       testKey,
		LiveModeKey:       liveKey,
		APIVersion:        requests.StripeVersionHeaderValue,
		PreviewAPIVersion: requests.StripePreviewVersionHeaderValue,
	}

	w := cmd.OutOrStdout()
	if strings.EqualFold(wc.format, "json") {
		b, err := json.MarshalIndent(out, "", "  ")
		if err != nil {
			return err
		}
		fmt.Fprintln(w, string(b))
	} else {
		printWhoamiText(w, out)
	}

	if !out.Authenticated {
		return errNotAuthenticated
	}
	return nil
}

func printWhoamiText(out io.Writer, data whoamiOutput) {
	w := tabwriter.NewWriter(out, 0, 0, 2, ' ', 0)
	defer w.Flush()

	fmt.Fprintf(w, "Profile:\t%s\n", data.ProfileName)

	if !data.Authenticated {
		fmt.Fprintln(w, "Authenticated:\tfalse")
		w.Flush()
		fmt.Fprintln(out, "Run `stripe login` to authenticate.")
		return
	}

	switch {
	case data.DisplayName != "" && data.AccountID != "":
		fmt.Fprintf(w, "Account:\t%s (%s)\n", data.DisplayName, data.AccountID)
	case data.DisplayName != "":
		fmt.Fprintf(w, "Account:\t%s\n", data.DisplayName)
	case data.AccountID != "":
		fmt.Fprintf(w, "Account:\t%s\n", data.AccountID)
	}

	if data.DeviceName != "" {
		fmt.Fprintf(w, "Device name:\t%s\n", data.DeviceName)
	}

	fmt.Fprintf(w, "Test mode key:\t%s\n", keyAvailabilityText(data.TestModeKey))
	fmt.Fprintf(w, "Live mode key:\t%s\n", keyAvailabilityText(data.LiveModeKey))
	fmt.Fprintf(w, "API version:\t%s\n", data.APIVersion)
	fmt.Fprintf(w, "Preview API version:\t%s\n", data.PreviewAPIVersion)
}

func keyAvailabilityText(k whoamiKeyInfo) string {
	if !k.Available {
		return "not available"
	}
	if k.ExpiresAt != nil {
		return fmt.Sprintf("available (expires %s)", *k.ExpiresAt)
	}
	return "available"
}

// resolveKeyInfo determines key availability and expiry for the given mode.
// HasAPIKey handles all sources (env var, --api-key flag, config file, keyring)
// without reading the secret, avoiding OS auth prompts on macOS.
func resolveKeyInfo(profile *config.Profile, livemode bool) whoamiKeyInfo {
	if !profile.HasAPIKey(livemode) {
		return whoamiKeyInfo{Available: false}
	}

	info := whoamiKeyInfo{Available: true}
	if !profile.HasOverrideAPIKey() {
		if t, err := profile.GetExpiresAt(livemode); err == nil {
			s := t.Format(config.DateStringFormat)
			info.ExpiresAt = &s
		}
	}
	return info
}
