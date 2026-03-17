package resource

import (
	"context"
	"fmt"
	"os"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/afero"
	"github.com/spf13/cobra"

	"github.com/stripe/stripe-cli/pkg/config"
	"github.com/stripe/stripe-cli/pkg/plugins"
)

// HandleResourcePluginConflict is called during initialization to set up
// resource/plugin command handling. It modifies the resource command to intercept
// unknown subcommands and delegate to the plugin.
//
// This should only be called when the plugin is confirmed to be installed.
func HandleResourcePluginConflict(rootCmd *cobra.Command, cfg *config.Config, commandName string) error {
	// Find the command
	var targetCmd *cobra.Command
	for _, cmd := range rootCmd.Commands() {
		if cmd.Use == commandName {
			targetCmd = cmd
			break
		}
	}

	if targetCmd == nil {
		return fmt.Errorf("%s command not found", commandName)
	}

	log.WithFields(log.Fields{
		"prefix":  "cmd.resource.HandleResourcePluginConflict",
		"command": commandName,
	}).Debug("Setting up plugin integration")

	// Store the original help function
	originalHelpFunc := targetCmd.HelpFunc()

	// Command path for this specific command (e.g., "stripe apps")
	expectedPath := fmt.Sprintf("stripe %s", commandName)

	// Override help ONLY for the target command itself
	// We check the command path to ensure we're not affecting subcommands
	targetCmd.SetHelpFunc(func(c *cobra.Command, args []string) {
		log.WithFields(log.Fields{
			"prefix":      "cmd.resource.HandleResourcePluginConflict.HelpFunc",
			"command":     c.Name(),
			"commandPath": c.CommandPath(),
		}).Debug("Help function called")

		// Only show combined help if this is exactly the target command
		// AND the user explicitly requested help (not triggered by unknown subcommand)
		if c.CommandPath() == expectedPath {
			// Check os.Args to see if this is an unknown subcommand scenario
			cmdArgs := ExtractCommandArgs(os.Args, commandName)
			if len(cmdArgs) > 0 {
				// User ran "stripe <command> <something>" where <something> is not a known resource
				// Check if it's a help request
				if cmdArgs[0] == "help" || cmdArgs[0] == "--help" || cmdArgs[0] == "-h" {
					// Explicit help request - show combined help
					log.WithFields(log.Fields{
						"prefix": "cmd.resource.HandleResourcePluginConflict.HelpFunc",
					}).Debug("Explicit help request, showing combined help")
					originalHelpFunc(c, args)
					fmt.Fprintf(c.OutOrStdout(), "\n")
					showPluginHelp(cfg, c.OutOrStdout(), commandName)
				} else {
					// Unknown subcommand triggered help - try the plugin
					log.WithFields(log.Fields{
						"prefix": "cmd.resource.HandleResourcePluginConflict.HelpFunc",
						"args":   cmdArgs,
					}).Debug("Unknown subcommand, trying plugin")

					// Try to execute the plugin directly
					pluginErr := TryPlugin(cfg, commandName, cmdArgs)
					if pluginErr != nil {
						// Plugin failed, show help as fallback
						originalHelpFunc(c, args)
					}
					// If plugin succeeded, TryPlugin will have exited
				}
			} else {
				// No args, user ran just "stripe <command>" - show combined help
				log.WithFields(log.Fields{
					"prefix": "cmd.resource.HandleResourcePluginConflict.HelpFunc",
				}).Debug("No args, showing combined help")
				originalHelpFunc(c, args)
				fmt.Fprintf(c.OutOrStdout(), "\n")
				showPluginHelp(cfg, c.OutOrStdout(), commandName)
			}
		} else {
			// This is a subcommand like "stripe <command> <subcommand>"
			// Show only the normal help for that subcommand
			log.WithFields(log.Fields{
				"prefix":  "cmd.resource.HandleResourcePluginConflict.HelpFunc",
				"command": c.CommandPath(),
			}).Debug("Showing subcommand help only (not combined)")
			originalHelpFunc(c, args)
		}
	})

	// Disable Cobra's built-in suggestions since we handle fallback ourselves
	targetCmd.DisableSuggestions = true

	// Mark that this command has plugin fallback available
	if targetCmd.Annotations == nil {
		targetCmd.Annotations = make(map[string]string)
	}
	targetCmd.Annotations["plugin_fallback"] = commandName

	return nil
}

