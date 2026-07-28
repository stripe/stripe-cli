package coopcmd

import (
	"bytes"
	"encoding/json"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/stripe/stripe-cli/pkg/coop"
)

func setupAgentCommandTest(t *testing.T) (*coop.Store, *coop.Session) {
	t.Helper()
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	store, err := coop.NewStore(coopConfigFolder())
	require.NoError(t, err)
	session := &coop.Session{
		SchemaVersion: coop.CurrentSessionSchemaVersion,
		ID:            "agent_test_session",
		Status:        coop.SessionActive,
		Settings:      map[string]string{"language": "node"},
		Steps: []coop.SessionStep{
			{
				StepDefinition: coop.StepDefinition{
					Key:   "step-1",
					Title: "Step 1",
				},
				Nodes: []coop.SessionNode{
					{
						NodeDefinition: coop.NodeDefinition{
							Key:   "node-1",
							Title: "Node 1",
							Type:  coop.NodeTestHelper,
						},
						State: coop.NodePending,
					},
				},
			},
		},
	}
	require.NoError(t, store.Write(session))
	return store, session
}

func TestCoopAgentStartWorkCommand(t *testing.T) {
	store, session := setupAgentCommandTest(t)
	cmd := newCoopAgentStartWorkCmd().cmd
	cmd.SetArgs([]string{"--session", session.ID, "--step", "1", "--note", "Starting"})

	output := captureStdout(t, func() {
		require.NoError(t, cmd.Execute())
	})

	var resp coop.CommandResponse
	require.NoError(t, json.Unmarshal([]byte(output), &resp))
	require.True(t, resp.OK)
	assert.Contains(t, resp.Next, "stripe coop agent report-work")

	loaded, err := store.Read(session.ID)
	require.NoError(t, err)
	node, err := loaded.NodeByNumber(1)
	require.NoError(t, err)
	assert.Equal(t, coop.NodeActive, node.State)
}

func TestCoopAgentReportCheckCommand(t *testing.T) {
	store, session := setupAgentCommandTest(t)
	_, err := store.Update(session.ID, func(session *coop.Session) error {
		return session.TransitionNode(1, coop.NodeActive)
	})
	require.NoError(t, err)

	cmd := newCoopAgentReportCheckCmd().cmd
	cmd.SetArgs([]string{"--session", session.ID, "--step", "1", "--check", "Manual checkout passed", "--passed"})

	output := captureStdout(t, func() {
		require.NoError(t, cmd.Execute())
	})

	var resp coop.CommandResponse
	require.NoError(t, json.Unmarshal([]byte(output), &resp))
	require.True(t, resp.OK)

	loaded, err := store.Read(session.ID)
	require.NoError(t, err)
	node, err := loaded.NodeByNumber(1)
	require.NoError(t, err)
	require.Len(t, node.Verifications, 1)
	assert.Equal(t, "Manual checkout passed", node.Verifications[0].Check)
	assert.True(t, node.Verifications[0].Passed)
}

func TestCoopAgentNextActionReturnsStructuredErrorForHelperFailure(t *testing.T) {
	store := &nextActionErrorStore{
		session: &coop.Session{
			ID:     "agent_test_session",
			Status: coop.SessionCompleted,
		},
	}

	stderr := captureStderr(t, func() {
		err := runCoopNextActionWithStore(store, "agent_test_session", "")
		require.Error(t, err)
		assert.IsType(t, RenderedError{}, err)
	})

	var resp coop.CommandResponse
	require.NoError(t, json.Unmarshal([]byte(stderr), &resp))
	assert.False(t, resp.OK)
	assert.Contains(t, resp.Error, "writing next-action suggestions")
	assert.Contains(t, resp.Error, "disk full")
	assert.Equal(t, "stripe coop agent next-action --session=agent_test_session", resp.Hint)
}

