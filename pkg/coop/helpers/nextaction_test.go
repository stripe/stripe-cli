package helpers

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/stripe/stripe-cli/pkg/coop"
)

func TestBuildSuggestionsIncludesCoreNextActions(t *testing.T) {
	session := &coop.Session{ID: "sess_123", Blueprint: "one-time-payment"}

	suggestions := BuildSuggestions(session, Environment{})

	assert.Equal(t, "summarize", suggestions[0].ID)
	assert.Equal(t, "add-integration", suggestions[1].ID)
	assert.Equal(t, "done", suggestions[2].ID)
}
