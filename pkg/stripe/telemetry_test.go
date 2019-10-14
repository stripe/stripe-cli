package stripe

import (
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/require"
)

func TestGetTelemetryInstance(t *testing.T) {
	t1 := GetTelemetryInstance()
	t2 := GetTelemetryInstance()
	require.Equal(t, t1, t2)
}

func TestSetCommandContext(t *testing.T) {
	tel := GetTelemetryInstance()
	cmd := &cobra.Command{
		Use: "foo",
	}
	tel.SetCommandContext(cmd)
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
