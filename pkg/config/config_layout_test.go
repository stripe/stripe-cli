package config

import (
	"bytes"
	"os"
	"testing"

	"github.com/spf13/viper"
	"github.com/stretchr/testify/require"
)

func captureStdout(t *testing.T, fn func() error) string {
	t.Helper()

	original := os.Stdout
	r, w, err := os.Pipe()
	require.NoError(t, err)
	os.Stdout = w

	fnErr := fn()

	w.Close()
	os.Stdout = original

	var buf bytes.Buffer
	buf.ReadFrom(r)

	require.NoError(t, fnErr)

	return buf.String()
}

func TestListProfilesReadsBothLayouts(t *testing.T) {
	c, _ := setupProfileConfig(t, v2ConfigFile+`
[legacy]
  display_name = 'Legacy Account'
`)
	c.Profile = Profile{ProfileName: "default"}

	output := captureStdout(t, c.ListProfiles)

	require.Contains(t, output, "V2 Account")
	require.Contains(t, output, "Collides With A Setting")
	require.Contains(t, output, "Legacy Account")
	require.Contains(t, output, "* V2 Account (active)")
}

func TestRemoveProfileInV2Layout(t *testing.T) {
	c, _ := setupProfileConfig(t, v2ConfigFile)

	require.NoError(t, c.RemoveProfile("installed_plugins"))

	require.False(t, viper.IsSet("profiles.installed_plugins"))
	require.Equal(t, "V2 Account", viper.GetString("profiles.default.display_name"))
	// Removing a profile that shares a name with a setting must not remove the
	// setting.
	require.Equal(t, []string{"apps"}, viper.GetStringSlice("installed_plugins"))
}

func TestRemoveAllProfilesInV2Layout(t *testing.T) {
	c, _ := setupProfileConfig(t, v2ConfigFile)

	require.NoError(t, c.RemoveAllProfiles())

	require.False(t, viper.IsSet("profiles.default"))
	require.False(t, viper.IsSet("profiles.installed_plugins"))
	require.Equal(t, []string{"apps"}, viper.GetStringSlice("installed_plugins"))
}

func TestRemoveAuthFieldsInV2Layout(t *testing.T) {
	c, _ := setupProfileConfig(t, `config_version = 2

[profiles.default]
  color = 'on'
  display_name = 'V2 Account'
  test_mode_api_key = 'sk_test_v2_key'
`)

	require.NoError(t, c.RemoveAuthFields("default"))

	require.False(t, viper.IsSet("profiles.default.test_mode_api_key"))
	require.Equal(t, "on", viper.GetString("profiles.default.color"))
}

func TestCopyProfileInV2Layout(t *testing.T) {
	c, _ := setupProfileConfig(t, v2ConfigFile)

	require.NoError(t, c.CopyProfile("default", "backup"))

	require.Equal(t, "sk_test_v2_key", viper.GetString("profiles.backup.test_mode_api_key"))
	require.Equal(t, "backup", viper.GetString("profiles.backup.profile_name"))
	require.False(t, viper.IsSet("backup"))
}

func TestPrintConfigReadsV2Layout(t *testing.T) {
	c, _ := setupProfileConfig(t, v2ConfigFile)
	c.Profile = Profile{ProfileName: "installed_plugins"}

	output := captureStdout(t, c.PrintConfig)

	require.Contains(t, output, "[installed_plugins]")
	require.Contains(t, output, "display_name=Collides With A Setting")
}
