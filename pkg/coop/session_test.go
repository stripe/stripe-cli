package coop

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newTestSession() *Session {
	return &Session{
		ID:        "test_abc123",
		Blueprint: "one-time-payment",
		Status:    SessionActive,
		Chapters: []SessionChapter{
			{
				Key:   "chapter-1",
				Title: "Chapter 1",
				Nodes: []SessionNode{
					{Key: "node-1", Title: "Step 1", State: StepPending},
					{Key: "node-2", Title: "Step 2", State: StepPending},
				},
			},
			{
				Key:   "chapter-2",
				Title: "Chapter 2",
				Nodes: []SessionNode{
					{Key: "node-3", Title: "Step 3", State: StepPending},
				},
			},
		},
	}
}

func TestTotalSteps(t *testing.T) {
	s := newTestSession()
	assert.Equal(t, 3, s.TotalSteps())
}

func TestNodeByNumber(t *testing.T) {
	s := newTestSession()

	node, err := s.NodeByNumber(1)
	require.NoError(t, err)
	assert.Equal(t, "node-1", node.Key)

	node, err = s.NodeByNumber(3)
	require.NoError(t, err)
	assert.Equal(t, "node-3", node.Key)

	_, err = s.NodeByNumber(0)
	assert.Error(t, err)

	_, err = s.NodeByNumber(4)
	assert.Error(t, err)
}

func TestTransitionStep(t *testing.T) {
	s := newTestSession()

	// pending -> active
	err := s.TransitionStep(1, StepActive)
	require.NoError(t, err)
	node, _ := s.NodeByNumber(1)
	assert.Equal(t, StepActive, node.State)
	assert.NotNil(t, node.StartedAt)

	// active -> review
	err = s.TransitionStep(1, StepReview)
	require.NoError(t, err)
	node, _ = s.NodeByNumber(1)
	assert.Equal(t, StepReview, node.State)

	// review -> done
	err = s.TransitionStep(1, StepDone)
	require.NoError(t, err)
	node, _ = s.NodeByNumber(1)
	assert.Equal(t, StepDone, node.State)

	// done -> active (invalid)
	err = s.TransitionStep(1, StepActive)
	assert.Error(t, err)
}

func TestTransitionStepSkip(t *testing.T) {
	s := newTestSession()

	// pending -> skipped
	err := s.TransitionStep(1, StepSkipped)
	require.NoError(t, err)
	node, _ := s.NodeByNumber(1)
	assert.Equal(t, StepSkipped, node.State)
}

func TestActiveNode(t *testing.T) {
	s := newTestSession()

	node, num := s.ActiveNode()
	assert.Nil(t, node)
	assert.Equal(t, 0, num)

	s.TransitionStep(2, StepActive)
	node, num = s.ActiveNode()
	assert.Equal(t, "node-2", node.Key)
	assert.Equal(t, 2, num)
}

func TestNextPendingStep(t *testing.T) {
	s := newTestSession()

	assert.Equal(t, 1, s.NextPendingStep(0))
	assert.Equal(t, 2, s.NextPendingStep(1))
	assert.Equal(t, 3, s.NextPendingStep(2))
	assert.Equal(t, 0, s.NextPendingStep(3))
}

func TestIsComplete(t *testing.T) {
	s := newTestSession()
	assert.False(t, s.IsComplete())

	for i := 1; i <= 3; i++ {
		s.TransitionStep(i, StepActive)
		s.TransitionStep(i, StepDone)
	}
	assert.True(t, s.IsComplete())
}

func TestStepSummary(t *testing.T) {
	s := newTestSession()
	s.TransitionStep(1, StepActive)
	s.TransitionStep(1, StepDone)

	summary := s.StepSummary()
	assert.Equal(t, 1, summary[StepDone])
	assert.Equal(t, 2, summary[StepPending])
}

func TestTransitionStepInvalidFromDone(t *testing.T) {
	s := newTestSession()
	s.TransitionStep(1, StepActive)
	s.TransitionStep(1, StepDone)

	// Can't go anywhere from done
	assert.Error(t, s.TransitionStep(1, StepActive))
	assert.Error(t, s.TransitionStep(1, StepPending))
	assert.Error(t, s.TransitionStep(1, StepReview))
}

func TestTransitionStepInvalidFromSkipped(t *testing.T) {
	s := newTestSession()
	s.TransitionStep(1, StepSkipped)

	// Can't go anywhere from skipped
	assert.Error(t, s.TransitionStep(1, StepActive))
	assert.Error(t, s.TransitionStep(1, StepDone))
}

