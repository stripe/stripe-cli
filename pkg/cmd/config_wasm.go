//go:build wasm
// +build wasm

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
	set   bool
}

func newConfigCmd() *configCmd {
	cc := &configCmd{
		config: &Config,
	}
	cc.cmd = &cobra.Command{
		Use:   "config",
		Short: "Manually change the config values for the CLI",
		Long: `config lets you set and unset specific configuration values for your profile if
you need more granular control over the configuration.`,
		Example: `stripe config --list
  stripe config --set color off
  stripe config --unset color`,
	}

	cc.cmd.Flags().BoolVar(&cc.list, "list", false, "List configs")
	cc.cmd.Flags().BoolVarP(&cc.edit, "edit", "e", false, "Open an editor to the config file")
	cc.cmd.Flags().StringVar(&cc.unset, "unset", "", "Unset a specific config field")
	cc.cmd.Flags().BoolVar(&cc.set, "set", false, "Set a config field to some value")

	cc.cmd.Flags().SetInterspersed(false) // allow args to happen after flags to enable 2 arguments to --set

	return cc
}
