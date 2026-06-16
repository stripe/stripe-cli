package coop

import (
	"fmt"
	"time"
)

// validTransitions defines allowed state transitions.
var validTransitions = map[NodeState][]NodeState{
	NodePending: {NodeActive, NodeSkipped},
	NodeActive:  {NodeReview, NodeDone, NodeSkipped},
	NodeReview:  {NodeDone, NodeActive, NodeSkipped}, // active = rejected, redo
}

// TotalNodes returns the total number of nodes across all steps.
func (s *Session) TotalNodes() int {
	count := 0
	for _, ch := range s.Steps {
		count += len(ch.Nodes)
	}
	return count
}

// NodeByNumber returns a pointer to the node at the given 1-based index,
// counting sequentially across steps. Returns an error if out of range.
func (s *Session) NodeByNumber(n int) (*SessionNode, error) {
	if n < 1 {
		return nil, fmt.Errorf("node number must be >= 1, got %d", n)
	}
	idx := 0
	for i := range s.Steps {
		for j := range s.Steps[i].Nodes {
			idx++
			if idx == n {
				return &s.Steps[i].Nodes[j], nil
			}
		}
	}
	return nil, fmt.Errorf("node %d out of range (session has %d nodes)", n, s.TotalNodes())
}

// StepByNodeNumber returns the step containing a 1-based node number.
func (s *Session) StepByNodeNumber(n int) (*SessionStep, int, int, error) {
	if n < 1 {
		return nil, -1, -1, fmt.Errorf("node number must be >= 1, got %d", n)
	}
	idx := 0
	for i := range s.Steps {
		for j := range s.Steps[i].Nodes {
			idx++
			if idx == n {
				return &s.Steps[i], i, j, nil
			}
		}
	}
	return nil, -1, -1, fmt.Errorf("node %d out of range (session has %d nodes)", n, s.TotalNodes())
}

// ReviewGranularityForNode returns the effective review granularity for a node.
func (s *Session) ReviewGranularityForNode(n int) ReviewGranularity {
	ch, _, _, err := s.StepByNodeNumber(n)
	if err != nil || ch.ReviewGranularity == "" {
		return ReviewGranularityNode
	}
	return ch.ReviewGranularity
}

// StepReadyForReview returns true when every reviewable node in a step is
// no longer pending or active.
func (s *Session) StepReadyForReview(stepIndex int) bool {
	if stepIndex < 0 || stepIndex >= len(s.Steps) {
		return false
	}
	hasReviewable := false
	for _, n := range s.Steps[stepIndex].Nodes {
		if n.AutoConfirm {
			continue
		}
		hasReviewable = true
		switch n.State {
		case NodeReview, NodeDone, NodeSkipped:
		default:
			return false
		}
	}
	return hasReviewable
}

// StepHasReview returns true if any node in the step is waiting for human review.
func (s *Session) StepHasReview(stepIndex int) bool {
	if stepIndex < 0 || stepIndex >= len(s.Steps) {
		return false
	}
	for _, n := range s.Steps[stepIndex].Nodes {
		if n.State == NodeReview {
			return true
		}
	}
	return false
}

// FirstReviewNodeInStep returns the 1-based node number of the first review node.
func (s *Session) FirstReviewNodeInStep(stepIndex int) int {
	if stepIndex < 0 || stepIndex >= len(s.Steps) {
		return 0
	}
	nodeNumber := 0
	for i := range s.Steps {
		for j := range s.Steps[i].Nodes {
			nodeNumber++
			if i == stepIndex && s.Steps[i].Nodes[j].State == NodeReview {
				return nodeNumber
			}
		}
	}
	return 0
}

// FirstActiveNodeInStep returns the 1-based node number of the first active node.
func (s *Session) FirstActiveNodeInStep(stepIndex int) int {
	if stepIndex < 0 || stepIndex >= len(s.Steps) {
		return 0
	}
	nodeNumber := 0
	for i := range s.Steps {
		for j := range s.Steps[i].Nodes {
			nodeNumber++
			if i == stepIndex && s.Steps[i].Nodes[j].State == NodeActive {
				return nodeNumber
			}
		}
	}
	return 0
}

// ActiveNode returns the first node in active state, or nil.
func (s *Session) ActiveNode() (*SessionNode, int) {
	idx := 0
	for i := range s.Steps {
		for j := range s.Steps[i].Nodes {
			idx++
			if s.Steps[i].Nodes[j].State == NodeActive {
				return &s.Steps[i].Nodes[j], idx
			}
		}
	}
	return nil, 0
}

// TransitionNode validates and applies a state transition on node n.
func (s *Session) TransitionNode(n int, to NodeState) error {
	node, err := s.NodeByNumber(n)
	if err != nil {
		return err
	}

	allowed, ok := validTransitions[node.State]
	if !ok {
		return fmt.Errorf("node %d is in terminal state %q, cannot transition", n, node.State)
	}

	valid := false
	for _, target := range allowed {
		if target == to {
			valid = true
			break
		}
	}
	if !valid {
		return fmt.Errorf("invalid transition: node %d is %q, cannot move to %q", n, node.State, to)
	}

	node.State = to
	now := time.Now().UTC()

	switch to {
	case NodeActive:
		if node.StartedAt == nil {
			node.StartedAt = &now
		}
		node.CompletedAt = nil
	case NodeDone:
		node.CompletedAt = &now
		node.RejectionNote = ""
	case NodeReview:
		node.CompletedAt = &now
	case NodeSkipped:
		node.CompletedAt = &now
	}

	return nil
}

// NextPendingNode returns the number of the next pending node after n, or 0.
func (s *Session) NextPendingNode(after int) int {
	idx := 0
	for i := range s.Steps {
		for j := range s.Steps[i].Nodes {
			idx++
			if idx > after && s.Steps[i].Nodes[j].State == NodePending {
				return idx
			}
		}
	}
	return 0
}

// IsComplete returns true when all nodes are done or skipped.
func (s *Session) IsComplete() bool {
	for i := range s.Steps {
		for j := range s.Steps[i].Nodes {
			state := s.Steps[i].Nodes[j].State
			if state != NodeDone && state != NodeSkipped {
				return false
			}
		}
	}
	return true
}

// NodeSummary returns counts of each state.
func (s *Session) NodeSummary() map[NodeState]int {
	summary := make(map[NodeState]int)
	for i := range s.Steps {
		for j := range s.Steps[i].Nodes {
			summary[s.Steps[i].Nodes[j].State]++
		}
	}
	return summary
}
