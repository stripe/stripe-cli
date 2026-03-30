package canary

import (
	"encoding/json"
	"os"
	"regexp"
	"strings"
	"testing"

	"github.com/stripe/stripe-cli/canary/testutil"
)

// =============================================================================
// Version & Help Tests
// =============================================================================

func TestOfflineVersion(t *testing.T) {
	runner := getRunner(t)

	result, err := runner.Run("version")
	if err != nil {
		fatalf(t, "Failed to run 'stripe version': %v", err)
	}

	if result.ExitCode != 0 {
		errorf(t, "Expected exit code 0, got %d. Stderr: %s", result.ExitCode, result.Stderr)
	}

	// Version output should contain "stripe version"
	if !strings.Contains(result.Stdout, "stripe version") {
		errorf(t, "Expected output to contain 'stripe version', got: %s", result.Stdout)
	}
}

func TestOfflineVersionFlag(t *testing.T) {
	runner := getRunner(t)

	result, err := runner.Run("--version")
	if err != nil {
		fatalf(t, "Failed to run 'stripe --version': %v", err)
	}

	if result.ExitCode != 0 {
		errorf(t, "Expected exit code 0, got %d. Stderr: %s", result.ExitCode, result.Stderr)
	}

	// Should contain version information
	if !strings.Contains(result.Stdout, "stripe") {
		errorf(t, "Expected output to contain 'stripe', got: %s", result.Stdout)
	}
}

func TestOfflineVersionFormat(t *testing.T) {
	runner := getRunner(t)

	result, err := runner.Run("version")
	if err != nil {
		fatalf(t, "Failed to run 'stripe version': %v", err)
	}

	if result.ExitCode != 0 {
		errorf(t, "Expected exit code 0, got %d", result.ExitCode)
	}

	// Version should match pattern like "stripe version x.y.z" or contain version info
	versionPattern := regexp.MustCompile(`\d+\.\d+\.\d+`)
	if !versionPattern.MatchString(result.Stdout) {
		// May be a dev build without version, just warn
		logSanitizedf(t, "Warning: Version output doesn't contain semver pattern: %s", result.Stdout)
	}
}

func TestOfflineHelp(t *testing.T) {
	runner := getRunner(t)

	result, err := runner.Run("--help")
	if err != nil {
		fatalf(t, "Failed to run 'stripe --help': %v", err)
	}

	if result.ExitCode != 0 {
		errorf(t, "Expected exit code 0, got %d. Stderr: %s", result.ExitCode, result.Stderr)
	}

	// Help output should list core commands
	expectedCommands := []string{"login", "listen", "trigger", "logs"}
	for _, cmd := range expectedCommands {
		if !strings.Contains(result.Stdout, cmd) {
			errorf(t, "Expected help output to contain '%s', got: %s", cmd, result.Stdout)
		}
	}
}

func TestOfflineHelpSubcommand(t *testing.T) {
	runner := getRunner(t)

	result, err := runner.Run("help")
	if err != nil {
		fatalf(t, "Failed to run 'stripe help': %v", err)
	}

	if result.ExitCode != 0 {
		errorf(t, "Expected exit code 0, got %d. Stderr: %s", result.ExitCode, result.Stderr)
	}

	// Help output should list core commands
	if !strings.Contains(result.Stdout, "listen") {
		errorf(t, "Expected help output to contain 'listen', got: %s", result.Stdout)
	}
}

// =============================================================================
// Completion Tests
// =============================================================================

func TestOfflineCompletionBash(t *testing.T) {
	runner := getRunner(t)

	result, err := runner.Run("completion", "--shell", "bash", "--write-to-stdout")
	if err != nil {
		fatalf(t, "Failed to run 'stripe completion --shell bash': %v", err)
	}

	if result.ExitCode != 0 {
		errorf(t, "Expected exit code 0, got %d. Stderr: %s", result.ExitCode, result.Stderr)
	}

	// Bash completion should contain shell functions
	if !strings.Contains(result.Stdout, "bash") && !strings.Contains(result.Stdout, "completion") {
		errorf(t, "Expected bash completion script, got: %s", result.Stdout)
	}
}

func TestOfflineCompletionZsh(t *testing.T) {
	runner := getRunner(t)

	result, err := runner.Run("completion", "--shell", "zsh", "--write-to-stdout")
	if err != nil {
		fatalf(t, "Failed to run 'stripe completion --shell zsh': %v", err)
	}

	if result.ExitCode != 0 {
		errorf(t, "Expected exit code 0, got %d. Stderr: %s", result.ExitCode, result.Stderr)
	}

	// Zsh completion should contain compdef or completion-related content
	if !strings.Contains(result.Stdout, "compdef") && !strings.Contains(result.Stdout, "zsh") {
		errorf(t, "Expected zsh completion script, got: %s", result.Stdout)
	}
}

