package cmd

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/spf13/cobra"

	"github.com/stripe/stripe-cli/pkg/validators"
)

// sentinelBegin and sentinelEnd mark the completion configuration block
// in shell config files (~/.zshrc, ~/.bashrc, ~/.bash_profile). This allows
// safe idempotent install/uninstall without corrupting the user's existing config.
const (
	sentinelBegin = "# begin stripe-completion"
	sentinelEnd   = "# end stripe-completion"
)

type completionCmd struct {
	cmd *cobra.Command

	shell         string
	writeToStdout bool
	install       bool
	uninstall     bool
}

func newCompletionCmd() *completionCmd {
	cc := &completionCmd{}

	cc.cmd = &cobra.Command{
		Use:   "completion",
		Short: "Generate bash, zsh, and fish completion scripts",
		Args:  validators.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			shell := cc.shell
			if shell == "" {
				shell = detectShell()
			}

			if cc.install || cc.uninstall {
				if shell == "" {
					return fmt.Errorf("could not automatically detect your shell. Please run the command with the `--shell` flag for bash, zsh, or fish")
				}
				if cc.install {
					return installCompletion(shell, os.UserHomeDir)
				}
				return uninstallCompletion(shell, os.UserHomeDir)
			}

			return selectShell(shell, cc.writeToStdout)
		},
	}

	cc.cmd.Flags().StringVar(&cc.shell, "shell", "", "The shell to generate completion commands for. Supports \"bash\", \"zsh\", or \"fish\"")
	cc.cmd.Flags().BoolVar(&cc.writeToStdout, "write-to-stdout", false, "Print completion script to stdout rather than creating a new file.")
	cc.cmd.Flags().BoolVar(&cc.install, "install", false, "Install completion script to ~/.stripe and configure your shell profile automatically")
	cc.cmd.Flags().BoolVar(&cc.uninstall, "uninstall", false, "Remove installed completion script and configuration from your shell profile")
	cc.cmd.MarkFlagsMutuallyExclusive("install", "uninstall")

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
	if selected == "" {
		selected = detectShell()
	}

	switch {
	case selected == "zsh":
		return genZsh(writeToStdout)
	case selected == "bash":
		return genBash(writeToStdout)
	case selected == "fish":
		return genFish(writeToStdout)
	default:
		return fmt.Errorf("Could not automatically detect your shell. Please run the command with the `--shell` flag for bash, zsh, or fish")
	}
}

func genZsh(writeToStdout bool) error {
	if writeToStdout {
		return rootCmd.GenZshCompletion(os.Stdout)
	}

	fmt.Println("Detected `zsh`, generating zsh completion file: stripe-completion.zsh")

	err := rootCmd.GenZshCompletionFile("stripe-completion.zsh")
	if err == nil {
		fmt.Printf("%s%s\n", instructionsHeader, zshCompletionInstructions)
	}

	return err
}

func genBash(writeToStdout bool) error {
	if writeToStdout {
		return rootCmd.GenBashCompletion(os.Stdout)
	}

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
}

func genFish(writeToStdout bool) error {
	if writeToStdout {
		// true enables completion descriptions (fish displays them inline during tab-complete)
		return rootCmd.GenFishCompletion(os.Stdout, true)
	}

	fmt.Println("Detected `fish`, generating fish completion file: stripe.fish")

	// true enables completion descriptions (fish displays them inline during tab-complete)
	err := rootCmd.GenFishCompletionFile("stripe.fish", true)
	if err == nil {
		fmt.Printf("%s%s\n", instructionsHeader, fishCompletionInstructions)
	}

	return err
}

func detectShell() string {
	shell := os.Getenv("SHELL")

	switch {
	case strings.Contains(shell, "zsh"):
		return "zsh"
	case strings.Contains(shell, "bash"):
		return "bash"
	case strings.Contains(shell, "fish"):
		return "fish"
	default:
		return ""
	}
}

// ---------------------------------------------------------------------------
// Auto-install/uninstall support
// ---------------------------------------------------------------------------

// getCompletionScriptDir returns the directory where completion scripts are stored.
func getCompletionScriptDir(homeDir string) string {
	return filepath.Join(homeDir, ".stripe")
}

// getShellConfigFile returns the path to the shell's configuration file.
// For fish, returns "" because fish auto-loads completions from a directory
// (~/.config/fish/completions/) and does not require a config file entry.
func getShellConfigFile(shell, homeDir string) string {
	switch shell {
	case "bash":
		if runtime.GOOS == "darwin" {
			return filepath.Join(homeDir, ".bash_profile")
		}
		return filepath.Join(homeDir, ".bashrc")
	case "zsh":
		return filepath.Join(homeDir, ".zshrc")
	default:
		return ""
	}
}

// getFishCompletionsDir returns the directory where fish completions are stored.
func getFishCompletionsDir(homeDir string) string {
	return filepath.Join(homeDir, ".config", "fish", "completions")
}

// completionScriptFilename returns the filename for the completion script.
// Fish uses "stripe.fish" (matching the command name) rather than
// "stripe-completion.fish" because fish auto-loads completions from
// ~/.config/fish/completions/ based on command name.
func completionScriptFilename(shell string) string {
	switch shell {
	case "bash":
		return "stripe-completion.bash"
	case "zsh":
		return "stripe-completion.zsh"
	case "fish":
		return "stripe.fish"
	default:
		return ""
	}
}

