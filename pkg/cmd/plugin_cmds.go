package cmd

import (
	"context"
	"errors"
	"fmt"
	"os"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/afero"
	"github.com/spf13/cobra"

	"github.com/stripe/stripe-cli/pkg/config"
	"github.com/stripe/stripe-cli/pkg/plugins"
	"github.com/stripe/stripe-cli/pkg/validators"
)

type pluginTemplateCmd struct {
	cfg        *config.Config
	cmd        *cobra.Command
	fs         afero.Fs
	ParsedArgs []string
}

// newPluginTemplateCmd is a generic plugin command template to dynamically use
// so that we can add any locally installed plugins as supported commands in the CLI
func newPluginTemplateCmd(config *config.Config, plugin *plugins.Plugin) *pluginTemplateCmd {
	ptc := &pluginTemplateCmd{}
	ptc.fs = afero.NewOsFs()
	ptc.cfg = config

	ptc.cmd = &cobra.Command{
		Use:   plugin.Shortname,
		Short: plugin.Shortdesc,
		RunE: func(cmd *cobra.Command, args []string) error {
			// "stripe [host_flags...] plugin_name [plugin_subcommands...] [plugin_flags...]" => "[plugin_subcommands...] [plugin_flags...]"
			pluginArgs := subsliceAfter(os.Args, cmd.Name())
			return ptc.runPluginCmd(cmd, pluginArgs)
		},
		Annotations: map[string]string{"scope": "plugin"},
		FParseErrWhitelist: cobra.FParseErrWhitelist{
			UnknownFlags: true,
		},
	}

	// override the CLI's help command and let the plugin supply the help text instead
	ptc.cmd.SetHelpFunc(func(c *cobra.Command, s []string) {
		var args []string
		if len(s) == 0 {
			// "stripe help plugin_name [plugin_subcommands...]" => "[plugin_subcommands...] --help"
			args = subsliceAfter(os.Args, c.Name())
			args = append(args, "--help")
			c.SetContext(context.Background())
		} else {
			// "stripe plugin_name [plugin_subcommands...] --help" => "[plugin_subcommands...] --help"
			args = subsliceAfter(s, c.Name())
		}
		ptc.runPluginCmd(c, args)
	})

	return ptc
}

// runPluginCmd hands off to the plugin itself to take over
func (ptc *pluginTemplateCmd) runPluginCmd(cmd *cobra.Command, args []string) error {
	ctx := withSIGTERMCancel(cmd.Context(), func() {
		log.WithFields(log.Fields{
			"prefix": "cmd.pluginCmd.runPluginCmd",
		}).Debug("Ctrl+C received, cleaning up...")
	})

	ptc.ParsedArgs = args

	fs := afero.NewOsFs()
	plugin, err := plugins.LookUpPlugin(ctx, ptc.cfg, fs, ptc.cmd.Name())

	if err != nil {
		return err
	}

	log.WithFields(log.Fields{
		"prefix": "cmd.pluginCmd.runPluginCmd",
	}).Debug("Running plugin...")

	err = plugin.Run(ctx, ptc.cfg, fs, ptc.ParsedArgs)
	plugins.CleanupAllClients()

	if err != nil {
		if err == validators.ErrAPIKeyNotConfigured {
			return errors.New("Install failed due to API key not configured. Please run `stripe login` or specify the `--api-key`")
		}

		log.WithFields(log.Fields{
			"prefix": "pluginTemplateCmd.runPluginCmd",
		}).Debug(fmt.Sprintf("Plugin command '%s' exited with error: %s", plugin.Shortname, err))

		// We can't return err because the plugin will have already printed the error message at
		// this point, and we can't return nil because the host will exit with code 0.
		os.Exit(1)
	}

	return nil
}

// Return a copy of sl strictly after the first occurrence of str, or empty slice if not found.
func subsliceAfter(sl []string, str string) []string {
	for i, s := range sl {
		if s == str {
			subsl := sl[i+1:]
			res := make([]string, len(subsl))
			copy(res, subsl)
			return res
		}
	}
	return make([]string, 0)
}
