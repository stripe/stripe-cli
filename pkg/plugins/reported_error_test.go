package plugins

import (
	"errors"
	"testing"

	"github.com/spf13/viper"
	"github.com/stretchr/testify/require"

	"github.com/stripe/stripe-cli/pkg/config"
)

// The caller exits on any error from Run, so it has to print the ones nobody else
// has. Refusing a plugin for being too old happens before the plugin is launched,
// which is what made that refusal exit 1 with no output at all.
func TestPluginAlreadyReportedIsFalseForARefusal(t *testing.T) {
	t.Cleanup(viper.Reset)
	viper.Set(config.ConfigVersionName, config.ConfigVersionV2)
	withConfigV2MinimumVersions(t, map[string]string{"apps": "2.0.0"})

	plugin := Plugin{Shortname: "apps"}
	err := plugin.refuseIfConfigTooNew("1.0.0")

	require.Error(t, err)
	require.False(t, PluginAlreadyReported(err),
		"a refusal happens before the plugin launches, so the caller must print it")
	require.Contains(t, err.Error(), "stripe plugin upgrade apps")
}

// An error the plugin process produced after starting is already on screen.
func TestPluginAlreadyReportedIsTrueForPluginOutput(t *testing.T) {
	inner := errors.New("the plugin printed this itself")

	require.True(t, PluginAlreadyReported(pluginReportedError{inner}))
	require.False(t, PluginAlreadyReported(inner))
	require.False(t, PluginAlreadyReported(nil))

	// Unwrapping has to keep working, so callers can still match on the cause.
	require.True(t, errors.Is(pluginReportedError{inner}, inner))
	require.Equal(t, inner.Error(), pluginReportedError{inner}.Error())
}
