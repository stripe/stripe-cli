package fixtures

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestInteropFallbackResolvesFixture(t *testing.T) {
	// v1.customer.created should resolve to the same fixture as customer.created
	interopContent, err := FixtureContents("v1.customer.created")
	require.NoError(t, err)

	directContent, err := FixtureContents("customer.created")
	require.NoError(t, err)

	assert.Equal(t, directContent, interopContent)
}

func TestInteropExactMatchTakesPrecedence(t *testing.T) {
	// v1.billing.meter.error_report_triggered has its own dedicated fixture file,
	// so it should resolve via exact match (not the interop fallback).
	content, err := FixtureContents("v1.billing.meter.error_report_triggered")
	require.NoError(t, err)
	assert.NotEmpty(t, content)
}

func TestInteropNonEligibleEventErrors(t *testing.T) {
	// An event not in the interop allowlist should produce an error.
	_, err := FixtureContents("v1.some.nonexistent.event")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not supported")
}

func TestInteropEventWithoutFixtureErrors(t *testing.T) {
	// financial_connections.account.created is in the interop allowlist but has no
	// fixture file, so it should produce an error.
	_, err := FixtureContents("v1.financial_connections.account.created")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not supported")
}

func TestEventNamesIncludesInteropWithFixtures(t *testing.T) {
	names := EventNames()

	// v1.customer.created should be present (customer.created has a fixture)
	assert.Contains(t, names, "v1.customer.created")

	// v1.financial_connections.account.created should NOT be present (no fixture)
	assert.NotContains(t, names, "v1.financial_connections.account.created")
}

func TestEventNamesIncludesDedicatedThinEvents(t *testing.T) {
	names := EventNames()

	// Dedicated thin-only events should still be present via exact match
	assert.Contains(t, names, "v1.billing.meter.error_report_triggered")
}
