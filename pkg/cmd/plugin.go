package cmd

import (
	"github.com/spf13/cobra"

	"github.com/stripe/stripe-cli/pkg/cmd/plugin"
	"github.com/stripe/stripe-cli/pkg/validators"
)

type pluginCmd struct {
	cmd *cobra.Command
}

func newPluginCmd() *pluginCmd {
	pc := &pluginCmd{}

	pc.cmd = &cobra.Command{
		Use:    "plugin",
		Hidden: true,
		Args:   validators.ExactArgs(1),
		Short:  "Interact with Stripe CLI plugins",
		Long:   "Interact with Stripe CLI plugins.",
	}

	pc.cmd.AddCommand(plugin.NewInstallCmd(&Config).Cmd)
	pc.cmd.AddCommand(plugin.NewUpgradeCmd(&Config).Cmd)
	pc.cmd.AddCommand(plugin.NewUninstallCmd(&Config).Cmd)

	return pc
}