func TestCoopAgentStartFollowupCreatesGuidedSession(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	store, err := coop.NewStore(coopConfigFolder())
	require.NoError(t, err)
	parent := &coop.Session{
		SchemaVersion: coop.CurrentSessionSchemaVersion,
		ID:            "parent_session",
		Blueprint:     "one-time-payment",
		Status:        coop.SessionCompleted,
		Settings:      map[string]string{"language": "node"},
		NextSteps: &coop.NextStepsState{
			Suggestions: []coop.NextStepSuggestion{
				{ID: "deploy-update", Title: "Deploy your changes"},
			},
		},
	}
	require.NoError(t, store.Write(parent))

	followupCmd := newCoopAgentStartFollowupCmd()
	ensureCalls := 0
	followupCmd.ensureSkill = func() error {
		ensureCalls++
		return errors.New("read-only repository")
	}
	cmd := followupCmd.cmd
	var stderr bytes.Buffer
	cmd.SetErr(&stderr)
	cmd.SetArgs([]string{"--session", parent.ID, "--action", "deploy-update", "--target", "Vercel"})

	output := captureStdout(t, func() {
		require.NoError(t, cmd.Execute())
	})

	var resp coopAgentRunResponse
	require.NoError(t, json.Unmarshal([]byte(output), &resp))
	require.True(t, resp.OK)
	assert.Equal(t, 1, ensureCalls)
	assert.Contains(t, stderr.String(), "unable to install the optional project-scoped Stripe skill; continuing without it")
	assert.Contains(t, resp.Message, "Deploy your changes")
	assert.Contains(t, resp.Next, "stripe coop agent start-work")
	assert.Contains(t, resp.AgentInstructions, "guided co-op follow-up")
	assert.Contains(t, resp.AgentInstructions, "Vercel")
	require.Len(t, resp.Nodes, 3)
	assert.Equal(t, "Inspect existing deploy config", resp.Nodes[0].Title)

	ids, err := store.List()
	require.NoError(t, err)
	require.Len(t, ids, 2)

	var child *coop.Session
	for _, id := range ids {
		if id == parent.ID {
			continue
		}
		child, err = store.Read(id)
		require.NoError(t, err)
	}
	require.NotNil(t, child)
	assert.Equal(t, "Deploy your changes", child.Blueprint)
	assert.Equal(t, parent.ID, child.ParentSessionID)
	assert.Equal(t, "deploy-update", child.ParentStepID)
	assert.Equal(t, "node", child.Settings["language"])
	assert.Equal(t, "Vercel", child.Settings["deploy_target"])
	assert.Equal(t, "deploy-update", child.Settings["guided_action"])
	assert.Len(t, child.Steps, 3)
}

func TestCoopAgentStartFollowupRequiresCompletedParent(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	store, err := coop.NewStore(coopConfigFolder())
	require.NoError(t, err)
	parent := &coop.Session{
		SchemaVersion: coop.CurrentSessionSchemaVersion,
		ID:            "parent_session",
		Status:        coop.SessionActive,
		NextSteps: &coop.NextStepsState{
			Suggestions: []coop.NextStepSuggestion{
				{ID: "deploy", Title: "Deploy with Stripe Projects"},
			},
		},
	}
	require.NoError(t, store.Write(parent))

	cmd := newCoopAgentStartFollowupCmd().cmd
	cmd.SilenceErrors = true
	cmd.SilenceUsage = true
	cmd.SetArgs([]string{"--session", parent.ID, "--action", "deploy"})

	stderr := captureStderr(t, func() {
		err := cmd.Execute()
		require.Error(t, err)
		assert.IsType(t, RenderedError{}, err)
	})

	var resp coop.CommandResponse
	require.NoError(t, json.Unmarshal([]byte(stderr), &resp))
	assert.False(t, resp.OK)
	assert.Contains(t, resp.Error, `parent session "parent_session" is not completed`)
}

