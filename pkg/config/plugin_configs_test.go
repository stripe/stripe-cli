package config

import (
	"testing"

	"github.com/spf13/viper"
	"github.com/stretchr/testify/require"
)

func TestPluginUpdatesEnabledPrecedence(t *testing.T) {
	const plugin = "apps"

	tests := []struct {
		name         string
		pluginValue  interface{}
		globalValue  interface{}
		wantForApps  bool
		wantForOther bool
	}{
		{
			name:         "unset defaults to off",
			wantForApps:  false,
			wantForOther: false,
		},
		{
			name:         "global on applies to every plugin",
			globalValue:  PluginConfigOn,
			wantForApps:  true,
			wantForOther: true,
		},
		{
			name:         "global off applies to every plugin",
			globalValue:  PluginConfigOff,
			wantForApps:  false,
			wantForOther: false,
		},
		{
			name:         "plugin on with no global setting",
			pluginValue:  PluginConfigOn,
			wantForApps:  true,
			wantForOther: false,
		},
		{
			name:         "plugin off overrides global on",
			pluginValue:  PluginConfigOff,
			globalValue:  PluginConfigOn,
			wantForApps:  false,
			wantForOther: true,
		},
		{
			name:         "plugin on overrides global off",
			pluginValue:  PluginConfigOn,
			globalValue:  PluginConfigOff,
			wantForApps:  true,
			wantForOther: false,
		},
		{
			// A value we cannot read as "on" leaves updates off, and still keeps the
			// global setting from answering for a plugin the user configured.
			name:         "unrecognized plugin value does not fall through to global on",
			pluginValue:  "yes",
			globalValue:  PluginConfigOn,
			wantForApps:  false,
			wantForOther: true,
		},
		{
			// Hand-editing the config file is how a non-string value gets here.
			name:         "non-string plugin value is not on",
			pluginValue:  true,
			wantForApps:  false,
			wantForOther: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			v := viper.New()
			if tt.pluginValue != nil {
				v.Set(PluginConfigKey(plugin, PluginConfigUpdatesField), tt.pluginValue)
			}
			if tt.globalValue != nil {
				v.Set(PluginConfigKey(PluginConfigGlobalScope, PluginConfigUpdatesField), tt.globalValue)
			}

			require.Equal(t, tt.wantForApps, pluginUpdatesEnabled(v, plugin))
			require.Equal(t, tt.wantForOther, pluginUpdatesEnabled(v, "projects"))
		})
	}
}

// An empty plugin name would build the key "plugin_configs..updates", so it has
// to be dropped rather than looked up. Callers should always name a plugin; this
// pins the behavior if one ever doesn't.
func TestPluginUpdatesEnabledIgnoresEmptyPluginName(t *testing.T) {
	v := viper.New()
	v.Set(PluginConfigKey(PluginConfigGlobalScope, PluginConfigUpdatesField), PluginConfigOn)

	require.True(t, pluginUpdatesEnabled(v, ""))

	v.Set(PluginConfigKey(PluginConfigGlobalScope, PluginConfigUpdatesField), PluginConfigOff)

	require.False(t, pluginUpdatesEnabled(v, ""))
}

// The setting is written by `stripe plugin auto-update` and read here, from two
// different packages. Go through the real write path to prove they agree.
func TestPluginUpdatesEnabledReadsWhatAutoUpdateWrites(t *testing.T) {
	c, _, cleanup := setupTestConfig(t)
	defer cleanup()

	require.False(t, PluginUpdatesEnabled("apps"))

	require.NoError(t, c.WriteConfigField(PluginConfigKey(PluginConfigGlobalScope, PluginConfigUpdatesField), PluginConfigOn))
	require.True(t, PluginUpdatesEnabled("apps"))

	require.NoError(t, c.WriteConfigField(PluginConfigKey("apps", PluginConfigUpdatesField), PluginConfigOff))
	require.False(t, PluginUpdatesEnabled("apps"))
	require.True(t, PluginUpdatesEnabled("projects"))
}
