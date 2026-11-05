package plugin

import (
	"github.com/spf13/cobra"

	"github.com/stripe/stripe-cli/pkg/errorcategory"
	"github.com/stripe/stripe-cli/pkg/plugins"
	"github.com/stripe/stripe-cli/pkg/stripe"
)

// validatePluginNames rejects invalid or repeated plugin names, so a command
// given several plugins fails before it changes any of them.
func validatePluginNames(pluginNames []string) error {
	seen := make(map[string]bool, len(pluginNames))
	for _, pluginName := range pluginNames {
		if err := plugins.ValidatePluginShortname(pluginName); err != nil {
			return err
		}
		if seen[pluginName] {
			return errorcategory.UserInputErrorf("plugin %q was specified more than once", pluginName)
		}
		seen[pluginName] = true
	}

	return nil
}

// setPluginNameTelemetry records pluginName on the invocation's telemetry
// metadata. Commands given several plugins call it before working on each one,
// so the events sent along the way name the plugin they describe.
func setPluginNameTelemetry(cmd *cobra.Command, pluginName string) {
	if m := stripe.GetEventMetadata(cmd.Context()); m != nil {
		m.SetPluginName(pluginName)
	}
}
