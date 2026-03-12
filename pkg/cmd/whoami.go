package cmd

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/spf13/cobra"

	stripecfg "github.com/stripe/stripe-cli/pkg/config"
)

func init() {
	rootCmd.AddCommand(newWhoamiCmd())
}

type whoamiOutput struct {
	ProjectName string `json:"project_name"`

	AccountID    string `json:"account_id,omitempty"`
	DisplayName  string `json:"display_name,omitempty"`
	DeviceName   string `json:"device_name,omitempty"`
	Color        string `json:"color,omitempty"`
	HasTestKey   bool   `json:"has_test_mode_api_key"`
	HasLiveKey   bool   `json:"has_live_mode_api_key"`
	TestKeyExp   string `json:"test_mode_key_expires_at,omitempty"`
	LiveKeyExp   string `json:"live_mode_key_expires_at,omitempty"`
	TestAPIKey   string `json:"test_mode_api_key,omitempty"`
	LiveAPIKey   string `json:"live_mode_api_key,omitempty"`
	ProfilesFile string `json:"profiles_file,omitempty"`
}

func newWhoamiCmd() *cobra.Command {
	var asJSON bool
	var showKeys bool

	cmd := &cobra.Command{
		Use:   "whoami",
		Short: "Show the currently selected Stripe CLI profile/environment",
		Long:  "Prints which Stripe CLI profile/environment you are currently operating against (project, account, display name, device, and key expiry).",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			// Config is the global CLI configuration in this package:
			// var Config config.Config (documented on pkg.go.dev)
			p := Config.GetProfile()
			if p == nil {
				return fmt.Errorf("no active profile found (try `stripe login` or check your config)")
			}

			out := whoamiOutput{
				ProjectName: cmd.Flag("project-name").Value.String(),
				DisplayName: p.GetDisplayName(),
				ProfilesFile: func() string {
					// Helpful for debugging which file you're reading from.
					// Safe even if empty.
					return Config.ProfilesFile
				}(),
			}

			if v, err := p.GetAccountID(); err == nil {
				out.AccountID = v
			}
			if v, err := p.GetDeviceName(); err == nil {
				out.DeviceName = v
			}
			if v, err := p.GetColor(); err == nil {
				out.Color = v
			}

			// Keys and expirations
			testKey, testKeyErr := p.GetAPIKey(false)
			liveKey, liveKeyErr := p.GetAPIKey(true)

			out.HasTestKey = testKeyErr == nil && testKey != ""
			out.HasLiveKey = liveKeyErr == nil && liveKey != ""

			if t, err := p.GetExpiresAt(false); err == nil && !t.IsZero() {
				out.TestKeyExp = t.Format(stripecfg.DateStringFormat)
			}

			if t, err := p.GetExpiresAt(true); err == nil && !t.IsZero() {
				out.LiveKeyExp = t.Format(stripecfg.DateStringFormat)
			}

			if showKeys {
				// Redact rather than dumping secrets.
				if out.HasTestKey {
					out.TestAPIKey = stripecfg.RedactAPIKey(testKey)
				}
				if out.HasLiveKey {
					out.LiveAPIKey = stripecfg.RedactAPIKey(liveKey)
				}
			}

			if asJSON {
				b, err := json.MarshalIndent(out, "", "")
				if err != nil {
					return err
				}
				fmt.Fprintln(cmd.OutOrStdout(), string(b))
				return nil
			}

			// Human output (boring, clear, and grep-friendly).
			fmt.Fprintf(cmd.OutOrStdout(), "project-name: %s\n", out.ProjectName)

			if out.DisplayName != "" {
				fmt.Fprintf(cmd.OutOrStdout(), "display_name: %s\n", out.DisplayName)
			}

			if out.AccountID != "" {
				fmt.Fprintf(cmd.OutOrStdout(), "account_id: %s\n", out.AccountID)
			}
			if out.DeviceName != "" {
				fmt.Fprintf(cmd.OutOrStdout(), "device_name: %s\n", out.DeviceName)
			}
			if out.Color != "" {
				fmt.Fprintf(cmd.OutOrStdout(), "color: %s\n", out.Color)
			}

			fmt.Fprintf(cmd.OutOrStdout(), "has_test_mode_api_key: %t\n", out.HasTestKey)
			if out.TestKeyExp != "" {
				fmt.Fprintf(cmd.OutOrStdout(), "test_mode_key_expires_at: %s\n", out.TestKeyExp)
			}

			fmt.Fprintf(cmd.OutOrStdout(), "has_live_mode_api_key: %t\n", out.HasLiveKey)
			if out.LiveKeyExp != "" {
				fmt.Fprintf(cmd.OutOrStdout(), "live_mode_key_expires_at: %s\n", out.LiveKeyExp)
			}

			if showKeys {
				if out.HasTestKey {
					fmt.Fprintf(cmd.OutOrStdout(), "test_mode_api_key: %s\n", out.TestAPIKey)
				}
				if out.HasLiveKey {
					fmt.Fprintf(cmd.OutOrStdout(), "live_mode_api_key: %s\n", out.LiveAPIKey)
				}
			}

			// Tiny extra clue: if the test key is expired, say it loudly.
			if out.TestKeyExp != "" {
				if exp, err := time.Parse(stripecfg.DateStringFormat, out.TestKeyExp); err == nil {
					if time.Now().After(exp.Add(24 * time.Hour)) {
						fmt.Fprintln(cmd.OutOrStdout(), "warning: test_mode_api_key appears expired (re-login may be required)")
					}
				}
			}
			return nil
		},
	}

	cmd.Flags().BoolVar(&asJSON, "json", false, "Output as JSON")
	cmd.Flags().BoolVar(&showKeys, "show-keys", false, "Include redacted API keys in output")
	return cmd
}
