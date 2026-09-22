package plugin

import (
	"bytes"
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
	assert.Equal(t, "Automatic updates are enabled for all plugins\n\nDisable them with 'stripe plugin auto-update --disable'\n", output.String())
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
	assert.Equal(t, "Automatic updates are disabled for all plugins\n\nEnable them with 'stripe plugin auto-update --enable'\n", output.String())
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
	assert.Equal(t, "Automatic updates are disabled for the Apps plugin\n\nEnable it with 'stripe plugin auto-update apps --enable'\nFollow the global setting with 'stripe plugin auto-update apps --unset' (current: disabled)\n", output.String())
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