// showPluginHelp displays the help output from the specified plugin
func showPluginHelp(cfg *config.Config, output interface{ Write([]byte) (int, error) }, pluginName string) {
	fs := afero.NewOsFs()
	plugin, err := plugins.LookUpPlugin(context.Background(), cfg, fs, pluginName)
	if err != nil {
		// Plugin not available, skip
		log.WithFields(log.Fields{
			"prefix": "cmd.resource.showPluginHelp",
			"plugin": pluginName,
			"error":  err,
		}).Debug("Could not load plugin for help display")
		return
	}

	// Add a clear separator and header
	fmt.Fprintf(output, "---\n")
	fmt.Fprintf(output, "Additional commands from the %s plugin:\n\n", pluginName)

	// Run the plugin with --help flag
	// Note: plugin.Run outputs directly to stdout, so it will appear after our header
	ctx := context.Background()
	err = plugin.Run(ctx, cfg, fs, []string{"--help"})
	plugins.CleanupAllClients()

	if err != nil {
		// Plugin help failed, show a message
		log.WithFields(log.Fields{
			"prefix": "cmd.resource.showPluginHelp",
			"plugin": pluginName,
			"error":  err,
		}).Debug("Plugin help failed")
		fmt.Fprintf(output, "  (Run 'stripe %s <command> --help' for plugin-specific help)\n", pluginName)
	}
}

// ExtractCommandArgs extracts the arguments after the specified command from os.Args
func ExtractCommandArgs(osArgs []string, commandName string) []string {
	for i, arg := range osArgs {
		if arg == commandName && i+1 < len(osArgs) {
			return osArgs[i+1:]
		}
	}
	return []string{}
}

// TryPlugin attempts to run the specified plugin with the given args.
// Returns nil if plugin exists and executed successfully.
// Returns error if plugin doesn't exist (so caller can show default error).
// Exits directly with os.Exit(1) if plugin exists but fails (plugin prints its own error).
func TryPlugin(cfg *config.Config, pluginName string, args []string) error {
	fs := afero.NewOsFs()
	plugin, err := plugins.LookUpPlugin(context.Background(), cfg, fs, pluginName)
	if err != nil {
		// Plugin not found - return error so caller can show default error message
		log.WithFields(log.Fields{
			"prefix": "cmd.resource.TryPlugin",
			"plugin": pluginName,
			"error":  err,
		}).Debug("Plugin not found")
		return fmt.Errorf("%s plugin not found: %w", pluginName, err)
	}

	log.WithFields(log.Fields{
		"prefix": "cmd.resource.TryPlugin",
		"plugin": pluginName,
		"args":   args,
	}).Debug("Running plugin")

	// Run the plugin with the provided args
	ctx := context.Background()
	err = plugin.Run(ctx, cfg, fs, args)
	plugins.CleanupAllClients()

	if err != nil {
		// Plugin found but execution failed
		// The plugin will have already printed its error message
		log.WithFields(log.Fields{
			"prefix": "cmd.resource.TryPlugin",
			"plugin": pluginName,
			"error":  err,
		}).Debug("Plugin execution failed")
		// Exit directly like plugin_cmds.go does - don't return error to avoid double-printing
		os.Exit(1)
	}

	// Plugin succeeded
	return nil
}

// Legacy function names for backwards compatibility
// These can be removed in a future refactor

// ExtractAppsArgs extracts the arguments after "apps" from os.Args
func ExtractAppsArgs(osArgs []string) []string {
	return ExtractCommandArgs(osArgs, "apps")
}

// TryAppsPlugin attempts to run the apps plugin with the given args
func TryAppsPlugin(cfg *config.Config, args []string) error {
	return TryPlugin(cfg, "apps", args)
}
