package plugins

import (
	"context"

	"github.com/stripe/stripe-cli/pkg/stripe"
)

// Event names reported for a plugin's lifecycle. They are named constants because
// more than one package reports them now, and a typo in one caller would quietly
// split a single event in two rather than fail.
const (
	PluginInstalledEvent   = "Plugin Installed"
	PluginUpgradedEvent    = "Plugin Upgraded"
	PluginUninstalledEvent = "Plugin Uninstalled"
)

// SendPluginLifecycleEvent sends a telemetry event for a plugin install, upgrade, or
// uninstall, and stamps the version onto the invocation's event metadata so the rest
// of the command's telemetry carries it too.
//
// A context without a telemetry client is normal rather than an error: telemetry can
// be disabled, and plugins are reachable from paths that never installed a client.
func SendPluginLifecycleEvent(ctx context.Context, eventName, pluginVersion string) {
	telemetryClient := stripe.GetTelemetryClient(ctx)
	if telemetryClient == nil {
		return
	}
	if m := stripe.GetEventMetadata(ctx); m != nil {
		m.SetPluginVersion(pluginVersion)
	}
	telemetryClient.SendEvent(ctx, eventName, pluginVersion)
}
