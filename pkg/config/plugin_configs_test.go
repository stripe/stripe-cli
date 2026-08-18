package config

import (
	"testing"

	"github.com/spf13/viper"
	"github.com/stretchr/testify/require"
)

func TestPluginAutoUpdateEnabled(t *testing.T) {
	viper.Reset()
	t.Cleanup(viper.Reset)

	cfg := &Config{}
	globalKey := PluginConfigKey(PluginConfigGlobalScope, PluginConfigUpdatesField)
	pluginKey := PluginConfigKey("apps", PluginConfigUpdatesField)

	require.False(t, cfg.PluginAutoUpdateEnabled("apps"), "updates should default to disabled")

	viper.Set(globalKey, "on")
	require.True(t, cfg.PluginAutoUpdateEnabled("apps"), "global enable should apply")

	viper.Set(pluginKey, "off")
	require.False(t, cfg.PluginAutoUpdateEnabled("apps"), "plugin disable should override global enable")

	viper.Set(globalKey, "off")
	viper.Set(pluginKey, "on")
	require.True(t, cfg.PluginAutoUpdateEnabled("apps"), "plugin enable should override global disable")
}