func TestTransitionStepInvalidPendingToDone(t *testing.T) {
	s := newTestSession()
	// Can't go directly from pending to done
	assert.Error(t, s.TransitionStep(1, StepDone))
}

func TestTransitionStepInvalidPendingToReview(t *testing.T) {
	s := newTestSession()
	// Can't go directly from pending to review
	assert.Error(t, s.TransitionStep(1, StepReview))
}

func TestTransitionStepActiveToDone(t *testing.T) {
	s := newTestSession()
	s.TransitionStep(1, StepActive)
	// Can go directly from active to done (--auto-confirm)
	err := s.TransitionStep(1, StepDone)
	assert.NoError(t, err)
	node, _ := s.NodeByNumber(1)
	assert.Equal(t, StepDone, node.State)
	assert.NotNil(t, node.CompletedAt)
}

func TestTransitionStepSetsTimestamps(t *testing.T) {
	s := newTestSession()

	s.TransitionStep(1, StepActive)
	node, _ := s.NodeByNumber(1)
	assert.NotNil(t, node.StartedAt)
	assert.Nil(t, node.CompletedAt)

	s.TransitionStep(1, StepReview)
	node, _ = s.NodeByNumber(1)
	assert.NotNil(t, node.CompletedAt)
}

func TestNextPendingStepSkipsNonPending(t *testing.T) {
	s := newTestSession()
	s.TransitionStep(1, StepActive)
	s.TransitionStep(2, StepSkipped)

	// After step 1 (active), next pending should be step 3 (skipping 2 which is skipped)
	assert.Equal(t, 3, s.NextPendingStep(1))
}

func TestIsCompleteWithSkipped(t *testing.T) {
	s := newTestSession()
	s.TransitionStep(1, StepActive)
	s.TransitionStep(1, StepDone)
	s.TransitionStep(2, StepSkipped)
	s.TransitionStep(3, StepActive)
	s.TransitionStep(3, StepDone)

	assert.True(t, s.IsComplete())
}

func TestNodeByNumberAcrossChapters(t *testing.T) {
	s := &Session{
		Chapters: []SessionChapter{
			{Key: "ch1", Nodes: []SessionNode{{Key: "a"}, {Key: "b"}}},
			{Key: "ch2", Nodes: []SessionNode{{Key: "c"}}},
			{Key: "ch3", Nodes: []SessionNode{{Key: "d"}, {Key: "e"}, {Key: "f"}}},
		},
	}

	node, _ := s.NodeByNumber(3)
	assert.Equal(t, "c", node.Key)

	node, _ = s.NodeByNumber(6)
	assert.Equal(t, "f", node.Key)

	assert.Equal(t, 6, s.TotalSteps())
}

func TestChapterByStepNumber(t *testing.T) {
	s := newTestSession()

	ch, chapterIndex, nodeIndex, err := s.ChapterByStepNumber(3)
	require.NoError(t, err)
	assert.Equal(t, "chapter-2", ch.Key)
	assert.Equal(t, 1, chapterIndex)
	assert.Equal(t, 0, nodeIndex)

	ch, chapterIndex, nodeIndex, err = s.ChapterByStepNumber(0)
	assert.Nil(t, ch)
	assert.Equal(t, -1, chapterIndex)
	assert.Equal(t, -1, nodeIndex)
	assert.Error(t, err)

	ch, chapterIndex, nodeIndex, err = s.ChapterByStepNumber(4)
	assert.Nil(t, ch)
	assert.Equal(t, -1, chapterIndex)
	assert.Equal(t, -1, nodeIndex)
	assert.Error(t, err)
}

func TestReviewGranularityForStep(t *testing.T) {
	s := newTestSession()
	s.Chapters[0].ReviewGranularity = ReviewGranularityChapter
	s.Chapters[1].ReviewGranularity = ReviewGranularityAuto

	assert.Equal(t, ReviewGranularityChapter, s.ReviewGranularityForStep(1))
	assert.Equal(t, ReviewGranularityChapter, s.ReviewGranularityForStep(2))
	assert.Equal(t, ReviewGranularityAuto, s.ReviewGranularityForStep(3))
	assert.Equal(t, ReviewGranularityStep, s.ReviewGranularityForStep(4))
}

