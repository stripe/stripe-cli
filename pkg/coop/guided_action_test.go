package coop

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGuidedActionByIDDeployCreatesStepFlow(t *testing.T) {
	action, err := GuidedActionByID(GuidedActionDeploy, "")
	require.NoError(t, err)

	assert.Equal(t, GuidedActionDeploy, action.ID)
	assert.Equal(t, "Deploy with Stripe Projects", action.Title)
	require.Len(t, action.Steps, 4)
	assert.Equal(t, "Detect deploy path", action.Steps[0].Title)
	assert.Equal(t, "Check Stripe Projects plugin", action.Steps[0].Nodes[0].Title)
	assert.Contains(t, action.AgentContext, "Stripe Projects CLI plugin")
}

func TestGuidedActionByIDDeployUpdateUsesTarget(t *testing.T) {
	action, err := GuidedActionByID(GuidedActionDeployUpdate, "Vercel")
	require.NoError(t, err)

	assert.Equal(t, GuidedActionDeployUpdate, action.ID)
	assert.Equal(t, "Deploy your changes", action.Title)
	require.Len(t, action.Steps, 3)
	assert.Contains(t, action.AgentContext, "existing Vercel deployment configuration")
	assert.Contains(t, action.Steps[0].Nodes[0].Description, "through Vercel")
	assert.Contains(t, action.Steps[1].Nodes[0].ReviewPrompt, "to Vercel")
}

func TestNewSessionFromGuidedActionIsParentedSession(t *testing.T) {
	action, err := GuidedActionByID(GuidedActionDeployUpdate, "Netlify")
	require.NoError(t, err)

	session := NewSessionFromGuidedAction(action, "coop_followup", GuidedActionSessionOptions{
		ParentSessionID: "parent_session",
		ParentStepID:    GuidedActionDeployUpdate,
		Settings:        map[string]string{"language": "go"},
		UsedSandbox:     true,
	})

	assert.Equal(t, CurrentSessionSchemaVersion, session.SchemaVersion)
	assert.Equal(t, "coop_followup", session.ID)
	assert.Equal(t, "Deploy your changes", session.Blueprint)
	assert.Equal(t, SessionActive, session.Status)
	assert.Equal(t, "parent_session", session.ParentSessionID)
	assert.Equal(t, GuidedActionDeployUpdate, session.ParentStepID)
	assert.Equal(t, "go", session.Settings["language"])
	assert.Equal(t, GuidedActionDeployUpdate, session.Settings["guided_action"])
	assert.True(t, session.UsedSandbox)
	assert.Len(t, session.Steps, 3)
	assert.Equal(t, NodePending, session.Steps[0].Nodes[0].State)
	assert.NotEmpty(t, session.Steps[0].Nodes[0].ReviewPrompt)
	assert.False(t, session.CreatedAt.IsZero())
}

func TestGuidedActionsAreNotBlueprints(t *testing.T) {
	ids, err := ListBlueprints()
	require.NoError(t, err)

	assert.NotContains(t, ids, GuidedActionDeploy)
	assert.NotContains(t, ids, GuidedActionDeployUpdate)

	_, err = LoadBlueprint(GuidedActionDeploy)
	assert.Error(t, err)
}
