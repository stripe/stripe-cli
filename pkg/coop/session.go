package coop

import (
	"fmt"
	"time"
)

// validTransitions defines allowed state transitions.
var validTransitions = map[StepState][]StepState{
	StepPending: {StepActive, StepSkipped},
	StepActive:  {StepReview, StepDone, StepSkipped},
	StepReview:  {StepDone, StepActive, StepSkipped}, // active = rejected, redo
}

// TotalSteps returns the total number of nodes across all chapters.
func (s *Session) TotalSteps() int {
	count := 0
	for _, ch := range s.Chapters {
		count += len(ch.Nodes)
	}
	return count
}

// NodeByNumber returns a pointer to the node at the given 1-based index,
// counting sequentially across chapters. Returns an error if out of range.
func (s *Session) NodeByNumber(n int) (*SessionNode, error) {
	if n < 1 {
		return nil, fmt.Errorf("step number must be >= 1, got %d", n)
	}
	idx := 0
	for i := range s.Chapters {
		for j := range s.Chapters[i].Nodes {
			idx++
			if idx == n {
				return &s.Chapters[i].Nodes[j], nil
			}
		}
	}
	return nil, fmt.Errorf("step %d out of range (session has %d steps)", n, s.TotalSteps())
}

// ActiveNode returns the first node in active state, or nil.
func (s *Session) ActiveNode() (*SessionNode, int) {
	idx := 0
	for i := range s.Chapters {
		for j := range s.Chapters[i].Nodes {
			idx++
			if s.Chapters[i].Nodes[j].State == StepActive {
				return &s.Chapters[i].Nodes[j], idx
			}
		}
	}
	return nil, 0
}

// TransitionStep validates and applies a state transition on step n.
func (s *Session) TransitionStep(n int, to StepState) error {
	node, err := s.NodeByNumber(n)
	if err != nil {
		return err
	}

	allowed, ok := validTransitions[node.State]
	if !ok {
		return fmt.Errorf("step %d is in terminal state %q, cannot transition", n, node.State)
	}

	valid := false
	for _, target := range allowed {
		if target == to {
			valid = true
			break
		}
	}
	if !valid {
		return fmt.Errorf("invalid transition: step %d is %q, cannot move to %q", n, node.State, to)
	}

	node.State = to
	now := time.Now().UTC()

	switch to {
	case StepActive:
		node.StartedAt = &now
	case StepDone:
		node.CompletedAt = &now
	case StepReview:
		node.CompletedAt = &now
	}

	return nil
}

// NextPendingStep returns the number of the next pending step after n, or 0.
func (s *Session) NextPendingStep(after int) int {
	idx := 0
	for i := range s.Chapters {
		for j := range s.Chapters[i].Nodes {
			idx++
			if idx > after && s.Chapters[i].Nodes[j].State == StepPending {
				return idx
			}
		}
	}
	return 0
}

// IsComplete returns true when all nodes are done or skipped.
func (s *Session) IsComplete() bool {
	for i := range s.Chapters {
		for j := range s.Chapters[i].Nodes {
			state := s.Chapters[i].Nodes[j].State
			if state != StepDone && state != StepSkipped {
				return false
			}
		}
	}
	return true
}

// StepSummary returns counts of each state.
func (s *Session) StepSummary() map[StepState]int {
	summary := make(map[StepState]int)
	for i := range s.Chapters {
		for j := range s.Chapters[i].Nodes {
			summary[s.Chapters[i].Nodes[j].State]++
		}
	}
	return summary
}
