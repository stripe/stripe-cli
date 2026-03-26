package plugin

import (
	"fmt"
	"slices"

	"github.com/spf13/cobra"

	"github.com/stripe/stripe-cli/pkg/config"
)

// ConfigCmd handles `stripe plugin config` for reading and writing plugin settings.
type ConfigCmd struct {
	cfg *config.Config
	Cmd *cobra.Command

	set   bool
	unset string
}

// NewConfigCmd creates the `stripe plugin config` command.
func NewConfigCmd(cfg *config.Config) *ConfigCmd {
	cc := &ConfigCmd{cfg: cfg}

	cc.Cmd = &cobra.Command{
		Hidden: true,
		Use:    "config [plugin]",
		Short:  "Read and write plugin configuration",
		Long: `Read and write configuration for plugins. Omit the plugin name to apply the setting globally.

Available fields:
  updates   Controls automatic background updates for a plugin. When set to "off",
            the CLI will not check for or download newer versions automatically.
            Accepted values: on, off (default: off)`,
		Example: `stripe plugin config --set updates on
  stripe plugin config --unset updates
  stripe plugin config apps --set updates off
  stripe plugin config apps --unset updates`,
		RunE: cc.run,
	}

	cc.Cmd.Flags().BoolVar(&cc.set, "set", false, "Set a config field to some value")
	cc.Cmd.Flags().StringVar(&cc.unset, "unset", "", "Unset a specific config field")

	return cc
}

func (cc *ConfigCmd) run(cmd *cobra.Command, args []string) error {
	switch {
	case cc.set && len(args) == 2:
		// stripe plugin config --set updates <value>  (global)
		return cc.setUpdates("__global", args[0], args[1])
	case cc.set && len(args) == 3:
		// stripe plugin config <plugin> --set updates <value>
		return cc.setUpdates(args[0], args[1], args[2])
	case cc.unset != "" && len(args) == 0:
		// stripe plugin config --unset updates  (global)
		return cc.cfg.DeleteConfigField(fmt.Sprintf("plugin_configs.__global.%s", cc.unset))
	case cc.unset != "" && len(args) == 1:
		// stripe plugin config <plugin> --unset updates
		pluginName := args[0]
		if !slices.Contains(cc.cfg.GetInstalledPlugins(), pluginName) {
			return fmt.Errorf("plugin %q is not installed", pluginName)
		}
		return cc.cfg.DeleteConfigField(fmt.Sprintf("plugin_configs.%s.%s", pluginName, cc.unset))
	default:
		return cmd.Help()
	}
}

func (cc *ConfigCmd) setUpdates(scope, field, value string) error {
	if field != "updates" {
		return fmt.Errorf("unknown config field %q", field)
	}
	if value != "on" && value != "off" {
		return fmt.Errorf("invalid value %q for updates — must be \"on\" or \"off\"", value)
	}
	if scope != "__global" {
		if !slices.Contains(cc.cfg.GetInstalledPlugins(), scope) {
			return fmt.Errorf("plugin %q is not installed", scope)
		}
	}
	return cc.cfg.WriteConfigField(fmt.Sprintf("plugin_configs.%s.%s", scope, field), value)
}
