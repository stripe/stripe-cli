package cmd

import (
	"github.com/spf13/cobra"
)

func registerHTTPCmds(rootCmd *cobra.Command) {
	rootCmd.AddCommand(newGetCmd(false).Cmd)
	rootCmd.AddCommand(newPostCmd(false).Cmd)
	rootCmd.AddCommand(newDeleteCmd(false).Cmd)

	for _, cmd := range rootCmd.Commands() {
		if cmd.Use == "preview" {
			cmd.AddCommand(newGetCmd(true).Cmd)
			cmd.AddCommand(newPostCmd(true).Cmd)
			cmd.AddCommand(newDeleteCmd(true).Cmd)
			break
		}
	}
}
