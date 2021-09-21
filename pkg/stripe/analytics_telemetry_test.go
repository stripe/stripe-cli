package stripe

import (
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/require"
)

func TestSetCobraCommandContext(t *testing.T) {
	tel := InitContext()
	cmd := &cobra.Command{
		Use: "foo",
	}
	tel.SetCobraCommandContext(cmd)
	require.Equal(t, "foo", tel.CommandPath)
}

func TestTelemetryOptedOut(t *testing.T) {
	require.False(t, telemetryOptedOut(""))
	require.False(t, telemetryOptedOut("0"))
	require.False(t, telemetryOptedOut("false"))
	require.False(t, telemetryOptedOut("False"))
	require.False(t, telemetryOptedOut("FALSE"))
	require.True(t, telemetryOptedOut("1"))
	require.True(t, telemetryOptedOut("true"))
	require.True(t, telemetryOptedOut("True"))
	require.True(t, telemetryOptedOut("TRUE"))
}
