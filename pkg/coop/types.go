// Package coop implements the co-op mode feature for collaborative
// AI agent + human developer Stripe integration building.
package coop

import (
	"encoding/json"
	"time"
)

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

// Verification is a single check the agent ran.
type Verification struct {
	Check  string `json:"check"`
	Passed bool   `json:"passed"`
}

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

// SessionNode combines an effective Workbench node with co-op progress.
type SessionNode struct {
	WorkbenchBlueprintNode
	ReviewPrompt   string          `json:"review_prompt,omitempty"`
	ReviewCommand  string          `json:"review_command,omitempty"`
	State          NodeState       `json:"state"`
	Activity       string          `json:"activity,omitempty"`
	Implementation *Implementation `json:"implementation,omitempty"`
	Verifications  []Verification  `json:"verifications,omitempty"`
	RejectionNote  string          `json:"rejection_note,omitempty"`
	StartedAt      *time.Time      `json:"started_at,omitempty"`
	CompletedAt    *time.Time      `json:"completed_at,omitempty"`
}

// UnmarshalJSON accepts the pre-Workbench session shape so in-progress sessions
// remain usable after the session model adopts Workbench node fields.
func (n *SessionNode) UnmarshalJSON(data []byte) error {
	type sessionNode SessionNode
	var decoded sessionNode
	if err := json.Unmarshal(data, &decoded); err != nil {
		return err
	}

	var legacy struct {
		Type        NodeType                     `json:"type"`
		AutoConfirm bool                         `json:"auto_confirm"`
		Request     *WorkbenchRequestFixture     `json:"request"`
		Requests    []WorkbenchRequestFixture    `json:"requests"`
		Events      []AsyncEvent                 `json:"events"`
		UIComponent *WorkbenchUIComponentDetails `json:"ui_component"`
	}
	if err := json.Unmarshal(data, &legacy); err != nil {
		return err
	}
	if decoded.NodeType == "" {
		decoded.NodeType = legacy.Type
	}
	decoded.IsInformationalNode = decoded.IsInformationalNode || legacy.AutoConfirm
	if decoded.APIRequestDetails == nil && legacy.Request != nil {
		decoded.APIRequestDetails = &WorkbenchAPIRequestDetails{Fixture: *legacy.Request}
	}
	if decoded.TestHelperDetails == nil && legacy.Requests != nil {
		decoded.TestHelperDetails = &WorkbenchTestHelperDetails{Requests: legacy.Requests}
	}
	if decoded.AsyncHandlerDetails == nil && legacy.Events != nil {
		decoded.AsyncHandlerDetails = &WorkbenchAsyncHandlerDetails{Events: legacy.Events}
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

func (n *SessionNode) Request() *WorkbenchRequestFixture {
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

// SessionStep combines an effective Workbench step with co-op progress.
type SessionStep struct {
	WorkbenchStepDefinition
	Nodes []SessionNode `json:"nodes"`
}

func (s *SessionStep) TitleText() string {
	return s.Title.DefaultMessage
}

// Session is the shared state file between agent and TUI.
type Session struct {
	SchemaVersion       int                           `json:"schema_version"`
	ID                  string                        `json:"id"`
	Blueprint           string                        `json:"blueprint"`
	BlueprintDefinition *WorkbenchBlueprintDefinition `json:"blueprint_definition,omitempty"`
	BlueprintPin        *BlueprintPin                 `json:"blueprint_pin,omitempty"`
	Status              SessionStatus                 `json:"status"`
	Settings            map[string]string             `json:"settings,omitempty"`
	Params              map[string]string             `json:"params,omitempty"`
	Steps               []SessionStep                 `json:"steps"`
	UsedSandbox         bool                          `json:"used_sandbox,omitempty"`
	NextSteps           *NextStepsState               `json:"next_steps,omitempty"`
	ParentSessionID     string                        `json:"parent_session_id,omitempty"`
	ParentStepID        string                        `json:"parent_step_id,omitempty"` // which next-step this session fulfills
	CreatedAt           time.Time                     `json:"created_at"`
	UpdatedAt           time.Time                     `json:"updated_at"`
	Version             int                           `json:"version"`
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

// CommandResponse is the JSON output format for agent-facing commands.
type CommandResponse struct {
	OK          bool                     `json:"ok"`
	SessionID   string                   `json:"session_id,omitempty"`
	Node        int                      `json:"node,omitempty"`
	State       string                   `json:"state,omitempty"`
	Message     string                   `json:"message,omitempty"`
	Next        string                   `json:"next,omitempty"`
	AgentPrompt string                   `json:"agent_prompt,omitempty"`
	APIRequest  *WorkbenchRequestFixture `json:"api_request,omitempty"`
	SDKExample  string                   `json:"sdk_example,omitempty"`
	Error       string                   `json:"error,omitempty"`
	Hint        string                   `json:"hint,omitempty"`
}
