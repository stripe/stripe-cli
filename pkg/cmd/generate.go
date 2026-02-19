package cmd

import (
	"context"
	"errors"
	"os"

	"github.com/spf13/afero"
	"github.com/spf13/cobra"

	"github.com/stripe/stripe-cli/pkg/plugins"
)

type generateCmd struct {
	cmd *cobra.Command
}

func newGenerateCmd(pluginList []string, nfs afero.Fs) (*generateCmd, error) {
	gc := &generateCmd{
		cmd: &cobra.Command{
			Hidden: true,
			Use:    "generate",
			Short:  "EXPERIMENTAL DO NOT USE",
		},
	}

	var appsPlugin *plugins.Plugin
	for _, p := range pluginList {
		plugin, err := plugins.LookUpPlugin(context.Background(), &Config, nfs, p)
		if err == nil && plugin.Shortname == "apps" {
			appsPlugin = &plugin
		}
	}

	if appsPlugin == nil {
		return nil, errors.New("apps plugin not found")
	}

	ptc := newPluginTemplateCmd(&Config, appsPlugin)

	ptc.cmd.RunE = func(cmd *cobra.Command, args []string) error {
		pluginArgs := subsliceAfter(os.Args, "generate")
		pluginArgs = append([]string{"add"}, pluginArgs...)
		return ptc.runPluginCmd(cmd, pluginArgs)
	}

	gc.cmd.AddCommand(ptc.cmd)

	return gc, nil
}
