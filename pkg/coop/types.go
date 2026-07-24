// Package coop implements the co-op mode feature for collaborative
// AI agent + human developer Stripe integration building.
package coop

import (
	"fmt"
	"regexp"
	"strings"
	"time"
)

var commandTemplatePlaceholder = regexp.MustCompile(`<[^>]+>`)

// NodeState represents the lifecycle state of a single blueprint node.
type NodeState string

const (
	CurrentSessionSchemaVersion = 2
)

const (
	NodePending NodeState = "pending"
	NodeActive  NodeState = "active"
	NodeReview  NodeState = "review"
	NodeDone    NodeState = "done"
	NodeSkipped NodeState = "skipped"
)

// NodeType represents the type of a blueprint node.
type NodeType string

const (
	NodeAPIRequest    NodeType = "apiRequest"
	NodeAsyncHandler  NodeType = "asyncHandler"
	NodeUIComponent   NodeType = "uiComponent"
	NodeTestHelper    NodeType = "testHelper"
	NodeCLICommand    NodeType = "cliCommand"
	NodeDashboard     NodeType = "dashboard"
	NodeSetUpWebhooks NodeType = "setUpWebhooks"
)

// SessionStatus represents the overall session lifecycle.
type SessionStatus string

const (
	SessionActive    SessionStatus = "active"
	SessionCompleted SessionStatus = "completed"
	SessionAborted   SessionStatus = "aborted"
)

// Implementation captures what the agent did for a node.
type Implementation struct {
	File    string `json:"file,omitempty"`
	Lines   string `json:"lines,omitempty"`
	Snippet string `json:"snippet,omitempty"`
	Note    string `json:"note,omitempty"`
}

// Verification is a single check the agent ran.
type Verification struct {
	Check  string `json:"check"`
	Passed bool   `json:"passed"`
}

// APIRequest describes the expected API call for a node.
type APIRequest struct {
	Path         string            `json:"path"`
	Method       string            `json:"method"`
	Headers      map[string]string `json:"headers,omitempty"`
	Params       interface{}       `json:"params,omitempty"`
	HiddenParams interface{}       `json:"hidden_params,omitempty"`
}

// TestHelperRequest describes an API-backed request used to advance test state.
type TestHelperRequest struct {
	Key string `json:"key"`
	APIRequest
}

// NodeDefinition is the source-derived static definition for a node.
type NodeDefinition struct {
	Type          NodeType            `json:"type"`
	Key           string              `json:"key"`
	Title         string              `json:"title"`
	Description   string              `json:"description,omitempty"`
	ReviewPrompt  string              `json:"review_prompt,omitempty"`
	ReviewCommand string              `json:"review_command,omitempty"`
	AutoConfirm   bool                `json:"auto_confirm,omitempty"`
	Request       *APIRequest         `json:"request,omitempty"`
	TestRequests  []TestHelperRequest `json:"requests,omitempty"`
	Events        []string            `json:"events,omitempty"`
}

// StepDefinition is the source-derived static definition for a step.
type StepDefinition struct {
	Key   string `json:"key"`
	Title string `json:"title"`
}

// SessionNode is a single action within a session step.
type SessionNode struct {
	NodeDefinition
	State          NodeState       `json:"state"`
	Activity       string          `json:"activity,omitempty"`
	Implementation *Implementation `json:"implementation,omitempty"`
	Verifications  []Verification  `json:"verifications,omitempty"`
	RejectionNote  string          `json:"rejection_note,omitempty"`
	StartedAt      *time.Time      `json:"started_at,omitempty"`
	CompletedAt    *time.Time      `json:"completed_at,omitempty"`
}

// SessionStep groups nodes under a titled step.
type SessionStep struct {
	StepDefinition
	Nodes []SessionNode `json:"nodes"`
}

// Session is the shared state file between agent and TUI.
type Session struct {
	SchemaVersion   int               `json:"schema_version"`
	ID              string            `json:"id"`
	Blueprint       string            `json:"blueprint"`
	Status          SessionStatus     `json:"status"`
	Settings        map[string]string `json:"settings,omitempty"`
	Params          map[string]string `json:"params,omitempty"`
	Steps           []SessionStep     `json:"steps"`
	UsedSandbox     bool              `json:"used_sandbox,omitempty"`
	NextSteps       *NextStepsState   `json:"next_steps,omitempty"`
	ParentSessionID string            `json:"parent_session_id,omitempty"`
	ParentStepID    string            `json:"parent_step_id,omitempty"` // which next-step this session fulfills
	CreatedAt       time.Time         `json:"created_at"`
	UpdatedAt       time.Time         `json:"updated_at"`
	Version         int               `json:"version"`
}

