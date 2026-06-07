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

// ChapterByStepNumber returns the chapter containing a 1-based step number.
func (s *Session) ChapterByStepNumber(n int) (*SessionChapter, int, int, error) {
	if n < 1 {
		return nil, -1, -1, fmt.Errorf("step number must be >= 1, got %d", n)
	}
	idx := 0
	for i := range s.Chapters {
		for j := range s.Chapters[i].Nodes {
			idx++
			if idx == n {
				return &s.Chapters[i], i, j, nil
			}
		}
	}
	return nil, -1, -1, fmt.Errorf("step %d out of range (session has %d steps)", n, s.TotalSteps())
}

// ReviewGranularityForStep returns the effective review granularity for a step.
func (s *Session) ReviewGranularityForStep(n int) ReviewGranularity {
	ch, _, _, err := s.ChapterByStepNumber(n)
	if err != nil || ch.ReviewGranularity == "" {
		return ReviewGranularityStep
	}
	return ch.ReviewGranularity
}

// ChapterReadyForReview returns true when every reviewable node in a chapter is
// no longer pending or active.
func (s *Session) ChapterReadyForReview(chapterIndex int) bool {
	if chapterIndex < 0 || chapterIndex >= len(s.Chapters) {
		return false
	}
	hasReviewable := false
	for _, n := range s.Chapters[chapterIndex].Nodes {
		if n.AutoConfirm {
			continue
		}
		hasReviewable = true
		switch n.State {
		case StepReview, StepDone, StepSkipped:
		default:
			return false
		}
	}
	return hasReviewable
}

// ChapterHasReview returns true if any node in the chapter is waiting for human review.
func (s *Session) ChapterHasReview(chapterIndex int) bool {
	if chapterIndex < 0 || chapterIndex >= len(s.Chapters) {
		return false
	}
	for _, n := range s.Chapters[chapterIndex].Nodes {
		if n.State == StepReview {
			return true
		}
	}
	return false
}

// FirstReviewStepInChapter returns the 1-based step number of the first review node.
func (s *Session) FirstReviewStepInChapter(chapterIndex int) int {
	if chapterIndex < 0 || chapterIndex >= len(s.Chapters) {
		return 0
	}
	step := 0
	for i := range s.Chapters {
		for j := range s.Chapters[i].Nodes {
			step++
			if i == chapterIndex && s.Chapters[i].Nodes[j].State == StepReview {
				return step
			}
		}
	}
	return 0
}

// FirstActiveStepInChapter returns the 1-based step number of the first active node.
func (s *Session) FirstActiveStepInChapter(chapterIndex int) int {
	if chapterIndex < 0 || chapterIndex >= len(s.Chapters) {
		return 0
	}
	step := 0
	for i := range s.Chapters {
		for j := range s.Chapters[i].Nodes {
			step++
			if i == chapterIndex && s.Chapters[i].Nodes[j].State == StepActive {
				return step
			}
		}
	}
	return 0
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
		node.CompletedAt = nil
	case StepDone:
		node.CompletedAt = &now
		node.RejectionNote = ""
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
