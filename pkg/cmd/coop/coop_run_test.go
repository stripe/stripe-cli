package coopcmd

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/stripe/stripe-cli/pkg/coop"
)

func TestNewCoopSessionAppliesSharedMetadata(t *testing.T) {
	previousOptions := options
	options = Options{ClaimURL: func() string { return "https://dashboard.stripe.com/sandbox/claim_test" }}
	t.Cleanup(func() { options = previousOptions })

	session := newCoopSession(
		&coop.Blueprint{ID: "one-time-payment"},
		"coop_123",
		"go",
		[]string{"framework=gin", "ignored"},
		"parent_123",
		"deploy",
	)

	require.Equal(t, "coop_123", session.ID)
	assert.Equal(t, "go", session.Settings["language"])
	assert.Equal(t, "gin", session.Settings["framework"])
	assert.NotContains(t, session.Settings, "ignored")
	assert.Equal(t, "parent_123", session.ParentSessionID)
	assert.Equal(t, "deploy", session.ParentStepID)
	assert.Equal(t, "https://dashboard.stripe.com/sandbox/claim_test", session.ClaimURL)
	assert.False(t, session.CreatedAt.IsZero())
}
