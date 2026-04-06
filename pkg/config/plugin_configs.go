package config

const (
	// PluginConfigGlobalScope is used as the scope when a setting applies to all plugins.
	PluginConfigGlobalScope = "__global"

	// PluginConfigUpdatesField is the config field name controlling automatic updates.
	PluginConfigUpdatesField = "updates"
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
	return "plugin_configs." + scope + "." + field
}
