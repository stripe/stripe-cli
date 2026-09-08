package plugins

import (
	"context"
	"encoding/json"
	"time"

	"github.com/stripe/stripe-cli/pkg/errorcategory"
	"github.com/stripe/stripe-cli/pkg/stripe"
)

const pluginCommandFinishedEventName = "Plugin command finished"

type pluginCommandFinishedEvent struct {
	PluginName      string `json:"plugin_name"`
	PluginVersion   string `json:"plugin_version,omitempty"`
	Outcome         string `json:"outcome"`
	DurationMS      int64  `json:"duration_ms"`
	FailureCategory string `json:"failure_category,omitempty"`
}

func withPluginTelemetryMetadata(ctx context.Context, pluginName, pluginVersion string) context.Context {
	metadata := stripe.GetEventMetadata(ctx)
	if metadata == nil {
		return ctx
	}
	pluginMetadata := *metadata
	pluginMetadata.SetPluginName(pluginName)
	pluginMetadata.SetPluginVersion(pluginVersion)
	return stripe.WithEventMetadata(ctx, &pluginMetadata)
}

func sendPluginCommandFinished(
	ctx context.Context,
	pluginName string,
	pluginVersion string,
	duration time.Duration,
	commandErr error,
) {
	telemetryClient := stripe.GetTelemetryClient(ctx)
	if telemetryClient == nil {
		return
	}

	event := pluginCommandFinishedEvent{
		PluginName:    pluginName,
		PluginVersion: pluginVersion,
		Outcome:       "success",
		DurationMS:    duration.Milliseconds(),
	}
	if commandErr != nil {
		event.Outcome = "error"
		if category, ok := errorcategory.Get(commandErr); ok {
			event.FailureCategory = string(category)
		} else {
			event.FailureCategory = "unknown"
		}
	}

	eventValue, err := json.Marshal(event)
	if err != nil {
		return
	}
	telemetryClient.SendEvent(ctx, pluginCommandFinishedEventName, string(eventValue))
}
