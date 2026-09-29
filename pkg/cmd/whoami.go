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
	livemode      bool
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
		Short: "Show what account you're logged in to and accounts you've authorized",
		Long: `Show what account you're logged in to and the accounts you've authorized.

Use --format json for output suitable for scripting or agent consumption.

Exit codes:
  0  Authenticated
  1  Not authenticated, or an error occurred`,
		Example: `stripe whoami
  stripe whoami --format json
  stripe whoami --project-name myproject --format json
  stripe whoami --context acct_123 --live`,
		RunE: wc.runWhoamiCmd,
	}

	wc.cmd.Flags().StringVar(&wc.format, "format", "", "Output format: 'json' for a stable JSON schema (suitable for scripting)")
	wc.cmd.Flags().BoolVar(&wc.livemode, "live", false, "Used with --context to report that account's live mode context instead of its sandbox (default: sandbox)")
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
		uat, err := config.RefreshUATIfNeeded(profile, uat)
		if err != nil {
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

	ac, err := resolveWhoamiActiveContext(wc.profile, accounts, wc.livemode)
	if err != nil {
		return err
	}

	var info requests.UserInfo
	if ac != nil {
		creds := stripe.NewOAKCredentials(uat, ac.AccountID, ac.Livemode)
		// Fail open: the user's info is less important than the authorized contexts
		info, _ = requests.GetUserInfo(cmd.Context(), wc.apiBaseURL, wc.profile, creds, ac.Livemode)
	}
	expiresAt, expiresAtErr := wc.profile.GetUATExpiresAt()

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
			fmt.Fprintf(tw, "Account\t%s · %s (%s)\n", out.DisplayName, displayModeText(out.Mode), ac.AccountID)
		} else {
			fmt.Fprintf(tw, "Account\t%s · %s\n", out.DisplayName, displayModeText(out.Mode))
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

	login.PrintAuthorizedContextsList(wc.profile, accounts)
	fmt.Fprintln(w)
	fmt.Fprintln(w, "Run 'stripe login' to change permissions or authorize access to additional accounts or sandboxes.")
	fmt.Fprintln(w, "Run 'stripe switch' to switch to a different account, or between live mode and a sandbox.")
	return nil
}

// resolveWhoamiActiveContext returns the context whoami should report on: the
// --context/STRIPE_CONTEXT override (validated against accounts, paired with
// the --live flag) when set, otherwise the persisted active context from
// 'stripe switch'.
func resolveWhoamiActiveContext(profile *config.Profile, accounts []config.AuthorizedAccount, livemode bool) (*config.ActiveContext, error) {
	override := profile.GetContextOverride()
	if override == "" {
		return profile.GetActiveContext()
	}

	mode := modeString(livemode)
	for _, a := range accounts {
		if a.ID != override {
			continue
		}
		if len(a.Modes) > 0 && !containsMode(a.Modes, mode) {
			return nil, errorcategory.UserInputErrorf("the account %q from --context/STRIPE_CONTEXT isn't authorized for %s mode. Run 'stripe whoami' without --context to see its available modes.", override, mode)
		}
		return &config.ActiveContext{AccountID: override, Livemode: livemode}, nil
	}

	return nil, errorcategory.UserInputErrorf("the account %q from --context/STRIPE_CONTEXT isn't among your authorized accounts. Run 'stripe whoami' to list them, or 'stripe login' to authorize it.", override)
}

func modeString(livemode bool) string {
	if livemode {
		return "live"
	}
	return "test"
}

func containsMode(modes []string, mode string) bool {
	for _, m := range modes {
		if m == mode {
			return true
		}
	}
	return false
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