// generateCompletionScript writes the completion script for the given shell into buf.
func generateCompletionScript(shell string, buf *bytes.Buffer) error {
	switch shell {
	case "bash":
		return rootCmd.GenBashCompletionV2(buf, true)
	case "zsh":
		return rootCmd.GenZshCompletion(buf)
	case "fish":
		return rootCmd.GenFishCompletion(buf, true)
	default:
		return fmt.Errorf("unsupported shell: %s", shell)
	}
}

// sourceLine returns the shell-specific line that loads the completion script.
func sourceLine(shell, scriptPath string) string {
	switch shell {
	case "bash", "zsh":
		return fmt.Sprintf("source %s", scriptPath)
	default:
		return ""
	}
}

// homeDirFunc is a function type that returns the user's home directory.
// Enables dependency injection during testing (see completion_test.go).
type homeDirFunc func() (string, error)

func installCompletion(shell string, getHomeDir homeDirFunc) error {
	homeDir, err := getHomeDir()
	if err != nil {
		return fmt.Errorf("could not determine home directory: %w", err)
	}

	// Determine script destination
	var scriptDir string
	if shell == "fish" {
		scriptDir = getFishCompletionsDir(homeDir)
	} else {
		scriptDir = getCompletionScriptDir(homeDir)
	}

	// Create directory
	if err := os.MkdirAll(scriptDir, 0755); err != nil {
		return fmt.Errorf("could not create directory %s: %w", scriptDir, err)
	}

	// Generate completion script
	var buf bytes.Buffer
	if err := generateCompletionScript(shell, &buf); err != nil {
		return fmt.Errorf("could not generate %s completion script: %w", shell, err)
	}

	// Write script file
	scriptPath := filepath.Join(scriptDir, completionScriptFilename(shell))
	if err := os.WriteFile(scriptPath, buf.Bytes(), 0644); err != nil {
		return fmt.Errorf("could not write completion script to %s: %w", scriptPath, err)
	}

	// For bash/zsh, add source line to shell config
	if shell != "fish" {
		configPath := getShellConfigFile(shell, homeDir)
		line := sourceLine(shell, scriptPath)
		if err := addSentinelBlock(configPath, line); err != nil {
			return fmt.Errorf("could not update %s: %w", configPath, err)
		}
		fmt.Printf("Completion installed for %s.\nScript written to: %s\nShell config updated: %s\nRestart your shell or run: %s\n", shell, scriptPath, configPath, line)

		// Warn about manually-added lines outside our sentinel block
		remnants := findManualRemnants(configPath, completionScriptFilename(shell))
		warnManualRemnants(configPath, remnants)
	} else {
		fmt.Printf("Completion installed for fish.\nScript written to: %s\nRestart your shell or open a new terminal session.\n", scriptPath)
	}

	return nil
}

func uninstallCompletion(shell string, getHomeDir homeDirFunc) error {
	homeDir, err := getHomeDir()
	if err != nil {
		return fmt.Errorf("could not determine home directory: %w", err)
	}

	// Determine script location
	var scriptPath string
	if shell == "fish" {
		scriptPath = filepath.Join(getFishCompletionsDir(homeDir), completionScriptFilename(shell))
	} else {
		scriptPath = filepath.Join(getCompletionScriptDir(homeDir), completionScriptFilename(shell))
	}

	// Remove script file (ignore if doesn't exist)
	if err := os.Remove(scriptPath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("could not remove completion script %s: %w", scriptPath, err)
	}

	// For bash/zsh, remove sentinel block from shell config
	if shell != "fish" {
		configPath := getShellConfigFile(shell, homeDir)
		if err := removeSentinelBlock(configPath); err != nil {
			return fmt.Errorf("could not update %s: %w", configPath, err)
		}

		fmt.Printf("Completion uninstalled for %s.\n", shell)

		// Warn about manually-added lines that survive uninstall
		remnants := findManualRemnants(configPath, completionScriptFilename(shell))
		if len(remnants) > 0 {
			fmt.Printf("\nWarning: your shell config file %s still references the completion script outside the managed block:\n", configPath)
			for _, r := range remnants {
				fmt.Printf("  line %d: %s\n", r.lineNumber, r.lineText)
			}
			fmt.Printf("Remove %s manually to fully disable shell completion.\n", pluralize(len(remnants), "this line", "these lines"))
		}
	} else {
		fmt.Printf("Completion uninstalled for %s.\n", shell)
	}

	return nil
}

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

// warnManualRemnants prints a warning about manually-added completion lines
// found outside the sentinel block. Does nothing if remnants is empty.
func warnManualRemnants(configPath string, remnants []manualRemnant) {
	if len(remnants) == 0 {
		return
	}

	fmt.Printf("\nWarning: found a manually-added completion reference outside the managed block in %s:\n", configPath)
	for _, r := range remnants {
		fmt.Printf("  line %d: %s\n", r.lineNumber, r.lineText)
	}
	fmt.Printf("You may want to remove %s manually to avoid loading completions twice.\n", pluralize(len(remnants), "this line", "these lines"))
}

func pluralize(n int, singular, plural string) string {
	if n == 1 {
		return singular
	}
	return plural
}
