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
	"github.com/stripe/stripe-cli/pkg/i18n"
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
		Use:     "whoami",
		Args:    validators.NoArgs,
		Short:   i18n.T("whoami.short"),
		Long:    i18n.T("whoami.long"),
		Example: i18n.T("whoami.example"),
		RunE:    wc.runWhoamiCmd,
	}

	wc.cmd.Flags().StringVar(&wc.format, "format", "", i18n.T("whoami.flags.format"))

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

	fmt.Fprint(w, i18n.Tf("whoami.output.profile", i18n.Args{"name": data.ProfileName}))

	if !data.Authenticated {
		fmt.Fprintln(w, i18n.T("whoami.output.authenticated_false"))
		w.Flush()
		fmt.Fprintln(out, i18n.T("whoami.output.login_hint"))
		return
	}

	switch {
	case data.DisplayName != "" && data.AccountID != "":
		fmt.Fprint(w, i18n.Tf("whoami.output.account_with_display_and_id", i18n.Args{"display_name": data.DisplayName, "account_id": data.AccountID}))
	case data.DisplayName != "":
		fmt.Fprint(w, i18n.Tf("whoami.output.account_with_display", i18n.Args{"display_name": data.DisplayName}))
	case data.AccountID != "":
		fmt.Fprint(w, i18n.Tf("whoami.output.account_with_id", i18n.Args{"account_id": data.AccountID}))
	}

	if data.DeviceName != "" {
		fmt.Fprint(w, i18n.Tf("whoami.output.device_name", i18n.Args{"name": data.DeviceName}))
	}

	fmt.Fprint(w, i18n.Tf("whoami.output.test_mode_key", i18n.Args{"status": keyAvailabilityText(data.TestModeKey)}))
	fmt.Fprint(w, i18n.Tf("whoami.output.live_mode_key", i18n.Args{"status": keyAvailabilityText(data.LiveModeKey)}))
	fmt.Fprint(w, i18n.Tf("whoami.output.api_version", i18n.Args{"version": data.APIVersion}))
	fmt.Fprint(w, i18n.Tf("whoami.output.preview_api_version", i18n.Args{"version": data.PreviewAPIVersion}))
}

func keyAvailabilityText(k whoamiKeyInfo) string {
	if !k.Available {
		return i18n.T("whoami.output.key_not_available")
	}
	if k.ExpiresAt != nil {
		return i18n.Tf("whoami.output.key_available_expires", i18n.Args{"date": *k.ExpiresAt})
	}
	return i18n.T("whoami.output.key_available")
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
