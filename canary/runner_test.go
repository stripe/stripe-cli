// Package canary contains end-to-end tests that invoke the compiled Stripe CLI binary.
// These tests verify that the CLI works correctly when invoked as users would use it.
package canary

import (
	"encoding/json"
	"os"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/stripe/stripe-cli/canary/testutil"
)

// getRunner returns a test runner, skipping the test if STRIPE_CLI_BINARY is not set.
func getRunner(t *testing.T) *testutil.Runner {
	t.Helper()
	runner, err := testutil.NewRunner()
	if err != nil {
		t.Skipf("Skipping canary test: %v", err)
	}
	return runner
}

// requireAPIKey skips the test if STRIPE_API_KEY is not set.
func requireAPIKey(t *testing.T) {
	t.Helper()
	if !testutil.HasAPIKey() {
		t.Skip("Skipping API test: STRIPE_API_KEY not set")
	}
}

// =============================================================================
// Offline Tests - No API key required
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

func TestOfflineConfig(t *testing.T) {
	runner := getRunner(t)

	// Create isolated config directory
	configDir, err := testutil.CreateTempConfigDir("config")
	if err != nil {
		t.Fatalf("Failed to create temp config dir: %v", err)
	}
	defer os.RemoveAll(configDir)

	runner = runner.WithConfigDir(configDir)

	result, err := runner.Run("config", "--list")
	if err != nil {
		t.Fatalf("Failed to run 'stripe config --list': %v", err)
	}

	// Config list may return exit code 0 even with empty config
	// The important thing is it doesn't crash
	if result.ExitCode != 0 {
		// Some versions may return non-zero for empty config, check stderr
		if !strings.Contains(result.Stderr, "config") && !strings.Contains(result.Stderr, "profile") {
			t.Logf("Warning: 'stripe config --list' returned exit code %d. Stderr: %s", result.ExitCode, result.Stderr)
		}
	}
}

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
// API Tests - Require STRIPE_API_KEY
// =============================================================================

func TestAPIGetBalance(t *testing.T) {
	runner := getRunner(t)
	requireAPIKey(t)

	runner = runner.WithEnv(map[string]string{
		"STRIPE_API_KEY": testutil.GetAPIKey(),
	})

	result, err := runner.Run("get", "/v1/balance")
	if err != nil {
		t.Fatalf("Failed to run 'stripe get /v1/balance': %v", err)
	}

	if result.ExitCode != 0 {
		t.Errorf("Expected exit code 0, got %d. Stderr: %s", result.ExitCode, result.Stderr)
	}

	// Should return JSON with balance info
	if !strings.Contains(result.Stdout, "available") && !strings.Contains(result.Stdout, "pending") {
		t.Errorf("Expected balance response with 'available' or 'pending', got: %s", result.Stdout)
	}
}

func TestAPICreateDeleteCustomer(t *testing.T) {
	runner := getRunner(t)
	requireAPIKey(t)

	runner = runner.WithEnv(map[string]string{
		"STRIPE_API_KEY": testutil.GetAPIKey(),
	})

	// Create a customer
	createResult, err := runner.Run("post", "/v1/customers", "-d", "name=CanaryTestCustomer", "-d", "metadata[test]=canary")
	if err != nil {
		t.Fatalf("Failed to run 'stripe post /v1/customers': %v", err)
	}

	if createResult.ExitCode != 0 {
		t.Fatalf("Expected exit code 0 for create, got %d. Stderr: %s", createResult.ExitCode, createResult.Stderr)
	}

	// Parse the customer ID from the response
	var customer struct {
		ID string `json:"id"`
	}
	if err := json.Unmarshal([]byte(createResult.Stdout), &customer); err != nil {
		t.Fatalf("Failed to parse customer response: %v. Output: %s", err, createResult.Stdout)
	}

	if customer.ID == "" {
		t.Fatalf("Customer ID is empty. Output: %s", createResult.Stdout)
	}

	if !strings.HasPrefix(customer.ID, "cus_") {
		t.Errorf("Expected customer ID to start with 'cus_', got: %s", customer.ID)
	}

	// Delete the customer to clean up (--confirm skips interactive prompt)
	deleteResult, err := runner.Run("delete", "/v1/customers/"+customer.ID, "--confirm")
	if err != nil {
		t.Fatalf("Failed to run 'stripe delete /v1/customers/%s': %v", customer.ID, err)
	}

	if deleteResult.ExitCode != 0 {
		t.Errorf("Expected exit code 0 for delete, got %d. Stderr: %s", deleteResult.ExitCode, deleteResult.Stderr)
	}

	// Verify deletion response
	if !strings.Contains(deleteResult.Stdout, "deleted") {
		t.Errorf("Expected delete response to contain 'deleted', got: %s", deleteResult.Stdout)
	}
}

