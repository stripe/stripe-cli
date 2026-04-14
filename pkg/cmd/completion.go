package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"

	"github.com/spf13/cobra"

	"github.com/stripe/stripe-cli/pkg/validators"
)

type completionCmd struct {
	cmd *cobra.Command

	shell         string
	writeToStdout bool
}

func newCompletionCmd() *completionCmd {
	cc := &completionCmd{}

	cc.cmd = &cobra.Command{
		Use:   "completion",
		Short: "Generate bash, zsh, and fish completion scripts",
		Args:  validators.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return selectShell(cc.shell, cc.writeToStdout)
		},
	}

	cc.cmd.Flags().StringVar(&cc.shell, "shell", "", "Shell to generate completions for: bash, zsh, or fish (auto-detected if omitted)")
	cc.cmd.Flags().BoolVar(&cc.writeToStdout, "write-to-stdout", false, "Print completion script to stdout rather than creating a new file.")

	_ = cc.cmd.RegisterFlagCompletionFunc("shell", func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		return []string{"bash", "zsh", "fish"}, cobra.ShellCompDirectiveNoFileComp
	})

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

	fishCompletionInstructions = `
1. Move ` + "`stripe.fish`" + ` to the fish completions directory:
    mkdir -p ~/.config/fish/completions
    mv stripe.fish ~/.config/fish/completions/stripe.fish

Fish automatically loads completions from this directory, so no additional
configuration is needed. Open a new terminal session and completions will
be available.`
)

func selectShell(shell string, writeToStdout bool) error {
	selected := shell
	autoDetected := false
	if selected == "" {
		selected = detectShell()
		autoDetected = selected != ""
	}

	switch selected {
	case "zsh":
		return genZsh(writeToStdout, autoDetected)
	case "bash":
		return genBash(writeToStdout, autoDetected)
	case "fish":
		return genFish(writeToStdout, autoDetected)
	default:
		if shell != "" {
			return fmt.Errorf("unsupported shell %q; supported shells are: bash, zsh, fish", shell)
		}
		return fmt.Errorf("could not automatically detect your shell; please run the command with the --shell flag for bash, zsh, or fish")
	}
}

func genZsh(writeToStdout bool, autoDetected bool) error {
	if writeToStdout {
		return rootCmd.GenZshCompletion(os.Stdout)
	}

	if autoDetected {
		fmt.Println("Detected `zsh`, generating zsh completion file: stripe-completion.zsh")
	} else {
		fmt.Println("Generating zsh completion file: stripe-completion.zsh")
	}

	err := rootCmd.GenZshCompletionFile("stripe-completion.zsh")
	if err == nil {
		fmt.Printf("%s%s\n", instructionsHeader, zshCompletionInstructions)
	}

	return err
}

func genBash(writeToStdout bool, autoDetected bool) error {
	if writeToStdout {
		return rootCmd.GenBashCompletion(os.Stdout)
	}

	if autoDetected {
		fmt.Println("Detected `bash`, generating bash completion file: stripe-completion.bash")
	} else {
		fmt.Println("Generating bash completion file: stripe-completion.bash")
	}

	err := rootCmd.GenBashCompletionFile("stripe-completion.bash")
	if err == nil {
		switch runtime.GOOS {
		case "darwin":
			fmt.Printf("%s%s\n", instructionsHeader, bashCompletionInstructionsMac)
		case "linux":
			fmt.Printf("%s%s\n", instructionsHeader, bashCompletionInstructionsLinux)
		}
	}

	return err
}

func genFish(writeToStdout bool, autoDetected bool) error {
	if writeToStdout {
		// true enables completion descriptions (fish displays them inline during tab-complete)
		return rootCmd.GenFishCompletion(os.Stdout, true)
	}

	if autoDetected {
		fmt.Println("Detected `fish`, generating fish completion file: stripe.fish")
	} else {
		fmt.Println("Generating fish completion file: stripe.fish")
	}

	// true enables completion descriptions (fish displays them inline during tab-complete)
	err := rootCmd.GenFishCompletionFile("stripe.fish", true)
	if err == nil {
		fmt.Printf("%s%s\n", instructionsHeader, fishCompletionInstructions)
	}

	return err
}

func detectShell() string {
	shell := filepath.Base(os.Getenv("SHELL"))

	switch shell {
	case "zsh", "bash", "fish":
		return shell
	default:
		return ""
	}
}
