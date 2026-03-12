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

// HandleAppsResourceAndPlugin is called during initialization to set up
// apps command handling. It modifies the apps command to intercept unknown
// subcommands and delegate to the plugin before Cobra shows suggestions.
func HandleAppsResourceAndPlugin(rootCmd *cobra.Command, cfg *config.Config) error {
	// Find the apps command
	var appsCmd *cobra.Command
	for _, cmd := range rootCmd.Commands() {
		if cmd.Use == "apps" {
			appsCmd = cmd
			break
		}
	}

	if appsCmd == nil {
		return nil // No apps command, nothing to do
	}

	// Check if apps plugin is installed and mark the command if so
	fs := afero.NewOsFs()
	_, err := plugins.LookUpPlugin(context.Background(), cfg, fs, "apps")
	if err != nil {
		// Plugin not found or error looking it up, just keep resource commands as-is
		log.WithFields(log.Fields{
			"prefix": "cmd.resource.HandleAppsResourceAndPlugin",
		}).Debug("Apps plugin not installed, using resource commands only")
		return nil
	}

	log.WithFields(log.Fields{
		"prefix": "cmd.resource.HandleAppsResourceAndPlugin",
	}).Debug("Apps plugin detected, setting up fallback handler")

	// Store the original help function
	originalHelpFunc := appsCmd.HelpFunc()

	// Override help ONLY for the apps command itself
	// We check the command path to ensure we're not affecting subcommands
	appsCmd.SetHelpFunc(func(c *cobra.Command, args []string) {
		log.WithFields(log.Fields{
			"prefix":      "cmd.resource.HandleAppsResourceAndPlugin.HelpFunc",
			"command":     c.Name(),
			"commandPath": c.CommandPath(),
		}).Debug("Help function called")

		// Only show combined help if this is exactly the "stripe apps" command
		// AND the user explicitly requested help (not triggered by unknown subcommand)
		if c.CommandPath() == "stripe apps" {
			// Check os.Args to see if this is an unknown subcommand scenario
			appsArgs := ExtractAppsArgs(os.Args)
			if len(appsArgs) > 0 {
				// User ran "stripe apps <something>" where <something> is not a known resource
				// Check if it's a help request
				if appsArgs[0] == "help" || appsArgs[0] == "--help" || appsArgs[0] == "-h" {
					// Explicit help request - show combined help
					log.WithFields(log.Fields{
						"prefix": "cmd.resource.HandleAppsResourceAndPlugin.HelpFunc",
					}).Debug("Explicit help request, showing combined help")
					originalHelpFunc(c, args)
					fmt.Fprintf(c.OutOrStdout(), "\n")
					showPluginHelp(cfg, c.OutOrStdout())
				} else {
					// Unknown subcommand triggered help - try the plugin
					log.WithFields(log.Fields{
						"prefix": "cmd.resource.HandleAppsResourceAndPlugin.HelpFunc",
						"args":   appsArgs,
					}).Debug("Unknown subcommand, trying plugin")

					// Try to execute the plugin directly
					pluginErr := TryAppsPlugin(cfg, appsArgs)
					if pluginErr != nil {
						// Plugin failed, show help as fallback
						originalHelpFunc(c, args)
					}
					// If plugin succeeded, TryAppsPlugin will have exited
				}
			} else {
				// No args, user ran just "stripe apps" - show combined help
				log.WithFields(log.Fields{
					"prefix": "cmd.resource.HandleAppsResourceAndPlugin.HelpFunc",
				}).Debug("No args, showing combined help")
				originalHelpFunc(c, args)
				fmt.Fprintf(c.OutOrStdout(), "\n")
				showPluginHelp(cfg, c.OutOrStdout())
			}
		} else {
			// This is a subcommand like "stripe apps secrets"
			// Show only the normal help for that subcommand
			log.WithFields(log.Fields{
				"prefix":  "cmd.resource.HandleAppsResourceAndPlugin.HelpFunc",
				"command": c.CommandPath(),
			}).Debug("Showing subcommand help only (not combined)")
			originalHelpFunc(c, args)
		}
	})

	// Disable Cobra's built-in suggestions since we handle fallback ourselves
	appsCmd.DisableSuggestions = true

	// Mark that this command has plugin fallback available
	if appsCmd.Annotations == nil {
		appsCmd.Annotations = make(map[string]string)
	}
	appsCmd.Annotations["plugin_fallback"] = "apps"

	return nil
}