func TestCoopAgentStartFollowupRequiresSuggestedAction(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	store, err := coop.NewStore(coopConfigFolder())
	require.NoError(t, err)
	parent := &coop.Session{
		SchemaVersion: coop.CurrentSessionSchemaVersion,
		ID:            "parent_session",
		Status:        coop.SessionCompleted,
		NextSteps: &coop.NextStepsState{
			Suggestions: []coop.NextStepSuggestion{
				{ID: "summarize", Title: "Write a STRIPE.md summary"},
			},
		},
	}
	require.NoError(t, store.Write(parent))

	cmd := newCoopAgentStartFollowupCmd().cmd
	cmd.SilenceErrors = true
	cmd.SilenceUsage = true
	cmd.SetArgs([]string{"--session", parent.ID, "--action", "deploy"})

	stderr := captureStderr(t, func() {
		err := cmd.Execute()
		require.Error(t, err)
		assert.IsType(t, RenderedError{}, err)
	})

	var resp coop.CommandResponse
	require.NoError(t, json.Unmarshal([]byte(stderr), &resp))
	assert.False(t, resp.OK)
	assert.Contains(t, resp.Error, `follow-up action "deploy" is not available for parent session "parent_session"`)
}

func TestCoopAgentStartFollowupRejectsCompletedAction(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	store, err := coop.NewStore(coopConfigFolder())
	require.NoError(t, err)
	parent := &coop.Session{
		SchemaVersion: coop.CurrentSessionSchemaVersion,
		ID:            "parent_session",
		Status:        coop.SessionCompleted,
		NextSteps: &coop.NextStepsState{
			Suggestions: []coop.NextStepSuggestion{
				{ID: "deploy", Title: "Deploy with Stripe Projects"},
			},
			Completed: []string{"deploy"},
		},
	}
	require.NoError(t, store.Write(parent))

	cmd := newCoopAgentStartFollowupCmd().cmd
	cmd.SilenceErrors = true
	cmd.SilenceUsage = true
	cmd.SetArgs([]string{"--session", parent.ID, "--action", "deploy"})

	stderr := captureStderr(t, func() {
		err := cmd.Execute()
		require.Error(t, err)
		assert.IsType(t, RenderedError{}, err)
	})

	var resp coop.CommandResponse
	require.NoError(t, json.Unmarshal([]byte(stderr), &resp))
	assert.False(t, resp.OK)
	assert.Contains(t, resp.Error, `follow-up action "deploy" is already completed for parent session "parent_session"`)
}

func TestCoopAgentStartFollowupRejectsUnknownAction(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	store, err := coop.NewStore(coopConfigFolder())
	require.NoError(t, err)
	parent := &coop.Session{
		SchemaVersion: coop.CurrentSessionSchemaVersion,
		ID:            "parent_session",
		Status:        coop.SessionCompleted,
	}
	require.NoError(t, store.Write(parent))

	cmd := newCoopAgentStartFollowupCmd().cmd
	cmd.SilenceErrors = true
	cmd.SilenceUsage = true
	cmd.SetArgs([]string{"--session", parent.ID, "--action", "unknown"})

	stderr := captureStderr(t, func() {
		err := cmd.Execute()
		require.Error(t, err)
		assert.IsType(t, RenderedError{}, err)
	})

	var resp coop.CommandResponse
	require.NoError(t, json.Unmarshal([]byte(stderr), &resp))
	assert.False(t, resp.OK)
	assert.Contains(t, resp.Error, `guided action "unknown" not found`)
	assert.Equal(t, "stripe coop agent start-followup --session=<session> --action=deploy", resp.Hint)
}

type nextActionErrorStore struct {
	session *coop.Session
}

func (s *nextActionErrorStore) Read(id string) (*coop.Session, error) {
	return s.session, nil
}

func (s *nextActionErrorStore) LatestSession() (*coop.Session, error) {
	return s.session, nil
}

func (s *nextActionErrorStore) Write(session *coop.Session) error {
	return errors.New("disk full")
}

func TestOutputAgentErrorEmitsStructuredJSON(t *testing.T) {
	// Failures before a workflow response exists (e.g. newWorkflowService/store
	// creation in start-work, report-work, etc.) must still emit structured JSON,
	// not a bare plain-text error, so an agent parsing stdout can recover.
	output := captureStdout(t, func() {
		err := outputAgentError(errors.New("creating store: disk full"))
		require.Error(t, err)
		assert.IsType(t, RenderedError{}, err)
	})

	var resp coop.CommandResponse
	require.NoError(t, json.Unmarshal([]byte(output), &resp))
	assert.False(t, resp.OK)
	assert.Contains(t, resp.Error, "creating store: disk full")
	assert.NotEmpty(t, resp.Next)
}
