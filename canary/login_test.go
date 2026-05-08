package canary

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/stripe/stripe-cli/canary/testutil"
)

// =============================================================================
// Login & Authentication Tests
// Tests for login help and non-interactive authentication methods (--api-key flag).
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

// TestAPIGlobalAPIKeyFlag verifies that the --api-key global flag provides
// non-interactive authentication without env vars or config files.
func TestAPIGlobalAPIKeyFlag(t *testing.T) {
	requireAPIKey(t)

	// Do NOT use WithEnv for STRIPE_API_KEY — we're testing the flag alone
	runner := getRunner(t)

	result, err := runner.Run("balance", "retrieve", "--api-key", testutil.GetAPIKey())
	if err != nil {
		fatalf(t, "Failed to run 'stripe balance retrieve --api-key ...': %v", err)
	}

	if result.ExitCode != 0 {
		errorf(t, "Expected exit code 0, got %d. Stderr: %s", result.ExitCode, result.Stderr)
	}

	// Should return valid JSON with balance info
	var data map[string]interface{}
	if err := json.Unmarshal([]byte(result.Stdout), &data); err != nil {
		errorf(t, "Output is not valid JSON: %v. Output: %s", err, result.Stdout)
	}

	if _, ok := data["available"]; !ok {
		errorf(t, "Expected balance response with 'available' field, got: %s", result.Stdout)
	}
}
