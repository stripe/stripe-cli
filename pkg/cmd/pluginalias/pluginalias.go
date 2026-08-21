// Package pluginalias declares the alternate names plugin commands answer to and
// recovers the arguments meant for the plugin binary. The command for an installed
// plugin and the hint command for one that is not installed yet both consult it, so
// an alias resolves and forwards the same way before and after the plugin is on disk.
package pluginalias

import (
	"slices"

	"github.com/spf13/cobra"

	"github.com/stripe/stripe-cli/pkg/cmdutil"
)

// Spec describes the alternate names a plugin command answers to.
type Spec struct {
	// Names are registered as Cobra aliases on the plugin's command.
	Names []string
	// ArgPrefix maps an alias to arguments that must be prepended when the command
	// is invoked through it, for aliases that stand in for a plugin subcommand
	// rather than for the plugin name itself.
	ArgPrefix map[string][]string
}

var specs = map[string]Spec{
	"directory": {
		Names: []string{
			"search",
			"directry",
			"directary",
			"direcotry", //nolint:misspell // Intentional typo alias.
			"diretory",
		},
		// "search" stands in for the plugin's own subcommand rather than for the
		// plugin name, so "stripe search <query>" has to forward as
		// "directory search <query>". The remaining aliases are name typos and
		// forward their arguments untouched.
		ArgPrefix: map[string][]string{"search": {"search"}},
	},
}

// For returns the alias spec for a plugin shortname, or the zero Spec for a plugin
// that has no aliases.
func For(pluginName string) Spec {
	return specs[pluginName]
}

// PluginArgs recovers the arguments intended for the plugin binary from the raw
// process arguments. Cobra has already consumed the flags and command names it
// recognizes, so read them back from argv instead:
// "stripe [host_flags...] directory [plugin_args...]" => "[plugin_args...]".
// Any subcommand the alias the user typed stands in for is then restored.
//
// cmd must be the plugin's root command, not one of its subcommand stubs, so that
// argv is sliced after the plugin name rather than after a subcommand.
func PluginArgs(argv []string, cmd *cobra.Command, pluginName string) []string {
	spec := For(pluginName)
	invokedAs := invokedName(argv, cmd, pluginName, spec)

	args := cmdutil.ArgsAfter(argv, invokedAs)

	if prefix, ok := spec.ArgPrefix[invokedAs]; ok {
		args = append(append([]string{}, prefix...), args...)
	}

	return args
}

// invokedName reports the token the user actually typed to reach cmd, which aliases
// mean is not always the plugin's own name. Cobra records that on the command it
// executed, but not on the target of `stripe help <cmd>` or on a plugin root whose
// subcommand stub ran instead, so fall back to the first name-or-alias in argv.
func invokedName(argv []string, cmd *cobra.Command, pluginName string, spec Spec) string {
	if cmd != nil && cmd.CalledAs() != "" {
		return cmd.CalledAs()
	}

	for _, arg := range argv {
		if arg == pluginName || slices.Contains(spec.Names, arg) {
			return arg
		}
	}

	return pluginName
}
