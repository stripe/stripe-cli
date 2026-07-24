package coop

import (
	"fmt"
	"strconv"
)

// StatusCommand returns the exact command for inspecting Co-op state.
func StatusCommand(sessionID string) string {
	if sessionID == "" {
		return "stripe coop status"
	}
	return fmt.Sprintf("stripe coop status --session=%s", sessionID)
}

// RunCommand returns the exact command for starting a blueprint session.
func RunCommand(blueprintID string) string {
	return fmt.Sprintf("stripe coop run %s", strconv.Quote(blueprintID))
}

// RunTemplate returns the continuation for a missing blueprint ID.
func RunTemplate() Continuation {
	return commandTemplate(
		"stripe coop run \"<blueprint>\"",
		commandInput("blueprint", "", "Blueprint ID returned by stripe coop recommend."),
	)
}

// StopCommand returns the exact command for ending Co-op state.
func StopCommand(sessionID string) string {
	if sessionID == "" {
		return "stripe coop stop"
	}
	return fmt.Sprintf("stripe coop stop --session=%s", sessionID)
}

// StartWorkCommand returns the exact command for activating a node.
func StartWorkCommand(sessionID string, nodeNumber int, note string) string {
	return fmt.Sprintf(
		"stripe coop agent start-work --session=%s --step=%d --note=%s",
		sessionID,
		nodeNumber,
		strconv.Quote(note),
	)
}

// AwaitReviewCommand returns the exact command for waiting on a node review.
func AwaitReviewCommand(sessionID string, nodeNumber int) string {
	return fmt.Sprintf("stripe coop agent await-review --session=%s --step=%d", sessionID, nodeNumber)
}

// NextActionCommand returns the exact command for waiting on or completing a
// post-session action.
func NextActionCommand(sessionID, completed string) string {
	command := fmt.Sprintf("stripe coop agent next-action --session=%s", sessionID)
	if completed != "" {
		command += " --completed=" + completed
	}
	return command
}

// StartFollowupCommand returns the exact command for starting a guided follow-up.
func StartFollowupCommand(sessionID, action, target string) string {
	command := fmt.Sprintf(
		"stripe coop agent start-followup --session=%s --action=%s",
		strconv.Quote(sessionID),
		strconv.Quote(action),
	)
	if target != "" {
		command += " --target=" + strconv.Quote(target)
	}
	return command
}

// SessionStepTemplate returns the common session/node continuation.
func SessionStepTemplate(action string) Continuation {
	return commandTemplate(
		fmt.Sprintf("stripe coop agent %s --session=\"<session>\" --step=<step>", action),
		commandInput("session", "--session", "Co-op session ID."),
		commandInput("step", "--step", "Positive 1-based node number."),
	)
}

// NextActionTemplate returns the continuation for a missing session ID.
func NextActionTemplate() Continuation {
	return commandTemplate(
		"stripe coop agent next-action --session=\"<session>\"",
		commandInput("session", "--session", "Co-op session ID."),
	)
}

// StartFollowupTemplate returns the continuation for missing follow-up inputs.
func StartFollowupTemplate(sessionID string) Continuation {
	if sessionID == "" {
		return commandTemplate(
			"stripe coop agent start-followup --session=\"<session>\" --action=\"<action>\"",
			commandInput("session", "--session", "Completed parent Co-op session ID."),
			commandInput("action", "--action", "Follow-up action offered by next-action."),
		)
	}
	return commandTemplate(
		fmt.Sprintf("stripe coop agent start-followup --session=%s --action=\"<action>\"", sessionID),
		commandInput("action", "--action", "Available follow-up action ID."),
	)
}

// ReportCheckTemplate returns the continuation for a missing check.
func ReportCheckTemplate(sessionID string, nodeNumber int) Continuation {
	return commandTemplate(
		fmt.Sprintf("stripe coop agent report-check --session=%s --step=%d --check=\"<what you verified>\" --passed", sessionID, nodeNumber),
		commandInput("check", "--check", "Concrete verification and its observed result."),
	)
}

// ReportWorkTemplate returns the continuation for reporting implementation.
func ReportWorkTemplate(sessionID string, nodeNumber int) Continuation {
	return commandTemplate(
		fmt.Sprintf("stripe coop agent report-work --session=%s --step=%d --note=\"<what you did>\"", sessionID, nodeNumber),
		commandInput("note", "--note", "Concrete summary of the completed implementation."),
	)
}

func commandTemplate(command string, inputs ...CommandInput) Continuation {
	return Continuation{NextTemplate: command, RequiredInputs: inputs}
}

func commandInput(name, flag, description string) CommandInput {
	return CommandInput{Name: name, Flag: flag, Description: description}
}