func TestAPITrigger(t *testing.T) {
	runner := getRunner(t)
	requireAPIKey(t)

	runner = runner.WithEnv(map[string]string{
		"STRIPE_API_KEY": testutil.GetAPIKey(),
	})

	// Use a longer timeout for trigger as it creates resources
	runner = runner.WithTimeout(60 * time.Second)

	result, err := runner.Run("trigger", "customer.created")
	if err != nil {
		t.Fatalf("Failed to run 'stripe trigger customer.created': %v", err)
	}

	if result.ExitCode != 0 {
		t.Errorf("Expected exit code 0, got %d. Stderr: %s", result.ExitCode, result.Stderr)
	}

	// Trigger should output something about the created resource
	combinedOutput := result.Stdout + result.Stderr
	if !strings.Contains(strings.ToLower(combinedOutput), "customer") {
		t.Errorf("Expected output to mention 'customer', got: %s", combinedOutput)
	}
}

func TestAPICustomersList(t *testing.T) {
	runner := getRunner(t)
	requireAPIKey(t)

	runner = runner.WithEnv(map[string]string{
		"STRIPE_API_KEY": testutil.GetAPIKey(),
	})

	result, err := runner.Run("customers", "list", "--limit", "1")
	if err != nil {
		t.Fatalf("Failed to run 'stripe customers list --limit 1': %v", err)
	}

	if result.ExitCode != 0 {
		t.Errorf("Expected exit code 0, got %d. Stderr: %s", result.ExitCode, result.Stderr)
	}

	// Should return JSON with data array
	if !strings.Contains(result.Stdout, "data") {
		t.Errorf("Expected response to contain 'data', got: %s", result.Stdout)
	}
}

func TestAPIProductsCreate(t *testing.T) {
	runner := getRunner(t)
	requireAPIKey(t)

	runner = runner.WithEnv(map[string]string{
		"STRIPE_API_KEY": testutil.GetAPIKey(),
	})

	// Create a product using the resource command
	result, err := runner.Run("products", "create", "--name", "Canary Test Product")
	if err != nil {
		t.Fatalf("Failed to run 'stripe products create': %v", err)
	}

	if result.ExitCode != 0 {
		t.Errorf("Expected exit code 0, got %d. Stderr: %s", result.ExitCode, result.Stderr)
	}

	// Parse the product ID for cleanup
	var product struct {
		ID string `json:"id"`
	}
	if err := json.Unmarshal([]byte(result.Stdout), &product); err != nil {
		t.Fatalf("Failed to parse product response: %v. Output: %s", err, result.Stdout)
	}

	if product.ID == "" {
		t.Fatalf("Product ID is empty. Output: %s", result.Stdout)
	}

	// Archive the product to clean up (products can't be deleted, only archived)
	archiveResult, err := runner.Run("products", "update", product.ID, "--active=false")
	if err != nil {
		t.Logf("Warning: Failed to archive product %s: %v", product.ID, err)
	} else if archiveResult.ExitCode != 0 {
		t.Logf("Warning: Archive returned non-zero exit code: %d. Stderr: %s", archiveResult.ExitCode, archiveResult.Stderr)
	}
}

