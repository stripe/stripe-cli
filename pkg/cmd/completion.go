package cmd

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/spf13/cobra"

	"github.com/stripe/stripe-cli/pkg/ansi"
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
		Long:  "Generate shell completion scripts. Use --install to automatically configure your shell profile, or run without flags to generate a script file manually.",
		Example: `  # Auto-install completions (detects your shell)
  stripe completion --install

  # Install for a specific shell
  stripe completion --install --shell zsh

  # Remove installed completions
  stripe completion --uninstall

  # Generate completion script to stdout
  stripe completion --shell bash --write-to-stdout`,
		Args: validators.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			shell := cc.shell
			if shell == "" {
				shell = detectShell()
			}

			if cc.install || cc.uninstall {
				if shell == "" {
					return fmt.Errorf("could not automatically detect your shell. Please run the command with the `--shell` flag for bash, zsh, or fish")
				}
				if shell != "bash" && shell != "zsh" && shell != "fish" {
					return fmt.Errorf("unsupported shell %q. Supported shells: bash, zsh, fish", shell)
				}
				if cc.install {
					return installCompletion(shell, os.UserHomeDir)
				}
				return uninstallCompletion(shell, os.UserHomeDir)
			}

			return selectShell(shell, cc.writeToStdout)
		},
	}

	cc.cmd.Flags().StringVar(&cc.shell, "shell", "", "Shell to generate completions for: bash, zsh, or fish (auto-detected if omitted)")
	cc.cmd.Flags().BoolVar(&cc.writeToStdout, "write-to-stdout", false, "Print completion script to stdout rather than creating a new file.")
	cc.cmd.Flags().BoolVar(&cc.install, "install", false, "Install completion script to ~/.stripe and configure your shell profile automatically")
	cc.cmd.Flags().BoolVar(&cc.uninstall, "uninstall", false, "Remove installed completion script and configuration from your shell profile")
	cc.cmd.MarkFlagsMutuallyExclusive("install", "uninstall")

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
		return fmt.Sprintf("source \"%s\"", scriptPath)
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

	// For bash/zsh, add source line to shell config (with diff preview + confirmation)
	if shell != "fish" {
		configPath := getShellConfigFile(shell, homeDir)
		line := sourceLine(shell, scriptPath)

		oldContent, perm, err := readConfigFile(configPath)
		if err != nil {
			return fmt.Errorf("could not read %s: %w", configPath, err)
		}

		newContent := computeAddSentinel(oldContent, line)

		if newContent != oldContent {
			ansi.RenderDiff(os.Stdout, configPath, oldContent, newContent)

			if !confirm(installConfirmFn, "Apply changes?") {
				fmt.Printf("Aborted. Completion script was written to %s but your shell config was not modified.\nTo activate manually, add this line to %s:\n  %s\n", scriptPath, configPath, line)
				return nil
			}

			if err := os.WriteFile(configPath, []byte(newContent), perm); err != nil {
				return fmt.Errorf("could not update %s: %w", configPath, err)
			}

			fmt.Printf("Completion installed for %s.\nScript written to: %s\nShell config updated: %s\nRestart your shell or run: %s\n", shell, scriptPath, configPath, line)
		} else {
			fmt.Printf("Completion already configured in %s.\nScript updated: %s\n", configPath, scriptPath)
		}

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

	// For bash/zsh, remove sentinel block from shell config (with diff preview + confirmation)
	if shell != "fish" {
		configPath := getShellConfigFile(shell, homeDir)

		oldContent, perm, err := readConfigFile(configPath)
		if err != nil {
			return fmt.Errorf("could not read %s: %w", configPath, err)
		}

		newContent, found := computeRemoveSentinel(oldContent)

		if found {
			ansi.RenderDiff(os.Stdout, configPath, oldContent, newContent)

			if !confirm(uninstallConfirmFn, "Apply changes?") {
				fmt.Printf("Aborted. Completion script was removed but your shell config was not modified.\nTo clean up manually, remove the block between \"%s\" and \"%s\" in %s.\n", sentinelBegin, sentinelEnd, configPath)
				return nil
			}

			if err := os.WriteFile(configPath, []byte(newContent), perm); err != nil {
				return fmt.Errorf("could not update %s: %w", configPath, err)
			}
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

// readConfigFile reads a shell config file and returns its content and
// permissions. If the file does not exist, returns ("", 0644, nil).
// Uses Open+Fstat to read content and permissions atomically from the
// same file descriptor.
func readConfigFile(path string) (string, os.FileMode, error) {
	f, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return "", 0644, nil
		}
		return "", 0, err
	}
	defer f.Close()

	info, err := f.Stat()
	if err != nil {
		return "", 0, err
	}

	data, err := io.ReadAll(f)
	if err != nil {
		return "", 0, err
	}

	return string(data), info.Mode().Perm(), nil
}

// computeAddSentinel returns the content with a sentinel block added or
// replaced. Pure function — no I/O. If the file contains orphaned or reversed
// markers, a new block is appended rather than attempting to repair.
func computeAddSentinel(content, line string) string {
	block := fmt.Sprintf("%s\n%s\n%s", sentinelBegin, line, sentinelEnd)

	beginIdx := strings.Index(content, sentinelBegin)
	endIdx := strings.Index(content, sentinelEnd)
	if beginIdx >= 0 && endIdx >= 0 && endIdx > beginIdx {
		endIdx += len(sentinelEnd)
		if endIdx < len(content) && content[endIdx] == '\n' {
			endIdx++
		}
		return content[:beginIdx] + block + "\n" + content[endIdx:]
	}

	if len(content) > 0 && !strings.HasSuffix(content, "\n") {
		content += "\n"
	}

	return content + block + "\n"
}

// computeRemoveSentinel returns the content with the sentinel block removed.
// Pure function — no I/O. Returns (result, true) if a block was found and
// removed, or ("", false) if no valid block exists.
func computeRemoveSentinel(content string) (string, bool) {
	beginIdx := strings.Index(content, sentinelBegin)
	endIdx := strings.Index(content, sentinelEnd)
	if beginIdx < 0 || endIdx < 0 || endIdx <= beginIdx {
		return "", false
	}

	endIdx += len(sentinelEnd)
	if endIdx < len(content) && content[endIdx] == '\n' {
		endIdx++
	}

	return content[:beginIdx] + content[endIdx:], true
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

// installConfirmFn and uninstallConfirmFn override the confirm prompt used
// during install/uninstall. nil means use the default stdin prompt.
// Override in tests to avoid blocking on stdin.
var (
	installConfirmFn   func(string) bool
	uninstallConfirmFn func(string) bool
)

// confirm calls fn if non-nil, otherwise falls back to defaultConfirm.
func confirm(fn func(string) bool, question string) bool {
	if fn != nil {
		return fn(question)
	}
	return defaultConfirm(question)
}

// defaultConfirm asks a yes/no question on stdout/stdin.
func defaultConfirm(question string) bool {
	fmt.Printf("%s [y/N] ", question)

	scanner := bufio.NewScanner(os.Stdin)
	if scanner.Scan() {
		answer := strings.TrimSpace(strings.ToLower(scanner.Text()))
		return answer == "y" || answer == "yes"
	}

	return false
}

func pluralize(n int, singular, plural string) string {
	if n == 1 {
		return singular
	}
	return plural
}
