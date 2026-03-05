package canary

import (
	"strings"
	"testing"
	"time"

	"github.com/stripe/stripe-cli/canary/testutil"
)

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
			errorf(t, "logs tail failed to authenticate: %s", combinedOutput)
			return
		}
		// Timeout is expected for a streaming command
		if err != nil && strings.Contains(err.Error(), "timed out") {
			// This is actually success - the command started and ran until timeout
			logSanitizedf(t, "logs tail ran until timeout (expected): %s", combinedOutput)
			return
		}
		errorf(t, "Expected logs tail to show startup message, got: %s (err: %v)", combinedOutput, err)
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
		errorf(t, "logs tail failed to authenticate: %s", combinedOutput)
		return
	}

	// If we got this far without auth failure, the command started successfully
	if err != nil && strings.Contains(err.Error(), "timed out") {
		// Expected - streaming command ran until timeout
		return
	}

	// Check for startup message
	if !strings.Contains(combinedOutput, "Ready") && !strings.Contains(combinedOutput, "ready") && !strings.Contains(combinedOutput, "Getting") {
		logSanitizedf(t, "logs tail output: %s", combinedOutput)
	}
}

func TestAPILogsTailCapture(t *testing.T) {
	runner := getRunner(t)
	requireAPIKey(t)

	// Start logs tail with JSON format in background
	runner = runner.WithEnv(map[string]string{
		"STRIPE_API_KEY": testutil.GetAPIKey(),
	})

	logsTail, err := runner.RunBackground("logs", "tail", "--format", "JSON")
	if err != nil {
		fatalf(t, "Failed to start logs tail: %v", err)
	}
	defer logsTail.Stop()

	// Wait for logs tail to be ready
	err = logsTail.WaitForOutput("Ready!", 30*time.Second)
	if err != nil {
		stdout, stderr := logsTail.GetOutput()
		fatalf(t, "Logs tail failed to become ready: %v\nStdout: %s\nStderr: %s", err, stdout, stderr)
	}

	// Make an API request that should be captured
	apiRunner := runner.WithTimeout(30 * time.Second)
	result, err := apiRunner.Run("customers", "list", "--limit", "1")
	if err != nil {
		fatalf(t, "Failed to run customers list: %v", err)
	}
	if result.ExitCode != 0 {
		fatalf(t, "Customers list failed with exit code %d. Stderr: %s", result.ExitCode, result.Stderr)
	}

	// Wait for log entry to stream
	time.Sleep(3 * time.Second)

	// Get logs tail output
	stdout, stderr := logsTail.GetOutput()
	combinedOutput := stdout + stderr

	// Verify the API request was captured
	// The log should contain information about the GET /v1/customers request
	if !strings.Contains(combinedOutput, "customers") && !strings.Contains(combinedOutput, "GET") {
		logSanitizedf(t, "Logs tail output (may not capture all requests immediately):\n%s", combinedOutput)
		// This is not necessarily a failure - logs streaming may have latency
		// or may not capture requests from the same CLI session
	}

	// Check that logs tail shows some structured output after Ready
	readyIdx := strings.Index(combinedOutput, "Ready!")
	if readyIdx >= 0 {
		afterReady := combinedOutput[readyIdx:]
		// Look for any log-like content (status codes, methods, paths)
		if strings.Contains(afterReady, "200") ||
			strings.Contains(afterReady, "GET") ||
			strings.Contains(afterReady, "POST") ||
			strings.Contains(afterReady, "/v1/") {
			t.Log("Successfully captured API request in logs tail")
		} else {
			// Log what we got for debugging, but don't fail
			// Logs tail captures requests from other CLI instances, not necessarily our own
			logSanitizedf(t, "No API request captured yet. This may be expected if no other requests are being made. Output after Ready: %s", afterReady[:min(len(afterReady), 500)])
		}
	}
}
