package cmd

import (
	"github.com/spf13/cobra"
	"github.com/stripe/stripe-cli/pkg/config"
)

type configCmd struct {
	cmd    *cobra.Command
	config *config.Config

	list  bool
	edit  bool
	unset bool
}

func newConfigCmd() *configCmd {
	cc := &configCmd{
		config: &Config,
	}
	cc.cmd = &cobra.Command{
		Use:   "config",
		Short: "Manually change the config values for the CLI",
		RunE:  cc.runConfigCmd,
	}

	cc.cmd.Flags().BoolVar(&cc.list, "list", false, "list configs")
	cc.cmd.Flags().BoolVarP(&cc.edit, "edit", "e", false, "open editor to the config file")
	cc.cmd.Flags().BoolVar(&cc.unset, "unset", false, "unset a specific config field")

	return cc
}

func (cc *configCmd) runConfigCmd(cmd *cobra.Command, args []string) error {
	switch ok := true; ok {
	case len(args) == 2:
		return cc.config.Profile.WriteConfigField(args[0], args[1])
	case len(args) == 1 && cc.unset:
		return cc.config.Profile.DeleteConfigField(args[0])
	case cc.list:
		cc.config.PrintConfig()
	case cc.edit:
		return cc.config.EditConfig()
	}

	return nil
}
