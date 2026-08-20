package config

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/spf13/viper"
	"github.com/stretchr/testify/require"
)

// writeConfigFileForMigration writes a config file to migrate and returns its
// path. It deliberately does not initialize viper: the migration reads the file
// directly.
func writeConfigFileForMigration(t *testing.T, contents string) string {
	t.Helper()

	path := filepath.Join(t.TempDir(), "config.toml")
	require.NoError(t, os.WriteFile(path, []byte(contents), 0600))

	return path
}

const v1ConfigFileToMigrate = `installed_plugins = ['apps', 'projects']
machine_uuid = 'f4a1c7de-0000-0000-0000-000000000000'

[plugin_configs.__global]
  updates = 'off'

[user_info]
  compartments = []

[default]
  account_id = 'acct_1'
  device_name = 'device-1'
  display_name = 'Account One'
  test_mode_api_key = 'sk_test_one_key'

["acme corp"]
  account_id = 'acct_2'
  display_name = 'Account Two'
  test_mode_api_key = 'sk_test_two_key'
`

func TestMigrateConfigFileMovesProfilesUnderProfilesTable(t *testing.T) {
	path := writeConfigFileForMigration(t, v1ConfigFileToMigrate)

	changed, err := MigrateConfigFile(path)
	require.NoError(t, err)
	require.True(t, changed)

	contents := string(helperLoadBytes(t, path))
	require.Contains(t, contents, "config_version = 2")
	require.Contains(t, contents, "[profiles.default]")
	require.Contains(t, contents, `[profiles."acme corp"]`)
	require.NotContains(t, contents, "\n[default]")
}

func TestMigrateConfigFileKeepsSettingsAtTopLevel(t *testing.T) {
	path := writeConfigFileForMigration(t, v1ConfigFileToMigrate)

	_, err := MigrateConfigFile(path)
	require.NoError(t, err)

	v := viper.New()
	v.SetConfigFile(path)
	require.NoError(t, v.ReadInConfig())

	require.Equal(t, []string{"apps", "projects"}, v.GetStringSlice(InstalledPluginsKey))
	require.Equal(t, "f4a1c7de-0000-0000-0000-000000000000", v.GetString(MachineUUIDKey))
	require.Equal(t, "off", v.GetString(PluginConfigKey(PluginConfigGlobalScope, PluginConfigUpdatesField)))
	require.True(t, v.IsSet(UserInfoName))
	require.Equal(t, ConfigVersionV2, v.GetInt(ConfigVersionName))
}

// The CLI has to be able to read what the migration writes. This is the seam
// between the two, so assert it directly rather than through the file contents.
func TestMigratedConfigIsReadableByProfile(t *testing.T) {
	path := writeConfigFileForMigration(t, v1ConfigFileToMigrate)

	_, err := MigrateConfigFile(path)
	require.NoError(t, err)

	setupProfileConfig(t, string(helperLoadBytes(t, path)))

	p := Profile{ProfileName: "default"}
	apiKey, err := p.GetAPIKey(false)
	require.NoError(t, err)
	require.Equal(t, "sk_test_one_key", apiKey)
	require.Equal(t, "Account One", p.GetDisplayName())

	second := Profile{ProfileName: "acme corp"}
	require.Equal(t, "Account Two", second.GetDisplayName())
}

func TestMigrateConfigFileLeavesBackup(t *testing.T) {
	path := writeConfigFileForMigration(t, v1ConfigFileToMigrate)

	_, err := MigrateConfigFile(path)
	require.NoError(t, err)

	backup := helperLoadBytes(t, path+ConfigBackupSuffix)
	require.Equal(t, v1ConfigFileToMigrate, string(backup))

	// Windows has no Unix permission bits, so the mode the file was created with
	// does not survive a Stat there.
	if runtime.GOOS != "windows" {
		info, err := os.Stat(path)
		require.NoError(t, err)
		require.Equal(t, os.FileMode(0600), info.Mode().Perm())
	}
}

func TestMigrateConfigFileIsIdempotent(t *testing.T) {
	path := writeConfigFileForMigration(t, v1ConfigFileToMigrate)

	changed, err := MigrateConfigFile(path)
	require.NoError(t, err)
	require.True(t, changed)

	firstPass := helperLoadBytes(t, path)

	changed, err = MigrateConfigFile(path)
	require.NoError(t, err)
	require.False(t, changed)
	require.Equal(t, firstPass, helperLoadBytes(t, path))
}

// viper lowercases every key it reads, so the migration reads raw TOML. A
// profile named with capitals must come out the other side unchanged.
func TestMigrateConfigFilePreservesProfileNameCase(t *testing.T) {
	path := writeConfigFileForMigration(t, `[AcmeCorp]
  display_name = 'Acme Corp'
  test_mode_api_key = 'sk_test_acme_key'
`)

	_, err := MigrateConfigFile(path)
	require.NoError(t, err)

	require.Contains(t, string(helperLoadBytes(t, path)), "[profiles.AcmeCorp]")
}

