package cmd

import (
	"github.com/spf13/cobra"

	"github.com/stripe/stripe-cli/pkg/cmd/plugin"
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
