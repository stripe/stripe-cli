package workflow

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/stripe/stripe-cli/pkg/coop"
)

func workflowNode(key, title string, state coop.NodeState) coop.SessionNode {
	return coop.SessionNode{
		WorkbenchBlueprintNode: coop.WorkbenchBlueprintNode{
			Key:   key,
			Title: coop.MessageDescriptor{DefaultMessage: title},
		},
		State: state,
	}
}

func workflowStep(key, title string, nodes ...coop.SessionNode) coop.SessionStep {
	return coop.SessionStep{
		WorkbenchStepDefinition: coop.WorkbenchStepDefinition{
			Key:   key,
			Title: coop.MessageDescriptor{DefaultMessage: title},
		},
		Nodes: nodes,
	}
}

func TestStartWorkTransitionsNodeAndReturnsTypedNextCommand(t *testing.T) {
	store, session := workflowTestStore(t)
	service := NewService(store, WithSnippetFetcher(func(path, method string, params interface{}, language string) (string, error) {
		return "", nil
	}))

	resp, err := service.StartWork(session.ID, 1, "Scanning")
	require.NoError(t, err)
	require.True(t, resp.OK)
	assert.Equal(t, "active", resp.State)
	assert.Contains(t, resp.Next, "stripe coop agent report-work")

	loaded, err := store.Read(session.ID)
	require.NoError(t, err)
	node, err := loaded.NodeByNumber(1)
	require.NoError(t, err)
	assert.Equal(t, coop.NodeActive, node.State)
	assert.Equal(t, "Scanning", node.Activity)
}

func TestReportWorkContinuesStepBeforeReview(t *testing.T) {
	store, session := workflowTestStore(t)
	service := NewService(store)

	_, err := service.StartWork(session.ID, 1, "First")
	require.NoError(t, err)
	resp, err := service.ReportWork(session.ID, 1, ReportWorkInput{File: "server.go", Note: "Done"}, false)
	require.NoError(t, err)
	require.True(t, resp.OK)
	assert.Equal(t, "review", resp.State)
	assert.Contains(t, resp.Message, "Continue the step")
	assert.Contains(t, resp.Next, "--step=2")
}

func TestReportWorkRoutesToAwaitReviewWhenStepReady(t *testing.T) {
	store, session := workflowTestStore(t)
	service := NewService(store)

	_, err := service.StartWork(session.ID, 1, "First")
	require.NoError(t, err)
	_, err = service.ReportWork(session.ID, 1, ReportWorkInput{File: "server.go"}, false)
	require.NoError(t, err)
	_, err = service.StartWork(session.ID, 2, "Second")
	require.NoError(t, err)
	resp, err := service.ReportWork(session.ID, 2, ReportWorkInput{File: "client.go"}, false)
	require.NoError(t, err)
	require.True(t, resp.OK)
	assert.Contains(t, resp.Message, "Step ready for review")
	assert.Contains(t, resp.Next, "stripe coop agent await-review")
}

func TestConfirmAndRequestChangesUseCentralWorkflow(t *testing.T) {
	store, session := workflowTestStore(t)
	service := NewService(store)

	_, err := service.StartWork(session.ID, 1, "First")
	require.NoError(t, err)
	_, err = service.ReportWork(session.ID, 1, ReportWorkInput{File: "server.go"}, false)
	require.NoError(t, err)

	updated, err := service.ConfirmReview(session.ID, []int{1})
	require.NoError(t, err)
	node, err := updated.NodeByNumber(1)
	require.NoError(t, err)
	assert.Equal(t, coop.NodeDone, node.State)
}

func TestConfirmReviewTreatsSkippedNodesAsTerminal(t *testing.T) {
	store, session := workflowTestStore(t)
	service := NewService(store)

	_, err := service.Skip(session.ID, 1, "Not needed")
	require.NoError(t, err)

	updated, err := service.ConfirmReview(session.ID, []int{1})
	require.NoError(t, err)
	node, err := updated.NodeByNumber(1)
	require.NoError(t, err)
	assert.Equal(t, coop.NodeSkipped, node.State)
}

func TestRequestChangesMovesReviewNodeBackToActive(t *testing.T) {
	store, session := workflowTestStore(t)
	service := NewService(store)

	_, err := service.StartWork(session.ID, 1, "First")
	require.NoError(t, err)
	_, err = service.ReportWork(session.ID, 1, ReportWorkInput{File: "server.go"}, false)
	require.NoError(t, err)
	updated, err := service.RequestChanges(session.ID, []int{1}, "Needs tests")
	require.NoError(t, err)
	node, err := updated.NodeByNumber(1)
	require.NoError(t, err)
	assert.Equal(t, coop.NodeActive, node.State)
	assert.Equal(t, "Needs tests", node.RejectionNote)
	assert.Nil(t, node.Implementation)
}

