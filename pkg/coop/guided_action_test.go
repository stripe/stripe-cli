package coop

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewSessionFromGuidedActionIsParentedSession(t *testing.T) {
	action := &GuidedAction{
		ID:    "test-followup",
		Title: "Test follow-up",
		Steps: []SessionStep{
			{
				WorkbenchStepDefinition: WorkbenchStepDefinition{
					Key:   "step",
					Title: MessageDescriptor{DefaultMessage: "Step"},
				},
				Nodes: []SessionNode{
					{
						WorkbenchBlueprintNode: WorkbenchBlueprintNode{
							NodeType: NodeTestHelper,
							Key:      "node",
							Title:    MessageDescriptor{DefaultMessage: "Node"},
						},
						ReviewPrompt: "Confirm the node completed.",
						State:        NodePending,
					},
				},
			},
		},
	}

	session := NewSessionFromGuidedAction(action, "coop_followup", GuidedActionSessionOptions{
		ParentSessionID: "parent_session",
		ParentStepID:    "test-followup",
		Settings:        map[string]string{"language": "go"},
		UsedSandbox:     true,
	})

	assert.Equal(t, CurrentSessionSchemaVersion, session.SchemaVersion)
	assert.Equal(t, "coop_followup", session.ID)
	assert.Equal(t, "Test follow-up", session.Blueprint)
	assert.Equal(t, SessionActive, session.Status)
	assert.Equal(t, "parent_session", session.ParentSessionID)
	assert.Equal(t, "test-followup", session.ParentStepID)
	assert.Equal(t, "go", session.Settings["language"])
	assert.Equal(t, "test-followup", session.Settings["guided_action"])
	assert.True(t, session.UsedSandbox)
	assert.Len(t, session.Steps, 1)
	assert.Equal(t, NodePending, session.Steps[0].Nodes[0].State)
	assert.NotEmpty(t, session.Steps[0].Nodes[0].ReviewPrompt)
	assert.False(t, session.CreatedAt.IsZero())
}
