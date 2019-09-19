package cmd

import (
	"fmt"
	"runtime"

	"github.com/spf13/cobra"

	"github.com/stripe/stripe-cli/pkg/ansi"
	"github.com/stripe/stripe-cli/pkg/validators"
)

type completionCmd struct {
	cmd *cobra.Command

	shell string
}

var installDirectory = map[string]map[string]string{
	"darwin": map[string]string{
		"zsh":  "/usr/local/share/zsh/site-functions",
		"bash": "/usr/local/etc/bash_completion.d",
	},
	"linux": map[string]string{
		"zsh":  "/usr/share/zsh/functions/Completion/Linux",
		"bash": "/usr/share/bash-completion/completions",
	},
}

func newCompletionCmd() *completionCmd {
	cc := &completionCmd{}

	cc.cmd = &cobra.Command{
		Use:   "completion",
		Short: "Generate bash and zsh completion scripts",
		Args:  validators.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			userOS := runtime.GOOS
			if userOS == "windows" {
				fmt.Println("Warning: autocompletion for Windows is currently not supported and may not work.")
				// Give them instructions for linux because Windows _might_ be using the linux fs under the hood
				userOS = "linux"
			}

			if cc.shell == "zsh" {
				fileName := "stripe-completion.zsh"
				fmt.Println(fmt.Sprintf("Generating zsh completion file in: %s", ansi.Bold(fileName)))
				fmt.Println("You'll need to either source this file or move it to zsh's autocomplete folder.")
				fmt.Println(fmt.Sprintf("To source, run: `source %s`", fileName))
				fmt.Println(fmt.Sprintf("To load automatically, move to zsh's autocomplete directory: `mv %s %s`", fileName, installDirectory[userOS]["zsh"]))
				return rootCmd.GenZshCompletionFile(fileName)
			}

			fileName := "stripe-completion.bash"
			fmt.Println(fmt.Sprintf("Generating bash completion file in: %s", ansi.Bold(fileName)))
			fmt.Println("You'll need to either source this file or move it to bash's autocomplete folder.")
			fmt.Println(fmt.Sprintf("To source, run: `source %s`", fileName))
			fmt.Println(fmt.Sprintf("To load automatically, move to bash's autocomplete directory: `mv %s %s`", fileName, installDirectory[userOS]["bash"]))
			return rootCmd.GenBashCompletionFile(fileName)
		},
	}

	cc.cmd.Flags().StringVar(&cc.shell, "shell", "bash", "The shell to generate completion commands for. This only supports \"bash\" or \"zsh\"")

	return cc
}
