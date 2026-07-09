package followups

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/stripe/stripe-cli/pkg/coop"
)

func TestGuidedActionByIDDeployCreatesStepFlow(t *testing.T) {
	action, err := GuidedActionByID(Deploy, "")
	require.NoError(t, err)

	assert.Equal(t, Deploy, action.ID)
	assert.Equal(t, "Deploy with Stripe Projects", action.Title)
	require.Len(t, action.Steps, 4)
	assert.Equal(t, "Detect deploy path", action.Steps[0].Title)
	assert.Equal(t, "Check Stripe Projects plugin", action.Steps[0].Nodes[0].Title)
	assert.Contains(t, action.AgentContext, "Stripe Projects CLI plugin")
}

func TestGuidedActionByIDDeployUpdateUsesTarget(t *testing.T) {
	action, err := GuidedActionByID(DeployUpdate, "Vercel")
	require.NoError(t, err)

	assert.Equal(t, DeployUpdate, action.ID)
	assert.Equal(t, "Deploy your changes", action.Title)
	require.Len(t, action.Steps, 3)
	assert.Contains(t, action.AgentContext, "existing Vercel deployment configuration")
	assert.Contains(t, action.Steps[0].Nodes[0].Description, "through Vercel")
	assert.Contains(t, action.Steps[1].Nodes[0].ReviewPrompt, "to Vercel")
}

func TestGuidedActionByIDRejectsUnknownAction(t *testing.T) {
	_, err := GuidedActionByID("unknown", "")

	require.Error(t, err)
	assert.Contains(t, err.Error(), `guided action "unknown" not found`)
}

func TestDeployFollowupsAreNotBlueprints(t *testing.T) {
	ids, err := coop.ListBlueprints()
	require.NoError(t, err)

	assert.NotContains(t, ids, Deploy)
	assert.NotContains(t, ids, DeployUpdate)

	_, err = coop.LoadBlueprint(Deploy)
	assert.Error(t, err)
}
