package canary

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/stripe/stripe-cli/canary/testutil"
)

// =============================================================================
// API Resource Tests - Require STRIPE_API_KEY
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
