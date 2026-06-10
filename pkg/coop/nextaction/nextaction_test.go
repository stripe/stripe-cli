package nextaction

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/stripe/stripe-cli/pkg/coop"
)

func TestBuildSuggestionsUsesStripeProjectsWhenConfigured(t *testing.T) {
	session := &coop.Session{ID: "sess_123", Blueprint: "one-time-payment"}

	suggestions := BuildSuggestions(session, Environment{HasStripeProjects: true})

	assert.Equal(t, "deploy", suggestions[0].ID)
	assert.Equal(t, "stripe.json found", suggestions[0].Reason)
	assert.Equal(t, "summarize", suggestions[1].ID)
	assert.Equal(t, "add-integration", suggestions[2].ID)
	assert.Equal(t, "done", suggestions[3].ID)
}

func TestBuildSuggestionsUsesExistingDeployTarget(t *testing.T) {
	session := &coop.Session{ID: "sess_123", Blueprint: "one-time-payment"}

	suggestions := BuildSuggestions(session, Environment{HasExistingDeploy: true, HasVercel: true})

	assert.Equal(t, "deploy-update", suggestions[0].ID)
	assert.Equal(t, "Detected: Vercel", suggestions[0].Reason)
}

func TestBuildResponseForDeploy(t *testing.T) {
	session := &coop.Session{
		ID:        "sess_123",
		Blueprint: "one-time-payment",
		Settings:  map[string]string{"language": "go"},
	}

	resp := BuildResponse(session, nil, "deploy")

	assert.True(t, resp.OK)
	assert.Equal(t, "sess_123", resp.SessionID)
	assert.Equal(t, "stripe coop run deploy-stripe-projects --language=go --parent-session=sess_123 --parent-step=deploy", resp.Next)
}
