package plugin

import (
	"bytes"
	"io"
	"path/filepath"
	"testing"

	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/stripe/stripe-cli/pkg/config"
	"github.com/stripe/stripe-cli/pkg/keyring"
)

func setupAutoUpdateTest(t *testing.T) (*config.Config, func()) {
	t.Helper()

	profilesFile := filepath.Join(t.TempDir(), "config.toml")
	cfg := &config.Config{
		Color:        "auto",
		LogLevel:     "info",
		ProfilesFile: profilesFile,
		Profile:      config.Profile{ProfileName: "default"},
	}
	cfg.InitConfig()
	config.KeyRing = keyring.NewMemoryStore(nil)

	return cfg, func() {
		viper.Reset()
	}
}

// -- global --enable --------------------------------------------------------

func TestGlobalEnable(t *testing.T) {
	cfg, cleanup := setupAutoUpdateTest(t)
	defer cleanup()

	ac := NewAutoUpdateCmd(cfg)
	ac.enable = true
	var output bytes.Buffer
	ac.Cmd.SetOut(&output)

	err := ac.run(ac.Cmd, []string{})
	require.NoError(t, err)
	assert.Equal(t, "on", viper.GetString(config.PluginConfigKey(config.PluginConfigGlobalScope, config.PluginConfigUpdatesField)))
	assert.Equal(t, "Automatic updates are enabled for all plugins\n\nDisable them with 'stripe plugin auto-update --disable'\nFollow each plugin's default with 'stripe plugin auto-update --unset'\n", output.String())
}

// A plugin's own choice answers before the global one, so changing the global
// setting has to say which plugins it does not reach.
func TestGlobalEnableListsPluginOverrides(t *testing.T) {
	cfg, cleanup := setupAutoUpdateTest(t)
	defer cleanup()

	require.NoError(t, cfg.WriteConfigField("installed_plugins", []string{"apps", "projects", "tools"}))
	require.NoError(t, cfg.WriteConfigField(config.PluginConfigKey("apps", config.PluginConfigUpdatesField), config.PluginConfigOff))
	require.NoError(t, cfg.WriteConfigField(config.PluginConfigKey("tools", config.PluginConfigUpdatesField), config.PluginConfigOn))

	ac := NewAutoUpdateCmd(cfg)
	ac.enable = true
	var output bytes.Buffer
	ac.Cmd.SetOut(&output)

	err := ac.run(ac.Cmd, []string{})
	require.NoError(t, err)
	assert.Equal(t, "Automatic updates are enabled for all plugins\n\nDisable them with 'stripe plugin auto-update --disable'\nFollow each plugin's default with 'stripe plugin auto-update --unset'\n\nNote: these plugins keep their own settings and are unaffected:\n  apps: disabled ('stripe plugin auto-update apps --unset' to clear)\n  tools: enabled ('stripe plugin auto-update tools --unset' to clear)\n", output.String())
}

// -- global --disable -------------------------------------------------------

func TestGlobalDisable(t *testing.T) {
	cfg, cleanup := setupAutoUpdateTest(t)
	defer cleanup()

	ac := NewAutoUpdateCmd(cfg)
	ac.disable = true
	var output bytes.Buffer
	ac.Cmd.SetOut(&output)

	err := ac.run(ac.Cmd, []string{})
	require.NoError(t, err)
	assert.Equal(t, "off", viper.GetString(config.PluginConfigKey(config.PluginConfigGlobalScope, config.PluginConfigUpdatesField)))
	assert.Equal(t, "Automatic updates are disabled for all plugins\n\nEnable them with 'stripe plugin auto-update --enable'\nFollow each plugin's default with 'stripe plugin auto-update --unset'\n", output.String())
}

// -- global --unset ---------------------------------------------------------

func TestGlobalUnset(t *testing.T) {
	cfg, cleanup := setupAutoUpdateTest(t)
	defer cleanup()

	require.NoError(t, cfg.WriteConfigField("installed_plugins", []string{"apps"}))
	require.NoError(t, cfg.WriteConfigField(config.PluginConfigKey(config.PluginConfigGlobalScope, config.PluginConfigUpdatesField), config.PluginConfigOn))
	require.NoError(t, cfg.WriteConfigField(config.PluginConfigKey("apps", config.PluginConfigUpdatesField), config.PluginConfigOff))

	ac := NewAutoUpdateCmd(cfg)
	ac.unset = true
	var output bytes.Buffer
	ac.Cmd.SetOut(&output)

	err := ac.run(ac.Cmd, []string{})
	require.NoError(t, err)
	assert.False(t, viper.IsSet(config.PluginConfigKey(config.PluginConfigGlobalScope, config.PluginConfigUpdatesField)))
	// Only the global choice is cleared; a per-plugin choice still answers for its plugin.
	assert.Equal(t, "off", viper.GetString(config.PluginConfigKey("apps", config.PluginConfigUpdatesField)))
	assert.Equal(t, "Automatic updates now follow each plugin's default\n\nEnable them with 'stripe plugin auto-update --enable'\nDisable them with 'stripe plugin auto-update --disable'\n\nNote: this plugin keeps its own setting and is unaffected:\n  apps: disabled ('stripe plugin auto-update apps --unset' to clear)\n", output.String())
}

// -- no flags → help --------------------------------------------------------

func TestNoFlags_ShowsHelp(t *testing.T) {
	cfg, cleanup := setupAutoUpdateTest(t)
	defer cleanup()

	ac := NewAutoUpdateCmd(cfg)

	err := ac.run(ac.Cmd, []string{})
	require.NoError(t, err)
	assert.False(t, viper.IsSet(config.PluginConfigKey(config.PluginConfigGlobalScope, config.PluginConfigUpdatesField)))
}

