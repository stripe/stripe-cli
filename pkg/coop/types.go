// Package coop implements the co-op mode feature for collaborative
// AI agent + human developer Stripe integration building.
package coop

import "time"

// StepState represents the lifecycle state of a single step/node.
type StepState string

const (
	StepPending StepState = "pending"
	StepActive  StepState = "active"
	StepReview  StepState = "review"
	StepDone    StepState = "done"
	StepSkipped StepState = "skipped"
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

// ReviewGranularity controls where human approval happens.
type ReviewGranularity string

const (
	ReviewGranularityStep    ReviewGranularity = "step"
	ReviewGranularityChapter ReviewGranularity = "chapter"
	ReviewGranularityAuto    ReviewGranularity = "auto"
)

// SessionStatus represents the overall session lifecycle.
type SessionStatus string

const (
	SessionActive    SessionStatus = "active"
	SessionCompleted SessionStatus = "completed"
	SessionAborted   SessionStatus = "aborted"
)

// Implementation captures what the agent did for a step.
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
	Key    string      `json:"key,omitempty"`
	Path   string      `json:"path"`
	Method string      `json:"method"`
	Params interface{} `json:"params,omitempty"`
}

// SessionNode is a single step within a session chapter.
type SessionNode struct {
	Key            string          `json:"key"`
	Type           NodeType        `json:"type"`
	Title          string          `json:"title"`
	Description    string          `json:"description,omitempty"`
	ReviewPrompt   string          `json:"review_prompt,omitempty"`
	ReviewRisk     string          `json:"review_risk,omitempty"`
	AutoConfirm    bool            `json:"auto_confirm,omitempty"`
	State          StepState       `json:"state"`
	Activity       string          `json:"activity,omitempty"`
	Implementation *Implementation `json:"implementation,omitempty"`
	Verifications  []Verification  `json:"verifications,omitempty"`
	Request        *APIRequest     `json:"request,omitempty"`
	Events         []string        `json:"events,omitempty"`
	RejectionNote  string          `json:"rejection_note,omitempty"`
	StartedAt      *time.Time      `json:"started_at,omitempty"`
	CompletedAt    *time.Time      `json:"completed_at,omitempty"`
}

// SessionChapter groups nodes under a titled section.
type SessionChapter struct {
	Key               string            `json:"key"`
	Title             string            `json:"title"`
	ReviewGranularity ReviewGranularity `json:"review_granularity,omitempty"`
	Nodes             []SessionNode     `json:"nodes"`
}

// Session is the shared state file between agent and TUI.
type Session struct {
	ID              string            `json:"id"`
	Blueprint       string            `json:"blueprint"`
	Status          SessionStatus     `json:"status"`
	Settings        map[string]string `json:"settings,omitempty"`
	Chapters        []SessionChapter  `json:"chapters"`
	ClaimURL        string            `json:"claim_url,omitempty"`
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
	OK        bool   `json:"ok"`
	SessionID string `json:"session_id,omitempty"`
	Step      int    `json:"step,omitempty"`
	State     string `json:"state,omitempty"`
	Message   string `json:"message,omitempty"`
	Next      string `json:"next,omitempty"`
	Error     string `json:"error,omitempty"`
	Hint      string `json:"hint,omitempty"`
}
