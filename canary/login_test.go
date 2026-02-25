package canary

import (
	"strings"
	"testing"
)

// =============================================================================
// Login Command Tests (Offline only - login requires browser)
// =============================================================================

func TestOfflineLoginHelp(t *testing.T) {
	runner := getRunner(t)

	result, err := runner.Run("login", "--help")
	if err != nil {
		fatalf(t, "Failed to run 'stripe login --help': %v", err)
	}

	if result.ExitCode != 0 {
		errorf(t, "Expected exit code 0, got %d. Stderr: %s", result.ExitCode, result.Stderr)
	}

	// Should show login-specific help
	if !strings.Contains(result.Stdout, "Login") && !strings.Contains(result.Stdout, "login") {
		errorf(t, "Expected help to mention 'login', got: %s", result.Stdout)
	}

	// Should mention interactive flag
	if !strings.Contains(result.Stdout, "interactive") {
		errorf(t, "Expected help to mention 'interactive' flag, got: %s", result.Stdout)
	}
}
