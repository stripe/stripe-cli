package coopcmd

import (
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
