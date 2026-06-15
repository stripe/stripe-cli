package workflow

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/stripe/stripe-cli/pkg/coop"
)

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
			{
				Key:               "step",
				Title:             "Step",
				ReviewGranularity: coop.ReviewGranularityStep,
				Nodes: []coop.SessionNode{
					{Key: "one", Title: "One", State: coop.NodePending},
					{Key: "two", Title: "Two", State: coop.NodePending},
				},
			},
		},
	}
	require.NoError(t, store.Write(session))
	return store, session
}
