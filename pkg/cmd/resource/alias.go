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

func GetCmdAlias(principle string) string {
	alias, ok := aliasedCmds[principle]
	if !ok {
		return ""
	}
	return alias
}

func GetAliases() map[string]string {
	return aliasedCmds
}

func HideAliasedCommands(rootCmd *cobra.Command) {
	for _, cmd := range rootCmd.Commands() {
		for principle, _ := range aliasedCmds {
			formattedPrinciple := GetResourceCmdName(principle)
			if cmd.Use == formattedPrinciple {
				cmd.Hidden = true
			}
		}
	}
}
