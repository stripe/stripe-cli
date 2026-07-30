package workflow

import (
	"fmt"

	"github.com/stripe/stripe-cli/pkg/coop"
)

// Resume reports the current non-blocking lifecycle continuation. It is
// read-only so duplicate wake-ups cannot advance the session twice.
func (s *Service) Resume(sessionID string) (coop.CommandResponse, error) {
	session, err := s.store.Read(sessionID)
	if err != nil {
		return sessionErrorResponse(err), nil
	}
	if session.Status == coop.SessionAborted {
		return errorResponse(
			fmt.Errorf("session %s is %s and cannot be resumed", session.ID, session.Status),
			"Inspect the session before choosing a recovery action.",
			coop.Continue(coop.StatusCommand(session.ID)),
		), nil
	}

	if session.Status == coop.SessionCompleted || session.IsComplete() {
		return nodeResponse(
			session.ID,
			session.TotalNodes(),
			string(coop.SessionCompleted),
			"Session work is complete. Continue with the exact next-action command.",
			nextAfterNode(session, session.TotalNodes()),
		), nil
	}

	if activeNode, nodeNumber := session.ActiveNode(); activeNode != nil {
		if isUnstartedRejection(activeNode) {
			return rejectedResponse(session, nodeNumber, activeNode), nil
		}
		// No next to hand back: the agent already has the context and should
		// simply carry on. Leave advance_allowed unset rather than promising a
		// command that is not there.
		response := nodeResponse(
			session.ID,
			nodeNumber,
			string(coop.NodeActive),
			fmt.Sprintf("Task %d is already active. Continue the current work; no resume command is needed.", nodeNumber),
			coop.Continuation{},
		)
		response.AdvanceAllowed = nil
		return response, nil
	}

	nodeNumber := 0
	for stepIndex := range session.Steps {
		step := &session.Steps[stepIndex]
		if session.StepReadyForReview(stepIndex) && session.StepHasReview(stepIndex) {
			reviewNode := session.FirstReviewNodeInStep(stepIndex)
			return nodeResponse(
				session.ID,
				reviewNode,
				string(coop.NodeReview),
				fmt.Sprintf("Step %q is waiting for human review.", step.TitleText()),
				waitFor(coop.AwaitReviewCommand(session.ID, reviewNode), s.awaitTimeout),
			), nil
		}
		for nodeIndex := range step.Nodes {
			nodeNumber++
			node := &step.Nodes[nodeIndex]
			if node.State == coop.NodePending {
				return nodeResponse(
					session.ID,
					nodeNumber,
					string(coop.NodePending),
					fmt.Sprintf("Resume with the next pending node: %s", node.TitleText()),
					coop.Continue(coop.StartWorkCommand(session.ID, nodeNumber, "Beginning: "+node.TitleText())),
				), nil
			}
		}
	}

	return errorResponse(
		fmt.Errorf("session %s has no resumable lifecycle state", session.ID),
		"Inspect the session before choosing a recovery action.",
		coop.Continue(coop.StatusCommand(session.ID)),
	), nil
}

func rejectedStepResponse(session *coop.Session, stepTitle string, nodeNumber int, node *coop.SessionNode) coop.CommandResponse {
	response := rejectedResponse(session, nodeNumber, node)
	response.Message = fmt.Sprintf("Step %q requested changes.", stepTitle)
	if node != nil && node.RejectionNote != "" {
		response.Message += fmt.Sprintf("\nFeedback: %s", node.RejectionNote)
	}
	response.Message += "\nRedo the step from the first affected task."
	return response
}

func rejectedResponse(session *coop.Session, nodeNumber int, node *coop.SessionNode) coop.CommandResponse {
	title := "affected task"
	if node != nil && node.TitleText() != "" {
		title = node.TitleText()
	}
	message := fmt.Sprintf("Task %d has requested changes. Redo the affected work.", nodeNumber)
	if node != nil && node.RejectionNote != "" {
		message += fmt.Sprintf("\nFeedback: %s", node.RejectionNote)
	}
	return nodeResponse(
		session.ID,
		nodeNumber,
		"rejected",
		message,
		coop.Continue(coop.StartWorkCommand(session.ID, nodeNumber, "Redoing: "+title)),
	)
}

// isUnstartedRejection reports a task the developer sent back that the agent
// has not picked up yet. RejectionNote survives until the task reaches Done, so
// the note alone is not enough: start-work sets Activity, and a non-empty
// Activity means the redo is already underway. Re-reporting the rejection then
// would restart work the agent is in the middle of.
func isUnstartedRejection(node *coop.SessionNode) bool {
	return node != nil && node.RejectionNote != "" && node.Activity == ""
}
