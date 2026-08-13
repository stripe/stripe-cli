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
	CurrentSessionSchemaVersion = 4
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
//
// Check is a short label — "Webhook signature verified", not a transcript.
// Detail carries the long output the agent wants to keep: command logs, the
// full reasoning, whatever explains the result. The TUI shows Check and reveals
// Detail on request, so a check that pastes its whole transcript into Check
// drowns the card. Sessions written before Detail existed have the transcript
// in Check; SummaryText splits one back out for them.
type Verification struct {
	Check  string `json:"check"`
	Detail string `json:"detail,omitempty"`
	Passed bool   `json:"passed"`
}

// DetailText returns the long-form output behind a check. Sessions written
// before Detail existed put everything in Check, so it falls back to that.
func (v Verification) DetailText() string {
	if v.Detail != "" {
		return v.Detail
	}
	return v.Check
}

// HasDetail reports whether there is more to read than the label.
func (v Verification) HasDetail() bool {
	return v.Detail != "" || len(v.Check) > verificationLabelBudget
}

// verificationLabelBudget is the length past which a Check is treated as a
// pasted transcript rather than as the one-line label the flag asks for.
const verificationLabelBudget = 120

type APIProcessingDetails struct {
	OutputField      string `json:"output_field,omitempty"`
	OutputFieldLabel string `json:"output_field_label,omitempty"`
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

// SessionNode combines an effective blueprint node with co-op progress.
type SessionNode struct {
	BlueprintNode
	ReviewPrompt   string          `json:"review_prompt,omitempty"`
	ReviewCommand  string          `json:"review_command,omitempty"`
	State          NodeState       `json:"state"`
	Activity       string          `json:"activity,omitempty"`
	Implementation *Implementation `json:"implementation,omitempty"`
	Outputs        NodeOutputs     `json:"outputs,omitempty"`
	Verifications  []Verification  `json:"verifications,omitempty"`
	RejectionNote  string          `json:"rejection_note,omitempty"`
	StartedAt      *time.Time      `json:"started_at,omitempty"`
	CompletedAt    *time.Time      `json:"completed_at,omitempty"`
}

// UnmarshalJSON accepts the legacy session shape so in-progress sessions remain
// usable after the session model adopts the blueprint API fields.
func (n *SessionNode) UnmarshalJSON(data []byte) error {
	type sessionNode SessionNode
	var decoded sessionNode
	if err := json.Unmarshal(data, &decoded); err != nil {
		return err
	}

	var legacy struct {
		Type        NodeType                     `json:"type"`
		AutoConfirm bool                         `json:"auto_confirm"`
		Request     *BlueprintRequestFixture     `json:"request"`
		Requests    []BlueprintRequestFixture    `json:"requests"`
		Events      []AsyncEvent                 `json:"events"`
		UIComponent *BlueprintUIComponentDetails `json:"ui_component"`
	}
	if err := json.Unmarshal(data, &legacy); err != nil {
		return err
	}
	if decoded.NodeType == "" {
		decoded.NodeType = legacy.Type
	}
	decoded.IsInformationalNode = decoded.IsInformationalNode || legacy.AutoConfirm
	if decoded.APIRequestDetails == nil && legacy.Request != nil {
		decoded.APIRequestDetails = &BlueprintAPIRequestDetails{Fixture: *legacy.Request}
	}
	if decoded.TestHelperDetails == nil && legacy.Requests != nil {
		decoded.TestHelperDetails = &BlueprintTestHelperDetails{Requests: legacy.Requests}
	}
	if decoded.AsyncHandlerDetails == nil && legacy.Events != nil {
		decoded.AsyncHandlerDetails = &BlueprintAsyncHandlerDetails{Events: legacy.Events}
	}
	if decoded.UIComponentDetails == nil {
		decoded.UIComponentDetails = legacy.UIComponent
	}
	*n = SessionNode(decoded)
	return nil
}

func (n *SessionNode) TitleText() string {
	return n.Title.DefaultMessage
}

func (n *SessionNode) DescriptionText() string {
	return n.Description.DefaultMessage
}

func (n *SessionNode) Request() *BlueprintRequestFixture {
	if n.APIRequestDetails == nil {
		return nil
	}
	return &n.APIRequestDetails.Fixture
}

func (n *SessionNode) Events() []AsyncEvent {
	if n.AsyncHandlerDetails == nil {
		return nil
	}
	return n.AsyncHandlerDetails.Events
}

func (n *SessionNode) TestRequests() []BlueprintRequestFixture {
	if n.TestHelperDetails == nil {
		return nil
	}
	return n.TestHelperDetails.Requests
}

// SessionStep combines an effective blueprint step with co-op progress.
type SessionStep struct {
	BlueprintStepDefinition
	Nodes []SessionNode `json:"nodes"`
}

func (s *SessionStep) TitleText() string {
	return s.Title.DefaultMessage
}

// DescriptionText returns the step's human-readable purpose, empty when the
// blueprint did not supply one or the session predates the field.
func (s *SessionStep) DescriptionText() string {
	return s.Description.DefaultMessage
}

// Session is the shared state file between agent and TUI.
type Session struct {
	SchemaVersion       int                  `json:"schema_version"`
	ID                  string               `json:"id"`
	Blueprint           string               `json:"blueprint"`
	BlueprintDefinition *BlueprintDefinition `json:"blueprint_definition,omitempty"`
	BlueprintPin        *BlueprintPin        `json:"blueprint_pin,omitempty"`
	Status              SessionStatus        `json:"status"`
	Settings            map[string]string    `json:"settings,omitempty"`
	Params              map[string]string    `json:"params,omitempty"`
	Steps               []SessionStep        `json:"steps"`
	UsedSandbox         bool                 `json:"used_sandbox,omitempty"`
	NextSteps           *NextStepsState      `json:"next_steps,omitempty"`
	// Cwd is the directory `coop start`/`coop run` was invoked from. Sessions
	// live in one flat global folder, so this is what distinguishes "the
	// session for the project I am looking at" from every other one.
	Cwd             string    `json:"cwd,omitempty"`
	ParentSessionID string    `json:"parent_session_id,omitempty"`
	ParentStepID    string    `json:"parent_step_id,omitempty"` // which next-step this session fulfills
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
	Version         int       `json:"version"`
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
	RequiredOutputs []RequiredOutput          `json:"required_outputs,omitempty"`
	AgentPrompt     string                    `json:"agent_prompt,omitempty"`
	APIRequest      *BlueprintRequestFixture  `json:"api_request,omitempty"`
	TestRequests    []BlueprintRequestFixture `json:"test_requests,omitempty"`
	Events          []AsyncEvent              `json:"events,omitempty"`
	SDKExample      string                    `json:"sdk_example,omitempty"`
	Error           string                    `json:"error,omitempty"`
	Recovery        *Recovery                 `json:"recovery,omitempty"`
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
