package cmd

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/stripe/stripe-cli/pkg/config"
	"github.com/stripe/stripe-cli/pkg/errorcategory"
	"github.com/stripe/stripe-cli/pkg/keyring"
)

// A config file holding the shapes a profile can take on disk: a normal one, a
// dotted one written before periods were rejected, the space-substituted
// duplicate CopyProfile leaves behind, a table with no display_name, and the
// machine-wide keys that sit alongside profiles at the top level.
const removeProfileTestConfig = `color = 'auto'
machine_uuid = 'bfe13def-d33f-4b4a-9939-dfc900eb635c'
project-name = 'default'

["bbq.sandbox"]
account_id = 'acct_123'
device_name = 'st-test'
display_name = 'BBQ sandbox'
test_mode_api_key = 'sk_test_dotted'

['bbq sandbox']
account_id = 'acct_123'
device_name = 'st-test'
display_name = 'BBQ sandbox'
profile_name = 'BBQ sandbox'

[default]
device_name = 'st-test'
display_name = 'Default'
test_mode_api_key = 'sk_test_default'

[no-display-name]
device_name = 'st-test'

[plugin_configs]
[plugin_configs.apps]
updates = 'off'
`

func setupRemoveProfileTest(t *testing.T, contents string) (*configCmd, string) {
	t.Helper()

	profilesFile := filepath.Join(t.TempDir(), "config.toml")
	require.NoError(t, os.WriteFile(profilesFile, []byte(contents), 0600))

	origProfilesFile := Config.ProfilesFile
	origProfileName := Config.Profile.ProfileName
	origKeyRing := config.KeyRing

	viper.Reset()
	Config.ProfilesFile = profilesFile
	Config.Profile.ProfileName = "default"
	config.KeyRing = keyring.NewMemoryStore(map[string][]byte{
		"bbq.sandbox.live_mode_api_key": []byte("rk_live_dotted"),
		"default.live_mode_api_key":     []byte("rk_live_default"),
	})
	Config.InitConfig()

	t.Cleanup(func() {
		Config.ProfilesFile = origProfilesFile
		Config.Profile.ProfileName = origProfileName
		config.KeyRing = origKeyRing
		viper.Reset()
	})

	cc := newConfigCmd()
	cc.autoConfirm = true

	return cc, profilesFile
}

// run executes the command and returns everything it wrote to stdout.
func runRemoveProfile(t *testing.T, cc *configCmd, name string) (string, error) {
	t.Helper()

	var out bytes.Buffer
	cc.cmd.SetOut(&out)
	cc.cmd.SetArgs([]string{"--remove-profile", name})
	if cc.autoConfirm {
		cc.cmd.SetArgs([]string{"--remove-profile", name, "--confirm"})
	}

	err := cc.cmd.Execute()

	return out.String(), err
}

// readBack reloads the config file from disk so assertions see what was actually
// persisted rather than in-memory viper state.
func readBack(t *testing.T, profilesFile string) *viper.Viper {
	t.Helper()

	v := viper.New()
	v.SetConfigFile(profilesFile)
	v.SetConfigType("toml")
	require.NoError(t, v.ReadInConfig())

	return v
}

