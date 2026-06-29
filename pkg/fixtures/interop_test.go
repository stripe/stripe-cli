package fixtures

import (
	"sort"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestInteropFallbackResolvesFixture(t *testing.T) {
	v1Content, err := FixtureContents("v1.customer.created")
	require.NoError(t, err)

	unprefixedContent, err := FixtureContents("customer.created")
	require.NoError(t, err)

	assert.Equal(t, unprefixedContent, v1Content)
}

func TestInteropExactMatchTakesPrecedence(t *testing.T) {
	content, err := FixtureContents("v1.billing.meter.error_report_triggered")
	require.NoError(t, err)
	assert.NotEmpty(t, content)
}

func TestInteropNonEligibleEventErrors(t *testing.T) {
	_, err := FixtureContents("v1.some.nonexistent.event")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not supported")
}

func TestInteropEventWithoutFixtureErrors(t *testing.T) {
	_, err := FixtureContents("v1.financial_connections.account.created")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not supported")
}

func TestEventNamesIncludesInteropWithFixtures(t *testing.T) {
	names := EventNames()
	assert.Contains(t, names, "v1.customer.created")
	assert.NotContains(t, names, "v1.financial_connections.account.created")
}

func TestEventNamesIncludesDedicatedThinEvents(t *testing.T) {
	names := EventNames()
	assert.Contains(t, names, "v1.billing.meter.error_report_triggered")
}

func TestEventNamesIsSorted(t *testing.T) {
	names := EventNames()
	assert.True(t, sort.StringsAreSorted(names))
}
