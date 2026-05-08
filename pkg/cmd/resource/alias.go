// Package resource provides auto-generated API resource commands.
package resource

import (
	"github.com/spf13/cobra"

	"github.com/stripe/stripe-cli/pkg/cmdutil"
)

// Aliases are mapped from resource name in the OpenAPI spec -> friendly name in the CLI
// Each alias causes a second resource command (the target / friendly name) to be generated, and uses post-processing
// to hide the principle resource name (OpenAPI spec)
var aliasedCmds = map[string]string{
	"line_item": "invoice_line_item",
}

// GetCmdAlias retrieves the alias for a given resource, if one is present; otherwise returns ""
func GetCmdAlias(principle string) string {
	alias, ok := aliasedCmds[principle]
	if !ok {
		return ""
	}
	return alias
}

// GetAliases retrieves the entire alias map, useful for testing
func GetAliases() map[string]string {
	return aliasedCmds
}

// HideAliasedCommands performs the post-processing on the command tree to hide
// resources that have an alias
func HideAliasedCommands(rootCmd *cobra.Command) {
	for principle := range aliasedCmds {
		if cmd, ok := cmdutil.FindSubCmd(rootCmd, GetResourceCmdName(principle)); ok {
			cmd.Hidden = true
		}
	}
}
