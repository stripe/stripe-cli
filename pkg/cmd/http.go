package cmd

import (
	"github.com/spf13/cobra"

	"github.com/stripe/stripe-cli/pkg/cmdutil"
)

func registerHTTPCmds(rootCmd *cobra.Command) {
	rootCmd.AddCommand(newGetCmd(false).Cmd)
	rootCmd.AddCommand(newPostCmd(false).Cmd)
	rootCmd.AddCommand(newDeleteCmd(false).Cmd)

	if preview, ok := cmdutil.FindSubCmd(rootCmd, "preview"); ok {
		preview.AddCommand(newGetCmd(true).Cmd)
		preview.AddCommand(newPostCmd(true).Cmd)
		preview.AddCommand(newDeleteCmd(true).Cmd)
	}
}
