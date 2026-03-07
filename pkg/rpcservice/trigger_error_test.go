// Package rpcservice provides tests for trigger error message quality.
//
// These tests ensure that error messages remain helpful and informative as the
// codebase evolves. They validate that users receive actionable guidance when
// they encounter errors, rather than cryptic messages.
//
// Why this matters:
// Error message quality can degrade silently during refactoring. Without these
// tests, helpful guidance can be accidentally removed, documentation links can
// break, or messages can become less clear over time. These tests protect the
// user experience by treating error messages as part of the API contract.
//
// What gets validated:
// - Error messages contain specific helpful phrases
// - Documentation links are present and correct
// - Alternative solutions are suggested
// - Event names are included in error messages
// - Common user mistakes are handled gracefully
package rpcservice

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stripe/stripe-cli/rpc"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// TestUnsupportedEventErrorMessage verifies the error message for unsupported events
// contains helpful guidance for users.
//
// The documentation link (https://docs.stripe.com/cli/fixtures) is critical - it's
// where users learn about custom fixtures. If this link breaks or changes, users
// lose access to a key feature. This test ensures the link stays in the error message.
func TestUnsupportedEventErrorMessage(t *testing.T) {
	ctx := withAuth(context.Background())

	conn, err := grpc.DialContext(ctx, "bufnet", grpc.WithContextDialer(bufDialer), grpc.WithTransportCredentials(insecure.NewCredentials()))
	require.NoError(t, err, "Failed to dial bufnet")
	defer conn.Close()

	client := rpc.NewStripeCLIClient(conn)
	baseURL = "https://api.stripe.com"

	resp, err := client.Trigger(ctx, &rpc.TriggerRequest{
		Event: "customer.nonexistent",
	})

	// Verify error occurs
	require.NotNil(t, err, "Expected error for unsupported event")
	require.Nil(t, resp, "Expected nil response for unsupported event")

	errMsg := err.Error()

	// Verify error message contains key components
	assert.Contains(t, errMsg, "not supported by Stripe CLI",
		"Error should indicate event is not supported")

	assert.Contains(t, errMsg, "customer.nonexistent",
		"Error should include the attempted event name")

	assert.Contains(t, errMsg, "Stripe API or Dashboard",
		"Error should suggest using Stripe API or Dashboard")

	assert.Contains(t, errMsg, "custom fixture",
		"Error should mention custom fixtures as an option")

	assert.Contains(t, errMsg, "https://docs.stripe.com/cli/fixtures",
		"Error should include link to fixture documentation")
}

// TestEmptyEventErrorMessage verifies the error message when no event name is provided.
// Empty event names should still provide helpful guidance, not just fail silently.
func TestEmptyEventErrorMessage(t *testing.T) {
	ctx := withAuth(context.Background())

	conn, err := grpc.DialContext(ctx, "bufnet", grpc.WithContextDialer(bufDialer), grpc.WithTransportCredentials(insecure.NewCredentials()))
	require.NoError(t, err, "Failed to dial bufnet")
	defer conn.Close()

	client := rpc.NewStripeCLIClient(conn)
	baseURL = "https://api.stripe.com"

	resp, err := client.Trigger(ctx, &rpc.TriggerRequest{
		Event: "",
	})

	// Verify error occurs
	require.NotNil(t, err, "Expected error for empty event")
	require.Nil(t, resp, "Expected nil response for empty event")

	// Verify error message is helpful even for empty events
	errMsg := err.Error()
	assert.Contains(t, errMsg, "not supported by Stripe CLI",
		"Even empty event errors should indicate the issue clearly")
}

// TestInvalidEventNameVariations tests various invalid event name patterns.
func TestInvalidEventNameVariations(t *testing.T) {
	ctx := withAuth(context.Background())

	conn, err := grpc.DialContext(ctx, "bufnet", grpc.WithContextDialer(bufDialer), grpc.WithTransportCredentials(insecure.NewCredentials()))
	require.NoError(t, err, "Failed to dial bufnet")
	defer conn.Close()

	client := rpc.NewStripeCLIClient(conn)
	baseURL = "https://api.stripe.com"

	testCases := []struct {
		name      string
		eventName string
	}{
		{"Typo in resource", "custmer.created"},
		{"Typo in action", "customer.crated"},
		{"Wrong action", "customer.destroyed"},
		{"Extra dots", "customer..created"},
		{"Missing action", "customer."},
		{"Only resource", "customer"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			resp, err := client.Trigger(ctx, &rpc.TriggerRequest{
				Event: tc.eventName,
			})

			assert.NotNil(t, err, "Expected error for invalid event: %s", tc.eventName)
			assert.Nil(t, resp, "Expected nil response for invalid event: %s", tc.eventName)
			assert.Contains(t, err.Error(), "not supported by Stripe CLI",
				"Error should indicate event is not supported")
		})
	}
}
