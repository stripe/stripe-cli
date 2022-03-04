package plugin

import (
	"testing"

	"github.com/stretchr/testify/require"
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
