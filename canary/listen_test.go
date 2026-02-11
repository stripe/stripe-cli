package canary

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/stripe/stripe-cli/canary/testutil"
)

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

func TestAPIListenForwardTo(t *testing.T) {
	runner := getRunner(t)
	requireAPIKey(t)

	// Set up webhook receiver
	var mu sync.Mutex
	var receivedReq *http.Request
	var receivedBody []byte

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		defer mu.Unlock()

		receivedReq = r
		body, _ := io.ReadAll(r.Body)
		receivedBody = body
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	// Start listen in background
	runner = runner.WithEnv(map[string]string{
		"STRIPE_API_KEY": testutil.GetAPIKey(),
	})

	listen, err := runner.RunBackground("listen", "--forward-to", server.URL)
	if err != nil {
		t.Fatalf("Failed to start listen: %v", err)
	}
	defer listen.Stop()

	// Wait for listen to be ready
	err = listen.WaitForOutput("Ready!", 30*time.Second)
	if err != nil {
		stdout, stderr := listen.GetOutput()
		t.Fatalf("Listen failed to become ready: %v\nStdout: %s\nStderr: %s", err, stdout, stderr)
	}

	// Trigger an event
	triggerRunner := runner.WithTimeout(60 * time.Second)
	result, err := triggerRunner.Run("trigger", "customer.created")
	if err != nil {
		t.Fatalf("Failed to run trigger: %v", err)
	}
	if result.ExitCode != 0 {
		t.Fatalf("Trigger failed with exit code %d. Stderr: %s", result.ExitCode, result.Stderr)
	}

	// Wait for webhook to arrive
	time.Sleep(3 * time.Second)

	// Validate received webhook
	mu.Lock()
	defer mu.Unlock()

	if receivedReq == nil {
		stdout, stderr := listen.GetOutput()
		t.Fatalf("Webhook not received. Listen stdout: %s\nListen stderr: %s", stdout, stderr)
	}

	// Validate HTTP method
	if receivedReq.Method != http.MethodPost {
		t.Errorf("Expected POST request, got %s", receivedReq.Method)
	}

	// Validate Stripe-Signature header is present and has correct format
	sig := receivedReq.Header.Get("Stripe-Signature")
	if sig == "" {
		t.Error("Stripe-Signature header is missing")
	} else if !strings.Contains(sig, "t=") || !strings.Contains(sig, "v1=") {
		t.Errorf("Stripe-Signature header has unexpected format: %s", sig)
	}

	// Validate body is valid JSON with expected event structure
	if len(receivedBody) == 0 {
		t.Error("Received empty body")
	} else {
		var event map[string]interface{}
		if err := json.Unmarshal(receivedBody, &event); err != nil {
			t.Errorf("Body is not valid JSON: %v", err)
		} else {
			// Check for required event fields
			if _, ok := event["id"]; !ok {
				t.Error("Event missing 'id' field")
			}
			if _, ok := event["type"]; !ok {
				t.Error("Event missing 'type' field")
			} else if event["type"] != "customer.created" {
				t.Errorf("Expected event type 'customer.created', got '%v'", event["type"])
			}
			if _, ok := event["data"]; !ok {
				t.Error("Event missing 'data' field")
			}
		}
	}
}

func TestAPIListenOutputFormat(t *testing.T) {
	runner := getRunner(t)
	requireAPIKey(t)

	// Start listen with JSON format in background
	runner = runner.WithEnv(map[string]string{
		"STRIPE_API_KEY": testutil.GetAPIKey(),
	})

	listen, err := runner.RunBackground("listen", "--format", "JSON")
	if err != nil {
		t.Fatalf("Failed to start listen: %v", err)
	}
	defer listen.Stop()

	// Wait for listen to be ready
	err = listen.WaitForOutput("Ready!", 30*time.Second)
	if err != nil {
		stdout, stderr := listen.GetOutput()
		t.Fatalf("Listen failed to become ready: %v\nStdout: %s\nStderr: %s", err, stdout, stderr)
	}

	// Trigger an event
	triggerRunner := runner.WithTimeout(60 * time.Second)
	result, err := triggerRunner.Run("trigger", "customer.created")
	if err != nil {
		t.Fatalf("Failed to run trigger: %v", err)
	}
	if result.ExitCode != 0 {
		t.Fatalf("Trigger failed with exit code %d. Stderr: %s", result.ExitCode, result.Stderr)
	}

	// Wait for event to be logged
	time.Sleep(3 * time.Second)

	// Get listen output
	stdout, stderr := listen.GetOutput()
	combinedOutput := stdout + stderr

	// Verify JSON event appears in output
	if !strings.Contains(combinedOutput, "customer.created") {
		t.Errorf("Expected output to contain 'customer.created' event, got:\n%s", combinedOutput)
	}

	// Try to find and parse a JSON object in the output (after "Ready!")
	readyIdx := strings.Index(combinedOutput, "Ready!")
	if readyIdx >= 0 {
		afterReady := combinedOutput[readyIdx:]
		// Look for JSON-like content
		if strings.Contains(afterReady, `"type"`) || strings.Contains(afterReady, "customer.created") {
			t.Logf("Successfully found event data in output after Ready")
		}
	}
}
