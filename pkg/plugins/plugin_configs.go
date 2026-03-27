package plugins

const (
	// PluginConfigGlobalScope is used as the scope when a setting applies to all plugins.
	PluginConfigGlobalScope = "__global"

	// PluginConfigUpdatesField is the config field name controlling automatic updates.
	PluginConfigUpdatesField = "updates"
)

// PluginConfigKey returns the key for a plugin config field.
// Use PluginConfigGlobalScope as scope to target all plugins.
// Use the plugin name as scope to target a specific plugin.
// Example: PluginConfigKey("__global", "updates") to read or set the global updates setting
// Example: PluginConfigKey("apps", "updates") to read or set the updates setting for the "apps" plugin
func PluginConfigKey(scope, field string) string {
	return "plugin_configs." + scope + "." + field
}
