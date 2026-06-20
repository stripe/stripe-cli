// Package coop implements the co-op mode feature for collaborative
// AI agent + human developer Stripe integration building.
package coop

import "time"

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

// AppRole describes an app responsibility a blueprint needs bound to concrete
// code, data, UI, or state in the user's project.
type AppRole struct {
	ID              string   `json:"id"`
	Kind            string   `json:"kind,omitempty"`
	Required        bool     `json:"required,omitempty"`
	Description     string   `json:"description,omitempty"`
	Examples        []string `json:"examples,omitempty"`
	ConsumedBy      []string `json:"consumed_by,omitempty"`
	Evidence        []string `json:"evidence,omitempty"`
	MissingBehavior string   `json:"missing_behavior,omitempty"`
}

// BlueprintSemantics captures durable Stripe product intent that can be supplied
// by local or remote blueprints and compiled into agent guidance.
type BlueprintSemantics struct {
	SourceOfTruth      *SourceOfTruthSemantics      `json:"source_of_truth,omitempty"`
	PaymentLifecycle   *PaymentLifecycleSemantics   `json:"payment_lifecycle,omitempty"`
	Connect            *ConnectSemantics            `json:"connect,omitempty"`
	EventRoles         []EventRoleSemantics         `json:"event_roles,omitempty"`
	ServerVerification *ServerVerificationSemantics `json:"server_verification,omitempty"`
	Assertions         []string                     `json:"assertions,omitempty"`
}

// SourceOfTruthSemantics tells agents which values come from the app domain
// instead of generated demo data.
type SourceOfTruthSemantics struct {
	Amount           string `json:"amount,omitempty"`
	LineItems        string `json:"line_items,omitempty"`
	Catalog          string `json:"catalog,omitempty"`
	Customer         string `json:"customer,omitempty"`
	ConnectedAccount string `json:"connected_account,omitempty"`
}

// PaymentLifecycleSemantics describes when payment-related app state is safe to
// finalize.
type PaymentLifecycleSemantics struct {
	StartsPayment                    bool     `json:"starts_payment,omitempty"`
	CompletionEvent                  string   `json:"completion_event,omitempty"`
	FailureEvents                    []string `json:"failure_events,omitempty"`
	PendingState                     string   `json:"pending_state,omitempty"`
	CompletedState                   string   `json:"completed_state,omitempty"`
	FulfillmentRequiresSignedWebhook bool     `json:"fulfillment_requires_signed_webhook,omitempty"`
}

// ConnectSemantics describes connected account prerequisites for Connect flows.
type ConnectSemantics struct {
	RequiresConnectedAccount bool   `json:"requires_connected_account,omitempty"`
	ConnectedAccountOwner    string `json:"connected_account_owner,omitempty"`
	OnboardingRequired       bool   `json:"onboarding_required,omitempty"`
	AccountLinkRequired      bool   `json:"account_link_required,omitempty"`
	CapabilityGate           string `json:"capability_gate,omitempty"`
	Source                   string `json:"source,omitempty"`
}

// EventRoleSemantics describes why an async event matters to the app.
type EventRoleSemantics struct {
	Event          string `json:"event,omitempty"`
	Role           string `json:"role,omitempty"`
	StateUpdate    string `json:"state_update,omitempty"`
	RequiresLookup bool   `json:"requires_lookup,omitempty"`
}

// ServerVerificationSemantics describes server-side checks needed for
// user-facing completion or return pages.
type ServerVerificationSemantics struct {
	Required    bool   `json:"required,omitempty"`
	StateSource string `json:"state_source,omitempty"`
	Reason      string `json:"reason,omitempty"`
}

// StepInfo is the agent-facing blueprint contract for a single node.
type StepInfo struct {
	Number        int                 `json:"number,omitempty"`
	Key           string              `json:"key"`
	Title         string              `json:"title"`
	Type          NodeType            `json:"type"`
	Description   string              `json:"description,omitempty"`
	ReviewPrompt  string              `json:"review_prompt,omitempty"`
	ReviewCommand string              `json:"review_command,omitempty"`
	AutoConfirm   bool                `json:"auto_confirm,omitempty"`
	APIRequest    *APIRequest         `json:"api_request,omitempty"`
	TestRequests  []TestHelperRequest `json:"requests,omitempty"`
	Events        []string            `json:"events,omitempty"`
	Semantics     *BlueprintSemantics `json:"semantics,omitempty"`
	AppRoles      []AppRole           `json:"app_roles,omitempty"`
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
	Semantics     *BlueprintSemantics `json:"semantics,omitempty"`
	AppRoles      []AppRole           `json:"app_roles,omitempty"`
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

// CommandResponse is the JSON output format for agent-facing commands.
type CommandResponse struct {
	OK                         bool        `json:"ok"`
	SessionID                  string      `json:"session_id,omitempty"`
	Node                       int         `json:"node,omitempty"`
	State                      string      `json:"state,omitempty"`
	Message                    string      `json:"message,omitempty"`
	Next                       string      `json:"next,omitempty"`
	AgentPrompt                string      `json:"agent_prompt,omitempty"`
	APIRequest                 *APIRequest `json:"api_request,omitempty"`
	SDKExample                 string      `json:"sdk_example,omitempty"`
	WebhookExample             string      `json:"webhook_example,omitempty"`
	AgentGuidance              string      `json:"agent_guidance,omitempty"`
	ImplementationRequirements []string    `json:"implementation_requirements,omitempty"`
	VerificationRequirements   []string    `json:"verification_requirements,omitempty"`
	QualityWarnings            []string    `json:"quality_warnings,omitempty"`
	BlueprintStep              *StepInfo   `json:"blueprint_step,omitempty"`
	Error                      string      `json:"error,omitempty"`
	Hint                       string      `json:"hint,omitempty"`
}
