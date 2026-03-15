package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"

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

// sentinelBegin and sentinelEnd mark the completion configuration block
// in shell config files (~/.zshrc, ~/.bashrc, ~/.bash_profile). This allows
// safe idempotent install/uninstall without corrupting the user's existing config.
const (
	sentinelBegin = "# begin stripe-completion — managed by stripe cli, do not edit"
	sentinelEnd   = "# end stripe-completion"
)

// addSentinelBlock adds or replaces a sentinel-delimited block in the given
// config file. If the file does not exist, it is created with mode 0644.
// Existing file permissions are preserved. The operation is idempotent:
// calling it twice with the same line produces the same result as calling
// it once. If the file contains orphaned or reversed markers, a new block
// is appended rather than attempting to repair the malformed state.
func addSentinelBlock(configPath, line string) error {
	block := fmt.Sprintf("%s\n%s\n%s", sentinelBegin, line, sentinelEnd)

	data, err := os.ReadFile(configPath)
	if err != nil {
		if os.IsNotExist(err) {
			// Create new file with just the sentinel block
			return os.WriteFile(configPath, []byte(block+"\n"), 0644)
		}
		return err
	}

	// Preserve existing file permissions
	perm := os.FileMode(0644)
	if info, statErr := os.Stat(configPath); statErr == nil {
		perm = info.Mode().Perm()
	}

	content := string(data)

	// Replace existing block if both markers are present in the correct order.
	// Orphaned or reversed markers are left untouched — we append instead.
	beginIdx := strings.Index(content, sentinelBegin)
	endIdx := strings.Index(content, sentinelEnd)
	if beginIdx >= 0 && endIdx >= 0 && endIdx > beginIdx {
		endIdx += len(sentinelEnd)
		// Include trailing newline if present
		if endIdx < len(content) && content[endIdx] == '\n' {
			endIdx++
		}
		content = content[:beginIdx] + block + "\n" + content[endIdx:]
		return os.WriteFile(configPath, []byte(content), perm)
	}

	// Append sentinel block
	if len(content) > 0 && !strings.HasSuffix(content, "\n") {
		content += "\n"
	}
	content += block + "\n"

	return os.WriteFile(configPath, []byte(content), perm)
}

// removeSentinelBlock removes the sentinel-delimited block from the given
// config file. If the file does not exist, this is a no-op. If the markers
// are orphaned or reversed, the file is left unchanged. Existing file
// permissions are preserved.
func removeSentinelBlock(configPath string) error {
	data, err := os.ReadFile(configPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}

	content := string(data)

	beginIdx := strings.Index(content, sentinelBegin)
	endIdx := strings.Index(content, sentinelEnd)
	if beginIdx < 0 || endIdx < 0 || endIdx <= beginIdx {
		// No valid sentinel block found, nothing to do
		return nil
	}

	// Preserve existing file permissions
	perm := os.FileMode(0644)
	if info, statErr := os.Stat(configPath); statErr == nil {
		perm = info.Mode().Perm()
	}

	endIdx += len(sentinelEnd)
	// Include trailing newline if present
	if endIdx < len(content) && content[endIdx] == '\n' {
		endIdx++
	}

	content = content[:beginIdx] + content[endIdx:]

	return os.WriteFile(configPath, []byte(content), perm)
}

// manualRemnant represents a line in a shell config file that references the
// completion script but is outside our sentinel-managed block.
type manualRemnant struct {
	lineNumber int    // 1-based, for display in user-facing warnings
	lineText   string // trimmed content of the matching line
}

// findManualRemnants scans a shell config file for lines referencing the
// completion script filename that are outside our sentinel block. This detects
// manually-added source/load lines that the user may need to clean up.
//
// Lines inside the sentinel block, blank lines, and comment lines (starting
// with #) are excluded from the scan. Returns nil if the file cannot be read
// or no matches are found.
func findManualRemnants(configPath, scriptFilename string) []manualRemnant {
	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil
	}

	var remnants []manualRemnant
	inSentinelBlock := false

	for i, line := range strings.Split(string(data), "\n") {
		trimmed := strings.TrimSpace(line)

		if trimmed == sentinelBegin {
			inSentinelBlock = true
			continue
		}
		if trimmed == sentinelEnd {
			inSentinelBlock = false
			continue
		}

		if inSentinelBlock {
			continue
		}

		// Skip blank lines and comments
		if trimmed == "" || strings.HasPrefix(trimmed, "#") {
			continue
		}

		if strings.Contains(trimmed, scriptFilename) {
			remnants = append(remnants, manualRemnant{
				lineNumber: i + 1,
				lineText:   trimmed,
			})
		}
	}

	return remnants
}
