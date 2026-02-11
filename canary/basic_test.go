package canary

import (
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
		t.Fatalf("Failed to run 'stripe version': %v", err)
	}

	if result.ExitCode != 0 {
		t.Errorf("Expected exit code 0, got %d. Stderr: %s", result.ExitCode, result.Stderr)
	}

	// Version output should contain "stripe version"
	if !strings.Contains(result.Stdout, "stripe version") {
		t.Errorf("Expected output to contain 'stripe version', got: %s", result.Stdout)
	}
}

func TestOfflineVersionFlag(t *testing.T) {
	runner := getRunner(t)

	result, err := runner.Run("--version")
	if err != nil {
		t.Fatalf("Failed to run 'stripe --version': %v", err)
	}

	if result.ExitCode != 0 {
		t.Errorf("Expected exit code 0, got %d. Stderr: %s", result.ExitCode, result.Stderr)
	}

	// Should contain version information
	if !strings.Contains(result.Stdout, "stripe") {
		t.Errorf("Expected output to contain 'stripe', got: %s", result.Stdout)
	}
}

func TestOfflineVersionFormat(t *testing.T) {
	runner := getRunner(t)

	result, err := runner.Run("version")
	if err != nil {
		t.Fatalf("Failed to run 'stripe version': %v", err)
	}

	if result.ExitCode != 0 {
		t.Errorf("Expected exit code 0, got %d", result.ExitCode)
	}

	// Version should match pattern like "stripe version x.y.z" or contain version info
	versionPattern := regexp.MustCompile(`\d+\.\d+\.\d+`)
	if !versionPattern.MatchString(result.Stdout) {
		// May be a dev build without version, just warn
		t.Logf("Warning: Version output doesn't contain semver pattern: %s", result.Stdout)
	}
}

func TestOfflineHelp(t *testing.T) {
	runner := getRunner(t)

	result, err := runner.Run("--help")
	if err != nil {
		t.Fatalf("Failed to run 'stripe --help': %v", err)
	}

	if result.ExitCode != 0 {
		t.Errorf("Expected exit code 0, got %d. Stderr: %s", result.ExitCode, result.Stderr)
	}

	// Help output should list core commands
	expectedCommands := []string{"login", "listen", "trigger", "logs"}
	for _, cmd := range expectedCommands {
		if !strings.Contains(result.Stdout, cmd) {
			t.Errorf("Expected help output to contain '%s', got: %s", cmd, result.Stdout)
		}
	}
}

func TestOfflineHelpSubcommand(t *testing.T) {
	runner := getRunner(t)

	result, err := runner.Run("help")
	if err != nil {
		t.Fatalf("Failed to run 'stripe help': %v", err)
	}

	if result.ExitCode != 0 {
		t.Errorf("Expected exit code 0, got %d. Stderr: %s", result.ExitCode, result.Stderr)
	}

	// Help output should list core commands
	if !strings.Contains(result.Stdout, "listen") {
		t.Errorf("Expected help output to contain 'listen', got: %s", result.Stdout)
	}
}

// =============================================================================
// Completion Tests
// =============================================================================

func TestOfflineCompletionBash(t *testing.T) {
	runner := getRunner(t)

	result, err := runner.Run("completion", "--shell", "bash", "--write-to-stdout")
	if err != nil {
		t.Fatalf("Failed to run 'stripe completion --shell bash': %v", err)
	}

	if result.ExitCode != 0 {
		t.Errorf("Expected exit code 0, got %d. Stderr: %s", result.ExitCode, result.Stderr)
	}

	// Bash completion should contain shell functions
	if !strings.Contains(result.Stdout, "bash") && !strings.Contains(result.Stdout, "completion") {
		t.Errorf("Expected bash completion script, got: %s", result.Stdout)
	}
}

func TestOfflineCompletionZsh(t *testing.T) {
	runner := getRunner(t)

	result, err := runner.Run("completion", "--shell", "zsh", "--write-to-stdout")
	if err != nil {
		t.Fatalf("Failed to run 'stripe completion --shell zsh': %v", err)
	}

	if result.ExitCode != 0 {
		t.Errorf("Expected exit code 0, got %d. Stderr: %s", result.ExitCode, result.Stderr)
	}

	// Zsh completion should contain compdef or completion-related content
	if !strings.Contains(result.Stdout, "compdef") && !strings.Contains(result.Stdout, "zsh") {
		t.Errorf("Expected zsh completion script, got: %s", result.Stdout)
	}
}

