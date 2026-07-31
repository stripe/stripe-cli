package coop

import (
	"fmt"
	"strconv"
	"strings"
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
		strconv.Quote(sanitizeCommandNote(note)),
	)
}

// sanitizeCommandNote keeps free text (node titles) safe to embed in an exact
// `next` command: anything matching the `<...>` placeholder syntax would make
// response validation reject the command as an unfilled template.
func sanitizeCommandNote(note string) string {
	return strings.NewReplacer("<", "", ">", "").Replace(note)
}

// AwaitReviewCommand returns the exact command for waiting on a node review.
func AwaitReviewCommand(sessionID string, nodeNumber int) string {
	return fmt.Sprintf("stripe coop agent await-review --session=%s --step=%d", sessionID, nodeNumber)
}

// ResumeCommand returns the exact command for reading the current lifecycle
// continuation without mutating the session.
func ResumeCommand(sessionID string) string {
	return fmt.Sprintf("stripe coop agent resume --session=%s", sessionID)
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
		commandInput("step", "--step", "Positive 1-based task number."),
	)
}

// NextActionTemplate returns the continuation for a missing session ID.
func NextActionTemplate() Continuation {
	return commandTemplate(
		"stripe coop agent next-action --session=\"<session>\"",
		commandInput("session", "--session", "Co-op session ID."),
	)
}

// ResumeTemplate returns the continuation for a missing session ID.
func ResumeTemplate() Continuation {
	return commandTemplate(
		"stripe coop agent resume --session=\"<session>\"",
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

// ReportWorkTemplate returns the report command template and every input that
// must be supplied for a node.
func ReportWorkTemplate(sessionID string, nodeNumber int, outputs []RequiredOutput) Continuation {
	continuation := commandTemplate(
		fmt.Sprintf("stripe coop agent report-work --session=%s --step=%d --note=\"<what you did>\"", sessionID, nodeNumber),
		commandInput("note", "--note", "Concrete summary of the completed implementation."),
	)
	for _, output := range outputs {
		selector := output.Selector()
		continuation.NextTemplate += " --output=" + strconv.Quote(selector+"=<"+selector+">")
		continuation.RequiredInputs = append(continuation.RequiredInputs, commandInput(
			selector, "--output", fmt.Sprintf("Value produced for the future blueprint reference %q.", selector),
		))
	}
	return continuation
}

// ReportWorkOutputTemplate returns the report template used when an output flag
// itself is malformed.
func ReportWorkOutputTemplate(sessionID string, nodeNumber int) Continuation {
	continuation := ReportWorkTemplate(sessionID, nodeNumber, nil)
	continuation.NextTemplate += " --output=\"<field=value>\""
	continuation.RequiredInputs = append(continuation.RequiredInputs, commandInput(
		"output", "--output", "Output selector and value in field=value or source:field=value form.",
	))
	return continuation
}

func commandTemplate(command string, inputs ...CommandInput) Continuation {
	return Continuation{NextTemplate: command, RequiredInputs: inputs}
}

func commandInput(name, flag, description string) CommandInput {
	return CommandInput{Name: name, Flag: flag, Description: description}
}
