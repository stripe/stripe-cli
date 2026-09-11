package config

import "github.com/spf13/viper"

const (
	// PluginConfigGlobalScope is used as the scope when a setting applies to all plugins.
	PluginConfigGlobalScope = "__global"

	// PluginConfigUpdatesField is the config field name controlling automatic updates.
	PluginConfigUpdatesField = "updates"

	// PluginConfigsKey is the top-level table holding every plugin config
	// section. It is a CLI setting, not a profile.
	PluginConfigsKey = "plugin_configs"

	// PluginConfigOn and PluginConfigOff are the values a plugin config toggle
	// holds. They are named because the writer and the reader live in different
	// packages and have to agree on the exact string.
	PluginConfigOn  = "on"
	PluginConfigOff = "off"
)

// isPluginConfigSection reports whether v is a plugin config section,
// i.e. a map of the form <scope>.<plugin config key>.
func isPluginConfigSection(v interface{}) bool {
	m, ok := v.(map[string]interface{})
	if !ok {
		return false
	}
	_, ok = m[PluginConfigUpdatesField]
	return ok
}

// PluginConfigKey returns the key for a plugin config field.
// Use PluginConfigGlobalScope as scope to target all plugins.
// Use the plugin name as scope to target a specific plugin.
// Example: PluginConfigKey("__global", "updates") to read or set the global updates setting
// Example: PluginConfigKey("apps", "updates") to read or set the updates setting for the "apps" plugin
func PluginConfigKey(scope, field string) string {
	return PluginConfigsKey + "." + scope + "." + field
}

// PluginUpdatesEnabled reports whether automatic updates are enabled for the
// named plugin, per `stripe plugin auto-update`.
//
// Updates are off unless the user turned them on. Nothing should download a new
// plugin version on its own just because no one said otherwise.
func PluginUpdatesEnabled(pluginName string) bool {
	return pluginUpdatesEnabled(viper.GetViper(), pluginName)
}

func pluginUpdatesEnabled(v *viper.Viper, pluginName string) bool {
	// A setting on the plugin itself answers alone, so `--disable` on one plugin
	// holds under a global `--enable` and the reverse. A value that is neither "on"
	// nor "off" counts as set: the user configured this plugin, which is enough to
	// keep the global setting from speaking for it, and anything unreadable as "on"
	// leaves updates off.
	if pluginName != "" {
		if enabled, isSet := pluginConfigToggle(v, pluginName, PluginConfigUpdatesField); isSet {
			return enabled
		}
	}

	enabled, _ := pluginConfigToggle(v, PluginConfigGlobalScope, PluginConfigUpdatesField)
	return enabled
}

// pluginConfigToggle reads one on/off plugin config field. isSet distinguishes a
// field the user turned off from one they never touched, which is what lets a
// per-plugin setting take precedence over the global one.
func pluginConfigToggle(v *viper.Viper, scope, field string) (enabled, isSet bool) {
	key := PluginConfigKey(scope, field)
	if !v.IsSet(key) {
		return false, false
	}

	return v.GetString(key) == PluginConfigOn, true
}