func TestOfflineCompletionHelp(t *testing.T) {
	runner := getRunner(t)

	result, err := runner.Run("completion", "--help")
	if err != nil {
		fatalf(t, "Failed to run 'stripe completion --help': %v", err)
	}

	if result.ExitCode != 0 {
		errorf(t, "Expected exit code 0, got %d. Stderr: %s", result.ExitCode, result.Stderr)
	}

	// Help should mention shell flag
	if !strings.Contains(result.Stdout, "shell") {
		errorf(t, "Expected help to mention 'shell', got: %s", result.Stdout)
	}
}

// =============================================================================
// Status Tests
// =============================================================================

func TestOfflineStatus(t *testing.T) {
	runner := getRunner(t)

	// Create isolated config directory
	configDir, err := testutil.CreateTempConfigDir("status")
	if err != nil {
		fatalf(t, "Failed to create temp config dir: %v", err)
	}
	defer os.RemoveAll(configDir)

	runner = runner.WithConfigDir(configDir)

	result, err := runner.Run("status")
	if err != nil {
		fatalf(t, "Failed to run 'stripe status': %v", err)
	}

	// Status command should run (may fail if not logged in, but shouldn't crash)
	// Just verify it produces some output
	combinedOutput := result.Stdout + result.Stderr
	if combinedOutput == "" {
		errorf(t, "Expected some output from 'stripe status', got none")
	}
}

// =============================================================================
// Error Handling Tests
// =============================================================================

func TestOfflineUnknownCommand(t *testing.T) {
	runner := getRunner(t)

	result, err := runner.Run("nonexistent-command-xyz")
	if err != nil {
		fatalf(t, "Failed to run command: %v", err)
	}

	// Should return non-zero exit code for unknown command
	if result.ExitCode == 0 {
		errorf(t, "Expected non-zero exit code for unknown command, got 0")
	}

	// Should show an error message
	combinedOutput := result.Stdout + result.Stderr
	if !strings.Contains(strings.ToLower(combinedOutput), "unknown") &&
		!strings.Contains(strings.ToLower(combinedOutput), "invalid") &&
		!strings.Contains(strings.ToLower(combinedOutput), "not") {
		errorf(t, "Expected error message about unknown command, got: stdout=%s stderr=%s", result.Stdout, result.Stderr)
	}
}

func TestOfflineUnknownFlag(t *testing.T) {
	runner := getRunner(t)

	result, err := runner.Run("--nonexistent-flag-xyz")
	if err != nil {
		fatalf(t, "Failed to run command: %v", err)
	}

	// Should return non-zero exit code for unknown flag
	if result.ExitCode == 0 {
		errorf(t, "Expected non-zero exit code for unknown flag, got 0")
	}
}

// =============================================================================
// Edge Cases
// =============================================================================

func TestOfflineEmptyArgs(t *testing.T) {
	runner := getRunner(t)

	// Running stripe with no args should show help or usage
	result, err := runner.Run()
	if err != nil {
		fatalf(t, "Failed to run 'stripe': %v", err)
	}

	// Should either show help (exit 0) or error (non-zero)
	// Just verify it doesn't crash and produces output
	combinedOutput := result.Stdout + result.Stderr
	if combinedOutput == "" {
		errorf(t, "Expected some output from 'stripe' with no args, got none")
	}
}

func TestOfflineMultipleFlags(t *testing.T) {
	runner := getRunner(t)

	// Test combining multiple flags
	result, err := runner.Run("--help", "--version")
	if err != nil {
		fatalf(t, "Failed to run command: %v", err)
	}

	// One of the flags should take precedence
	// Just verify it doesn't crash
	_ = result
}

func TestOfflineCommandWithHelp(t *testing.T) {
	runner := getRunner(t)

	// Test subcommand help
	result, err := runner.Run("listen", "--help")
	if err != nil {
		fatalf(t, "Failed to run 'stripe listen --help': %v", err)
	}

	if result.ExitCode != 0 {
		errorf(t, "Expected exit code 0, got %d. Stderr: %s", result.ExitCode, result.Stderr)
	}

	// Should show listen-specific help
	if !strings.Contains(result.Stdout, "listen") {
		errorf(t, "Expected output to contain 'listen', got: %s", result.Stdout)
	}
}

func TestOfflineTriggerHelp(t *testing.T) {
	runner := getRunner(t)

	result, err := runner.Run("trigger", "--help")
	if err != nil {
		fatalf(t, "Failed to run 'stripe trigger --help': %v", err)
	}

	if result.ExitCode != 0 {
		errorf(t, "Expected exit code 0, got %d. Stderr: %s", result.ExitCode, result.Stderr)
	}

	// Should show trigger-specific help
	if !strings.Contains(result.Stdout, "trigger") {
		errorf(t, "Expected output to contain 'trigger', got: %s", result.Stdout)
	}
}

