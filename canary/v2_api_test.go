package canary

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/stripe/stripe-cli/canary/testutil"
)

// =============================================================================
// V2 API Resource Tests - Require STRIPE_API_KEY
// =============================================================================

func TestAPIV2CoreEventsList(t *testing.T) {
	requireAPIKey(t)
	runner := getRunner(t, testutil.WithEnv(map[string]string{
		"STRIPE_API_KEY": testutil.GetAPIKey(),
	}))

	result, err := runner.Run("v2", "core", "events", "list", "--limit", "2")
	if err != nil {
		fatalf(t, "Failed to run 'stripe v2 core events list': %v", err)
	}

	if result.ExitCode != 0 {
		errorf(t, "Expected exit code 0, got %d. Stderr: %s", result.ExitCode, result.Stderr)
	}

	var data map[string]interface{}
	if err := json.Unmarshal([]byte(result.Stdout), &data); err != nil {
		errorf(t, "Output is not valid JSON: %v. Output: %s", err, result.Stdout)
	}

	if _, ok := data["data"]; !ok {
		errorf(t, "Expected response to contain 'data' field, got: %s", result.Stdout)
	}
}

func TestAPIV2CoreEventDestinationsList(t *testing.T) {
	requireAPIKey(t)
	runner := getRunner(t, testutil.WithEnv(map[string]string{
		"STRIPE_API_KEY": testutil.GetAPIKey(),
	}))

	result, err := runner.Run("v2", "core", "event_destinations", "list")
	if err != nil {
		fatalf(t, "Failed to run 'stripe v2 core event_destinations list': %v", err)
	}

	if result.ExitCode != 0 {
		errorf(t, "Expected exit code 0, got %d. Stderr: %s", result.ExitCode, result.Stderr)
	}

	var data map[string]interface{}
	if err := json.Unmarshal([]byte(result.Stdout), &data); err != nil {
		errorf(t, "Output is not valid JSON: %v. Output: %s", err, result.Stdout)
	}

	if _, ok := data["data"]; !ok {
		errorf(t, "Expected response to contain 'data' field, got: %s", result.Stdout)
	}
}

func TestAPIV2BillingMeterEventSessionCreate(t *testing.T) {
	requireAPIKey(t)
	runner := getRunner(t, testutil.WithEnv(map[string]string{
		"STRIPE_API_KEY": testutil.GetAPIKey(),
	}))

	result, err := runner.Run("v2", "billing", "meter_event_sessions", "create")
	if err != nil {
		fatalf(t, "Failed to run 'stripe v2 billing meter_event_sessions create': %v", err)
	}

	if result.ExitCode != 0 {
		errorf(t, "Expected exit code 0, got %d. Stderr: %s", result.ExitCode, result.Stderr)
	}

	var data map[string]interface{}
	if err := json.Unmarshal([]byte(result.Stdout), &data); err != nil {
		errorf(t, "Output is not valid JSON: %v. Output: %s", err, result.Stdout)
	}

	if _, ok := data["authentication_token"]; !ok {
		errorf(t, "Expected response to contain 'authentication_token' field, got: %s", result.Stdout)
	}
}

func TestAPIV2RawGet(t *testing.T) {
	requireAPIKey(t)
	runner := getRunner(t, testutil.WithEnv(map[string]string{
		"STRIPE_API_KEY": testutil.GetAPIKey(),
	}))

	result, err := runner.Run("get", "/v2/core/events", "-d", `{"limit": 1}`)
	if err != nil {
		fatalf(t, "Failed to run 'stripe get /v2/core/events': %v", err)
	}

	if result.ExitCode != 0 {
		errorf(t, "Expected exit code 0, got %d. Stderr: %s", result.ExitCode, result.Stderr)
	}

	var data map[string]interface{}
	if err := json.Unmarshal([]byte(result.Stdout), &data); err != nil {
		errorf(t, "Output is not valid JSON: %v. Output: %s", err, result.Stdout)
	}

	if _, ok := data["data"]; !ok {
		errorf(t, "Expected response to contain 'data' field, got: %s", result.Stdout)
	}
}

func TestAPIV2RawPost(t *testing.T) {
	requireAPIKey(t)
	runner := getRunner(t, testutil.WithEnv(map[string]string{
		"STRIPE_API_KEY": testutil.GetAPIKey(),
	}))

	result, err := runner.Run("post", "/v2/billing/meter_event_session")
	if err != nil {
		fatalf(t, "Failed to run 'stripe post /v2/billing/meter_event_session': %v", err)
	}

	if result.ExitCode != 0 {
		errorf(t, "Expected exit code 0, got %d. Stderr: %s", result.ExitCode, result.Stderr)
	}

	var data map[string]interface{}
	if err := json.Unmarshal([]byte(result.Stdout), &data); err != nil {
		errorf(t, "Output is not valid JSON: %v. Output: %s", err, result.Stdout)
	}

	if _, ok := data["authentication_token"]; !ok {
		errorf(t, "Expected response to contain 'authentication_token' field, got: %s", result.Stdout)
	}
}

func TestAPIV2TriggerMeterNoMeterFound(t *testing.T) {
	requireAPIKey(t)
	runner := getRunner(t, testutil.WithEnv(map[string]string{
		"STRIPE_API_KEY": testutil.GetAPIKey(),
	}))

	runner = runner.WithTimeout(60 * time.Second)

	result, err := runner.Run("trigger", "v1.billing.meter.no_meter_found")
	if err != nil {
		fatalf(t, "Failed to run 'stripe trigger v1.billing.meter.no_meter_found': %v", err)
	}

	if result.ExitCode != 0 {
		errorf(t, "Expected exit code 0, got %d. Stderr: %s", result.ExitCode, result.Stderr)
	}

	combinedOutput := result.Stdout + result.Stderr
	if !strings.Contains(strings.ToLower(combinedOutput), "meter") {
		errorf(t, "Expected output to mention 'meter', got: %s", combinedOutput)
	}
}