// The case that matters most here: a dotted profile is read back as a nested
// table, so the enumeration over top-level tables never sees it. Removal has to
// address it by key path, and must not take the sibling profile that shares its
// first segment.
func TestConfigRemoveProfileWithDottedName(t *testing.T) {
	cc, profilesFile := setupRemoveProfileTest(t, removeProfileTestConfig)

	out, err := runRemoveProfile(t, cc, "bbq.sandbox")
	require.NoError(t, err)
	assert.Contains(t, out, `Removed profile "bbq.sandbox"`)

	v := readBack(t, profilesFile)
	assert.False(t, v.IsSet("bbq.sandbox.display_name"), "dotted profile should be gone from the file")

	// The space-substituted duplicate is a separate profile and must survive.
	assert.Equal(t, "BBQ sandbox", v.GetString("bbq sandbox.display_name"))
	assert.Equal(t, "Default", v.GetString("default.display_name"))
	assert.Equal(t, "st-test", v.GetString("no-display-name.device_name"))

	// Machine-wide settings are untouched.
	assert.Equal(t, "bfe13def-d33f-4b4a-9939-dfc900eb635c", v.GetString("machine_uuid"))
	assert.Equal(t, "off", v.GetString("plugin_configs.apps.updates"))

	// The keyring entry is namespaced by profile name, so it has to go too, and
	// only for the profile that was removed.
	_, err = config.KeyRing.Get("bbq.sandbox.live_mode_api_key")
	assert.ErrorIs(t, err, keyring.ErrKeyNotFound)
	remaining, err := config.KeyRing.Get("default.live_mode_api_key")
	require.NoError(t, err)
	assert.Equal(t, []byte("rk_live_default"), remaining)
}

// Removing the space-substituted duplicate must not disturb the dotted original,
// which is the same assertion in the other direction.
func TestConfigRemoveProfileLeavesDottedSibling(t *testing.T) {
	cc, profilesFile := setupRemoveProfileTest(t, removeProfileTestConfig)

	_, err := runRemoveProfile(t, cc, "bbq sandbox")
	require.NoError(t, err)

	v := readBack(t, profilesFile)
	assert.False(t, v.IsSet("bbq sandbox.display_name"))
	assert.Equal(t, "BBQ sandbox", v.GetString("bbq.sandbox.display_name"))
	assert.Equal(t, "sk_test_dotted", v.GetString("bbq.sandbox.test_mode_api_key"))
}

func TestConfigRemoveProfileNormalName(t *testing.T) {
	cc, profilesFile := setupRemoveProfileTest(t, removeProfileTestConfig)

	_, err := runRemoveProfile(t, cc, "default")
	require.NoError(t, err)

	v := readBack(t, profilesFile)
	assert.False(t, v.IsSet("default.display_name"))
	assert.Equal(t, "BBQ sandbox", v.GetString("bbq.sandbox.display_name"))
}

// A profile written before display_name was always set is invisible to
// isProfile, so it needs the same key-path fallback the dotted case uses.
func TestConfigRemoveProfileWithoutDisplayName(t *testing.T) {
	cc, profilesFile := setupRemoveProfileTest(t, removeProfileTestConfig)

	_, err := runRemoveProfile(t, cc, "no-display-name")
	require.NoError(t, err)

	v := readBack(t, profilesFile)
	assert.False(t, v.IsSet("no-display-name.device_name"))
}

// A profile can also be addressed by its profile_name attribute, which is what
// CopyProfile records and what `stripe login list` displays.
func TestConfigRemoveProfileByProfileNameAttribute(t *testing.T) {
	cc, profilesFile := setupRemoveProfileTest(t, removeProfileTestConfig)

	_, err := runRemoveProfile(t, cc, "BBQ sandbox")
	require.NoError(t, err)

	v := readBack(t, profilesFile)
	assert.False(t, v.IsSet("bbq sandbox.display_name"))
}

func TestConfigRemoveProfileNotFound(t *testing.T) {
	cc, profilesFile := setupRemoveProfileTest(t, removeProfileTestConfig)
	before := readFile(t, profilesFile)

	_, err := runRemoveProfile(t, cc, "nope")
	require.Error(t, err)
	assert.Contains(t, err.Error(), `no profile named "nope" was found`)

	category, ok := errorcategory.Get(err)
	require.True(t, ok)
	assert.Equal(t, errorcategory.UserInput, category)

	assert.Equal(t, before, readFile(t, profilesFile), "a failed removal must not rewrite the file")
}

