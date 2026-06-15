package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"

	"github.com/spf13/cobra"

	"github.com/stripe/stripe-cli/pkg/i18n"
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
		Short: i18n.T("completion.short"),
		Args:  validators.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return selectShell(cc.shell, cc.writeToStdout)
		},
	}

	cc.cmd.Flags().StringVar(&cc.shell, "shell", "", i18n.T("completion.flags.shell"))
	cc.cmd.Flags().BoolVar(&cc.writeToStdout, "write-to-stdout", false, i18n.T("completion.flags.write_to_stdout"))

	_ = cc.cmd.RegisterFlagCompletionFunc("shell", func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		return []string{"bash", "zsh", "fish"}, cobra.ShellCompDirectiveNoFileComp
	})

	return cc
}


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
			return fmt.Errorf("%s", i18n.Tf("completion.errors.unsupported_shell", i18n.Args{"shell": shell}))
		}
		return fmt.Errorf("%s", i18n.T("completion.errors.cannot_detect_shell"))
	}
}

func genZsh(writeToStdout bool, autoDetected bool) error {
	if writeToStdout {
		return rootCmd.GenZshCompletion(os.Stdout)
	}

	if autoDetected {
		fmt.Println(i18n.T("completion.output.zsh_detected"))
	} else {
		fmt.Println(i18n.T("completion.output.zsh_generating"))
	}

	err := rootCmd.GenZshCompletionFile("stripe-completion.zsh")
	if err == nil {
		fmt.Printf("%s%s\n", i18n.T("completion.output.instructions_header"), i18n.T("completion.output.zsh_instructions"))
	}

	return err
}

func genBash(writeToStdout bool, autoDetected bool) error {
	if writeToStdout {
		return rootCmd.GenBashCompletion(os.Stdout)
	}

	if autoDetected {
		fmt.Println(i18n.T("completion.output.bash_detected"))
	} else {
		fmt.Println(i18n.T("completion.output.bash_generating"))
	}

	err := rootCmd.GenBashCompletionFile("stripe-completion.bash")
	if err == nil {
		switch runtime.GOOS {
		case "darwin":
			fmt.Printf("%s%s\n", i18n.T("completion.output.instructions_header"), i18n.T("completion.output.bash_instructions_mac"))
		case "linux":
			fmt.Printf("%s%s\n", i18n.T("completion.output.instructions_header"), i18n.T("completion.output.bash_instructions_linux"))
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
		fmt.Println(i18n.T("completion.output.fish_detected"))
	} else {
		fmt.Println(i18n.T("completion.output.fish_generating"))
	}

	// true enables completion descriptions (fish displays them inline during tab-complete)
	err := rootCmd.GenFishCompletionFile("stripe.fish", true)
	if err == nil {
		fmt.Printf("%s%s\n", i18n.T("completion.output.instructions_header"), i18n.T("completion.output.fish_instructions"))
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
