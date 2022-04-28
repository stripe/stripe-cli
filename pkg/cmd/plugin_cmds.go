package cmd

import (
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
		Use:         plugin.Shortname,
		Short:       plugin.Shortdesc,
		RunE:        ptc.runPluginCmd,
		Annotations: map[string]string{"scope": "plugin"},
		FParseErrWhitelist: cobra.FParseErrWhitelist{
			UnknownFlags: true,
		},
	}

	// override the CLI's help command and let the plugin supply the help text instead
	ptc.cmd.SetHelpCommand(&cobra.Command{
		Use:    "no-help",
		Hidden: true,
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

	ptc.ParsedArgs = os.Args[2:]

	fs := afero.NewOsFs()
	plugin, err := plugins.LookUpPlugin(ctx, ptc.cfg, fs, cmd.CalledAs())

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
	}

	return nil
}
