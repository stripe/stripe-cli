package config

import (
	"strings"
	"testing"

	"github.com/spf13/viper"
	"github.com/stretchr/testify/require"
)

// The v2 layout keeps every profile under the reserved profiles table, which is
// what makes a profile name unable to collide with a setting. Note that
// installed_plugins is both a real setting here and a profile name.
const v2ConfigFile = `config_version = 2
installed_plugins = ['apps']

[profiles.default]
  account_id = 'acct_v2'
  device_name = 'device-v2'
  display_name = 'V2 Account'
  test_mode_api_key = 'sk_test_v2_key'
  test_mode_key_expires_at = '2026-01-02'

[profiles.installed_plugins]
  display_name = 'Collides With A Setting'
  test_mode_api_key = 'sk_test_collide_key'
`

const v1ConfigFile = `installed_plugins = ['apps']

[default]
  account_id = 'acct_v1'
  device_name = 'device-v1'
  display_name = 'V1 Account'
  test_mode_api_key = 'sk_test_v1_key'
  test_mode_key_expires_at = '2026-01-02'
`

func TestReadProfileFromV2Layout(t *testing.T) {
	setupProfileConfig(t, v2ConfigFile)

	p := Profile{ProfileName: "default"}

	deviceName, err := p.GetDeviceName()
	require.NoError(t, err)
	require.Equal(t, "device-v2", deviceName)

	accountID, err := p.GetAccountID()
	require.NoError(t, err)
	require.Equal(t, "acct_v2", accountID)

	apiKey, err := p.GetAPIKey(false)
	require.NoError(t, err)
	require.Equal(t, "sk_test_v2_key", apiKey)

	require.Equal(t, "V2 Account", p.GetDisplayName())
	require.True(t, p.HasAPIKey(false))
}

func TestReadProfileFallsBackToV1Layout(t *testing.T) {
	setupProfileConfig(t, v1ConfigFile)

	p := Profile{ProfileName: "default"}

	deviceName, err := p.GetDeviceName()
	require.NoError(t, err)
	require.Equal(t, "device-v1", deviceName)

	apiKey, err := p.GetAPIKey(false)
	require.NoError(t, err)
	require.Equal(t, "sk_test_v1_key", apiKey)

	require.Equal(t, "V1 Account", p.GetDisplayName())
}

// A migrated file can still pick up a top-level profile table, because an older
// CLI or plugin writing the same file does not know about the profiles table.
// The migrated copy is the one that wins.
func TestReadProfilePrefersV2LayoutOverV1Shadow(t *testing.T) {
	setupProfileConfig(t, v2ConfigFile+`
[default]
  device_name = 'shadow-device'
  display_name = 'Shadow Account'
  test_mode_api_key = 'sk_test_shadow_key'
`)

	p := Profile{ProfileName: "default"}

	deviceName, err := p.GetDeviceName()
	require.NoError(t, err)
	require.Equal(t, "device-v2", deviceName)

	apiKey, err := p.GetAPIKey(false)
	require.NoError(t, err)
	require.Equal(t, "sk_test_v2_key", apiKey)
}

// The point of the restructure: a profile named after a setting no longer
// overwrites that setting.
func TestProfileNamedAfterSettingDoesNotCollide(t *testing.T) {
	c, _ := setupProfileConfig(t, v2ConfigFile)

	require.Equal(t, []string{"apps"}, c.GetInstalledPlugins())

	p := Profile{ProfileName: "installed_plugins"}
	require.Equal(t, "Collides With A Setting", p.GetDisplayName())
}

// Legacy field names have to keep resolving in a migrated file, since the
// migration moves fields without renaming them.
func TestLegacyFieldAliasResolvesInV2Layout(t *testing.T) {
	setupProfileConfig(t, `config_version = 2

[profiles.default]
  display_name = 'Legacy Account'
  secret_key = 'sk_test_legacy_key'
`)

	p := Profile{ProfileName: "default"}

	require.True(t, p.HasAPIKey(false))

	apiKey, err := p.GetAPIKey(false)
	require.NoError(t, err)
	require.Equal(t, "sk_test_legacy_key", apiKey)
}

func TestWriteConfigFieldUsesV2LayoutWhenMigrated(t *testing.T) {
	_, profilesFile := setupProfileConfig(t, v2ConfigFile)

	p := Profile{ProfileName: "default"}
	require.NoError(t, p.WriteConfigField("color", "on"))

	require.Equal(t, "on", viper.GetString("profiles.default.color"))

	contents := string(helperLoadBytes(t, profilesFile))
	require.Contains(t, contents, "[profiles.default]")
	// A flat write into a migrated file would create a second, shadow copy of
	// the profile at the top level.
	require.NotContains(t, contents, "\n[default]")
}

func TestWriteConfigFieldStaysFlatWhenNotMigrated(t *testing.T) {
	_, profilesFile := setupProfileConfig(t, v1ConfigFile)

	p := Profile{ProfileName: "default"}
	require.NoError(t, p.WriteConfigField("color", "on"))

	require.Equal(t, "on", viper.GetString("default.color"))

	contents := string(helperLoadBytes(t, profilesFile))
	require.NotContains(t, contents, "profiles")
}

