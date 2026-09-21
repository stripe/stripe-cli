package plugins

import (
	"context"
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/stripe/stripe-cli/pkg/stripe"
)

type recordedTelemetryEvent struct {
	name  string
	value string
}

type recordingTelemetryClient struct {
	events []recordedTelemetryEvent
}

func (c *recordingTelemetryClient) SendAPIRequestEvent(context.Context, string, bool) (*http.Response, error) {
	return nil, nil
}

func (c *recordingTelemetryClient) SendEvent(_ context.Context, name, value string) {
	c.events = append(c.events, recordedTelemetryEvent{name: name, value: value})
}

func TestSendPluginLifecycleEventReportsEventAndStampsMetadata(t *testing.T) {
	client := &recordingTelemetryClient{}
	metadata := &stripe.CLIAnalyticsEventMetadata{}
	ctx := stripe.WithEventMetadata(stripe.WithTelemetryClient(context.Background(), client), metadata)

	SendPluginLifecycleEvent(ctx, PluginUpgradedEvent, "1.2.3")

	require.Equal(t, []recordedTelemetryEvent{{name: "Plugin Upgraded", value: "1.2.3"}}, client.events)
	// Stamped on the metadata as well as sent, so the rest of the invocation's
	// telemetry reports the version too.
	require.Equal(t, "1.2.3", metadata.PluginVersion)
}

// Telemetry can be disabled, and plugins are reachable from paths that never
// installed a client, so a missing client has to be a no-op rather than a panic.
func TestSendPluginLifecycleEventWithoutTelemetryClient(t *testing.T) {
	metadata := &stripe.CLIAnalyticsEventMetadata{}
	ctx := stripe.WithEventMetadata(context.Background(), metadata)

	SendPluginLifecycleEvent(ctx, PluginInstalledEvent, "1.2.3")

	// Nothing was sent, so nothing should claim a version was reported either.
	require.Empty(t, metadata.PluginVersion)
}

// A context with a client but no event metadata is what the gRPC and provision
// paths can hand over; the event still has to go out.
func TestSendPluginLifecycleEventWithoutEventMetadata(t *testing.T) {
	client := &recordingTelemetryClient{}
	ctx := stripe.WithTelemetryClient(context.Background(), client)

	SendPluginLifecycleEvent(ctx, PluginUninstalledEvent, "1.2.3")

	require.Equal(t, []recordedTelemetryEvent{{name: "Plugin Uninstalled", value: "1.2.3"}}, client.events)
}
