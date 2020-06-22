package cmd

import (
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"runtime"

	"github.com/stripe/stripe-cli/pkg/validators"
)

type completionCmd struct {
	cmd *cobra.Command

	shell string
}

func newCompletionCmd() *completionCmd {
	cc := &completionCmd{}

	cc.cmd = &cobra.Command{
		Use:   "completion",
		Short: "Generate bash and zsh completion scripts",
		Args:  validators.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return selectShell(cc.shell)
		},
	}

	cc.cmd.Flags().StringVar(&cc.shell, "shell", "", "The shell to generate completion commands for. Supports \"bash\" or \"zsh\"")

	return cc
}

const (
	instructionsHeader = `
Suggested next steps:
---------------------`

	zshCompletionInstructions = `
1. Move ` + "`stripe-completion.zsh`" + ` to the correct location:
    mkdir -p ~/.stripe
    mv stripe-completion.zsh ~/.stripe

2. Add the following lines to your ` + "`.zshrc`" + ` enabling shell completion for Stripe:
    fpath=(~/.stripe $fpath)
    autoload -Uz compinit && compinit -i

3. Source your ` + "`.zshrc`" + ` or open a new terminal session:
    source ~/.zshrc`

	bashCompletionInstructionsMac = `
Set up bash autocompletion on your system:
1. Install the bash autocompletion package:
     brew install bash-completion
2. Follow the post-install instructions displayed by Homebrew; add a line like the following to your bash profile:
     [[ -r "/usr/local/etc/profile.d/bash_completion.sh" ]] && . "/usr/local/etc/profile.d/bash_completion.sh"

Set up Stripe autocompletion:
3. Move ` + "`stripe-completion.bash`" + ` to the correct location:
    mkdir -p ~/.stripe
    mv stripe-completion.bash ~/.stripe

4. Add the following line to your bash profile, so that Stripe autocompletion will be enabled every time you start a new terminal session:
    source ~/.stripe/stripe-completion.bash

5. Either restart your terminal, or run the following command in your current session to enable immediately:
    source ~/.stripe/stripe-completion.bash`

	bashCompletionInstructionsLinux = `
1. Ensure bash autocompletion is installed on your system. Often, this means verifying that ` + "`/etc/profile.d/bash_completion.sh`" + ` exists, and is sourced by your bash profile; the location of this file varies across distributions of Linux.

2. Move ` + "`stripe-completion.bash`" + ` to the correct location:
    mkdir -p ~/.stripe
    mv stripe-completion.bash ~/.stripe

3. Add the following line to your bash profile, so that Stripe autocompletion will be enabled every time you start a new terminal session:
    source ~/.stripe/stripe-completion.bash

4. Either restart your terminal, or run the following command in your current session to enable immediately:
    source ~/.stripe/stripe-completion.bash`
)

func selectShell(shell string) error {
	selected := shell
	if selected == "" {
		selected = detectShell()
	}

	switch {
	case selected == "zsh":
		fmt.Println("Detected `zsh`, generating zsh completion file: stripe-completion.zsh")
		err := rootCmd.GenZshCompletionFile("stripe-completion.zsh")
		if err == nil {
			fmt.Printf("%s%s\n", instructionsHeader, zshCompletionInstructions)
		}
		return err
	case selected == "bash":
		fmt.Println("Detected `bash`, generating bash completion file: stripe-completion.bash")
		err := rootCmd.GenBashCompletionFile("stripe-completion.bash")
		if err == nil {
			if runtime.GOOS == "darwin" {
				fmt.Printf("%s%s\n", instructionsHeader, bashCompletionInstructionsMac)
			} else if runtime.GOOS == "linux" {
				fmt.Printf("%s%s\n", instructionsHeader, bashCompletionInstructionsLinux)
			}
		}
		return err
	default:
		return fmt.Errorf("Could not automatically detect your shell. Please run the command with the `--shell` flag for either bash or zsh")
	}
}

func detectShell() string {
	shell := os.Getenv("SHELL")

	switch {
	case strings.Contains(shell, "zsh"):
		return "zsh"
	case strings.Contains(shell, "bash"):
		return "bash"
	default:
		return ""
	}
}
