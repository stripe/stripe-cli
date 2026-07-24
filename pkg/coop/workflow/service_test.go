package workflow

import (
	"fmt"
	"sync"
	"testing"
	"time"

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

func TestStartWorkExactRetryCommandIsConcurrentAndIdempotent(t *testing.T) {
	store, session := twoStepWorkflowTestStore(t)
	service := NewService(store)

	const workers = 8
	var wg sync.WaitGroup
	errs := make(chan error, workers)
	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func(worker int) {
			defer wg.Done()
			resp, err := service.StartWork(session.ID, 1, fmt.Sprintf("retry-%d", worker))
			if err == nil && !resp.OK {
				err = fmt.Errorf("start-work response failed: %s", resp.Error)
			}
			errs <- err
		}(i)
	}
	wg.Wait()
	close(errs)
	for err := range errs {
		require.NoError(t, err)
	}

	loaded, err := store.Read(session.ID)
	require.NoError(t, err)
	node, err := loaded.NodeByNumber(1)
	require.NoError(t, err)
	assert.Equal(t, coop.NodeActive, node.State)
	assert.Contains(t, node.Activity, "retry-")
}

func TestAwaitReviewKeepsIntentionalTenMinuteBoundAndExactRetry(t *testing.T) {
	assert.Equal(t, 10*time.Minute, AwaitTimeout)
	store, session := twoStepWorkflowTestStore(t)
	service := NewService(store)
	_, err := service.StartWork(session.ID, 1, "First")
	require.NoError(t, err)
	_, err = service.ReportWork(session.ID, 1, ReportWorkInput{File: "server.go"}, false)
	require.NoError(t, err)

	current := time.Unix(1_000, 0)
	service = NewService(store,
		WithAwaitTimeout(time.Second),
		WithClock(func() time.Time { return current }, func(d time.Duration) { current = current.Add(d) }),
	)
	resp, err := service.AwaitReview(session.ID, 1)

	require.NoError(t, err)
	require.True(t, resp.OK)
	assert.Equal(t, "timeout", resp.State)
	assert.Equal(t, "stripe coop agent await-review --session=workflow_two_step --step=1", resp.Next)
	age, err := store.HeartbeatAge(session.ID)
	require.NoError(t, err)
	assert.Equal(t, time.Duration(-1), age)
}

func TestLateReviewDecisionReleasesWaiterWithExactNextCommand(t *testing.T) {
	store, session := twoStepWorkflowTestStore(t)
	service := NewService(store, WithAwaitTimeout(3*time.Second))
	_, err := service.StartWork(session.ID, 1, "First")
	require.NoError(t, err)
	_, err = service.ReportWork(session.ID, 1, ReportWorkInput{File: "server.go"}, false)
	require.NoError(t, err)

	type awaitResult struct {
		resp coop.CommandResponse
		err  error
	}
	done := make(chan awaitResult, 1)
	go func() {
		resp, err := service.AwaitReview(session.ID, 1)
		done <- awaitResult{resp: resp, err: err}
	}()

	deadline := time.Now().Add(2 * time.Second)
	for {
		age, err := store.HeartbeatAge(session.ID)
		require.NoError(t, err)
		if age >= 0 {
			break
		}
		require.True(t, time.Now().Before(deadline), "await-review did not publish its heartbeat")
		time.Sleep(5 * time.Millisecond)
	}
	_, err = service.ConfirmReview(session.ID, []int{1})
	require.NoError(t, err)

	result := <-done
	require.NoError(t, result.err)
	require.True(t, result.resp.OK)
	assert.Equal(t, "confirmed", result.resp.State)
	assert.Equal(t, `stripe coop agent start-work --session=workflow_two_step --step=2 --note="Beginning: Two"`, result.resp.Next)
}