func TestWriteProfileUsesV2LayoutWhenMigrated(t *testing.T) {
	setupProfileConfig(t, v2ConfigFile)

	p := Profile{
		ProfileName:    "default",
		DeviceName:     "rewritten-device",
		TestModeAPIKey: "sk_test_rewritten_key",
		DisplayName:    "Rewritten Account",
	}
	require.NoError(t, p.CreateProfile())

	require.Equal(t, "rewritten-device", viper.GetString("profiles.default.device_name"))
	require.Equal(t, "sk_test_rewritten_key", viper.GetString("profiles.default.test_mode_api_key"))
	require.False(t, viper.IsSet("default.device_name"))

	// Settings that share the top level with profiles must survive the write.
	require.Equal(t, []string{"apps"}, viper.GetStringSlice("installed_plugins"))
	require.Equal(t, ConfigVersionV2, viper.GetInt(ConfigVersionName))
}

// A delete has to clear the field from both layouts, or a stale copy is left
// behind in a file that contains both.
func TestDeleteConfigFieldClearsBothLayouts(t *testing.T) {
	setupProfileConfig(t, v2ConfigFile+`
[default]
  device_name = 'shadow-device'
  display_name = 'Shadow Account'
`)

	p := Profile{ProfileName: "default"}
	require.NoError(t, p.DeleteConfigField(DeviceNameName))

	require.False(t, viper.IsSet("profiles.default.device_name"))
	require.False(t, viper.IsSet("default.device_name"))
	require.Equal(t, "V2 Account", viper.GetString("profiles.default.display_name"))
}

// A config_version this binary does not know could only have been written by a
// newer CLI, which is a routine situation the moment a v3 ships: a pinned CLI in
// CI, or a synced home directory, reads the file a newer binary migrated.
const unsupportedVersionConfigFile = `config_version = 3

[profiles.default]
  device_name = 'device-v3'
  display_name = 'V3 Account'
  test_mode_api_key = 'sk_test_v3_key'
`

func readVersionFrom(t *testing.T, contents string) (int, error) {
	t.Helper()

	v := viper.New()
	v.SetConfigType("toml")
	require.NoError(t, v.ReadConfig(strings.NewReader(contents)))

	return configVersion(v)
}

func TestConfigVersionTreatsAbsentKeyAsV1(t *testing.T) {
	version, err := readVersionFrom(t, v1ConfigFile)
	require.NoError(t, err)
	require.Equal(t, ConfigVersionV1, version)
}

// Viper truncates a float before we ever see it, so these are rejected by the
// range check rather than by a cast failure. A value that truncates into the
// supported range is deliberately left alone.
func TestConfigVersionRejectsValuesThatAreNotVersionNumbers(t *testing.T) {
	for _, raw := range []string{"'abc'", "0", "0.3", "-1", "-1.12", "''"} {
		t.Run(raw, func(t *testing.T) {
			_, err := readVersionFrom(t, "config_version = "+raw+"\n")
			require.ErrorContains(t, err, "is not a version number")
		})
	}
}

func TestConfigVersionRejectsVersionsNewerThanSupported(t *testing.T) {
	_, err := readVersionFrom(t, unsupportedVersionConfigFile)
	require.ErrorContains(t, err, "understands up to 2")
}

// Reads stay best-effort on an unknown version: both layouts are tried, so the
// profile a newer CLI wrote is still found and read-only commands keep working.
func TestReadProfileStillWorksWhenVersionIsNewerThanSupported(t *testing.T) {
	setupProfileConfig(t, unsupportedVersionConfigFile)

	p := Profile{ProfileName: "default"}

	deviceName, err := p.GetDeviceName()
	require.NoError(t, err)
	require.Equal(t, "device-v3", deviceName)

	require.Equal(t, "V3 Account", p.GetDisplayName())
}

// Writes are refused instead, because both candidate layouts are a guess at that
// point and the flat guess is silently shadowed by the nested copy.
func TestWriteRefusedWhenVersionIsNewerThanSupported(t *testing.T) {
	_, profilesFile := setupProfileConfig(t, unsupportedVersionConfigFile)

	p := Profile{ProfileName: "default"}
	require.ErrorContains(t, p.WriteConfigField("color", "on"), "understands up to 2")

	// The refusal has to leave the file untouched, not half-written.
	require.Equal(t, unsupportedVersionConfigFile, string(helperLoadBytes(t, profilesFile)))
}

func TestWriteRefusedWhenVersionIsNotAVersionNumber(t *testing.T) {
	setupProfileConfig(t, `config_version = 'abc'

[profiles.default]
  display_name = 'Malformed Version'
`)

	p := Profile{ProfileName: "default"}
	require.ErrorContains(t, p.WriteConfigField("color", "on"), "is not a version number")
}

// An unknown version is not "migrated", so nothing downstream mistakes it for the
// v2 layout it happens to resemble.
func TestIsMigratedOnlyAtTheSupportedVersion(t *testing.T) {
	setupProfileConfig(t, unsupportedVersionConfigFile)
	require.False(t, isMigrated(viper.GetViper()))
}