// =============================================================================
// Map Tests
// =============================================================================

func TestOfflineMapRoot(t *testing.T) {
	runner := getRunner(t)

	result, err := runner.Run("--map")
	if err != nil {
		fatalf(t, "Failed to run 'stripe --map': %v", err)
	}

	if result.ExitCode != 0 {
		errorf(t, "Expected exit code 0, got %d. Stderr: %s", result.ExitCode, result.Stderr)
	}

	// Should contain the root command and core subcommands
	if !strings.Contains(result.Stdout, "stripe") {
		errorf(t, "Expected output to contain 'stripe', got: %s", result.Stdout)
	}

	// Should contain key commands in tree format with box-drawing characters
	expectedCommands := []string{"listen", "trigger", "login", "logs"}
	for _, cmd := range expectedCommands {
		if !strings.Contains(result.Stdout, cmd) {
			errorf(t, "Expected map output to contain '%s', got: %s", cmd, result.Stdout)
		}
	}

	// Should use box-drawing characters
	if !strings.Contains(result.Stdout, "├──") && !strings.Contains(result.Stdout, "└──") {
		errorf(t, "Expected tree output with box-drawing characters, got: %s", result.Stdout)
	}
}

func TestOfflineMapSubcommand(t *testing.T) {
	runner := getRunner(t)

	result, err := runner.Run("--map", "logs")
	if err != nil {
		fatalf(t, "Failed to run 'stripe --map logs': %v", err)
	}

	if result.ExitCode != 0 {
		errorf(t, "Expected exit code 0, got %d. Stderr: %s", result.ExitCode, result.Stderr)
	}

	// Root line should show the scoped command path
	if !strings.Contains(result.Stdout, "stripe logs") {
		errorf(t, "Expected scoped tree header 'stripe logs', got: %s", result.Stdout)
	}

	// Should contain the tail subcommand
	if !strings.Contains(result.Stdout, "tail") {
		errorf(t, "Expected map output to contain 'tail', got: %s", result.Stdout)
	}
}

func TestOfflineMapFlagAfterCommand(t *testing.T) {
	runner := getRunner(t)

	// --map after the command name should also work
	result, err := runner.Run("logs", "--map")
	if err != nil {
		fatalf(t, "Failed to run 'stripe logs --map': %v", err)
	}

	if result.ExitCode != 0 {
		errorf(t, "Expected exit code 0, got %d. Stderr: %s", result.ExitCode, result.Stderr)
	}

	if !strings.Contains(result.Stdout, "stripe logs") {
		errorf(t, "Expected scoped tree header 'stripe logs', got: %s", result.Stdout)
	}
}

func TestOfflineMapUnknownCommand(t *testing.T) {
	runner := getRunner(t)

	// --map with unknown command should warn and show root tree
	result, err := runner.Run("--map", "nonexistent-xyz")
	if err != nil {
		fatalf(t, "Failed to run 'stripe --map nonexistent-xyz': %v", err)
	}

	if result.ExitCode != 0 {
		errorf(t, "Expected exit code 0, got %d. Stderr: %s", result.ExitCode, result.Stderr)
	}

	// Should show root tree as fallback
	if !strings.Contains(result.Stdout, "stripe") {
		errorf(t, "Expected root tree fallback, got: %s", result.Stdout)
	}

	// Should show warning on stderr
	if !strings.Contains(result.Stderr, "Unknown command") {
		errorf(t, "Expected 'Unknown command' warning on stderr, got: %s", result.Stderr)
	}
}

func TestOfflineMapInvalidModeShowsError(t *testing.T) {
	runner := getRunner(t)

	// --map=false is not a valid mode and should show an error
	result, err := runner.Run("--map=false")
	if err != nil {
		fatalf(t, "Failed to run 'stripe --map=false': %v", err)
	}

	// Should NOT produce tree output
	if strings.Contains(result.Stdout, "├──") || strings.Contains(result.Stdout, "└──") {
		errorf(t, "Expected --map=false to NOT produce tree output, got: %s", result.Stdout)
	}

	// Should show unknown mode error on stderr
	if !strings.Contains(result.Stderr, "Unknown --map mode") {
		errorf(t, "Expected unknown mode error on stderr, got: %s", result.Stderr)
	}
}

