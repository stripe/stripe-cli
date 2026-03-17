package plugins

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/stripe/stripe-cli/pkg/stripe"
)

func TestEcho(t *testing.T) {
	ctx := context.Background()
	coreCLIHelper := NewCoreCLIHelper(ctx)
	output, err := coreCLIHelper.Echo("test")
	require.NoError(t, err)
	require.Equal(t, "test", output)
}

func TestSendAnalytics(t *testing.T) {
	// Test with no telemetry client in context (should not error)
	ctx := context.Background()
	coreCLIHelper := NewCoreCLIHelper(ctx)
	err := coreCLIHelper.SendAnalytics("test_event", "test_value")
	require.NoError(t, err)
}

func TestSendAnalyticsWithTelemetryClient(t *testing.T) {
	// Test with a NoOp telemetry client
	ctx := context.Background()
	telemetryClient := &stripe.NoOpTelemetryClient{}
	ctx = stripe.WithTelemetryClient(ctx, telemetryClient)

	coreCLIHelper := NewCoreCLIHelper(ctx)
	err := coreCLIHelper.SendAnalytics("test_event", "test_value")
	require.NoError(t, err)
}