// NextStepsState tracks post-completion suggestions and selection.
type NextStepsState struct {
	Suggestions []NextStepSuggestion `json:"suggestions"`
	Selected    string               `json:"selected,omitempty"`
	Completed   []string             `json:"completed,omitempty"`
}

// NextStepSuggestion is a post-completion recommendation.
type NextStepSuggestion struct {
	ID          string `json:"id"`
	Title       string `json:"title"`
	Description string `json:"description"`
	Reason      string `json:"reason,omitempty"`
}

// CommandInput describes a value an agent must provide before executing a
// command template.
type CommandInput struct {
	Name        string `json:"name"`
	Flag        string `json:"flag,omitempty"`
	Description string `json:"description"`
}

// Continuation tells an agent either what to run next or what inputs are
// needed to complete a command template.
type Continuation struct {
	Next               string         `json:"next,omitempty"`
	NextTemplate       string         `json:"next_template,omitempty"`
	RequiredInputs     []CommandInput `json:"required_inputs,omitempty"`
	WaitTimeoutSeconds int            `json:"wait_timeout_seconds,omitempty"`
}

// Continue returns a continuation for an immediately executable command.
func Continue(next string) Continuation {
	return Continuation{Next: next}
}

// WithWaitTimeout returns a continuation with its advertised shell timeout.
func (c Continuation) WithWaitTimeout(seconds int) Continuation {
	c.WaitTimeoutSeconds = seconds
	return c
}

// Recovery turns a continuation into the common agent failure contract.
func (c Continuation) Recovery(hint string) *Recovery {
	return &Recovery{Hint: hint, Continuation: c}
}

// Recovery is the single recovery contract for all agent-facing failures.
type Recovery struct {
	Hint string `json:"hint"`
	Continuation
}

// CommandResponse is the JSON output format for agent-facing commands.
type CommandResponse struct {
	OK        bool   `json:"ok"`
	SessionID string `json:"session_id,omitempty"`
	Node      int    `json:"node,omitempty"`
	State     string `json:"state,omitempty"`
	Message   string `json:"message,omitempty"`
	Continuation
	AgentPrompt string      `json:"agent_prompt,omitempty"`
	APIRequest  *APIRequest `json:"api_request,omitempty"`
	SDKExample  string      `json:"sdk_example,omitempty"`
	Error       string      `json:"error,omitempty"`
	Recovery    *Recovery   `json:"recovery,omitempty"`
}

// Validate checks the invariants agents rely on when interpreting a response.
func (r CommandResponse) Validate() error {
	if r.OK {
		if r.Error != "" || r.Recovery != nil {
			return fmt.Errorf("successful response cannot contain error recovery")
		}
		return r.validate(true)
	}
	if strings.TrimSpace(r.Error) == "" {
		return fmt.Errorf("failed response must contain error")
	}
	if r.Next != "" || r.NextTemplate != "" || len(r.RequiredInputs) > 0 || r.WaitTimeoutSeconds != 0 {
		return fmt.Errorf("failed response must put continuation data inside recovery")
	}
	if r.Recovery == nil {
		return fmt.Errorf("failed response must contain recovery")
	}
	if strings.TrimSpace(r.Recovery.Hint) == "" {
		return fmt.Errorf("recovery must contain hint")
	}
	return r.Recovery.validate(false)
}

func (c Continuation) validate(allowEmpty bool) error {
	if c.WaitTimeoutSeconds < 0 {
		return fmt.Errorf("wait_timeout_seconds cannot be negative")
	}
	if c.Next != "" && c.NextTemplate != "" {
		return fmt.Errorf("response cannot contain both next and next_template")
	}
	if c.Next == "" && c.NextTemplate == "" {
		if c.WaitTimeoutSeconds != 0 {
			return fmt.Errorf("wait_timeout_seconds requires a continuation")
		}
		if allowEmpty {
			return nil
		}
		return fmt.Errorf("recovery must contain next or next_template")
	}
	if c.Next != "" {
		if len(c.RequiredInputs) > 0 {
			return fmt.Errorf("exact next command cannot require inputs")
		}
		if commandTemplatePlaceholder.MatchString(c.Next) {
			return fmt.Errorf("exact next command contains a template placeholder")
		}
		return nil
	}
	if len(c.RequiredInputs) == 0 {
		return fmt.Errorf("next_template must describe required_inputs")
	}
	if !commandTemplatePlaceholder.MatchString(c.NextTemplate) {
		return fmt.Errorf("next_template must contain a placeholder")
	}
	for _, input := range c.RequiredInputs {
		if strings.TrimSpace(input.Name) == "" || strings.TrimSpace(input.Description) == "" {
			return fmt.Errorf("required_inputs must contain name and description")
		}
	}
	return nil
}