func TestResumeIsReadOnlyAndTracksCurrentLifecycleState(t *testing.T) {
	store, session := twoStepWorkflowTestStore(t)
	service := NewService(store)

	resp, err := service.Resume(session.ID)
	require.NoError(t, err)
	assert.Equal(t, `stripe coop agent start-work --session=workflow_two_step --step=1 --note="Beginning: One"`, resp.Next)

	_, err = service.StartWork(session.ID, 1, "First")
	require.NoError(t, err)
	_, err = service.ReportWork(session.ID, 1, ReportWorkInput{File: "server.go"}, false)
	require.NoError(t, err)
	resp, err = service.Resume(session.ID)
	require.NoError(t, err)
	assert.Equal(t, "review", resp.State)
	assert.Equal(t, "stripe coop agent await-review --session=workflow_two_step --step=1", resp.Next)

	_, err = service.RequestChanges(session.ID, []int{1}, "Reuse the saved price")
	require.NoError(t, err)
	resp, err = service.Resume(session.ID)
	require.NoError(t, err)
	assert.Equal(t, "rejected", resp.State)
	assert.Equal(t, `stripe coop agent start-work --session=workflow_two_step --step=1 --note="Redoing: One"`, resp.Next)

	_, err = service.StartWork(session.ID, 1, "Redoing: One")
	require.NoError(t, err)
	resp, err = service.Resume(session.ID)
	require.NoError(t, err)
	assert.Equal(t, "active", resp.State)
	assert.Empty(t, resp.Next)

	_, err = service.ReportWork(session.ID, 1, ReportWorkInput{File: "server.go"}, false)
	require.NoError(t, err)
	_, err = service.ConfirmReview(session.ID, []int{1})
	require.NoError(t, err)
	resp, err = service.Resume(session.ID)
	require.NoError(t, err)
	assert.Equal(t, `stripe coop agent start-work --session=workflow_two_step --step=2 --note="Beginning: Two"`, resp.Next)

	loaded, err := store.Read(session.ID)
	require.NoError(t, err)
	node, err := loaded.NodeByNumber(2)
	require.NoError(t, err)
	assert.Equal(t, coop.NodePending, node.State, "resume must not mutate session state")

	_, err = service.StartWork(session.ID, 2, "Second")
	require.NoError(t, err)
	_, err = service.ReportWork(session.ID, 2, ReportWorkInput{File: "client.go"}, false)
	require.NoError(t, err)
	_, err = service.ConfirmReview(session.ID, []int{2})
	require.NoError(t, err)
	resp, err = service.Resume(session.ID)
	require.NoError(t, err)
	assert.Equal(t, "completed", resp.State)
	assert.Equal(t, "stripe coop agent next-action --session=workflow_two_step", resp.Next)
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
			{
				StepDefinition: coop.StepDefinition{Key: "add-integration", Title: "Add integration"},
				Nodes: []coop.SessionNode{
					{
						NodeDefinition: coop.NodeDefinition{Key: "add-integration", Title: "Add integration"},
						State:          coop.NodeActive,
					},
				},
			},
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
			{
				StepDefinition: coop.StepDefinition{
					Key:   "step",
					Title: "Step",
				},
				Nodes: []coop.SessionNode{
					{
						NodeDefinition: coop.NodeDefinition{Key: "one", Title: "One"},
						State:          coop.NodePending,
					},
					{
						NodeDefinition: coop.NodeDefinition{Key: "two", Title: "Two"},
						State:          coop.NodePending,
					},
				},
			},
		},
	}
	require.NoError(t, store.Write(session))
	return store, session
}

func twoStepWorkflowTestStore(t *testing.T) (*coop.Store, *coop.Session) {
	t.Helper()
	store, err := coop.NewStoreAt(t.TempDir())
	require.NoError(t, err)
	session := &coop.Session{
		SchemaVersion: coop.CurrentSessionSchemaVersion,
		ID:            "workflow_two_step",
		Blueprint:     "test",
		Status:        coop.SessionActive,
		Steps: []coop.SessionStep{
			{
				StepDefinition: coop.StepDefinition{Key: "one", Title: "One"},
				Nodes: []coop.SessionNode{{
					NodeDefinition: coop.NodeDefinition{Key: "one", Title: "One"},
					State:          coop.NodePending,
				}},
			},
			{
				StepDefinition: coop.StepDefinition{Key: "two", Title: "Two"},
				Nodes: []coop.SessionNode{{
					NodeDefinition: coop.NodeDefinition{Key: "two", Title: "Two"},
					State:          coop.NodePending,
				}},
			},
		},
	}
	require.NoError(t, store.Write(session))
	return store, session
}