func TestAgentWorkflowRejectsInactiveSessions(t *testing.T) {
	tests := []struct {
		name string
		run  func(*Service, string) (coop.CommandResponse, error)
	}{
		{
			name: "start work",
			run: func(service *Service, sessionID string) (coop.CommandResponse, error) {
				return service.StartWork(sessionID, 1, "Starting")
			},
		},
		{
			name: "report work",
			run: func(service *Service, sessionID string) (coop.CommandResponse, error) {
				return service.ReportWork(sessionID, 1, ReportWorkInput{File: "server.go"}, false)
			},
		},
		{
			name: "report check",
			run: func(service *Service, sessionID string) (coop.CommandResponse, error) {
				return service.ReportCheck(sessionID, 1, "Manual checkout passed", true)
			},
		},
		{
			name: "skip",
			run: func(service *Service, sessionID string) (coop.CommandResponse, error) {
				return service.Skip(sessionID, 1, "Not needed")
			},
		},
		{
			name: "await review",
			run: func(service *Service, sessionID string) (coop.CommandResponse, error) {
				return service.AwaitReview(sessionID, 1)
			},
		},
	}

	for _, status := range []coop.SessionStatus{coop.SessionCompleted, coop.SessionAborted} {
		t.Run(string(status), func(t *testing.T) {
			for _, tt := range tests {
				t.Run(tt.name, func(t *testing.T) {
					store, session := workflowTestStore(t)
					service := NewService(store)
					_, err := store.Update(session.ID, func(session *coop.Session) error {
						session.Status = status
						return nil
					})
					require.NoError(t, err)

					resp, err := tt.run(service, session.ID)
					require.NoError(t, err)
					assert.False(t, resp.OK)
					assert.Contains(t, resp.Error, "session workflow_test is "+string(status)+" and cannot be advanced")

					loaded, err := store.Read(session.ID)
					require.NoError(t, err)
					node, err := loaded.NodeByNumber(1)
					require.NoError(t, err)
					assert.Equal(t, coop.NodePending, node.State)
				})
			}
		})
	}
}

func TestReviewWorkflowRejectsInactiveSessions(t *testing.T) {
	tests := []struct {
		name string
		run  func(*Service, string) error
	}{
		{
			name: "confirm review",
			run: func(service *Service, sessionID string) error {
				_, err := service.ConfirmReview(sessionID, []int{1})
				return err
			},
		},
		{
			name: "request changes",
			run: func(service *Service, sessionID string) error {
				_, err := service.RequestChanges(sessionID, []int{1}, "Needs tests")
				return err
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store, session := workflowTestStore(t)
			service := NewService(store)
			_, err := store.Update(session.ID, func(session *coop.Session) error {
				session.Status = coop.SessionAborted
				return nil
			})
			require.NoError(t, err)

			err = tt.run(service, session.ID)
			require.Error(t, err)
			assert.Contains(t, err.Error(), "session workflow_test is aborted and cannot be advanced")
		})
	}
}

func TestCompletedParentedSessionRoutesNextActionToParent(t *testing.T) {
	store, err := coop.NewStoreAt(t.TempDir())
	require.NoError(t, err)
	parent := &coop.Session{
		SchemaVersion: coop.CurrentSessionSchemaVersion,
		ID:            "parent_session",
		Blueprint:     "one-time-payment",
		Status:        coop.SessionCompleted,
	}
	require.NoError(t, store.Write(parent))
	child := &coop.Session{
		SchemaVersion:   coop.CurrentSessionSchemaVersion,
		ID:              "child_session",
		Blueprint:       "follow-up-integration",
		Status:          coop.SessionActive,
		ParentSessionID: "parent_session",
		ParentStepID:    "add-integration",
		Steps: []coop.SessionStep{
			workflowStep("add-integration", "Add integration",
				workflowNode("add-integration", "Add integration", coop.NodeActive),
			),
		},
	}
	require.NoError(t, store.Write(child))
	service := NewService(store)

	resp, err := service.ReportWork(child.ID, 1, ReportWorkInput{File: "server.go", Note: "Added another integration"}, true)

	require.NoError(t, err)
	require.True(t, resp.OK)
	assert.Equal(t, "stripe coop agent next-action --session=parent_session --completed=add-integration", resp.Next)
}

func workflowTestStore(t *testing.T) (*coop.Store, *coop.Session) {
	t.Helper()
	store, err := coop.NewStoreAt(t.TempDir())
	require.NoError(t, err)
	session := &coop.Session{
		SchemaVersion: coop.CurrentSessionSchemaVersion,
		ID:            "workflow_test",
		Blueprint:     "test",
		Status:        coop.SessionActive,
		Steps: []coop.SessionStep{
			workflowStep("step", "Step",
				workflowNode("one", "One", coop.NodePending),
				workflowNode("two", "Two", coop.NodePending),
			),
		},
	}
	require.NoError(t, store.Write(session))
	return store, session
}
