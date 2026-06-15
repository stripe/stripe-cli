package cmd

import (
	"github.com/spf13/cobra"

	"github.com/stripe/stripe-cli/pkg/cmd/plugin"
	"github.com/stripe/stripe-cli/pkg/i18n"
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
		Short:  i18n.T("plugin.short"),
		Long:   i18n.T("plugin.long"),
	}

	pc.cmd.AddCommand(listCmd.Cmd)
	pc.cmd.AddCommand(plugin.NewInstallCmd(&Config).Cmd)
	pc.cmd.AddCommand(plugin.NewUpgradeCmd(&Config).Cmd)
	pc.cmd.AddCommand(plugin.NewUninstallCmd(&Config).Cmd)
	pc.cmd.AddCommand(plugin.NewAutoUpdateCmd(&Config).Cmd)

	return pc
}