func TestOfflineCompletionHelp(t *testing.T) {
	runner := getRunner(t)

	result, err := runner.Run("completion", "--help")
	if err != nil {
		t.Fatalf("Failed to run 'stripe completion --help': %v", err)
	}

	if result.ExitCode != 0 {
		t.Errorf("Expected exit code 0, got %d. Stderr: %s", result.ExitCode, result.Stderr)
	}

	// Help should mention shell flag
	if !strings.Contains(result.Stdout, "shell") {
		t.Errorf("Expected help to mention 'shell', got: %s", result.Stdout)
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
		t.Fatalf("Failed to create temp config dir: %v", err)
	}
	defer os.RemoveAll(configDir)

	runner = runner.WithConfigDir(configDir)

	result, err := runner.Run("status")
	if err != nil {
		t.Fatalf("Failed to run 'stripe status': %v", err)
	}

	// Status command should run (may fail if not logged in, but shouldn't crash)
	// Just verify it produces some output
	combinedOutput := result.Stdout + result.Stderr
	if combinedOutput == "" {
		t.Errorf("Expected some output from 'stripe status', got none")
	}
}

// =============================================================================
// Error Handling Tests
// =============================================================================

func TestOfflineUnknownCommand(t *testing.T) {
	runner := getRunner(t)

	result, err := runner.Run("nonexistent-command-xyz")
	if err != nil {
		t.Fatalf("Failed to run command: %v", err)
	}

	// Should return non-zero exit code for unknown command
	if result.ExitCode == 0 {
		t.Errorf("Expected non-zero exit code for unknown command, got 0")
	}

	// Should show an error message
	combinedOutput := result.Stdout + result.Stderr
	if !strings.Contains(strings.ToLower(combinedOutput), "unknown") &&
		!strings.Contains(strings.ToLower(combinedOutput), "invalid") &&
		!strings.Contains(strings.ToLower(combinedOutput), "not") {
		t.Errorf("Expected error message about unknown command, got: stdout=%s stderr=%s", result.Stdout, result.Stderr)
	}
}

func TestOfflineUnknownFlag(t *testing.T) {
	runner := getRunner(t)

	result, err := runner.Run("--nonexistent-flag-xyz")
	if err != nil {
		t.Fatalf("Failed to run command: %v", err)
	}

	// Should return non-zero exit code for unknown flag
	if result.ExitCode == 0 {
		t.Errorf("Expected non-zero exit code for unknown flag, got 0")
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
		t.Fatalf("Failed to run 'stripe': %v", err)
	}

	// Should either show help (exit 0) or error (non-zero)
	// Just verify it doesn't crash and produces output
	combinedOutput := result.Stdout + result.Stderr
	if combinedOutput == "" {
		t.Errorf("Expected some output from 'stripe' with no args, got none")
	}
}

func TestOfflineMultipleFlags(t *testing.T) {
	runner := getRunner(t)

	// Test combining multiple flags
	result, err := runner.Run("--help", "--version")
	if err != nil {
		t.Fatalf("Failed to run command: %v", err)
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
		t.Fatalf("Failed to run 'stripe listen --help': %v", err)
	}

	if result.ExitCode != 0 {
		t.Errorf("Expected exit code 0, got %d. Stderr: %s", result.ExitCode, result.Stderr)
	}

	// Should show listen-specific help
	if !strings.Contains(result.Stdout, "listen") {
		t.Errorf("Expected output to contain 'listen', got: %s", result.Stdout)
	}
}

func TestOfflineTriggerHelp(t *testing.T) {
	runner := getRunner(t)

	result, err := runner.Run("trigger", "--help")
	if err != nil {
		t.Fatalf("Failed to run 'stripe trigger --help': %v", err)
	}

	if result.ExitCode != 0 {
		t.Errorf("Expected exit code 0, got %d. Stderr: %s", result.ExitCode, result.Stderr)
	}

	// Should show trigger-specific help
	if !strings.Contains(result.Stdout, "trigger") {
		t.Errorf("Expected output to contain 'trigger', got: %s", result.Stdout)
	}
}
