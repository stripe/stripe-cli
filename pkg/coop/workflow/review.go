package workflow

import (
	"fmt"
	"strings"
	"time"

	"github.com/stripe/stripe-cli/pkg/coop"
	"github.com/stripe/stripe-cli/pkg/coop/helpers"
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

// AwaitReview waits for the developer's review of the step containing
// nodeNumber. It returns a waiting response rather than blocking indefinitely
// so an agent harness cannot kill it mid-wait; the agent runs it again until
// advance_allowed is true.
func (s *Service) AwaitReview(sessionID string, nodeNumber int) (coop.CommandResponse, error) {
	session, err := s.store.Read(sessionID)
	if err != nil {
		return sessionErrorResponse(err), nil
	}
	if err := requireActiveSession(session); err != nil {
		// A completed session is not a failure here. The developer can confirm
		// the final task between two await calls, and the agent is told to keep
		// re-running await until advance_allowed is true — so the common case
		// would otherwise be a non-zero exit with an empty stdout, which reads
		// to an agent as a broken session. Resume turns it into the next-action
		// command, and still reports a genuinely aborted session as an error.
		return s.Resume(sessionID)
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
			if activeNode, _ := session.NodeByNumber(activeNodeNumber); isUnstartedRejection(activeNode) {
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

// awaitStepReview blocks for at most one await interval. If the developer has
// not acted by then it returns a waiting response asking the agent to run the
// same command again, rather than holding the process long enough for an agent
// harness to kill it. The developer's total review time stays unbounded because
// the state being polled lives on disk, not in this process.
func (s *Service) awaitStepReview(sessionID, stepTitle string, stepIndex, nodeNumber int) (coop.CommandResponse, error) {
	if err := s.store.WriteHeartbeat(sessionID); err != nil {
		return coop.CommandResponse{}, err
	}
	// Removed on every return, including waiting. Retaining it between calls
	// would not stop the TUI reporting the agent idle anyway — that check falls
	// through to the last session change, which is frozen for the whole review —
	// and would leave a killed agent indistinguishable from a patient one.
	defer func() {
		_ = s.store.RemoveHeartbeat(sessionID)
	}()

	start := s.now()
	deadline := start.Add(s.awaitTimeout)
	lastProgress := start

	for {
		// Poll before checking the deadline so a very short interval still
		// performs one read: a decision can land while the process starts.
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
		if !session.StepHasReview(stepIndex) {
			return confirmedResponse(session, nodeNumber), nil
		}

		now := s.now()
		waited := helpers.WaitedInReview(session, stepIndex, now)
		if s.progress != nil && now.Sub(lastProgress) >= awaitProgressEvery {
			lastProgress = now
			fmt.Fprintf(s.progress, "Waiting for the developer to review %q (%s so far)...\n",
				stepTitle, formatWait(waited))
		}
		if now.After(deadline) {
			// Deliberately leave the heartbeat in place. Clearing it on every
			// interval would make the TUI flash "agent appears idle" during the
			// gap between calls for any review over two minutes. A genuinely
			// dead agent still goes stale inside the TUI's freshness window.
			return waitingResponse(sessionID, nodeNumber, s.awaitTimeout, waited,
				helpers.StepReviewPrompt(session, stepIndex)), nil
		}
	}
}
