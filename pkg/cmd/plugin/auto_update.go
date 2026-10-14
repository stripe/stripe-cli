package plugin

import (
	"fmt"
	"io"
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
	unset   bool
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
A per-plugin choice overrides the global choice; both override the plugin's default.
Use --unset to clear a choice so the next one applies.`,
		Example: `stripe plugin auto-update --enable
  stripe plugin auto-update --disable
  stripe plugin auto-update --unset
  stripe plugin auto-update apps --enable
  stripe plugin auto-update apps --disable
  stripe plugin auto-update apps --unset`,
		Args:   validators.MaximumNArgs(1),
		RunE:   ac.run,
		Hidden: true,
	}

	ac.Cmd.Flags().BoolVar(&ac.enable, "enable", false, "Enable automatic updates")
	ac.Cmd.Flags().BoolVar(&ac.disable, "disable", false, "Disable automatic updates")
	ac.Cmd.Flags().BoolVar(&ac.unset, "unset", false, "Clear the choice so the global setting or plugin default applies")
	ac.Cmd.MarkFlagsMutuallyExclusive("enable", "disable", "unset")

	return ac
}

func (ac *AutoUpdateCmd) run(cmd *cobra.Command, args []string) error {
	if !ac.enable && !ac.disable && !ac.unset {
		return cmd.Help()
	}

	scope := config.PluginConfigGlobalScope
	if len(args) == 1 {
		scope = args[0]
		if !slices.Contains(ac.cfg.GetInstalledPlugins(), scope) {
			return errorcategory.Errorf(errorcategory.UserInput, "plugin %q is not installed", scope)
		}
	}

	key := config.PluginConfigKey(scope, config.PluginConfigUpdatesField)
	var err error
	switch {
	case ac.unset:
		err = ac.cfg.DeleteConfigField(key)
	case ac.enable:
		err = ac.cfg.WriteConfigField(key, config.PluginConfigOn)
	default:
		err = ac.cfg.WriteConfigField(key, config.PluginConfigOff)
	}
	if err != nil {
		return err
	}

	ac.printSettings(cmd, scope)
	return nil
}

func (ac *AutoUpdateCmd) printSettings(cmd *cobra.Command, scope string) {
	out := cmd.OutOrStdout()
	if ac.unset {
		printUnsetSettings(out, scope)
		return
	}

	action := "Enable"
	state := "disabled"
	if ac.enable {
		action = "Disable"
		state = "enabled"
	}

	if scope == config.PluginConfigGlobalScope {
		fmt.Fprintf(out, "Automatic updates are %s for all plugins\n\n", state)
		fmt.Fprintf(out, "%s them with 'stripe plugin auto-update --%s'\n", action, strings.ToLower(action))
		fmt.Fprintln(out, "Follow each plugin's default with 'stripe plugin auto-update --unset'")
		return
	}

	fmt.Fprintf(out, "Automatic updates are %s for the %s plugin\n\n", state, displayPluginName(scope))
	fmt.Fprintf(out, "%s it with 'stripe plugin auto-update %s --%s'\n", action, scope, strings.ToLower(action))
	fmt.Fprintf(out, "Follow the global setting with 'stripe plugin auto-update %s --unset' (current: %s)\n", scope, globalUpdatesState())
}

// printUnsetSettings reports what a scope follows once its own choice is cleared.
func printUnsetSettings(out io.Writer, scope string) {
	if scope == config.PluginConfigGlobalScope {
		fmt.Fprint(out, "Automatic updates now follow each plugin's default\n\n")
		fmt.Fprintln(out, "Enable them with 'stripe plugin auto-update --enable'")
		fmt.Fprintln(out, "Disable them with 'stripe plugin auto-update --disable'")
		return
	}

	fmt.Fprintf(out, "Automatic updates for the %s plugin now follow the global setting (current: %s)\n\n", displayPluginName(scope), globalUpdatesState())
	fmt.Fprintf(out, "Enable it with 'stripe plugin auto-update %s --enable'\n", scope)
	fmt.Fprintf(out, "Disable it with 'stripe plugin auto-update %s --disable'\n", scope)
}

// globalUpdatesState describes the global choice, which is what a plugin without
// a choice of its own follows.
func globalUpdatesState() string {
	switch {
	case config.PluginUpdatesEnabled("", false):
		return "enabled"
	case !config.PluginUpdatesEnabled("", true):
		return "disabled"
	default:
		return "plugin default"
	}
}

func displayPluginName(scope string) string {
	return strings.ToUpper(scope[:1]) + scope[1:]
}
