package cmd

import (
	"github.com/spf13/cobra"
	"github.com/stripe/stripe-cli/validators"
)

type completionCmd struct {
	cmd *cobra.Command

	shell string
}

func newCompletionCmd() *completionCmd {
	cc := &completionCmd{}

	cc.cmd = &cobra.Command{
		Use:   "completion",
		Short: "Generates bash and zsh completion scripts",
		Args:  validators.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			if cc.shell == "zsh" {
				return rootCmd.GenZshCompletionFile("stripe-completion.zsh")
			}
			return rootCmd.GenBashCompletionFile("stripe-completion.bash")
		},
	}

	cc.cmd.Flags().StringVar(&cc.shell, "shell", "bash", "The shell to generate completion commands for. This only supports \"bash\" or \"zsh\"")

	return cc
}
