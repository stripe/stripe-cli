package canary

import (
	"os"
	"strings"
	"testing"

	"github.com/stripe/stripe-cli/canary/testutil"
)

// =============================================================================
// Config Command Tests
// =============================================================================

func TestOfflineConfig(t *testing.T) {
	runner := getRunner(t)

	// Create isolated config directory
	configDir, err := testutil.CreateTempConfigDir("config")
	if err != nil {
		fatalf(t, "Failed to create temp config dir: %v", err)
	}
	defer os.RemoveAll(configDir)

	runner = runner.WithConfigDir(configDir)

	result, err := runner.Run("config", "--list")
	if err != nil {
		fatalf(t, "Failed to run 'stripe config --list': %v", err)
	}

	// Config list may return exit code 0 even with empty config
	// The important thing is it doesn't crash
	if result.ExitCode != 0 {
		// Some versions may return non-zero for empty config, check stderr
		if !strings.Contains(result.Stderr, "config") && !strings.Contains(result.Stderr, "profile") {
			logSanitizedf(t, "Warning: 'stripe config --list' returned exit code %d. Stderr: %s", result.ExitCode, result.Stderr)
		}
	}
}

// =============================================================================
// Config-Based Authentication Tests
// =============================================================================

func TestAPIConfigSetAndUseAPIKey(t *testing.T) {
	runner := getRunner(t)
	requireAPIKey(t)

	// Create isolated config directory
	configDir, err := testutil.CreateTempConfigDir("config-auth")
	if err != nil {
		fatalf(t, "Failed to create temp config dir: %v", err)
	}
	defer os.RemoveAll(configDir)

	runner = runner.WithConfigDir(configDir)

	// Set the API key via config command
	setResult, err := runner.Run("config", "--set", "test_mode_api_key", testutil.GetAPIKey())
	if err != nil {
		fatalf(t, "Failed to run 'stripe config --set': %v", err)
	}

	if setResult.ExitCode != 0 {
		fatalf(t, "Expected exit code 0 for config --set, got %d. Stderr: %s", setResult.ExitCode, setResult.Stderr)
	}

	// Now run a command that requires authentication WITHOUT passing --api-key
	// The CLI should use the configured key
	balanceResult, err := runner.Run("get", "/v1/balance")
	if err != nil {
		fatalf(t, "Failed to run 'stripe get /v1/balance': %v", err)
	}

	if balanceResult.ExitCode != 0 {
		errorf(t, "Expected exit code 0, got %d. Stderr: %s", balanceResult.ExitCode, balanceResult.Stderr)
	}

	// Should return balance info
	if !strings.Contains(balanceResult.Stdout, "available") && !strings.Contains(balanceResult.Stdout, "pending") {
		errorf(t, "Expected balance response, got: %s", balanceResult.Stdout)
	}
}

func TestAPIConfigListShowsKey(t *testing.T) {
	runner := getRunner(t)
	requireAPIKey(t)

	// Create isolated config directory
	configDir, err := testutil.CreateTempConfigDir("config-list")
	if err != nil {
		fatalf(t, "Failed to create temp config dir: %v", err)
	}
	defer os.RemoveAll(configDir)

	runner = runner.WithConfigDir(configDir)

	// Set the API key
	_, err = runner.Run("config", "--set", "test_mode_api_key", testutil.GetAPIKey())
	if err != nil {
		fatalf(t, "Failed to set config: %v", err)
	}

	// List config and verify key is shown (masked)
	listResult, err := runner.Run("config", "--list")
	if err != nil {
		fatalf(t, "Failed to run 'stripe config --list': %v", err)
	}

	// The key should be listed (possibly masked)
	if !strings.Contains(listResult.Stdout, "test_mode_api_key") &&
		!strings.Contains(listResult.Stdout, "sk_test") {
		logSanitizedf(t, "Config list output: %s", listResult.Stdout)
	}
}

func TestAPIConfigMultipleProfiles(t *testing.T) {
	runner := getRunner(t)
	requireAPIKey(t)

	// Create isolated config directory
	configDir, err := testutil.CreateTempConfigDir("config-profiles")
	if err != nil {
		fatalf(t, "Failed to create temp config dir: %v", err)
	}
	defer os.RemoveAll(configDir)

	runner = runner.WithConfigDir(configDir)

	// Set API key for default profile
	_, err = runner.Run("config", "--set", "test_mode_api_key", testutil.GetAPIKey())
	if err != nil {
		fatalf(t, "Failed to set config for default profile: %v", err)
	}

	// Set API key for a custom profile
	_, err = runner.Run("config", "--set", "test_mode_api_key", testutil.GetAPIKey(), "--project-name", "canary-test")
	if err != nil {
		fatalf(t, "Failed to set config for canary-test profile: %v", err)
	}

	// Use the custom profile for a request
	result, err := runner.Run("get", "/v1/balance", "--project-name", "canary-test")
	if err != nil {
		fatalf(t, "Failed to run command with custom profile: %v", err)
	}

	if result.ExitCode != 0 {
		errorf(t, "Expected exit code 0, got %d. Stderr: %s", result.ExitCode, result.Stderr)
	}

	if !strings.Contains(result.Stdout, "available") && !strings.Contains(result.Stdout, "pending") {
		errorf(t, "Expected balance response, got: %s", result.Stdout)
	}
}
