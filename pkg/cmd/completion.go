package cmd

import (
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"github.com/stripe/stripe-cli/pkg/validators"
	"runtime"
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

var zsh_completion_instructions = `
Suggested next steps:
----------------------
1. Move ` + "`stripe-completion.zsh`" + `to the correct location:
    mkdir -p ~/.stripe
    mv stripe-completion.zsh ~/.stripe

2. Add the following lines to your ` + "`.zshrc`" + ` enabling shell completion for Stripe:
    fpath=(~/.stripe $fpath)
    autoload -Uz compinit && compinit -i

3. Source your ` + "`.zshrc`" + `or open a new terminal session:
    source ~/.zshrc
`

var bash_completion_instructions_mac = `
1. (If you've never installed bash completion before) Install bash completion on your system.
    brew install bash-completion

2. (If you've never installed bash completion before) Follow the post-instructions that brew displays, and add a line like the following to your bash profile:
    [[ -r "/usr/local/etc/profile.d/bash_completion.sh" ]] && . "/usr/local/etc/profile.d/bash_completion.sh"

3. Move ` + "`stripe-completion.bash`" + `to the correct location:
    mkdir -p ~/.stripe
    mv stripe-completion.bash ~/.stripe

4. Add the following line to your bash profile, so that Stripe autocompletion will be enabled every time you start a new terminal session.
    source ~/.stripe/stripe-completion.bash

5. And either restart your terminal, or run the following command in your current session, to enable immediately.
    source ~/.stripe/stripe-completion.bash
`

var bash_completion_instructions_linux = `
1. Make sure bash completion is installed on your system. Usually this means checking that a file such as ` + "`/etc/profile.d/bash_completion.sh`" + ` exists, and is sourced by your bash profile, but the location of this file varies across flavors of Linux.

2. Move ` + "`stripe-completion.bash`" + `to the correct location:
    mkdir -p ~/.stripe
    mv stripe-completion.bash ~/.stripe

3. Add the following line to your bash profile, so that Stripe autocompletion will be enabled every time you start a new terminal session.
    source ~/.stripe/stripe-completion.bash

4. And either restart your terminal, or run the following command in your current session, to enable immediately.
    source ~/.stripe/stripe-completion.bash
`

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
                  fmt.Println(zsh_completion_instructions)
                }
                return err
	case selected == "bash":
		fmt.Println("Detected `bash`, generating bash completion file: stripe-completion.bash")
                err := rootCmd.GenBashCompletionFile("stripe-completion.bash")
                if err == nil {
                  if runtime.GOOS == "darwin" {
                    fmt.Println(bash_completion_instructions_mac)
                  } else if runtime.GOOS == "linux" {
                    fmt.Println(bash_completion_instructions_linux)
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
