package coop

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func testSessionNode(key, title string, state NodeState) SessionNode {
	return SessionNode{
		NodeDefinition: NodeDefinition{Key: key, Title: title},
		State:          state,
	}
}

func newTestSession() *Session {
	return &Session{
		ID:        "test_abc123",
		Blueprint: "one-time-payment",
		Status:    SessionActive,
		Steps: []SessionStep{
			{
				StepDefinition: StepDefinition{Key: "step-1", Title: "Step 1"},
				Nodes: []SessionNode{
					testSessionNode("node-1", "Step 1", NodePending),
					testSessionNode("node-2", "Step 2", NodePending),
				},
			},
			{
				StepDefinition: StepDefinition{Key: "step-2", Title: "Step 2"},
				Nodes: []SessionNode{
					testSessionNode("node-3", "Step 3", NodePending),
				},
			},
		},
	}
}

func TestTotalNodes(t *testing.T) {
	s := newTestSession()
	assert.Equal(t, 3, s.TotalNodes())
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

func TestTransitionNode(t *testing.T) {
	s := newTestSession()

	// pending -> active
	err := s.TransitionNode(1, NodeActive)
	require.NoError(t, err)
	node, _ := s.NodeByNumber(1)
	assert.Equal(t, NodeActive, node.State)
	assert.NotNil(t, node.StartedAt)

	// active -> review
	err = s.TransitionNode(1, NodeReview)
	require.NoError(t, err)
	node, _ = s.NodeByNumber(1)
	assert.Equal(t, NodeReview, node.State)

	// review -> done
	err = s.TransitionNode(1, NodeDone)
	require.NoError(t, err)
	node, _ = s.NodeByNumber(1)
	assert.Equal(t, NodeDone, node.State)

	// done -> active (invalid)
	err = s.TransitionNode(1, NodeActive)
	assert.Error(t, err)
}

func TestTransitionNodeSkip(t *testing.T) {
	s := newTestSession()

	// pending -> skipped
	err := s.TransitionNode(1, NodeSkipped)
	require.NoError(t, err)
	node, _ := s.NodeByNumber(1)
	assert.Equal(t, NodeSkipped, node.State)
}

func TestActiveNode(t *testing.T) {
	s := newTestSession()

	node, num := s.ActiveNode()
	assert.Nil(t, node)
	assert.Equal(t, 0, num)

	s.TransitionNode(2, NodeActive)
	node, num = s.ActiveNode()
	assert.Equal(t, "node-2", node.Key)
	assert.Equal(t, 2, num)
}

func TestNextPendingNode(t *testing.T) {
	s := newTestSession()

	assert.Equal(t, 1, s.NextPendingNode(0))
	assert.Equal(t, 2, s.NextPendingNode(1))
	assert.Equal(t, 3, s.NextPendingNode(2))
	assert.Equal(t, 0, s.NextPendingNode(3))
}

func TestIsComplete(t *testing.T) {
	s := newTestSession()
	assert.False(t, s.IsComplete())

	for i := 1; i <= 3; i++ {
		s.TransitionNode(i, NodeActive)
		s.TransitionNode(i, NodeDone)
	}
	assert.True(t, s.IsComplete())
}

func TestNodeSummary(t *testing.T) {
	s := newTestSession()
	s.TransitionNode(1, NodeActive)
	s.TransitionNode(1, NodeDone)

	summary := s.NodeSummary()
	assert.Equal(t, 1, summary[NodeDone])
	assert.Equal(t, 2, summary[NodePending])
}

func TestTransitionNodeInvalidFromDone(t *testing.T) {
	s := newTestSession()
	s.TransitionNode(1, NodeActive)
	s.TransitionNode(1, NodeDone)

	// Can't go anywhere from done
	assert.Error(t, s.TransitionNode(1, NodeActive))
	assert.Error(t, s.TransitionNode(1, NodePending))
	assert.Error(t, s.TransitionNode(1, NodeReview))
}

func TestTransitionNodeInvalidFromSkipped(t *testing.T) {
	s := newTestSession()
	s.TransitionNode(1, NodeSkipped)

	// Can't go anywhere from skipped
	assert.Error(t, s.TransitionNode(1, NodeActive))
	assert.Error(t, s.TransitionNode(1, NodeDone))
}

func TestTransitionNodeInvalidPendingToDone(t *testing.T) {
	s := newTestSession()
	// Can't go directly from pending to done
	assert.Error(t, s.TransitionNode(1, NodeDone))
}

func TestTransitionNodeInvalidPendingToReview(t *testing.T) {
	s := newTestSession()
	// Can't go directly from pending to review
	assert.Error(t, s.TransitionNode(1, NodeReview))
}

func TestTransitionNodeActiveToDone(t *testing.T) {
	s := newTestSession()
	s.TransitionNode(1, NodeActive)
	// Can go directly from active to done (--auto-confirm)
	err := s.TransitionNode(1, NodeDone)
	assert.NoError(t, err)
	node, _ := s.NodeByNumber(1)
	assert.Equal(t, NodeDone, node.State)
	assert.NotNil(t, node.CompletedAt)
}

func TestTransitionNodeSetsTimestamps(t *testing.T) {
	s := newTestSession()

	s.TransitionNode(1, NodeActive)
	node, _ := s.NodeByNumber(1)
	assert.NotNil(t, node.StartedAt)
	assert.Nil(t, node.CompletedAt)

	s.TransitionNode(1, NodeReview)
	node, _ = s.NodeByNumber(1)
	assert.NotNil(t, node.CompletedAt)
}

func TestTransitionNodePreservesOriginalStartedAt(t *testing.T) {
	s := newTestSession()

	require.NoError(t, s.TransitionNode(1, NodeActive))
	node, _ := s.NodeByNumber(1)
	firstStartedAt := node.StartedAt
	require.NotNil(t, firstStartedAt)

	require.NoError(t, s.TransitionNode(1, NodeReview))
	require.NoError(t, s.TransitionNode(1, NodeActive))
	node, _ = s.NodeByNumber(1)
	require.NotNil(t, node.StartedAt)
	assert.True(t, node.StartedAt.Equal(*firstStartedAt))
	assert.Nil(t, node.CompletedAt)
}

func TestTransitionNodeSkippedSetsCompletedAt(t *testing.T) {
	s := newTestSession()

	require.NoError(t, s.TransitionNode(1, NodeSkipped))
	node, _ := s.NodeByNumber(1)
	assert.NotNil(t, node.CompletedAt)
}

func TestNextPendingNodeSkipsNonPending(t *testing.T) {
	s := newTestSession()
	s.TransitionNode(1, NodeActive)
	s.TransitionNode(2, NodeSkipped)

	// After step 1 (active), next pending should be step 3 (skipping 2 which is skipped)
	assert.Equal(t, 3, s.NextPendingNode(1))
}

func TestIsCompleteWithSkipped(t *testing.T) {
	s := newTestSession()
	s.TransitionNode(1, NodeActive)
	s.TransitionNode(1, NodeDone)
	s.TransitionNode(2, NodeSkipped)
	s.TransitionNode(3, NodeActive)
	s.TransitionNode(3, NodeDone)

	assert.True(t, s.IsComplete())
}

func TestNodeByNumberAcrossSteps(t *testing.T) {
	s := &Session{
		Steps: []SessionStep{
			{StepDefinition: StepDefinition{Key: "ch1"}, Nodes: []SessionNode{testSessionNode("a", "", ""), testSessionNode("b", "", "")}},
			{StepDefinition: StepDefinition{Key: "ch2"}, Nodes: []SessionNode{testSessionNode("c", "", "")}},
			{StepDefinition: StepDefinition{Key: "ch3"}, Nodes: []SessionNode{testSessionNode("d", "", ""), testSessionNode("e", "", ""), testSessionNode("f", "", "")}},
		},
	}

	node, _ := s.NodeByNumber(3)
	assert.Equal(t, "c", node.Key)

	node, _ = s.NodeByNumber(6)
	assert.Equal(t, "f", node.Key)

	assert.Equal(t, 6, s.TotalNodes())
}

func TestStepByNodeNumber(t *testing.T) {
	s := newTestSession()

	ch, stepIndex, nodeIndex, err := s.StepByNodeNumber(3)
	require.NoError(t, err)
	assert.Equal(t, "step-2", ch.Key)
	assert.Equal(t, 1, stepIndex)
	assert.Equal(t, 0, nodeIndex)

	ch, stepIndex, nodeIndex, err = s.StepByNodeNumber(0)
	assert.Nil(t, ch)
	assert.Equal(t, -1, stepIndex)
	assert.Equal(t, -1, nodeIndex)
	assert.Error(t, err)

	ch, stepIndex, nodeIndex, err = s.StepByNodeNumber(4)
	assert.Nil(t, ch)
	assert.Equal(t, -1, stepIndex)
	assert.Equal(t, -1, nodeIndex)
	assert.Error(t, err)
}

func TestStepReadyForReview(t *testing.T) {
	s := newTestSession()
	assert.False(t, s.StepReadyForReview(-1))
	assert.False(t, s.StepReadyForReview(99))

	s.Steps[0].Nodes[0].State = NodeReview
	s.Steps[0].Nodes[1].State = NodePending
	assert.False(t, s.StepReadyForReview(0))

	s.Steps[0].Nodes[1].State = NodeSkipped
	assert.True(t, s.StepReadyForReview(0))

	s.Steps[0].Nodes[0].State = NodeActive
	assert.False(t, s.StepReadyForReview(0))
}

func TestStepReadyForReviewIgnoresAutoConfirmNodes(t *testing.T) {
	s := &Session{
		Steps: []SessionStep{
			{Nodes: []SessionNode{
				testSessionNode("a", "", NodeReview),
				{
					NodeDefinition: NodeDefinition{Key: "auto", AutoConfirm: true},
					State:          NodePending,
				},
			}},
			{Nodes: []SessionNode{
				{
					NodeDefinition: NodeDefinition{Key: "only-auto", AutoConfirm: true},
					State:          NodePending,
				},
			}},
		},
	}

	assert.True(t, s.StepReadyForReview(0))
	assert.False(t, s.StepReadyForReview(1))
}

func TestStepReviewStateHelpers(t *testing.T) {
	s := newTestSession()
	s.Steps[0].Nodes[0].State = NodeDone
	s.Steps[0].Nodes[1].State = NodeReview
	s.Steps[1].Nodes[0].State = NodeActive

	assert.True(t, s.StepHasReview(0))
	assert.False(t, s.StepHasReview(1))
	assert.False(t, s.StepHasReview(-1))

	assert.Equal(t, 2, s.FirstReviewNodeInStep(0))
	assert.Equal(t, 0, s.FirstReviewNodeInStep(1))
	assert.Equal(t, 0, s.FirstReviewNodeInStep(99))

	assert.Equal(t, 0, s.FirstActiveNodeInStep(0))
	assert.Equal(t, 3, s.FirstActiveNodeInStep(1))
	assert.Equal(t, 0, s.FirstActiveNodeInStep(-1))
}

func TestActiveNodeReturnsFirst(t *testing.T) {
	s := newTestSession()
	s.TransitionNode(1, NodeActive)
	s.Steps[1].Nodes[0].State = NodeActive // manually set step 3 active too

	node, num := s.ActiveNode()
	// Should return the FIRST active node
	assert.Equal(t, "node-1", node.Key)
	assert.Equal(t, 1, num)
}

func TestTransitionNodeReviewToActive(t *testing.T) {
	s := newTestSession()
	s.TransitionNode(1, NodeActive)
	s.TransitionNode(1, NodeReview)

	// Rejection: review -> active
	err := s.TransitionNode(1, NodeActive)
	assert.NoError(t, err)
	node, _ := s.NodeByNumber(1)
	assert.Equal(t, NodeActive, node.State)
}

func TestTransitionNodeActiveToSkipped(t *testing.T) {
	s := newTestSession()
	s.TransitionNode(1, NodeActive)

	err := s.TransitionNode(1, NodeSkipped)
	assert.NoError(t, err)
	node, _ := s.NodeByNumber(1)
	assert.Equal(t, NodeSkipped, node.State)
}

func TestIsCompleteAllSkipped(t *testing.T) {
	s := newTestSession()
	for i := 1; i <= s.TotalNodes(); i++ {
		s.TransitionNode(i, NodeSkipped)
	}
	assert.True(t, s.IsComplete())
}

func TestIsCompletePartialDone(t *testing.T) {
	s := newTestSession()
	s.TransitionNode(1, NodeActive)
	s.TransitionNode(1, NodeDone)
	// Steps 2 and 3 still pending
	assert.False(t, s.IsComplete())
}

func TestNodeByNumberNegative(t *testing.T) {
	s := newTestSession()
	_, err := s.NodeByNumber(-1)
	assert.Error(t, err)
}

func TestTransitionNodeActiveToActive(t *testing.T) {
	s := newTestSession()
	s.TransitionNode(1, NodeActive)
	// Can't transition active -> active (not in valid transitions)
	err := s.TransitionNode(1, NodeActive)
	assert.Error(t, err)
}

func TestTransitionNodeReviewToSkipped(t *testing.T) {
	s := newTestSession()
	s.TransitionNode(1, NodeActive)
	s.TransitionNode(1, NodeReview)

	err := s.TransitionNode(1, NodeSkipped)
	assert.NoError(t, err)
	node, _ := s.NodeByNumber(1)
	assert.Equal(t, NodeSkipped, node.State)
}

func TestIsCompleteEmptySession(t *testing.T) {
	// A session with no nodes must not report complete (avoids showing a fresh
	// or malformed empty session as "Integration complete").
	assert.False(t, (&Session{}).IsComplete())
	assert.False(t, (&Session{Steps: []SessionStep{{}}}).IsComplete())
}
