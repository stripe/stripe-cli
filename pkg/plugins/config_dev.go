//go:build localdev
// +build localdev

package plugins

func init() {
	PluginDev = true
	PluginsPath = ""
}
