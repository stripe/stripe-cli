package cmd

import (
	"strings"

	"github.com/spf13/cobra"

	"github.com/stripe/stripe-cli/pkg/cmd/plugin"
	"github.com/stripe/stripe-cli/pkg/plugins"
	"github.com/stripe/stripe-cli/pkg/stripe"
	"github.com/stripe/stripe-cli/pkg/validators"
)

type pluginCmd struct {
	cmd *cobra.Command
}

func newPluginCmd() *pluginCmd {
	pc := &pluginCmd{}
	listCmd := plugin.NewListCmd(&Config)

	pc.cmd = &cobra.Command{
		Use:    "plugin",
		Hidden: true,
		Args:   validators.ExactArgs(1),
		Short:  "Interact with Stripe CLI plugins",
		Long:   "Interact with Stripe CLI plugins.",
	}

	pc.cmd.AddCommand(listCmd.Cmd)
	pc.cmd.AddCommand(plugin.NewInstallCmd(&Config).Cmd)
	pc.cmd.AddCommand(plugin.NewUpgradeCmd(&Config).Cmd)
	pc.cmd.AddCommand(plugin.NewUninstallCmd(&Config).Cmd)
	pc.cmd.AddCommand(plugin.NewAutoUpdateCmd(&Config).Cmd)

	return pc
}

// Stamp the validated install/upgrade target before the root starts invocation
// telemetry. An install argument's version is never a plugin name.
func setPluginCommandTelemetryMetadata(cmd *cobra.Command, args []string) {
	parent := cmd.Parent()
	if parent == nil || parent.Name() != "plugin" || parent.Parent() != cmd.Root() || len(args) != 1 {
		return
	}
	name := args[0]
	switch cmd.Name() {
	case "install":
		name, _, _ = strings.Cut(name, "@")
	case "upgrade":
	default:
		return
	}
	if plugins.ValidatePluginShortname(name) != nil {
		return
	}
	if metadata := stripe.GetEventMetadata(cmd.Context()); metadata != nil {
		metadata.SetPluginName(name)
	}
}
