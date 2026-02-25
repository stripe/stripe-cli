// Package canary contains end-to-end tests that invoke the compiled Stripe CLI binary.
// These tests verify that the CLI works correctly when invoked as users would use it.
package canary

import (
	"fmt"
	"testing"

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

// sanitize removes sensitive data from output before logging.
// Always use this when logging command output in tests.
func sanitize(s string) string {
	return testutil.SanitizeOutput(s)
}

// logSanitized logs a message with sanitized output.
func logSanitized(t *testing.T, format string, args ...interface{}) {
	t.Helper()
	msg := fmt.Sprintf(format, args...)
	t.Log(sanitize(msg))
}

// fatalf logs a fatal error with sanitized output.
func fatalf(t *testing.T, format string, args ...interface{}) {
	t.Helper()
	msg := fmt.Sprintf(format, args...)
	t.Fatal(sanitize(msg))
}

// errorf logs an error with sanitized output.
func errorf(t *testing.T, format string, args ...interface{}) {
	t.Helper()
	msg := fmt.Sprintf(format, args...)
	t.Error(sanitize(msg))
}

// min returns the smaller of two integers.
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
