package plugin

import (
	"fmt"
	"slices"

	"github.com/spf13/cobra"

	"github.com/stripe/stripe-cli/pkg/config"
	"github.com/stripe/stripe-cli/pkg/plugins"
)

// AutoUpdateCmd handles `stripe plugin auto-update` for enabling/disabling automatic plugin updates.
type AutoUpdateCmd struct {
	cfg *config.Config
	Cmd *cobra.Command

	enable  bool
	disable bool
}

// NewAutoUpdateCmd creates the `stripe plugin auto-update` command.
func NewAutoUpdateCmd(cfg *config.Config) *AutoUpdateCmd {
	ac := &AutoUpdateCmd{cfg: cfg}

	ac.Cmd = &cobra.Command{
		Use:   "auto-update [plugin]",
		Short: "Enable or disable automatic updates for a plugin",
		Long: `Enable or disable automatic background updates for a plugin.

When disabled, the CLI will not check for or download newer versions automatically.
Omit the plugin name to apply the setting globally to all plugins.`,
		Example: `stripe plugin auto-update --enable
  stripe plugin auto-update --disable
  stripe plugin auto-update apps --enable
  stripe plugin auto-update apps --disable`,
		RunE: ac.run,
	}

	ac.Cmd.Flags().BoolVar(&ac.enable, "enable", false, "Enable automatic updates")
	ac.Cmd.Flags().BoolVar(&ac.disable, "disable", false, "Disable automatic updates")
	ac.Cmd.MarkFlagsMutuallyExclusive("enable", "disable")

	return ac
}

func (ac *AutoUpdateCmd) run(cmd *cobra.Command, args []string) error {
	if !ac.enable && !ac.disable {
		return cmd.Help()
	}

	scope := plugins.PluginConfigGlobalScope
	if len(args) == 1 {
		scope = args[0]
		if !slices.Contains(ac.cfg.GetInstalledPlugins(), scope) {
			return fmt.Errorf("plugin %q is not installed", scope)
		}
	}

	value := "off"
	if ac.enable {
		value = "on"
	}

	return ac.cfg.WriteConfigField(plugins.PluginConfigKey(scope, plugins.PluginConfigUpdatesField), value)
}
