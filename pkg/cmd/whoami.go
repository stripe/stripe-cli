package cmd

import (
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/spf13/cobra"

	"github.com/stripe/stripe-cli/pkg/config"
	"github.com/stripe/stripe-cli/pkg/errorcategory"
	"github.com/stripe/stripe-cli/pkg/login"
	"github.com/stripe/stripe-cli/pkg/requests"
	"github.com/stripe/stripe-cli/pkg/stripe"
	"github.com/stripe/stripe-cli/pkg/validators"
)

// errNotAuthenticated is returned by whoami when no credentials are found.
// root.go recognizes this sentinel to suppress duplicate error output while
// still exiting non-zero.
var errNotAuthenticated = errorcategory.New(errorcategory.Auth, "not authenticated")

type whoamiCmd struct {
	cmd           *cobra.Command
	profile       *config.Profile
	format        string
	accessBaseURL string
	apiBaseURL    string
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

// whoamiOAuthOutput mirrors exactly what the OAuth text output shows: the
// active context (account, mode, email, role, expiry) and the list of
// authorized accounts. Fields not surfaced in the text output (e.g. profile
// name, API version) are intentionally omitted here.
type whoamiOAuthOutput struct {
	DisplayName        string                     `json:"display_name,omitempty"`
	AccountID          string                     `json:"account_id,omitempty"`
	Mode               string                     `json:"mode,omitempty"`
	Email              string                     `json:"email,omitempty"`
	Role               string                     `json:"role,omitempty"`
	ExpiresAt          *int64                     `json:"expires_at,omitempty"`
	AuthorizedAccounts []config.AuthorizedAccount `json:"authorized_accounts,omitempty"`
}

func newWhoamiCmd() *whoamiCmd {
	wc := &whoamiCmd{
		profile: &Config.Profile,
	}

	wc.cmd = &cobra.Command{
		Use:   "whoami",
		Args:  validators.NoArgs,
		Short: "Show the current Stripe auth context",
		Long: `Display the current authentication context for the Stripe CLI.

Reads credentials from the config file and keychain — no API calls are made.

Use --format json for output suitable for scripting or agent consumption. The
schema is stable: test_mode_key and live_mode_key are always present regardless
of auth context, and authenticated: false indicates no usable credentials exist.

Exit codes:
  0  Authenticated (at least one key is available)
  1  Not authenticated, or an error occurred`,
		Example: `stripe whoami
  stripe whoami --format json
  stripe whoami --project-name myproject --format json`,
		RunE: wc.runWhoamiCmd,
	}

	wc.cmd.Flags().StringVar(&wc.format, "format", "", "Output format: 'json' for a stable JSON schema (suitable for scripting)")
	wc.cmd.Flags().StringVar(&wc.accessBaseURL, "access-base", login.DefaultAccessBaseURL, "Sets the access base URL")
	wc.cmd.Flags().MarkHidden("access-base") //nolint:errcheck
	wc.cmd.Flags().StringVar(&wc.apiBaseURL, "api-base", stripe.DefaultAPIBaseURL, "Sets the API base URL")
	wc.cmd.Flags().MarkHidden("api-base") //nolint:errcheck

	return wc
}

func (wc *whoamiCmd) runWhoamiCmd(cmd *cobra.Command, args []string) error {
	profile := wc.profile

	uat, _ := profile.GetUAT()
	if strings.HasPrefix(uat, "oak_") {
		if err := login.ValidateAccessBaseURL(wc.accessBaseURL); err != nil {
			return err
		}
		if err := stripe.ValidateAPIBaseURL(wc.apiBaseURL); err != nil {
			return err
		}
		return wc.runWhoamiOAuth(cmd, uat)
	}

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

const expiryDisplayFormat = "Jan 2, 2006 at 3:04 PM"

func (wc *whoamiCmd) runWhoamiOAuth(cmd *cobra.Command, uat string) error {
	w := cmd.OutOrStdout()

	accounts, err := login.ListAuthorizedAccounts(cmd.Context(), wc.accessBaseURL, uat)
	if err != nil {
		return fmt.Errorf("failed to fetch authorized accounts: %w", err)
	}

	ac, _ := config.GetActiveContext()

	var info requests.UserInfo
	if ac != nil {
		creds := stripe.NewOAKCredentials(uat, ac.AccountID, ac.Livemode)
		// Fail open: the user's info is less important than the authorized contexts
		info, _ = requests.GetUserInfo(cmd.Context(), wc.apiBaseURL, wc.profile, creds, ac.Livemode)
	}
	expiresAt, expiresAtErr := config.GetUATExpiresAt()

	out := buildOAuthWhoamiOutput(accounts, ac, info, expiresAt, expiresAtErr == nil)

	if strings.EqualFold(wc.format, "json") {
		b, err := json.MarshalIndent(out, "", "  ")
		if err != nil {
			return err
		}
		fmt.Fprintln(w, string(b))
		return nil
	}

	if ac != nil {
		tw := tabwriter.NewWriter(w, 0, 0, 3, ' ', 0)
		if out.Email != "" {
			fmt.Fprintf(tw, "User\t%s\n", out.Email)
		}
		if out.DisplayName != ac.AccountID {
			fmt.Fprintf(tw, "Context\t%s · %s (%s)\n", out.DisplayName, displayModeText(out.Mode), ac.AccountID)
		} else {
			fmt.Fprintf(tw, "Context\t%s · %s\n", out.DisplayName, displayModeText(out.Mode))
		}
		if out.Role != "" {
			fmt.Fprintf(tw, "Role\t%s\n", out.Role)
		}
		if expiresAtErr == nil {
			fmt.Fprintf(tw, "Expires\t%s\n", expiresAt.Local().Format(expiryDisplayFormat))
		}
		tw.Flush()
		fmt.Fprintln(w)
	}

	login.PrintAuthorizedContextsList(accounts)
	return nil
}

// buildOAuthWhoamiOutput assembles the OAuth whoami JSON output. It's a pure
// function so the mapping can be unit tested without making network calls.
func buildOAuthWhoamiOutput(accounts []config.AuthorizedAccount, ac *config.ActiveContext, info requests.UserInfo, expiresAt time.Time, hasExpiry bool) whoamiOAuthOutput {
	out := whoamiOAuthOutput{
		AuthorizedAccounts: accounts,
	}

	if ac == nil {
		return out
	}

	out.AccountID = ac.AccountID
	out.DisplayName = ac.AccountID
	for _, a := range accounts {
		if a.ID == ac.AccountID && a.Name != "" {
			out.DisplayName = a.Name
			break
		}
	}

	out.Email = info.Email
	out.Role = info.Role

	if ac.Livemode {
		out.Mode = "live"
	} else {
		out.Mode = "test"
	}

	if hasExpiry {
		ts := expiresAt.Unix()
		out.ExpiresAt = &ts
	}

	return out
}

// displayModeText maps the internal "test"/"live" mode value to the CLI's
// sandbox-oriented display terminology.
func displayModeText(mode string) string {
	if mode == "test" {
		return "sandbox"
	}
	return mode
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

	fmt.Fprintf(w, "Sandbox key:\t%s\n", keyAvailabilityText(data.TestModeKey))
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