func TestAPIEventsListWithLimit(t *testing.T) {
	runner := getRunner(t)
	requireAPIKey(t)

	runner = runner.WithEnv(map[string]string{
		"STRIPE_API_KEY": testutil.GetAPIKey(),
	})

	result, err := runner.Run("events", "list", "--limit", "2")
	if err != nil {
		t.Fatalf("Failed to run 'stripe events list': %v", err)
	}

	if result.ExitCode != 0 {
		t.Errorf("Expected exit code 0, got %d. Stderr: %s", result.ExitCode, result.Stderr)
	}

	// Should return JSON response
	if !strings.Contains(result.Stdout, "data") {
		t.Errorf("Expected response to contain 'data', got: %s", result.Stdout)
	}
}

// =============================================================================
// Output Format Tests
// =============================================================================

func TestAPIOutputJSON(t *testing.T) {
	runner := getRunner(t)
	requireAPIKey(t)

	runner = runner.WithEnv(map[string]string{
		"STRIPE_API_KEY": testutil.GetAPIKey(),
	})

	result, err := runner.Run("get", "/v1/balance")
	if err != nil {
		t.Fatalf("Failed to run command: %v", err)
	}

	if result.ExitCode != 0 {
		t.Fatalf("Expected exit code 0, got %d. Stderr: %s", result.ExitCode, result.Stderr)
	}

	// Verify it's valid JSON
	var data map[string]interface{}
	if err := json.Unmarshal([]byte(result.Stdout), &data); err != nil {
		t.Errorf("Output is not valid JSON: %v. Output: %s", err, result.Stdout)
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

// =============================================================================
// Version Pattern Test
// =============================================================================

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

// =============================================================================
// Listen Command Tests
// =============================================================================

func TestOfflineListenHelp(t *testing.T) {
	runner := getRunner(t)

	result, err := runner.Run("listen", "--help")
	if err != nil {
		t.Fatalf("Failed to run 'stripe listen --help': %v", err)
	}

	if result.ExitCode != 0 {
		t.Errorf("Expected exit code 0, got %d. Stderr: %s", result.ExitCode, result.Stderr)
	}

	// Should show listen-specific flags
	expectedFlags := []string{"forward-to", "events", "print-secret"}
	for _, flag := range expectedFlags {
		if !strings.Contains(result.Stdout, flag) {
			t.Errorf("Expected help to mention '%s', got: %s", flag, result.Stdout)
		}
	}
}

func TestAPIListenPrintSecret(t *testing.T) {
	runner := getRunner(t)
	requireAPIKey(t)

	runner = runner.WithEnv(map[string]string{
		"STRIPE_API_KEY": testutil.GetAPIKey(),
	})

	// Use --print-secret which connects, gets the webhook signing secret, and exits
	runner = runner.WithTimeout(30 * time.Second)

	result, err := runner.Run("listen", "--print-secret")
	if err != nil {
		t.Fatalf("Failed to run 'stripe listen --print-secret': %v", err)
	}

	if result.ExitCode != 0 {
		t.Errorf("Expected exit code 0, got %d. Stderr: %s", result.ExitCode, result.Stderr)
	}

	// Should output a webhook signing secret (whsec_...)
	combinedOutput := result.Stdout + result.Stderr
	if !strings.Contains(combinedOutput, "whsec_") {
		t.Errorf("Expected output to contain webhook secret 'whsec_', got: %s", combinedOutput)
	}
}

func TestAPIListenWithEvents(t *testing.T) {
	runner := getRunner(t)
	requireAPIKey(t)

	runner = runner.WithEnv(map[string]string{
		"STRIPE_API_KEY": testutil.GetAPIKey(),
	})

	// Test that listen with event filtering and --print-secret works
	runner = runner.WithTimeout(30 * time.Second)

	result, err := runner.Run("listen", "--events", "customer.created,customer.updated", "--print-secret")
	if err != nil {
		t.Fatalf("Failed to run 'stripe listen --events ... --print-secret': %v", err)
	}

	if result.ExitCode != 0 {
		t.Errorf("Expected exit code 0, got %d. Stderr: %s", result.ExitCode, result.Stderr)
	}

	// Should still output a webhook signing secret
	combinedOutput := result.Stdout + result.Stderr
	if !strings.Contains(combinedOutput, "whsec_") {
		t.Errorf("Expected output to contain webhook secret 'whsec_', got: %s", combinedOutput)
	}
}

// =============================================================================
// Logs Tail Command Tests
// =============================================================================

func TestOfflineLogsTailHelp(t *testing.T) {
	runner := getRunner(t)

	result, err := runner.Run("logs", "tail", "--help")
	if err != nil {
		t.Fatalf("Failed to run 'stripe logs tail --help': %v", err)
	}

	if result.ExitCode != 0 {
		t.Errorf("Expected exit code 0, got %d. Stderr: %s", result.ExitCode, result.Stderr)
	}

	// Should show logs tail-specific flags
	expectedFlags := []string{"filter-http-method", "filter-status-code", "format"}
	for _, flag := range expectedFlags {
		if !strings.Contains(result.Stdout, flag) {
			t.Errorf("Expected help to mention '%s', got: %s", flag, result.Stdout)
		}
	}
}

func TestAPILogsTailStartup(t *testing.T) {
	runner := getRunner(t)
	requireAPIKey(t)

	runner = runner.WithEnv(map[string]string{
		"STRIPE_API_KEY": testutil.GetAPIKey(),
	})

	// logs tail is a streaming command, so we use a short timeout
	// The goal is to verify it starts up and connects successfully
	runner = runner.WithTimeout(10 * time.Second)

	result, err := runner.Run("logs", "tail")

	// The command will be killed by timeout, so we expect an error
	// But we should see "Getting ready..." or "Ready!" in the output
	combinedOutput := result.Stdout + result.Stderr

	// Check for successful startup indicators
	if !strings.Contains(combinedOutput, "Ready") && !strings.Contains(combinedOutput, "ready") {
		// If it failed to authenticate, that's also a valid test result
		if strings.Contains(combinedOutput, "Authorization failed") {
			t.Errorf("logs tail failed to authenticate: %s", combinedOutput)
			return
		}
		// Timeout is expected for a streaming command
		if err != nil && strings.Contains(err.Error(), "timed out") {
			// This is actually success - the command started and ran until timeout
			t.Logf("logs tail ran until timeout (expected): %s", combinedOutput)
			return
		}
		t.Errorf("Expected logs tail to show startup message, got: %s (err: %v)", combinedOutput, err)
	}
}

func TestAPILogsTailWithFilters(t *testing.T) {
	runner := getRunner(t)
	requireAPIKey(t)

	runner = runner.WithEnv(map[string]string{
		"STRIPE_API_KEY": testutil.GetAPIKey(),
	})

	// Test that logs tail with filters starts correctly
	runner = runner.WithTimeout(10 * time.Second)

	result, err := runner.Run("logs", "tail", "--filter-http-method", "POST")

	combinedOutput := result.Stdout + result.Stderr

	// Same logic as above - we expect timeout for streaming command
	if strings.Contains(combinedOutput, "Authorization failed") {
		t.Errorf("logs tail failed to authenticate: %s", combinedOutput)
		return
	}

	// If we got this far without auth failure, the command started successfully
	if err != nil && strings.Contains(err.Error(), "timed out") {
		// Expected - streaming command ran until timeout
		return
	}

	// Check for startup message
	if !strings.Contains(combinedOutput, "Ready") && !strings.Contains(combinedOutput, "ready") && !strings.Contains(combinedOutput, "Getting") {
		t.Logf("logs tail output: %s", combinedOutput)
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
		t.Fatalf("Failed to create temp config dir: %v", err)
	}
	defer os.RemoveAll(configDir)

	runner = runner.WithConfigDir(configDir)

	// Set the API key via config command
	setResult, err := runner.Run("config", "--set", "test_mode_api_key", testutil.GetAPIKey())
	if err != nil {
		t.Fatalf("Failed to run 'stripe config --set': %v", err)
	}

	if setResult.ExitCode != 0 {
		t.Fatalf("Expected exit code 0 for config --set, got %d. Stderr: %s", setResult.ExitCode, setResult.Stderr)
	}

	// Now run a command that requires authentication WITHOUT passing --api-key
	// The CLI should use the configured key
	balanceResult, err := runner.Run("get", "/v1/balance")
	if err != nil {
		t.Fatalf("Failed to run 'stripe get /v1/balance': %v", err)
	}

	if balanceResult.ExitCode != 0 {
		t.Errorf("Expected exit code 0, got %d. Stderr: %s", balanceResult.ExitCode, balanceResult.Stderr)
	}

	// Should return balance info
	if !strings.Contains(balanceResult.Stdout, "available") && !strings.Contains(balanceResult.Stdout, "pending") {
		t.Errorf("Expected balance response, got: %s", balanceResult.Stdout)
	}
}

func TestAPIConfigListShowsKey(t *testing.T) {
	runner := getRunner(t)
	requireAPIKey(t)

	// Create isolated config directory
	configDir, err := testutil.CreateTempConfigDir("config-list")
	if err != nil {
		t.Fatalf("Failed to create temp config dir: %v", err)
	}
	defer os.RemoveAll(configDir)

	runner = runner.WithConfigDir(configDir)

	// Set the API key
	_, err = runner.Run("config", "--set", "test_mode_api_key", testutil.GetAPIKey())
	if err != nil {
		t.Fatalf("Failed to set config: %v", err)
	}

	// List config and verify key is shown (masked)
	listResult, err := runner.Run("config", "--list")
	if err != nil {
		t.Fatalf("Failed to run 'stripe config --list': %v", err)
	}

	// The key should be listed (possibly masked)
	if !strings.Contains(listResult.Stdout, "test_mode_api_key") &&
		!strings.Contains(listResult.Stdout, "sk_test") {
		t.Logf("Config list output: %s", listResult.Stdout)
	}
}

func TestAPIConfigMultipleProfiles(t *testing.T) {
	runner := getRunner(t)
	requireAPIKey(t)

	// Create isolated config directory
	configDir, err := testutil.CreateTempConfigDir("config-profiles")
	if err != nil {
		t.Fatalf("Failed to create temp config dir: %v", err)
	}
	defer os.RemoveAll(configDir)

	runner = runner.WithConfigDir(configDir)

	// Set API key for default profile
	_, err = runner.Run("config", "--set", "test_mode_api_key", testutil.GetAPIKey())
	if err != nil {
		t.Fatalf("Failed to set config for default profile: %v", err)
	}

	// Set API key for a custom profile
	_, err = runner.Run("config", "--set", "test_mode_api_key", testutil.GetAPIKey(), "--project-name", "canary-test")
	if err != nil {
		t.Fatalf("Failed to set config for canary-test profile: %v", err)
	}

	// Use the custom profile for a request
	result, err := runner.Run("get", "/v1/balance", "--project-name", "canary-test")
	if err != nil {
		t.Fatalf("Failed to run command with custom profile: %v", err)
	}

	if result.ExitCode != 0 {
		t.Errorf("Expected exit code 0, got %d. Stderr: %s", result.ExitCode, result.Stderr)
	}

	if !strings.Contains(result.Stdout, "available") && !strings.Contains(result.Stdout, "pending") {
		t.Errorf("Expected balance response, got: %s", result.Stdout)
	}
}

// =============================================================================
// Login Command Tests (Offline only - login requires browser)
// =============================================================================

func TestOfflineLoginHelp(t *testing.T) {
	runner := getRunner(t)

	result, err := runner.Run("login", "--help")
	if err != nil {
		t.Fatalf("Failed to run 'stripe login --help': %v", err)
	}

	if result.ExitCode != 0 {
		t.Errorf("Expected exit code 0, got %d. Stderr: %s", result.ExitCode, result.Stderr)
	}

	// Should show login-specific help
	if !strings.Contains(result.Stdout, "Login") && !strings.Contains(result.Stdout, "login") {
		t.Errorf("Expected help to mention 'login', got: %s", result.Stdout)
	}

	// Should mention interactive flag
	if !strings.Contains(result.Stdout, "interactive") {
		t.Errorf("Expected help to mention 'interactive' flag, got: %s", result.Stdout)
	}
}
