package workflow

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/stripe/stripe-cli/pkg/coop"
)

// A developer can press "request changes" between two await-review calls. The
// node is then active with a rejection note rather than in review, and the next
// call must surface the feedback instead of advancing past the rejected work.
func TestAwaitReviewReturnsRejectionRequestedBetweenInvocations(t *testing.T) {
	store, session := workflowTestStore(t)
	service := NewService(store)

	for _, node := range []int{1, 2} {
		_, err := store.Update(session.ID, func(s *coop.Session) error {
			if err := s.TransitionNode(node, coop.NodeActive); err != nil {
				return err
			}
			return s.TransitionNode(node, coop.NodeReview)
		})
		require.NoError(t, err)
	}

	_, err := service.RequestChanges(session.ID, []int{1}, "Use the sandbox price id")
	require.NoError(t, err)

	resp, err := service.AwaitReview(session.ID, 2)
	require.NoError(t, err)

	assert.Equal(t, "rejected", resp.State)
	assert.Contains(t, resp.Message, "Use the sandbox price id")
	assert.Contains(t, resp.Next, "start-work")
	assert.Contains(t, resp.Next, "--step=1", "must redo the rejected task, not advance")
}

// A step can legitimately have one task awaiting review while another is still
// active. That is not a rejection and must not be reported as one.
func TestAwaitReviewTreatsActiveNodeWithoutFeedbackAsProgress(t *testing.T) {
	store, session := workflowTestStore(t)
	service := NewService(store)

	_, err := store.Update(session.ID, func(s *coop.Session) error {
		if err := s.TransitionNode(1, coop.NodeActive); err != nil {
			return err
		}
		if err := s.TransitionNode(1, coop.NodeReview); err != nil {
			return err
		}
		return s.TransitionNode(2, coop.NodeActive)
	})
	require.NoError(t, err)

	resp, err := service.AwaitReview(session.ID, 1)
	require.NoError(t, err)

	assert.NotEqual(t, "rejected", resp.State)
}

// RejectionNote survives until a task reaches Done, so the note alone does not
// mean the developer is still waiting. Once the agent has restarted the task,
// re-reporting the rejection would restart work already underway.
func TestAwaitReviewDoesNotRepeatRejectionOnceRedoStarted(t *testing.T) {
	store, session := workflowTestStore(t)
	service := NewService(store)

	for _, node := range []int{1, 2} {
		_, err := service.StartWork(session.ID, node, "Working")
		require.NoError(t, err)
		_, err = service.ReportWork(session.ID, node, ReportWorkInput{Note: "Done"}, false)
		require.NoError(t, err)
	}

	_, err := service.RequestChanges(session.ID, []int{1}, "Use the sandbox price id")
	require.NoError(t, err)
	_, err = service.StartWork(session.ID, 1, "Redoing: One")
	require.NoError(t, err)

	resp, err := service.AwaitReview(session.ID, 2)
	require.NoError(t, err)

	assert.NotEqual(t, "rejected", resp.State)
}

// The developer can confirm the final task between two await calls. The agent
// is told to keep re-running await, so a completed session must hand back the
// next-action command rather than a non-zero error with an empty stdout, which
// an agent reads as a broken session.
func TestAwaitReviewOnCompletedSessionPointsAtNextAction(t *testing.T) {
	store, session := workflowTestStore(t)
	service := NewService(store)

	for _, node := range []int{1, 2} {
		_, err := service.StartWork(session.ID, node, "Working")
		require.NoError(t, err)
		_, err = service.ReportWork(session.ID, node, ReportWorkInput{Note: "Done"}, false)
		require.NoError(t, err)
	}
	_, err := service.ConfirmReview(session.ID, []int{1, 2})
	require.NoError(t, err)

	resp, err := service.AwaitReview(session.ID, 2)
	require.NoError(t, err)

	assert.True(t, resp.OK)
	require.NotNil(t, resp.AdvanceAllowed)
	assert.True(t, *resp.AdvanceAllowed)
	assert.Contains(t, resp.Next, "next-action")
}

// An aborted session is a genuine failure and must stay one.
func TestAwaitReviewOnAbortedSessionStaysAnError(t *testing.T) {
	store, session := workflowTestStore(t)
	service := NewService(store)

	_, err := store.Update(session.ID, func(s *coop.Session) error {
		s.Status = coop.SessionAborted
		return nil
	})
	require.NoError(t, err)

	resp, err := service.AwaitReview(session.ID, 1)
	require.NoError(t, err)

	assert.False(t, resp.OK)
	assert.Nil(t, resp.AdvanceAllowed)
}