// A near-miss on a dotted name must not be treated as a hit on its first
// segment, which is the failure mode viper's key flattening invites.
func TestConfigRemoveProfileDoesNotMatchPartialDottedName(t *testing.T) {
	cc, profilesFile := setupRemoveProfileTest(t, removeProfileTestConfig)

	_, err := runRemoveProfile(t, cc, "bbq")
	require.Error(t, err)
	assert.Contains(t, err.Error(), `no profile named "bbq" was found`)

	v := readBack(t, profilesFile)
	assert.Equal(t, "BBQ sandbox", v.GetString("bbq.sandbox.display_name"))
}

// Machine-wide keys share the top level with profiles, so a name collision must
// not let this command delete them.
func TestConfigRemoveProfileRejectsReservedKeys(t *testing.T) {
	for _, name := range []string{"machine_uuid", "plugin_configs", "color", "project-name", "installed_plugins", "user_info"} {
		t.Run(name, func(t *testing.T) {
			cc, profilesFile := setupRemoveProfileTest(t, removeProfileTestConfig)
			before := readFile(t, profilesFile)

			_, err := runRemoveProfile(t, cc, name)
			require.Error(t, err)
			assert.Contains(t, err.Error(), "reserved config key")
			assert.Equal(t, before, readFile(t, profilesFile))
		})
	}
}

// plugin_configs is a nested table, so a key path under it would otherwise
// resolve like a dotted profile name.
func TestConfigRemoveProfileRejectsPluginConfigSubtable(t *testing.T) {
	cc, profilesFile := setupRemoveProfileTest(t, removeProfileTestConfig)

	_, err := runRemoveProfile(t, cc, "plugin_configs.apps")
	require.Error(t, err)

	v := readBack(t, profilesFile)
	assert.Equal(t, "off", v.GetString("plugin_configs.apps.updates"))
}

func TestConfigRemoveProfileRequiresConfirmation(t *testing.T) {
	cc, profilesFile := setupRemoveProfileTest(t, removeProfileTestConfig)
	cc.autoConfirm = false
	before := readFile(t, profilesFile)

	var out bytes.Buffer
	cc.cmd.SetOut(&out)
	cc.cmd.SetIn(strings.NewReader("no\n"))
	cc.cmd.SetArgs([]string{"--remove-profile", "bbq.sandbox"})
	require.NoError(t, cc.cmd.Execute())

	assert.Contains(t, out.String(), "Aborted. No changes were made.")
	assert.Equal(t, before, readFile(t, profilesFile))
}

func TestConfigRemoveProfileProceedsOnYes(t *testing.T) {
	cc, profilesFile := setupRemoveProfileTest(t, removeProfileTestConfig)
	cc.autoConfirm = false

	var out bytes.Buffer
	cc.cmd.SetOut(&out)
	cc.cmd.SetIn(strings.NewReader("yes\n"))
	cc.cmd.SetArgs([]string{"--remove-profile", "bbq.sandbox"})
	require.NoError(t, cc.cmd.Execute())

	v := readBack(t, profilesFile)
	assert.False(t, v.IsSet("bbq.sandbox.display_name"))
}

// With no terminal to prompt at, removal must refuse rather than assume yes.
// os.Stdin under `go test` is not a terminal, which is the same condition as a
// piped or CI invocation.
func TestConfigRemoveProfileRefusesWhenNonInteractive(t *testing.T) {
	cc, profilesFile := setupRemoveProfileTest(t, removeProfileTestConfig)
	cc.autoConfirm = false
	before := readFile(t, profilesFile)

	var out bytes.Buffer
	cc.cmd.SetOut(&out)
	cc.cmd.SetIn(os.Stdin)
	cc.cmd.SetArgs([]string{"--remove-profile", "bbq.sandbox"})

	err := cc.cmd.Execute()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "re-run with --confirm")

	category, ok := errorcategory.Get(err)
	require.True(t, ok)
	assert.Equal(t, errorcategory.UserInput, category)

	assert.Equal(t, before, readFile(t, profilesFile))
}

func readFile(t *testing.T, name string) []byte {
	t.Helper()

	data, err := os.ReadFile(name)
	require.NoError(t, err)

	return data
}
