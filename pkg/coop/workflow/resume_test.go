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

func TestStartWorkRetryIsConcurrentAndIdempotent(t *testing.T) {
	store, session := twoStepResumeTestStore(t)
	service := NewService(store)

	const workers = 8
	var wg sync.WaitGroup
	errs := make(chan error, workers)
	for worker := 0; worker < workers; worker++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			response, err := service.StartWork(session.ID, 1, fmt.Sprintf("retry-%d", worker))
			if err == nil && !response.OK {
				err = fmt.Errorf("start-work response failed: %s", response.Error)
			}
			errs <- err
		}()
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

func TestLateReviewDecisionReleasesWaiterWithExactNextCommand(t *testing.T) {
	store, session := twoStepResumeTestStore(t)
	service := NewService(store, WithAwaitTimeout(3*time.Second))
	_, err := service.StartWork(session.ID, 1, "First")
	require.NoError(t, err)
	_, err = service.ReportWork(session.ID, 1, ReportWorkInput{File: "server.go", Note: "Implemented server work"}, false)
	require.NoError(t, err)

	type awaitResult struct {
		response coop.CommandResponse
		err      error
	}
	done := make(chan awaitResult, 1)
	go func() {
		response, err := service.AwaitReview(session.ID, 1)
		done <- awaitResult{response: response, err: err}
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
	require.True(t, result.response.OK)
	assert.Equal(t, "confirmed", result.response.State)
	assert.Equal(t, `stripe coop agent start-work --session=workflow_resume --step=2 --note="Beginning: Two"`, result.response.Next)
}

func TestResumeIsReadOnlyAndTracksCurrentLifecycleState(t *testing.T) {
	store, session := twoStepResumeTestStore(t)
	service := NewService(store)

	response, err := service.Resume(session.ID)
	require.NoError(t, err)
	assert.Equal(t, `stripe coop agent start-work --session=workflow_resume --step=1 --note="Beginning: One"`, response.Next)

	_, err = service.StartWork(session.ID, 1, "First")
	require.NoError(t, err)
	_, err = service.ReportWork(session.ID, 1, ReportWorkInput{File: "server.go", Note: "Implemented server work"}, false)
	require.NoError(t, err)
	response, err = service.Resume(session.ID)
	require.NoError(t, err)
	assert.Equal(t, string(coop.NodeReview), response.State)
	assert.Equal(t, "stripe coop agent await-review --session=workflow_resume --step=1", response.Next)
	assert.Equal(t, int(AwaitTimeout.Seconds()), response.WaitTimeoutSeconds)

	_, err = service.RequestChanges(session.ID, []int{1}, "Reuse the saved price")
	require.NoError(t, err)
	response, err = service.Resume(session.ID)
	require.NoError(t, err)
	assert.Equal(t, "rejected", response.State)
	assert.Equal(t, `stripe coop agent start-work --session=workflow_resume --step=1 --note="Redoing: One"`, response.Next)

	_, err = service.StartWork(session.ID, 1, "Redoing: One")
	require.NoError(t, err)
	response, err = service.Resume(session.ID)
	require.NoError(t, err)
	assert.Equal(t, string(coop.NodeActive), response.State)
	assert.Empty(t, response.Next)

	_, err = service.ReportWork(session.ID, 1, ReportWorkInput{File: "server.go", Note: "Reworked server code"}, false)
	require.NoError(t, err)
	_, err = service.ConfirmReview(session.ID, []int{1})
	require.NoError(t, err)
	response, err = service.Resume(session.ID)
	require.NoError(t, err)
	assert.Equal(t, `stripe coop agent start-work --session=workflow_resume --step=2 --note="Beginning: Two"`, response.Next)

	loaded, err := store.Read(session.ID)
	require.NoError(t, err)
	node, err := loaded.NodeByNumber(2)
	require.NoError(t, err)
	assert.Equal(t, coop.NodePending, node.State, "resume must not mutate session state")

	_, err = service.StartWork(session.ID, 2, "Second")
	require.NoError(t, err)
	_, err = service.ReportWork(session.ID, 2, ReportWorkInput{File: "client.go", Note: "Implemented client work"}, false)
	require.NoError(t, err)
	_, err = service.ConfirmReview(session.ID, []int{2})
	require.NoError(t, err)
	response, err = service.Resume(session.ID)
	require.NoError(t, err)
	assert.Equal(t, string(coop.SessionCompleted), response.State)
	assert.Equal(t, "stripe coop agent next-action --session=workflow_resume", response.Next)
	require.NoError(t, response.Validate())
}

func twoStepResumeTestStore(t *testing.T) (*coop.Store, *coop.Session) {
	t.Helper()
	store, err := coop.NewStoreAt(t.TempDir())
	require.NoError(t, err)
	session := &coop.Session{
		SchemaVersion: coop.CurrentSessionSchemaVersion,
		ID:            "workflow_resume",
		Blueprint:     "test",
		Status:        coop.SessionActive,
		Steps: []coop.SessionStep{
			workflowStep("one", "One", workflowNode("one", "One", coop.NodePending)),
			workflowStep("two", "Two", workflowNode("two", "Two", coop.NodePending)),
		},
	}
	require.NoError(t, store.Write(session))
	return store, session
}

// Selecting Finish hands the agent "stripe coop stop". Once it runs that, the
// session is completed — and Resume used to answer a completed session with the
// next-action command, so the stop hook held the turn open and the agent could
// never exit.
func TestResumeReportsNothingToRunAfterDeveloperFinishes(t *testing.T) {
	store, session := workflowTestStore(t)
	service := NewService(store)

	_, err := store.Update(session.ID, func(s *coop.Session) error {
		s.Status = coop.SessionCompleted
		s.NextSteps = &coop.NextStepsState{Completed: []string{coop.FinishActionID}}
		return nil
	})
	require.NoError(t, err)

	response, err := service.Resume(session.ID)
	require.NoError(t, err)

	assert.True(t, response.OK)
	assert.Empty(t, response.Next, "a finished session has nothing left to hand back")
	assert.Nil(t, response.AdvanceAllowed)
}

// A completed session the developer has not closed out still owes a next action.
func TestResumeStillOffersNextActionBeforeFinish(t *testing.T) {
	store, session := workflowTestStore(t)
	service := NewService(store)

	_, err := store.Update(session.ID, func(s *coop.Session) error {
		for i := range s.Steps {
			for j := range s.Steps[i].Nodes {
				s.Steps[i].Nodes[j].State = coop.NodeDone
			}
		}
		s.Status = coop.SessionCompleted
		return nil
	})
	require.NoError(t, err)

	response, err := service.Resume(session.ID)
	require.NoError(t, err)

	assert.Contains(t, response.Next, "next-action")
}
