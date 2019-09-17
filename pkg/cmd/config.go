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
	unset string
	set bool
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
	cc.cmd.Flags().StringVar(&cc.unset, "unset", "", "unset a specific config field")
	cc.cmd.Flags().BoolVar(&cc.set, "set string string", false, "set a config field to some value")

	cc.cmd.Flags().SetInterspersed(false) // allow args to happen after flags to enable 2 arguments to --set

	return cc
}

func (cc *configCmd) runConfigCmd(cmd *cobra.Command, args []string) error {
	switch ok := true; ok {
	case cc.set && len(args) == 2:
		return cc.config.Profile.WriteConfigField(args[0], args[1])
	case cc.unset != "":
		return cc.config.Profile.DeleteConfigField(cc.unset)
	case cc.list:
		return cc.config.PrintConfig()
	case cc.edit:
		return cc.config.EditConfig()
	default:
		// no flags set or unrecognized flags/args
		return cc.cmd.Help()
	}

	return nil
}
