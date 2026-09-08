package plugins

import (
	"context"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/stripe/stripe-cli/pkg/errorcategory"
	"github.com/stripe/stripe-cli/pkg/stripe"
)

type recordedTelemetryEvent struct {
	name          string
	value         string
	pluginName    string
	pluginVersion string
}

type recordingTelemetryClient struct {
	events []recordedTelemetryEvent
}

func (c *recordingTelemetryClient) SendAPIRequestEvent(context.Context, string, bool) (*http.Response, error) {
	return nil, nil
}

func (c *recordingTelemetryClient) SendEvent(ctx context.Context, name, value string) {
	event := recordedTelemetryEvent{name: name, value: value}
	if metadata := stripe.GetEventMetadata(ctx); metadata != nil {
		event.pluginName = metadata.PluginName
		event.pluginVersion = metadata.PluginVersion
	}
	c.events = append(c.events, event)
}

func TestSendPluginCommandFinishedSuccess(t *testing.T) {
	client := &recordingTelemetryClient{}
	ctx := stripe.WithTelemetryClient(context.Background(), client)

	sendPluginCommandFinished(ctx, "projects", "1.2.3", 1500*time.Millisecond, nil)

	require.Equal(t, []recordedTelemetryEvent{{
		name:  pluginCommandFinishedEventName,
		value: `{"plugin_name":"projects","plugin_version":"1.2.3","outcome":"success","duration_ms":1500}`,
	}}, client.events)
}

func TestSendPluginCommandFinishedError(t *testing.T) {
	client := &recordingTelemetryClient{}
	ctx := stripe.WithTelemetryClient(context.Background(), client)

	sendPluginCommandFinished(
		ctx,
		"projects",
		"1.2.3",
		250*time.Millisecond,
		errorcategory.New(errorcategory.Auth, "sensitive error"),
	)

	require.Equal(t, []recordedTelemetryEvent{{
		name:  pluginCommandFinishedEventName,
		value: `{"plugin_name":"projects","plugin_version":"1.2.3","outcome":"error","duration_ms":250,"failure_category":"auth"}`,
	}}, client.events)
	require.NotContains(t, client.events[0].value, "sensitive error")
}

func TestSendPluginCommandFinishedDoesNothingWithoutTelemetryClient(t *testing.T) {
	require.NotPanics(t, func() {
		sendPluginCommandFinished(context.Background(), "projects", "1.2.3", time.Second, nil)
	})
}

func TestWithPluginTelemetryMetadataDoesNotModifyParentContext(t *testing.T) {
	metadata := stripe.NewEventMetadata()
	metadata.SetPluginName("caller")
	metadata.SetPluginVersion("2.0.0")
	ctx := stripe.WithEventMetadata(context.Background(), metadata)

	pluginCtx := withPluginTelemetryMetadata(ctx, "projects", "1.2.3")
	require.Equal(t, "caller", metadata.PluginName)
	require.Equal(t, "2.0.0", metadata.PluginVersion)
	require.Equal(t, "projects", stripe.GetEventMetadata(pluginCtx).PluginName)
	require.Equal(t, "1.2.3", stripe.GetEventMetadata(pluginCtx).PluginVersion)
}
