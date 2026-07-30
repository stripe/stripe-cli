package workflow

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/stripe/stripe-cli/pkg/coop"
	"github.com/stripe/stripe-cli/pkg/coop/helpers"
)

func (s *Service) reportWorkResponse(session *coop.Session, node *coop.SessionNode, nodeNumber int, targetState coop.NodeState) coop.CommandResponse {
	if targetState == coop.NodeReview {
		step, stepIndex, _, err := session.StepByNodeNumber(nodeNumber)
		if err == nil && !session.StepReadyForReview(stepIndex) {
			return nodeResponse(
				session.ID, nodeNumber, string(coop.NodeReview),
				fmt.Sprintf("Ready: %s. Continue the step before asking for human review.", node.TitleText()),
				coop.Continue(nextInStepOrStatus(session, stepIndex, nodeNumber)),
			)
		}
		if err == nil {
			return nodeResponse(
				session.ID, nodeNumber, string(coop.NodeReview),
				fmt.Sprintf("Step ready for review: %s. Run relevant checks, keep useful servers running, share local URLs or test data, then await review.", step.TitleText()),
				waitFor(coop.AwaitReviewCommand(session.ID, nodeNumber), s.awaitTimeout),
			)
		}
		return nodeResponse(
			session.ID, nodeNumber, string(coop.NodeReview),
			fmt.Sprintf("Ready for review: %s", node.TitleText()),
			waitFor(coop.AwaitReviewCommand(session.ID, nodeNumber), s.awaitTimeout),
		)
	}

	msg := fmt.Sprintf("Completed: %s", node.TitleText())
	next := nextAfterNode(session, nodeNumber)
	if session.IsComplete() {
		msg += " All tasks complete. Run next-action so the developer can choose what happens next."
	}
	return nodeResponse(session.ID, nodeNumber, string(targetState), msg, next)
}

func waitFor(command string, timeout time.Duration) coop.Continuation {
	return coop.Continue(command).WithWaitTimeout(int(timeout.Seconds()))
}

func nodeResponse(sessionID string, nodeNumber int, state, message string, continuation coop.Continuation) coop.CommandResponse {
	return coop.CommandResponse{
		OK:             true,
		SessionID:      sessionID,
		Node:           nodeNumber,
		State:          state,
		AdvanceAllowed: advanceAllowed(true),
		Message:        message,
		Continuation:   continuation,
	}
}

// advanceAllowed returns a pointer so call sites that never set the field omit
// it rather than emitting a false that would stall the agent.
func advanceAllowed(v bool) *bool {
	return &v
}

func nextAfterNode(session *coop.Session, nodeNumber int) coop.Continuation {
	if nextNodeNumber := session.NextPendingNode(nodeNumber); nextNodeNumber > 0 {
		nextNode, _ := session.NodeByNumber(nextNodeNumber)
		return coop.Continue(coop.StartWorkCommand(session.ID, nextNodeNumber, "Beginning: "+nextNode.TitleText()))
	}
	if session.IsComplete() {
		var command string
		if session.ParentSessionID != "" && session.ParentStepID != "" {
			command = coop.NextActionCommand(session.ParentSessionID, session.ParentStepID)
		} else {
			command = coop.NextActionCommand(session.ID, "")
		}
		return waitFor(command, helpers.NextActionInterval)
	}
	return coop.Continue(coop.StatusCommand(session.ID))
}

func nextInStepOrStatus(session *coop.Session, stepIndex, afterNode int) string {
	if nextNodeNumber := helpers.NextPendingNodeInStep(session, stepIndex+1, afterNode); nextNodeNumber > 0 {
		nextNode, _ := session.NodeByNumber(nextNodeNumber)
		return coop.StartWorkCommand(session.ID, nextNodeNumber, "Beginning: "+nextNode.TitleText())
	}
	return coop.StatusCommand(session.ID)
}

func alreadyMovedResponse(session *coop.Session, nodeNumber int, state coop.NodeState) coop.CommandResponse {
	msg := fmt.Sprintf("Task %d is already %s.", nodeNumber, state)
	if session.IsComplete() {
		msg = fmt.Sprintf("Task %d confirmed. All tasks done. Run next-action now.", nodeNumber)
	}
	return nodeResponse(session.ID, nodeNumber, string(state), msg, nextAfterNode(session, nodeNumber))
}

func confirmedResponse(session *coop.Session, nodeNumber int) coop.CommandResponse {
	return nodeResponse(
		session.ID, nodeNumber, "confirmed",
		fmt.Sprintf("Task %d confirmed by developer. Proceed to the next task.", nodeNumber),
		nextAfterNode(session, nodeNumber),
	)
}

// waitingResponse says the developer simply has not finished reviewing yet. It
// avoids failure vocabulary — a response that reads as an error makes models
// abandon the loop or skip ahead — and repeats the invoked command verbatim so
// re-running takes no judgment.
func waitingResponse(sessionID string, nodeNumber int, timeout, waited time.Duration, reviewPrompt string) coop.CommandResponse {
	message := fmt.Sprintf(
		"The developer is still reviewing (%s so far). This is expected and nothing is wrong.\n"+
			"Do not start the next task, do not report new work, and do not ask a question here.\n"+
			"Run the command in \"next\" again now to keep waiting.", formatWait(waited))
	if reviewPrompt != "" {
		message += fmt.Sprintf("\nThe developer is checking: %s", reviewPrompt)
	}
	response := nodeResponse(
		sessionID, nodeNumber, "waiting", message,
		waitFor(coop.AwaitReviewCommand(sessionID, nodeNumber), timeout),
	)
	response.AdvanceAllowed = advanceAllowed(false)
	response.WaitedSeconds = int(waited.Round(time.Second).Seconds())
	response.ReviewPrompt = reviewPrompt
	return response
}

func formatWait(d time.Duration) string {
	d = d.Round(time.Second)
	if d < time.Minute {
		return fmt.Sprintf("%ds", int(d.Seconds()))
	}
	return fmt.Sprintf("%dm%ds", int(d.Minutes()), int((d % time.Minute).Seconds()))
}

func sessionErrorResponse(err error) coop.CommandResponse {
	return errorResponse(
		err,
		"Inspect the current Co-op session before retrying.",
		coop.Continue(coop.StatusCommand("")),
	)
}

func errorResponse(err error, hint string, continuation coop.Continuation) coop.CommandResponse {
	return coop.CommandResponse{
		OK:       false,
		Error:    err.Error(),
		Recovery: continuation.Recovery(hint),
	}
}

func mergeNodeOutputs(node *coop.SessionNode, reported coop.NodeOutputs) error {
	if len(reported) == 0 {
		return nil
	}
	if node.Outputs == nil {
		node.Outputs = coop.NodeOutputs{}
	}
	for source, values := range reported {
		if strings.TrimSpace(source) == "" {
			return fmt.Errorf("output source cannot be empty")
		}
		if node.Outputs[source] == nil {
			node.Outputs[source] = map[string]json.RawMessage{}
		}
		for field, value := range values {
			if strings.TrimSpace(field) == "" {
				return fmt.Errorf("output field cannot be empty")
			}
			if !json.Valid(value) {
				return fmt.Errorf("output %q is not valid JSON", field)
			}
			node.Outputs[source][field] = append(json.RawMessage(nil), value...)
		}
	}
	return nil
}

func formatNodeNumbers(nodes []int) string {
	values := make([]string, len(nodes))
	for i, node := range nodes {
		values[i] = strconv.Itoa(node)
	}
	return strings.Join(values, ", ")
}
