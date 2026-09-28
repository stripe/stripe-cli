package plugin

import (
	"fmt"
	"slices"
	"strings"

	"github.com/spf13/cobra"

	"github.com/stripe/stripe-cli/pkg/config"
	"github.com/stripe/stripe-cli/pkg/errorcategory"
	"github.com/stripe/stripe-cli/pkg/validators"
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
		Long: `Enable or disable automatic updates before a plugin runs.

Each plugin defines whether automatic updates are enabled by default.
Omit the plugin name to apply your choice globally to all plugins.
A per-plugin choice overrides the global choice; both override the plugin's default.`,
		Example: `stripe plugin auto-update --enable
  stripe plugin auto-update --disable
  stripe plugin auto-update apps --enable
  stripe plugin auto-update apps --disable`,
		Args:   validators.MaximumNArgs(1),
		RunE:   ac.run,
		Hidden: true,
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

	scope := config.PluginConfigGlobalScope
	if len(args) == 1 {
		scope = args[0]
		if !slices.Contains(ac.cfg.GetInstalledPlugins(), scope) {
			return errorcategory.Errorf(errorcategory.UserInput, "plugin %q is not installed", scope)
		}
	}

	value := config.PluginConfigOff
	if ac.enable {
		value = config.PluginConfigOn
	}

	if err := ac.cfg.WriteConfigField(config.PluginConfigKey(scope, config.PluginConfigUpdatesField), value); err != nil {
		return err
	}

	ac.printSettings(cmd, scope)
	return nil
}

func (ac *AutoUpdateCmd) printSettings(cmd *cobra.Command, scope string) {
	action := "Enable"
	state := "disabled"
	if ac.enable {
		action = "Disable"
		state = "enabled"
	}

	out := cmd.OutOrStdout()
	if scope == config.PluginConfigGlobalScope {
		fmt.Fprintf(out, "Automatic updates are %s for all plugins\n\n", state)
		fmt.Fprintf(out, "%s them with 'stripe plugin auto-update --%s'\n", action, strings.ToLower(action))
		return
	}

	pluginName := strings.ToUpper(scope[:1]) + scope[1:]
	globalState := "plugin default"
	if config.PluginUpdatesEnabled("", false) {
		globalState = "enabled"
	} else if !config.PluginUpdatesEnabled("", true) {
		globalState = "disabled"
	}

	fmt.Fprintf(out, "Automatic updates are %s for the %s plugin\n\n", state, pluginName)
	fmt.Fprintf(out, "%s it with 'stripe plugin auto-update %s --%s'\n", action, scope, strings.ToLower(action))
	fmt.Fprintf(out, "Follow the global setting with 'stripe plugin auto-update %s --unset' (current: %s)\n", scope, globalState)
}