// An older CLI or plugin writing a migrated file does not know about the
// profiles table, so it leaves a v1 table behind. Folding it in is the only way
// the file converges back to one copy of each profile.
func TestMigrateConfigFileFoldsShadowTable(t *testing.T) {
	path := writeConfigFileForMigration(t, `config_version = 2

[profiles.default]
  device_name = 'nested-device'
  display_name = 'Nested Account'

[default]
  device_name = 'shadow-device'
  test_mode_api_key = 'sk_test_shadow_key'
`)

	changed, err := MigrateConfigFile(path)
	require.NoError(t, err)
	require.True(t, changed)

	v := viper.New()
	v.SetConfigFile(path)
	require.NoError(t, v.ReadInConfig())

	// The nested copy is the one the current CLI reads and writes, so it wins.
	require.Equal(t, "nested-device", v.GetString("profiles.default.device_name"))
	// A field only the old writer knew about is not thrown away.
	require.Equal(t, "sk_test_shadow_key", v.GetString("profiles.default.test_mode_api_key"))
	require.False(t, v.IsSet("default.device_name"))
}

// Nothing stops `stripe login --project-name profiles` on a v1 CLI, so the
// reserved table name can already be taken by a profile.
func TestMigrateConfigFileHandlesProfileNamedProfiles(t *testing.T) {
	path := writeConfigFileForMigration(t, `[profiles]
  display_name = 'Confusing Account'
  test_mode_api_key = 'sk_test_confusing_key'
`)

	changed, err := MigrateConfigFile(path)
	require.NoError(t, err)
	require.True(t, changed)

	v := viper.New()
	v.SetConfigFile(path)
	require.NoError(t, v.ReadInConfig())

	require.Equal(t, "sk_test_confusing_key", v.GetString("profiles.profiles.test_mode_api_key"))
}

// An unrecognized table might be anything, including a setting a newer CLI
// writes. Dual-read means leaving it alone costs nothing, while guessing wrong
// would move a setting into the profiles table.
func TestMigrateConfigFileLeavesUnrecognizedTableAlone(t *testing.T) {
	path := writeConfigFileForMigration(t, `[some_future_setting]
  enabled = true

[default]
  display_name = 'Account One'
  test_mode_api_key = 'sk_test_one_key'
`)

	_, err := MigrateConfigFile(path)
	require.NoError(t, err)

	v := viper.New()
	v.SetConfigFile(path)
	require.NoError(t, v.ReadInConfig())

	require.True(t, v.GetBool("some_future_setting.enabled"))
	require.Equal(t, "Account One", v.GetString("profiles.default.display_name"))
}

// A dotted profile name parses as a nested table, so it is not recognizable as a
// profile. It stays where it is and keeps working through the v1 read fallback.
func TestMigrateConfigFileLeavesDottedProfileAlone(t *testing.T) {
	path := writeConfigFileForMigration(t, `[example.project]
  display_name = 'Dotted Account'
  test_mode_api_key = 'sk_test_dotted_key'
`)

	_, err := MigrateConfigFile(path)
	require.NoError(t, err)

	v := viper.New()
	v.SetConfigFile(path)
	require.NoError(t, v.ReadInConfig())

	require.Equal(t, "sk_test_dotted_key", v.GetString("example.project.test_mode_api_key"))
	require.False(t, v.IsSet("profiles.example"))
}

// A profile abandoned part-way through login has no display_name, so the
// migration cannot use isProfile's rule to find it.
func TestMigrateConfigFileMovesProfileWithoutDisplayName(t *testing.T) {
	path := writeConfigFileForMigration(t, `[default]
  device_name = 'partial-device'
`)

	_, err := MigrateConfigFile(path)
	require.NoError(t, err)

	require.Contains(t, string(helperLoadBytes(t, path)), "[profiles.default]")
}

func TestMigrateConfigFileStampsVersionOnFileWithNoProfiles(t *testing.T) {
	path := writeConfigFileForMigration(t, "installed_plugins = ['apps']\n")

	changed, err := MigrateConfigFile(path)
	require.NoError(t, err)
	require.True(t, changed)

	contents := string(helperLoadBytes(t, path))
	require.Contains(t, contents, "config_version = 2")
	require.Contains(t, contents, "installed_plugins")
}

func TestMigrateConfigFileMissingFile(t *testing.T) {
	changed, err := MigrateConfigFile(filepath.Join(t.TempDir(), "config.toml"))
	require.NoError(t, err)
	require.False(t, changed)
}

