package resource

import (
	"github.com/spf13/cobra"
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
	for _, cmd := range rootCmd.Commands() {
		for principle := range aliasedCmds {
			formattedPrinciple := GetResourceCmdName(principle)
			if cmd.Use == formattedPrinciple {
				cmd.Hidden = true
			}
		}
	}
}
