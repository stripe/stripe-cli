package cmd

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/afero"
	"github.com/spf13/cobra"

	"github.com/stripe/stripe-cli/pkg/config"
	"github.com/stripe/stripe-cli/pkg/plugins"
	"github.com/stripe/stripe-cli/pkg/validators"
)

var pluginTelemetryFlagsWithValues = map[string]struct{}{
	"--api-key":      {},
	"--color":        {},
	"--config":       {},
	"--device-name":  {},
	"--log-level":    {},
	"--project-name": {},
	"-p":             {},
}

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

	// Add subcommand stubs from manifest metadata so they appear in --map and help
	addPluginSubcommandStubs(ptc.cmd, plugin.Commands, ptc)

	return ptc
}

// addPluginSubcommandStubs recursively creates cobra.Command stubs from
// manifest CommandInfo metadata. These stubs exist for --map display and
// shell completion; actual execution is always delegated to the plugin binary.
func addPluginSubcommandStubs(parent *cobra.Command, commands []plugins.CommandInfo, ptc *pluginTemplateCmd) {
	for _, ci := range commands {
		if ci.Name == "" {
			continue
		}
		sub := &cobra.Command{
			Use:   ci.Name,
			Short: ci.Desc,
			RunE: func(cmd *cobra.Command, args []string) error {
				pluginArgs := subsliceAfter(os.Args, ptc.cmd.Name())
				return ptc.runPluginCmd(ptc.cmd, pluginArgs)
			},
			Annotations: map[string]string{"scope": "plugin"},
			FParseErrWhitelist: cobra.FParseErrWhitelist{
				UnknownFlags: true,
			},
		}
		parent.AddCommand(sub)
		addPluginSubcommandStubs(sub, ci.Commands, ptc)
	}
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

	err = plugin.Run(ctx, ptc.cfg, fs, ptc.ParsedArgs, "")
	plugins.CleanupAllClients()

	if err != nil {
		if err == validators.ErrAPIKeyNotConfigured {
			return errors.New("install failed due to API key not configured, please run `stripe login` or specify the `--api-key`")
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

func resolvePluginTelemetryCommandPath(cmd *cobra.Command, argv []string) string {
	if cmd == nil {
		return ""
	}

	basePath := cmd.CommandPath()
	pluginRoot := pluginRootCommand(cmd)
	if pluginRoot == nil {
		return basePath
	}

	// If Cobra already resolved plugin subcommand stubs from the manifest,
	// trust that path instead of trying to infer it from argv.
	if len(strings.Fields(basePath)) > len(strings.Fields(pluginRoot.CommandPath())) {
		return basePath
	}

	pluginArgs := subsliceAfter(argv, pluginRoot.Name())
	subcommand := firstPluginTelemetrySubcommand(pluginArgs)
	if subcommand == "" {
		return basePath
	}

	return basePath + " " + subcommand
}

func pluginRootCommand(cmd *cobra.Command) *cobra.Command {
	if cmd == nil || !plugins.IsPluginCommand(cmd) {
		return nil
	}

	pluginRoot := cmd
	for pluginRoot.Parent() != nil && plugins.IsPluginCommand(pluginRoot.Parent()) {
		pluginRoot = pluginRoot.Parent()
	}

	return pluginRoot
}

func firstPluginTelemetrySubcommand(args []string) string {
	skipNext := false

	for _, arg := range args {
		if skipNext {
			skipNext = false
			continue
		}

		if arg == "--" {
			break
		}

		if arg == "--help" || arg == "-h" || arg == "--version" || arg == "-v" {
			continue
		}

		if strings.HasPrefix(arg, "--") {
			flagName, _, hasValue := strings.Cut(arg, "=")
			if _, ok := pluginTelemetryFlagsWithValues[flagName]; ok && !hasValue {
				skipNext = true
			}
			continue
		}

		if strings.HasPrefix(arg, "-") && arg != "-" {
			if _, ok := pluginTelemetryFlagsWithValues[arg]; ok {
				skipNext = true
			}
			continue
		}

		return arg
	}

	return ""
}