func TestChapterReadyForReview(t *testing.T) {
	s := newTestSession()
	assert.False(t, s.ChapterReadyForReview(-1))
	assert.False(t, s.ChapterReadyForReview(99))

	s.Chapters[0].Nodes[0].State = StepReview
	s.Chapters[0].Nodes[1].State = StepPending
	assert.False(t, s.ChapterReadyForReview(0))

	s.Chapters[0].Nodes[1].State = StepSkipped
	assert.True(t, s.ChapterReadyForReview(0))

	s.Chapters[0].Nodes[0].State = StepActive
	assert.False(t, s.ChapterReadyForReview(0))
}

func TestChapterReadyForReviewIgnoresAutoConfirmNodes(t *testing.T) {
	s := &Session{
		Chapters: []SessionChapter{
			{Nodes: []SessionNode{
				{Key: "a", State: StepReview},
				{Key: "auto", State: StepPending, AutoConfirm: true},
			}},
			{Nodes: []SessionNode{
				{Key: "only-auto", State: StepPending, AutoConfirm: true},
			}},
		},
	}

	assert.True(t, s.ChapterReadyForReview(0))
	assert.False(t, s.ChapterReadyForReview(1))
}

func TestChapterReviewStateHelpers(t *testing.T) {
	s := newTestSession()
	s.Chapters[0].Nodes[0].State = StepDone
	s.Chapters[0].Nodes[1].State = StepReview
	s.Chapters[1].Nodes[0].State = StepActive

	assert.True(t, s.ChapterHasReview(0))
	assert.False(t, s.ChapterHasReview(1))
	assert.False(t, s.ChapterHasReview(-1))

	assert.Equal(t, 2, s.FirstReviewStepInChapter(0))
	assert.Equal(t, 0, s.FirstReviewStepInChapter(1))
	assert.Equal(t, 0, s.FirstReviewStepInChapter(99))

	assert.Equal(t, 0, s.FirstActiveStepInChapter(0))
	assert.Equal(t, 3, s.FirstActiveStepInChapter(1))
	assert.Equal(t, 0, s.FirstActiveStepInChapter(-1))
}

func TestActiveNodeReturnsFirst(t *testing.T) {
	s := newTestSession()
	s.TransitionStep(1, StepActive)
	s.Chapters[1].Nodes[0].State = StepActive // manually set step 3 active too

	node, num := s.ActiveNode()
	// Should return the FIRST active node
	assert.Equal(t, "node-1", node.Key)
	assert.Equal(t, 1, num)
}

func TestTransitionStepReviewToActive(t *testing.T) {
	s := newTestSession()
	s.TransitionStep(1, StepActive)
	s.TransitionStep(1, StepReview)

	// Rejection: review -> active
	err := s.TransitionStep(1, StepActive)
	assert.NoError(t, err)
	node, _ := s.NodeByNumber(1)
	assert.Equal(t, StepActive, node.State)
}

func TestTransitionStepActiveToSkipped(t *testing.T) {
	s := newTestSession()
	s.TransitionStep(1, StepActive)

	err := s.TransitionStep(1, StepSkipped)
	assert.NoError(t, err)
	node, _ := s.NodeByNumber(1)
	assert.Equal(t, StepSkipped, node.State)
}

func TestIsCompleteAllSkipped(t *testing.T) {
	s := newTestSession()
	for i := 1; i <= s.TotalSteps(); i++ {
		s.TransitionStep(i, StepSkipped)
	}
	assert.True(t, s.IsComplete())
}

func TestIsCompletePartialDone(t *testing.T) {
	s := newTestSession()
	s.TransitionStep(1, StepActive)
	s.TransitionStep(1, StepDone)
	// Steps 2 and 3 still pending
	assert.False(t, s.IsComplete())
}

func TestNodeByNumberNegative(t *testing.T) {
	s := newTestSession()
	_, err := s.NodeByNumber(-1)
	assert.Error(t, err)
}

func TestTransitionStepActiveToActive(t *testing.T) {
	s := newTestSession()
	s.TransitionStep(1, StepActive)
	// Can't transition active -> active (not in valid transitions)
	err := s.TransitionStep(1, StepActive)
	assert.Error(t, err)
}

func TestTransitionStepReviewToSkipped(t *testing.T) {
	s := newTestSession()
	s.TransitionStep(1, StepActive)
	s.TransitionStep(1, StepReview)

	err := s.TransitionStep(1, StepSkipped)
	assert.NoError(t, err)
	node, _ := s.NodeByNumber(1)
	assert.Equal(t, StepSkipped, node.State)
}