func TestMigrateConfigFileRejectsUnparseableFile(t *testing.T) {
	contents := "this is not = = toml\n"
	path := writeConfigFileForMigration(t, contents)

	changed, err := MigrateConfigFile(path)
	require.Error(t, err)
	require.False(t, changed)

	// The file the CLI cannot parse is still the file the user has.
	require.Equal(t, contents, string(helperLoadBytes(t, path)))
	require.NoFileExists(t, path+ConfigBackupSuffix)
}

func TestNeedsMigrationForV1Config(t *testing.T) {
	setupProfileConfig(t, v1ConfigFile)

	require.True(t, NeedsMigration())
	require.False(t, IsMigrated())
}

func TestNeedsMigrationForV2Config(t *testing.T) {
	setupProfileConfig(t, v2ConfigFile)

	require.False(t, NeedsMigration())
	require.True(t, IsMigrated())
}

// A migrated file that has picked up a top-level profile still has work to do:
// folding the stray copy back in is what stops the two from diverging.
func TestNeedsMigrationForV2ConfigWithShadowTable(t *testing.T) {
	setupProfileConfig(t, v2ConfigFile+`
[legacy]
  display_name = 'Written By An Old Plugin'
`)

	require.True(t, NeedsMigration())
}

// A setting that happens to be a table is not a profile, so it is not work.
func TestNeedsMigrationIgnoresSettingTables(t *testing.T) {
	setupProfileConfig(t, `config_version = 2

[profiles.default]
  display_name = 'V2 Account'

[plugin_configs.__global]
  updates = 'off'

[user_info]
  compartments = []
`)

	require.False(t, NeedsMigration())
}

// Writing the version out explicitly is how a user opts out for good.
func TestConfigPinnedToV1IsLeftAlone(t *testing.T) {
	pinned := `config_version = 1

[default]
  display_name = 'Pinned Account'
  test_mode_api_key = 'sk_test_pinned_key'
`
	_, profilesFile := setupProfileConfig(t, pinned)

	require.False(t, NeedsMigration())

	changed, err := MigrateConfigFile(profilesFile)
	require.NoError(t, err)
	require.False(t, changed)
	require.Equal(t, pinned, string(helperLoadBytes(t, profilesFile)))
}

// The migration rewrites the config file, so it has to refuse the same symlinked
// destinations the normal write path refuses.
func TestMigrateConfigFileRefusesSymlink(t *testing.T) {
	profilesFile, victimFile := setupSymlinkedProfilesFile(t)

	changed, err := MigrateConfigFile(profilesFile)
	require.ErrorContains(t, err, "symlink")
	require.False(t, changed)

	require.Equal(t, "original = true\n", string(helperLoadBytes(t, victimFile)))
	require.NoFileExists(t, profilesFile+ConfigBackupSuffix)
}

func TestMigrateConfigFileRefusesSymlinkedParent(t *testing.T) {
	profilesFile, victimDir := setupProfilesFileWithSymlinkedParent(t)
	require.NoError(t, os.WriteFile(profilesFile, []byte(v1ConfigFileToMigrate), 0600))

	changed, err := MigrateConfigFile(profilesFile)
	require.ErrorContains(t, err, "symlink")
	require.False(t, changed)

	require.Equal(t, v1ConfigFileToMigrate, string(helperLoadBytes(t, filepath.Join(victimDir, "config.toml"))))
}

func TestVerifyPlanCatchesADroppedField(t *testing.T) {
	plan := &migrationPlan{
		profiles: map[string]map[string]interface{}{
			"default": {"display_name": "Account One", "test_mode_api_key": "sk_test_one_key"},
		},
		settings: map[string]interface{}{},
	}

	err := verifyPlan(plan, []byte("config_version = 2\n\n[profiles.default]\n  display_name = 'Account One'\n"))
	require.ErrorContains(t, err, "lost field")
}

func TestVerifyPlanCatchesAChangedProfileName(t *testing.T) {
	plan := &migrationPlan{
		profiles: map[string]map[string]interface{}{
			"AcmeCorp": {"display_name": "Acme Corp"},
		},
		settings: map[string]interface{}{},
	}

	err := verifyPlan(plan, []byte("config_version = 2\n\n[profiles.acmecorp]\n  display_name = 'Acme Corp'\n"))
	require.ErrorContains(t, err, "missing from the migrated config")
}

func TestVerifyPlanCatchesADroppedSetting(t *testing.T) {
	plan := &migrationPlan{
		profiles: map[string]map[string]interface{}{},
		settings: map[string]interface{}{InstalledPluginsKey: []interface{}{"apps"}},
	}

	err := verifyPlan(plan, []byte("config_version = 2\n\n[profiles]\n"))
	require.ErrorContains(t, err, "is missing from the migrated config")
}
