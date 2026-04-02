package plugin

import (
	"path/filepath"
	"testing"

	"github.com/99designs/keyring"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/stripe/stripe-cli/pkg/config"
	"github.com/stripe/stripe-cli/pkg/plugins"
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
	config.KeyRing = keyring.NewArrayKeyring(nil)

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

	err := ac.run(ac.Cmd, []string{})
	require.NoError(t, err)
	assert.Equal(t, "on", viper.GetString(plugins.PluginConfigKey(plugins.PluginConfigGlobalScope, plugins.PluginConfigUpdatesField)))
}

// -- global --disable -------------------------------------------------------

func TestGlobalDisable(t *testing.T) {
	cfg, cleanup := setupAutoUpdateTest(t)
	defer cleanup()

	ac := NewAutoUpdateCmd(cfg)
	ac.disable = true

	err := ac.run(ac.Cmd, []string{})
	require.NoError(t, err)
	assert.Equal(t, "off", viper.GetString(plugins.PluginConfigKey(plugins.PluginConfigGlobalScope, plugins.PluginConfigUpdatesField)))
}

// -- no flags → help --------------------------------------------------------

func TestNoFlags_ShowsHelp(t *testing.T) {
	cfg, cleanup := setupAutoUpdateTest(t)
	defer cleanup()

	ac := NewAutoUpdateCmd(cfg)

	err := ac.run(ac.Cmd, []string{})
	require.NoError(t, err)
	assert.False(t, viper.IsSet(plugins.PluginConfigKey(plugins.PluginConfigGlobalScope, plugins.PluginConfigUpdatesField)))
}

// -- per-plugin --enable ----------------------------------------------------

func TestPluginEnable(t *testing.T) {
	cfg, cleanup := setupAutoUpdateTest(t)
	defer cleanup()

	require.NoError(t, cfg.WriteConfigField("installed_plugins", []string{"apps"}))

	ac := NewAutoUpdateCmd(cfg)
	ac.enable = true

	err := ac.run(ac.Cmd, []string{"apps"})
	require.NoError(t, err)
	assert.Equal(t, "on", viper.GetString(plugins.PluginConfigKey("apps", plugins.PluginConfigUpdatesField)))
}

// -- per-plugin --disable ---------------------------------------------------

func TestPluginDisable(t *testing.T) {
	cfg, cleanup := setupAutoUpdateTest(t)
	defer cleanup()

	require.NoError(t, cfg.WriteConfigField("installed_plugins", []string{"apps"}))

	ac := NewAutoUpdateCmd(cfg)
	ac.disable = true

	err := ac.run(ac.Cmd, []string{"apps"})
	require.NoError(t, err)
	assert.Equal(t, "off", viper.GetString(plugins.PluginConfigKey("apps", plugins.PluginConfigUpdatesField)))
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
