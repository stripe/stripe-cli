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

func newUpgradeCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:    "upgrade",
		Hidden: true,
		Short:  "Upgrade a Stripe CLI plugin",
	}
	upgrade := plugin.NewUpgradeCmd(&Config).Cmd
	upgrade.Use = "plugin <plugin>"
	cmd.AddCommand(upgrade)
	return cmd
}

// Stamp the validated install, upgrade, or uninstall target before invocation
// telemetry starts. An install argument's version is never a plugin name.
func setPluginCommandTelemetryMetadata(cmd *cobra.Command, args []string) {
	parent := cmd.Parent()
	if parent == nil || parent.Parent() != cmd.Root() || len(args) != 1 {
		return
	}
	name := args[0]
	switch parent.Name() {
	case "plugin":
		switch cmd.Name() {
		case "install":
			name, _, _ = strings.Cut(name, "@")
		case "upgrade", "uninstall":
		default:
			return
		}
	case "upgrade":
		if cmd.Name() != "plugin" {
			return
		}
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