// showPluginHelp displays the help output from the apps plugin
func showPluginHelp(cfg *config.Config, output interface{ Write([]byte) (int, error) }) {
	fs := afero.NewOsFs()
	plugin, err := plugins.LookUpPlugin(context.Background(), cfg, fs, "apps")
	if err != nil {
		// Plugin not available, skip
		log.WithFields(log.Fields{
			"prefix": "cmd.resource.showPluginHelp",
			"error":  err,
		}).Debug("Could not load plugin for help display")
		return
	}

	// Add a clear separator and header
	fmt.Fprintf(output, "---\n")
	fmt.Fprintf(output, "Additional commands from the apps plugin:\n\n")

	// Run the plugin with --help flag
	// Note: plugin.Run outputs directly to stdout, so it will appear after our header
	ctx := context.Background()
	err = plugin.Run(ctx, cfg, fs, []string{"--help"})
	plugins.CleanupAllClients()

	if err != nil {
		// Plugin help failed, show a message
		log.WithFields(log.Fields{
			"prefix": "cmd.resource.showPluginHelp",
			"error":  err,
		}).Debug("Plugin help failed")
		fmt.Fprintf(output, "  (Run 'stripe apps <command> --help' for plugin-specific help)\n")
	}
}

// ExtractAppsArgs extracts the arguments after "apps" from os.Args
func ExtractAppsArgs(osArgs []string) []string {
	for i, arg := range osArgs {
		if arg == "apps" && i+1 < len(osArgs) {
			return osArgs[i+1:]
		}
	}
	return []string{}
}

// TryAppsPlugin attempts to run the apps plugin with the given args.
// Returns nil if plugin exists and executed successfully.
// Returns error if plugin doesn't exist (so caller can show default error).
// Exits directly with os.Exit(1) if plugin exists but fails (plugin prints its own error).
func TryAppsPlugin(cfg *config.Config, args []string) error {
	fs := afero.NewOsFs()
	plugin, err := plugins.LookUpPlugin(context.Background(), cfg, fs, "apps")
	if err != nil {
		// Plugin not found - return error so caller can show default error message
		log.WithFields(log.Fields{
			"prefix": "cmd.resource.TryAppsPlugin",
			"error":  err,
		}).Debug("Apps plugin not found")
		return fmt.Errorf("apps plugin not found: %w", err)
	}

	log.WithFields(log.Fields{
		"prefix": "cmd.resource.TryAppsPlugin",
		"args":   args,
	}).Debug("Running apps plugin")

	// Run the plugin with the provided args
	ctx := context.Background()
	err = plugin.Run(ctx, cfg, fs, args)
	plugins.CleanupAllClients()

	if err != nil {
		// Plugin found but execution failed
		// The plugin will have already printed its error message
		log.WithFields(log.Fields{
			"prefix": "cmd.resource.TryAppsPlugin",
			"error":  err,
		}).Debug("Plugin execution failed")
		// Exit directly like plugin_cmds.go does - don't return error to avoid double-printing
		os.Exit(1)
	}

	// Plugin succeeded
	return nil
}

// RemoveAppsCmd is kept for backwards compatibility but is now a no-op
func RemoveAppsCmd(rootCmd *cobra.Command) error {
	// No longer removes the apps command
	// The actual handling is done by HandleAppsResourceAndPlugin
	return nil
}
