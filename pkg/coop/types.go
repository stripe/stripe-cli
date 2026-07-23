// Package coop implements the co-op mode feature for collaborative
// AI agent + human developer Stripe integration building.
package coop

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
	"time"
)

var commandTemplatePlaceholder = regexp.MustCompile(`<[^>]+>`)

// NodeState represents the lifecycle state of a single blueprint node.
type NodeState string

const (
	CurrentSessionSchemaVersion = 3
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

// NodeOutputs stores values produced by a node. The outer key identifies the
// result source ("default" for a node's primary result, or a named/numeric
// request result); the inner key is the field path referenced by a blueprint.
type NodeOutputs map[string]map[string]json.RawMessage

// Verification is a single check the agent ran.
type Verification struct {
	Check  string `json:"check"`
	Passed bool   `json:"passed"`
}

// APIRequest describes the expected API call for a node.
type APIRequest struct {
	Key               string                `json:"key,omitempty"`
	Path              string                `json:"path"`
	Method            string                `json:"method"`
	Headers           map[string]string     `json:"headers,omitempty"`
	Params            interface{}           `json:"params,omitempty"`
	HiddenParams      interface{}           `json:"hidden_params,omitempty"`
	ExpectedErrorType string                `json:"expected_error_type,omitempty"`
	ProcessingDetails *APIProcessingDetails `json:"processing_details,omitempty"`
	RegenerateEnv     bool                  `json:"regenerate_env,omitempty"`
}

type APIProcessingDetails struct {
	OutputField      string `json:"output_field,omitempty"`
	OutputFieldLabel string `json:"output_field_label,omitempty"`
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
	Events        []AsyncEvent        `json:"events,omitempty"`
	UIComponent   *UIComponentDetails `json:"ui_component,omitempty"`
}

type AsyncEvent struct {
	ConnectedAccountID string                 `json:"connected_account_id,omitempty"`
	EventCount         int                    `json:"event_count,omitempty"`
	EventData          map[string]interface{} `json:"event_data,omitempty"`
	EventPayloadType   string                 `json:"event_payload_type,omitempty"`
	EventType          string                 `json:"event_type"`
	ObjectID           string                 `json:"object_id,omitempty"`
	OnNodeComplete     *NodeReference         `json:"on_node_complete,omitempty"`
}

type NodeReference struct {
	NodeKey string `json:"node_key"`
	StepKey string `json:"step_key"`
}

type UIComponentReference struct {
	ID      string `json:"id"`
	Version string `json:"version"`
}

type UIComponentDetails struct {
	Display             string                 `json:"display,omitempty"`
	DisplayComponentRef *UIComponentReference  `json:"display_component_ref,omitempty"`
	StripeElementRef    map[string]interface{} `json:"stripe_element_ref,omitempty"`
	Options             []UIComponentOption    `json:"options,omitempty"`
}

type UIComponentOption struct {
	Type     string       `json:"type"`
	Title    string       `json:"title"`
	Link     string       `json:"link,omitempty"`
	Requests []APIRequest `json:"requests,omitempty"`
}

type BlueprintPin struct {
	ID               string             `json:"id"`
	Key              string             `json:"key"`
	Title            string             `json:"title,omitempty"`
	BlueprintVersion int                `json:"blueprint_version"`
	TemplateVersion  int                `json:"template_version"`
	Steps            []BlueprintStepPin `json:"steps"`
	Digest           string             `json:"digest"`
}

type BlueprintStepPin struct {
	Key             string `json:"key"`
	StepVersion     int    `json:"step_version"`
	TemplateVersion int    `json:"template_version"`
}

// StepDefinition is the source-derived static definition for a step.
type StepDefinition struct {
	Key     string            `json:"key"`
	Title   string            `json:"title"`
	Outputs []BlueprintOutput `json:"outputs,omitempty"`
}

type BlueprintOutput struct {
	Name   string                 `json:"name"`
	Source string                 `json:"source"`
	Schema map[string]interface{} `json:"schema,omitempty"`
}

// SessionNode is a single action within a session step.
type SessionNode struct {
	NodeDefinition
	State          NodeState       `json:"state"`
	Activity       string          `json:"activity,omitempty"`
	Implementation *Implementation `json:"implementation,omitempty"`
	Outputs        NodeOutputs     `json:"outputs,omitempty"`
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
	BlueprintPin    *BlueprintPin     `json:"blueprint_pin,omitempty"`
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

// RequiredOutput describes a node result that a future blueprint node
// references. Source is empty for the node's primary result.
type RequiredOutput struct {
	Source string `json:"source,omitempty"`
	Field  string `json:"field"`
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
	RequiredOutputs []RequiredOutput    `json:"required_outputs,omitempty"`
	AgentPrompt     string              `json:"agent_prompt,omitempty"`
	APIRequest      *APIRequest         `json:"api_request,omitempty"`
	TestRequests    []TestHelperRequest `json:"test_requests,omitempty"`
	Events          []string            `json:"events,omitempty"`
	SDKExample      string              `json:"sdk_example,omitempty"`
	Error           string              `json:"error,omitempty"`
	Recovery        *Recovery           `json:"recovery,omitempty"`
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
