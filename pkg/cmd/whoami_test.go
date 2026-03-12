package cmd

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/99designs/keyring"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/require"

	stripecfg "github.com/stripe/stripe-cli/pkg/config"
)

func writeTempConfig(t *testing.T, contents string) string {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "config.toml")
	err := os.WriteFile(path, []byte(contents), 0o600)
	require.NoError(t, err)
	return path
}

func setupWhoamiConfig(t *testing.T) (profilesFile string) {
	t.Helper()

	viper.Reset()
	// Ensure tests are hermetic; Profile.GetAPIKey() prefers STRIPE_API_KEY.
	t.Setenv("STRIPE_API_KEY", "")
	t.Setenv("STRIPE_DEVICE_NAME", "device-from-env")

	profilesFile = writeTempConfig(t, `[test]
account_id = "acct_123"
display_name = "Alice"
test_mode_api_key = "sk_test_1234abcd"
test_mode_key_expires_at = "2099-01-02"
live_mode_key_expires_at = "2099-02-03"
`)

	// Configure the CLI to read from our temp config.
	Config.ProfilesFile = profilesFile
	Config.Profile.ProfileName = "test"
	// Ensure no leftover state from other tests/environment overrides config/keyring.
	Config.Profile.APIKey = ""
	Config.Profile.AccountID = ""
	Config.Profile.DisplayName = ""
	Config.Profile.DeviceName = ""

	// Load config into viper (GetAPIKey(false) requires ReadInConfig() to succeed).
	viper.SetConfigFile(profilesFile)
	viper.SetConfigType("toml")
	require.NoError(t, viper.ReadInConfig())

	// Use an in-memory keyring to simulate a stored live key.
	stripecfg.KeyRing = keyring.NewArrayKeyring([]keyring.Item{{
		Key:  "test.live_mode_api_key",
		Data: []byte("rk_live_0000000001"),
	}})

	return profilesFile
}

func runWhoami(t *testing.T, profilesFile string, asJSON bool, showKeys bool) (string, error) {
	t.Helper()

	c := newWhoamiCmd()
	// In production this flag is a persistent flag on the root command. The
	// implementation reads it from the command, so add it here for unit tests.
	c.Flags().String("project-name", "test", "the project name to read from for config")

	require.NoError(t, c.Flags().Set("project-name", "test"))
	require.NoError(t, c.Flags().Set("json", map[bool]string{true: "true", false: "false"}[asJSON]))
	require.NoError(t, c.Flags().Set("show-keys", map[bool]string{true: "true", false: "false"}[showKeys]))

	buf := new(bytes.Buffer)
	c.SetOut(buf)

	err := c.RunE(c, []string{})
	return buf.String(), err
}

func TestWhoami_JSON_NoKeys(t *testing.T) {
	profilesFile := setupWhoamiConfig(t)
	out, err := runWhoami(t, profilesFile, true, false)
	require.NoError(t, err)

	var m map[string]any
	require.NoError(t, json.Unmarshal([]byte(out), &m))

	require.Equal(t, "test", m["project_name"])
	require.Equal(t, "acct_123", m["account_id"])
	require.Equal(t, "Alice", m["display_name"])
	require.Equal(t, "device-from-env", m["device_name"])
	require.Equal(t, "auto", m["color"])
	require.Equal(t, true, m["has_test_mode_api_key"])
	require.Equal(t, true, m["has_live_mode_api_key"])
	require.Equal(t, "2099-01-02", m["test_mode_key_expires_at"])
	require.Equal(t, "2099-02-03", m["live_mode_key_expires_at"])
	require.Equal(t, profilesFile, m["profiles_file"])

	_, hasTestKey := m["test_mode_api_key"]
	_, hasLiveKey := m["live_mode_api_key"]
	require.False(t, hasTestKey)
	require.False(t, hasLiveKey)
}

func TestWhoami_JSON_ShowKeys_Redacted(t *testing.T) {
	profilesFile := setupWhoamiConfig(t)
	out, err := runWhoami(t, profilesFile, true, true)
	require.NoError(t, err)

	var m map[string]any
	require.NoError(t, json.Unmarshal([]byte(out), &m))

	require.Equal(t, stripecfg.RedactAPIKey("sk_test_1234abcd"), m["test_mode_api_key"])
	require.Equal(t, stripecfg.RedactAPIKey("rk_live_0000000001"), m["live_mode_api_key"])
}

func TestWhoami_HumanOutput(t *testing.T) {
	profilesFile := setupWhoamiConfig(t)
	out, err := runWhoami(t, profilesFile, false, false)
	require.NoError(t, err)

	require.Contains(t, out, "project-name: test\n")
	require.Contains(t, out, "display_name: Alice\n")
	require.Contains(t, out, "account_id: acct_123\n")
	require.Contains(t, out, "device_name: device-from-env\n")
	require.Contains(t, out, "color: auto\n")
	require.Contains(t, out, "has_test_mode_api_key: true\n")
	require.Contains(t, out, "test_mode_key_expires_at: 2099-01-02\n")
	require.Contains(t, out, "has_live_mode_api_key: true\n")
	require.Contains(t, out, "live_mode_key_expires_at: 2099-02-03\n")
	require.NotContains(t, out, "\ntest_mode_api_key: ")
	require.NotContains(t, out, "\nlive_mode_api_key: ")
}

func TestWhoami_HumanOutput_ShowKeys(t *testing.T) {
	profilesFile := setupWhoamiConfig(t)
	out, err := runWhoami(t, profilesFile, false, true)
	require.NoError(t, err)

	require.Contains(t, out, "test_mode_api_key: "+stripecfg.RedactAPIKey("sk_test_1234abcd")+"\n")
	require.Contains(t, out, "live_mode_api_key: "+stripecfg.RedactAPIKey("rk_live_0000000001")+"\n")
}
