package plugin

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/stripe/stripe-cli/pkg/stripe"
)

func TestParseArg(t *testing.T) {
	// No version
	plugin, version := parseInstallArg("apps")
	require.Equal(t, "apps", plugin)
	require.Equal(t, "", version)

	// Version
	plugin, version = parseInstallArg("apps@2.0.1")
	require.Equal(t, "apps", plugin)
	require.Equal(t, "2.0.1", version)
}

func TestSetInstallTelemetryMetadata(t *testing.T) {
	installCmd := &InstallCmd{}
	metadata := stripe.NewEventMetadata()
	ctx := stripe.WithEventMetadata(context.Background(), metadata)

	installCmd.setInstallTelemetryMetadata(ctx, "apps")

	require.Equal(t, "apps", metadata.PluginName)
}