func TestOfflineMapCompact(t *testing.T) {
	runner := getRunner(t)

	result, err := runner.Run("--map=compact")
	if err != nil {
		fatalf(t, "Failed to run 'stripe --map=compact': %v", err)
	}

	if result.ExitCode != 0 {
		errorf(t, "Expected exit code 0, got %d. Stderr: %s", result.ExitCode, result.Stderr)
	}

	// Should contain tree structure
	if !strings.Contains(result.Stdout, "stripe") {
		errorf(t, "Expected compact tree output, got: %s", result.Stdout)
	}

	if !strings.Contains(result.Stdout, "├──") && !strings.Contains(result.Stdout, "└──") {
		errorf(t, "Expected tree with box-drawing characters, got: %s", result.Stdout)
	}

	// Compact mode should have shorter lines than tree mode (no descriptions).
	// Verify a known command name appears without its description on the same line.
	// We check that "listen" appears but its description doesn't follow it.
	lines := strings.Split(result.Stdout, "\n")
	for _, line := range lines {
		if strings.Contains(line, "listen") {
			// In tree mode, the line would contain "listen  Listen for..." — in compact, just "listen"
			if strings.Contains(line, "Listen for") {
				errorf(t, "Compact mode should not include descriptions, but found: %s", line)
			}
			break
		}
	}
}

func TestOfflineMapCompactSubcommand(t *testing.T) {
	runner := getRunner(t)

	result, err := runner.Run("issuing", "--map=compact")
	if err != nil {
		fatalf(t, "Failed to run 'stripe issuing --map=compact': %v", err)
	}

	if result.ExitCode != 0 {
		errorf(t, "Expected exit code 0, got %d. Stderr: %s", result.ExitCode, result.Stderr)
	}

	if !strings.Contains(result.Stdout, "stripe issuing") {
		errorf(t, "Expected scoped compact tree, got: %s", result.Stdout)
	}
}

func TestOfflineMapPaths(t *testing.T) {
	runner := getRunner(t)

	result, err := runner.Run("--map=paths")
	if err != nil {
		fatalf(t, "Failed to run 'stripe --map=paths': %v", err)
	}

	if result.ExitCode != 0 {
		errorf(t, "Expected exit code 0, got %d. Stderr: %s", result.ExitCode, result.Stderr)
	}

	// Should be a flat list of paths, no tree characters
	if strings.Contains(result.Stdout, "├──") || strings.Contains(result.Stdout, "└──") {
		errorf(t, "Paths mode should not contain tree characters, got: %s", result.Stdout)
	}

	// Each line should start with "stripe"
	lines := strings.Split(strings.TrimSpace(result.Stdout), "\n")
	if len(lines) == 0 {
		fatalf(t, "Expected at least one path line, got none")
	}
	for _, line := range lines {
		if !strings.HasPrefix(line, "stripe ") {
			errorf(t, "Expected each path to start with 'stripe ', got: %s", line)
		}
	}

	// Should contain at least some known commands
	if !strings.Contains(result.Stdout, "stripe listen") {
		errorf(t, "Expected 'stripe listen' in paths output, got: %s", result.Stdout)
	}
}

func TestOfflineMapJSON(t *testing.T) {
	runner := getRunner(t)

	result, err := runner.Run("--map=json")
	if err != nil {
		fatalf(t, "Failed to run 'stripe --map=json': %v", err)
	}

	if result.ExitCode != 0 {
		errorf(t, "Expected exit code 0, got %d. Stderr: %s", result.ExitCode, result.Stderr)
	}

	// Should be valid JSON
	var parsed map[string]interface{}
	if err := json.Unmarshal([]byte(result.Stdout), &parsed); err != nil {
		errorf(t, "Expected valid JSON output, got parse error: %v\nOutput: %s", err, result.Stdout)
	}

	// Root should have "name" field
	name, ok := parsed["name"]
	if !ok {
		errorf(t, "Expected JSON to have 'name' field, got: %s", result.Stdout)
	}
	if name != "stripe" {
		errorf(t, "Expected root name 'stripe', got: %v", name)
	}

	// Root should have "commands" array
	commands, ok := parsed["commands"]
	if !ok {
		errorf(t, "Expected JSON to have 'commands' field, got: %s", result.Stdout)
	}
	cmdArr, ok := commands.([]interface{})
	if !ok || len(cmdArr) == 0 {
		errorf(t, "Expected non-empty commands array, got: %v", commands)
	}
}

func TestOfflineMapInvalidMode(t *testing.T) {
	runner := getRunner(t)

	result, err := runner.Run("--map=invalid")
	if err != nil {
		fatalf(t, "Failed to run 'stripe --map=invalid': %v", err)
	}

	// Should print error to stderr about unknown mode
	if !strings.Contains(result.Stderr, "Unknown --map mode") {
		errorf(t, "Expected 'Unknown --map mode' on stderr, got: %s", result.Stderr)
	}

	// Should NOT produce tree output (falls through to normal CLI behavior)
	if strings.Contains(result.Stdout, "├──") || strings.Contains(result.Stdout, "└──") {
		errorf(t, "Expected no tree output for invalid mode, got: %s", result.Stdout)
	}
}
