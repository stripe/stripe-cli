package workflow

import (
	"fmt"
	"strings"
	"time"

	"github.com/stripe/stripe-cli/pkg/coop"
)

func (s *Service) ConfirmReview(sessionID string, nodeNumbers []int) (*coop.Session, error) {
	return s.store.Update(sessionID, func(session *coop.Session) error {
		if err := requireActiveSession(session); err != nil {
			return err
		}
		for _, nodeNumber := range nodeNumbers {
			node, err := session.NodeByNumber(nodeNumber)
			if err != nil {
				return err
			}
			if node.State == coop.NodeDone || node.State == coop.NodeSkipped {
				continue
			}
			if err := session.TransitionNode(nodeNumber, coop.NodeDone); err != nil {
				return err
			}
		}
		if session.IsComplete() {
			session.Status = coop.SessionCompleted
		}
		return nil
	})
}

func (s *Service) RequestChanges(sessionID string, nodeNumbers []int, note string) (*coop.Session, error) {
	if strings.TrimSpace(note) == "" {
		return nil, fmt.Errorf("request changes note is required")
	}
	return s.store.Update(sessionID, func(session *coop.Session) error {
		if err := requireActiveSession(session); err != nil {
			return err
		}
		for _, nodeNumber := range nodeNumbers {
			node, err := session.NodeByNumber(nodeNumber)
			if err != nil {
				return err
			}
			if node.State != coop.NodeActive {
				if err := session.TransitionNode(nodeNumber, coop.NodeActive); err != nil {
					return err
				}
				node, _ = session.NodeByNumber(nodeNumber)
			}
			node.RejectionNote = note
			node.Implementation = nil
			node.Outputs = nil
			node.Verifications = nil
		}
		return nil
	})
}

func (s *Service) AwaitReview(sessionID string, nodeNumber int) (coop.CommandResponse, error) {
	session, err := s.store.Read(sessionID)
	if err != nil {
		return sessionErrorResponse(err), nil
	}
	if err := requireActiveSession(session); err != nil {
		return sessionErrorResponse(err), nil
	}
	node, err := session.NodeByNumber(nodeNumber)
	if err != nil {
		return sessionErrorResponse(err), nil
	}

	if node.IsInformationalNode && node.State == coop.NodeReview {
		return s.autoConfirm(sessionID, nodeNumber)
	}

	// The developer can request changes between two await-review calls, which
	// moves the node out of review and back to active with a rejection note.
	// Falling through to alreadyMovedResponse would point the agent at the next
	// *pending* task, silently skipping the rejected work and dropping the
	// feedback. Requiring a rejection note keeps a step where one task is
	// awaited while another is legitimately active from reading as a rejection.
	if step, stepIndex, _, stepErr := session.StepByNodeNumber(nodeNumber); stepErr == nil {
		if activeNodeNumber := session.FirstActiveNodeInStep(stepIndex); activeNodeNumber > 0 {
			if activeNode, _ := session.NodeByNumber(activeNodeNumber); activeNode != nil && activeNode.RejectionNote != "" {
				return rejectedStepResponse(session, step.TitleText(), activeNodeNumber, activeNode), nil
			}
		}
	}

	if node.State == coop.NodeReview {
		step, stepIndex, _, err := session.StepByNodeNumber(nodeNumber)
		if err != nil {
			return sessionErrorResponse(err), nil
		}
		if !session.StepReadyForReview(stepIndex) {
			return nodeResponse(
				session.ID, nodeNumber, string(coop.NodeReview),
				fmt.Sprintf("Task %d is ready. Continue the step before asking for human review.", nodeNumber),
				coop.Continue(nextInStepOrStatus(session, stepIndex, nodeNumber)),
			), nil
		}
		return s.awaitStepReview(session.ID, step.TitleText(), stepIndex, nodeNumber)
	}
	// Node is not in review (auto-confirm handled above, review handled in the
	// block above): it has already moved on. Review always waits at step
	// granularity via awaitStepReview.
	return alreadyMovedResponse(session, nodeNumber, node.State), nil
}

func (s *Service) autoConfirm(sessionID string, nodeNumber int) (coop.CommandResponse, error) {
	session, err := s.ConfirmReview(sessionID, []int{nodeNumber})
	if err != nil {
		return sessionErrorResponse(err), nil
	}
	return nodeResponse(
		session.ID, nodeNumber, "confirmed",
		fmt.Sprintf("Task %d auto-confirmed. Proceed to the next task.", nodeNumber),
		nextAfterNode(session, nodeNumber),
	), nil
}

func (s *Service) awaitStepReview(sessionID, stepTitle string, stepIndex, nodeNumber int) (coop.CommandResponse, error) {
	if err := s.store.WriteHeartbeat(sessionID); err != nil {
		return coop.CommandResponse{}, err
	}
	defer func() {
		_ = s.store.RemoveHeartbeat(sessionID)
	}()

	deadline := s.now().Add(s.awaitTimeout)
	for {
		if s.now().After(deadline) {
			return timeoutResponse(sessionID, nodeNumber, s.awaitTimeout), nil
		}
		s.sleep(500 * time.Millisecond)
		if err := s.store.WriteHeartbeat(sessionID); err != nil {
			return coop.CommandResponse{}, err
		}

		session, err := s.store.Read(sessionID)
		if err != nil {
			return coop.CommandResponse{}, err
		}
		if activeNodeNumber := session.FirstActiveNodeInStep(stepIndex); activeNodeNumber > 0 {
			activeNode, _ := session.NodeByNumber(activeNodeNumber)
			return rejectedStepResponse(session, stepTitle, activeNodeNumber, activeNode), nil
		}
		if session.StepHasReview(stepIndex) {
			continue
		}
		return confirmedResponse(session, nodeNumber), nil
	}
}