// -- per-plugin --enable ----------------------------------------------------

func TestPluginEnable(t *testing.T) {
	cfg, cleanup := setupAutoUpdateTest(t)
	defer cleanup()

	require.NoError(t, cfg.WriteConfigField("installed_plugins", []string{"apps"}))
	require.NoError(t, cfg.WriteConfigField(config.PluginConfigKey(config.PluginConfigGlobalScope, config.PluginConfigUpdatesField), config.PluginConfigOn))

	ac := NewAutoUpdateCmd(cfg)
	ac.enable = true
	var output bytes.Buffer
	ac.Cmd.SetOut(&output)

	err := ac.run(ac.Cmd, []string{"apps"})
	require.NoError(t, err)
	assert.Equal(t, "on", viper.GetString(config.PluginConfigKey("apps", config.PluginConfigUpdatesField)))
	assert.Equal(t, "Automatic updates are enabled for the Apps plugin\n\nDisable it with 'stripe plugin auto-update apps --disable'\nFollow the global setting with 'stripe plugin auto-update apps --unset' (current: enabled)\n", output.String())
}

// -- per-plugin --disable ---------------------------------------------------

func TestPluginDisable(t *testing.T) {
	cfg, cleanup := setupAutoUpdateTest(t)
	defer cleanup()

	require.NoError(t, cfg.WriteConfigField("installed_plugins", []string{"apps"}))

	ac := NewAutoUpdateCmd(cfg)
	ac.disable = true
	var output bytes.Buffer
	ac.Cmd.SetOut(&output)

	err := ac.run(ac.Cmd, []string{"apps"})
	require.NoError(t, err)
	assert.Equal(t, "off", viper.GetString(config.PluginConfigKey("apps", config.PluginConfigUpdatesField)))
	assert.Equal(t, "Automatic updates are disabled for the Apps plugin\n\nEnable it with 'stripe plugin auto-update apps --enable'\nFollow the global setting with 'stripe plugin auto-update apps --unset' (current: plugin default)\n", output.String())
}

// -- per-plugin --unset -----------------------------------------------------

func TestPluginUnset(t *testing.T) {
	cfg, cleanup := setupAutoUpdateTest(t)
	defer cleanup()

	require.NoError(t, cfg.WriteConfigField("installed_plugins", []string{"apps"}))
	require.NoError(t, cfg.WriteConfigField(config.PluginConfigKey(config.PluginConfigGlobalScope, config.PluginConfigUpdatesField), config.PluginConfigOn))
	require.NoError(t, cfg.WriteConfigField(config.PluginConfigKey("apps", config.PluginConfigUpdatesField), config.PluginConfigOff))

	ac := NewAutoUpdateCmd(cfg)
	var output bytes.Buffer
	ac.Cmd.SetOut(&output)
	// Parsed rather than set on the struct: the per-plugin output advertises this exact
	// command line, so it has to be one the CLI accepts.
	ac.Cmd.SetArgs([]string{"apps", "--unset"})

	require.NoError(t, ac.Cmd.Execute())
	assert.False(t, viper.IsSet(config.PluginConfigKey("apps", config.PluginConfigUpdatesField)))
	assert.Equal(t, "on", viper.GetString(config.PluginConfigKey(config.PluginConfigGlobalScope, config.PluginConfigUpdatesField)))
	assert.Equal(t, "Automatic updates for the Apps plugin now follow the global setting (current: enabled)\n\nEnable it with 'stripe plugin auto-update apps --enable'\nDisable it with 'stripe plugin auto-update apps --disable'\n", output.String())
}

// Clearing a choice that was never made is already the state the user asked for.
func TestPluginUnsetWithNothingSet(t *testing.T) {
	cfg, cleanup := setupAutoUpdateTest(t)
	defer cleanup()

	require.NoError(t, cfg.WriteConfigField("installed_plugins", []string{"apps"}))

	ac := NewAutoUpdateCmd(cfg)
	ac.unset = true
	var output bytes.Buffer
	ac.Cmd.SetOut(&output)

	err := ac.run(ac.Cmd, []string{"apps"})
	require.NoError(t, err)
	assert.False(t, viper.IsSet(config.PluginConfigKey("apps", config.PluginConfigUpdatesField)))
	assert.Equal(t, "Automatic updates for the Apps plugin now follow the global setting (current: plugin default)\n\nEnable it with 'stripe plugin auto-update apps --enable'\nDisable it with 'stripe plugin auto-update apps --disable'\n", output.String())
}

func TestUnsetExcludesEnableAndDisable(t *testing.T) {
	for _, other := range []string{"--enable", "--disable"} {
		t.Run(other, func(t *testing.T) {
			cfg, cleanup := setupAutoUpdateTest(t)
			defer cleanup()

			ac := NewAutoUpdateCmd(cfg)
			ac.Cmd.SetOut(io.Discard)
			ac.Cmd.SetErr(io.Discard)
			ac.Cmd.SetArgs([]string{"--unset", other})

			require.ErrorContains(t, ac.Cmd.Execute(), "none of the others can be")
			assert.False(t, viper.IsSet(config.PluginConfigKey(config.PluginConfigGlobalScope, config.PluginConfigUpdatesField)))
		})
	}
}

// -- per-plugin not installed -----------------------------------------------

func TestPluginNotInstalled(t *testing.T) {
	cfg, cleanup := setupAutoUpdateTest(t)
	defer cleanup()

	ac := NewAutoUpdateCmd(cfg)
	ac.enable = true

	err := ac.run(ac.Cmd, []string{"apps"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not installed")
}
