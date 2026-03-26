package plugin

import (
	"path/filepath"
	"testing"

	"github.com/99designs/keyring"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/stripe/stripe-cli/pkg/config"
)

func setupPluginConfigTest(t *testing.T) (*config.Config, func()) {
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

func newTestConfigCmd(cfg *config.Config) *ConfigCmd {
	cc := NewConfigCmd(cfg)
	return cc
}

// -- global --set -----------------------------------------------------------

func TestGlobalSet_UpdatesOn(t *testing.T) {
	cfg, cleanup := setupPluginConfigTest(t)
	defer cleanup()

	cc := newTestConfigCmd(cfg)
	cc.set = true

	err := cc.run(cc.Cmd, []string{"updates", "on"})
	require.NoError(t, err)
	assert.Equal(t, "on", viper.GetString("plugin_configs.__global.updates"))
}

func TestGlobalSet_UpdatesOff(t *testing.T) {
	cfg, cleanup := setupPluginConfigTest(t)
	defer cleanup()

	cc := newTestConfigCmd(cfg)
	cc.set = true

	err := cc.run(cc.Cmd, []string{"updates", "off"})
	require.NoError(t, err)
	assert.Equal(t, "off", viper.GetString("plugin_configs.__global.updates"))
}

func TestGlobalSet_InvalidValue(t *testing.T) {
	cfg, cleanup := setupPluginConfigTest(t)
	defer cleanup()

	cc := newTestConfigCmd(cfg)
	cc.set = true

	err := cc.run(cc.Cmd, []string{"updates", "maybe"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid value")
}

func TestGlobalSet_UnknownField(t *testing.T) {
	cfg, cleanup := setupPluginConfigTest(t)
	defer cleanup()

	cc := newTestConfigCmd(cfg)
	cc.set = true

	err := cc.run(cc.Cmd, []string{"unknown_field", "on"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unknown config field")
}

// -- global --unset ---------------------------------------------------------

func TestGlobalUnset_Updates(t *testing.T) {
	cfg, cleanup := setupPluginConfigTest(t)
	defer cleanup()

	require.NoError(t, cfg.WriteConfigField("plugin_configs.__global.updates", "off"))

	cc := newTestConfigCmd(cfg)
	cc.unset = "updates"

	err := cc.run(cc.Cmd, []string{})
	require.NoError(t, err)
	assert.False(t, viper.IsSet("plugin_configs.__global.updates"))
}

// -- per-plugin --set -------------------------------------------------------

func TestPluginSet_UpdatesOff(t *testing.T) {
	cfg, cleanup := setupPluginConfigTest(t)
	defer cleanup()

	require.NoError(t, cfg.WriteConfigField("installed_plugins", []string{"apps"}))

	cc := newTestConfigCmd(cfg)
	cc.set = true

	err := cc.run(cc.Cmd, []string{"apps", "updates", "off"})
	require.NoError(t, err)
	assert.Equal(t, "off", viper.GetString("plugin_configs.apps.updates"))
}

func TestPluginSet_NotInstalled(t *testing.T) {
	cfg, cleanup := setupPluginConfigTest(t)
	defer cleanup()

	cc := newTestConfigCmd(cfg)
	cc.set = true

	err := cc.run(cc.Cmd, []string{"apps", "updates", "off"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not installed")
}

func TestPluginSet_InvalidValue(t *testing.T) {
	cfg, cleanup := setupPluginConfigTest(t)
	defer cleanup()

	require.NoError(t, cfg.WriteConfigField("installed_plugins", []string{"apps"}))

	cc := newTestConfigCmd(cfg)
	cc.set = true

	err := cc.run(cc.Cmd, []string{"apps", "updates", "maybe"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid value")
}

// -- per-plugin --unset -----------------------------------------------------

func TestPluginUnset_Updates(t *testing.T) {
	cfg, cleanup := setupPluginConfigTest(t)
	defer cleanup()

	require.NoError(t, cfg.WriteConfigField("installed_plugins", []string{"apps"}))
	require.NoError(t, cfg.WriteConfigField("plugin_configs.apps.updates", "off"))

	cc := newTestConfigCmd(cfg)
	cc.unset = "updates"

	err := cc.run(cc.Cmd, []string{"apps"})
	require.NoError(t, err)
	assert.False(t, viper.IsSet("plugin_configs.apps.updates"))
}

func TestPluginUnset_NotInstalled(t *testing.T) {
	cfg, cleanup := setupPluginConfigTest(t)
	defer cleanup()

	cc := newTestConfigCmd(cfg)
	cc.unset = "updates"

	err := cc.run(cc.Cmd, []string{"apps"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not installed")
}
