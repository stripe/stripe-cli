package coop

import (
	"embed"
	"encoding/json"
	"fmt"
	"strings"
)

//go:embed blueprints/*.json
var blueprintFS embed.FS

// BlueprintNode is a node definition within a blueprint step.
type BlueprintNode struct {
	Type          NodeType    `json:"type"`
	Key           string      `json:"key"`
	Title         string      `json:"title"`
	Description   string      `json:"description,omitempty"`
	ReviewPrompt  string      `json:"review_prompt,omitempty"`
	ReviewCommand string      `json:"review_command,omitempty"`
	AutoConfirm   bool        `json:"auto_confirm,omitempty"`
	Request       *APIRequest `json:"request,omitempty"`
	Events        []string    `json:"events,omitempty"`
}

// BlueprintStep is a step definition within a blueprint.
type BlueprintStep struct {
	Key               string            `json:"key"`
	Title             string            `json:"title"`
	Description       string            `json:"description,omitempty"`
	ReviewGranularity ReviewGranularity `json:"review_granularity,omitempty"`
	Required          bool              `json:"required,omitempty"`
	Nodes             []BlueprintNode   `json:"nodes"`
}

// Blueprint is the CLI-friendly representation of a Workbench Blueprint.
type Blueprint struct {
	ID          string             `json:"id"`
	Title       string             `json:"title"`
	Description string             `json:"description,omitempty"`
	Prompt      string             `json:"prompt,omitempty"`
	Type        string             `json:"type"`
	Products    []string           `json:"products,omitempty"`
	Settings    []BlueprintSetting `json:"settings,omitempty"`
	Params      []BlueprintParam   `json:"params,omitempty"`
	Steps       []BlueprintStep    `json:"steps"`
}

// BlueprintSetting defines a configurable setting for a blueprint.
type BlueprintSetting struct {
	Key         string   `json:"key"`
	Title       string   `json:"title"`
	Description string   `json:"description,omitempty"`
	Type        string   `json:"type"`
	Default     string   `json:"default,omitempty"`
	Options     []string `json:"options,omitempty"`
}

// BlueprintParam defines a variable value used while implementing a blueprint.
type BlueprintParam struct {
	Key         string   `json:"key"`
	Title       string   `json:"title"`
	Description string   `json:"description,omitempty"`
	Type        string   `json:"type"`
	Default     string   `json:"default,omitempty"`
	Options     []string `json:"options,omitempty"`
}

// LoadBlueprint loads a blueprint by ID from the embedded filesystem.
// If no exact match is found, it tries prefix matching (e.g. "deploy" → "deploy-stripe-projects").
func LoadBlueprint(id string) (*Blueprint, error) {
	filename := fmt.Sprintf("blueprints/%s.json", id)
	data, err := blueprintFS.ReadFile(filename)
	if err != nil {
		// Try prefix match
		match, matchErr := prefixMatchBlueprint(id)
		if matchErr != nil {
			return nil, matchErr
		}
		data, err = blueprintFS.ReadFile(fmt.Sprintf("blueprints/%s.json", match))
		if err != nil {
			return nil, fmt.Errorf("blueprint %q not found", id)
		}
	}

	var bp Blueprint
	if err := json.Unmarshal(data, &bp); err != nil {
		return nil, fmt.Errorf("parsing blueprint %q: %w", id, err)
	}

	return &bp, nil
}

func prefixMatchBlueprint(prefix string) (string, error) {
	ids, err := ListBlueprints()
	if err != nil {
		return "", fmt.Errorf("blueprint %q not found", prefix)
	}

	var matches []string
	for _, id := range ids {
		if strings.HasPrefix(id, prefix) {
			matches = append(matches, id)
		}
	}

	switch len(matches) {
	case 0:
		return "", fmt.Errorf("blueprint %q not found", prefix)
	case 1:
		return matches[0], nil
	default:
		return "", fmt.Errorf("ambiguous blueprint prefix %q matches: %s", prefix, strings.Join(matches, ", "))
	}
}

// ListBlueprints returns all available blueprint IDs.
func ListBlueprints() ([]string, error) {
	entries, err := blueprintFS.ReadDir("blueprints")
	if err != nil {
		return nil, err
	}

	var ids []string
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".json") {
			continue
		}
		ids = append(ids, strings.TrimSuffix(e.Name(), ".json"))
	}
	return ids, nil
}

// ListBlueprintsWithMetadata returns all blueprints with their metadata.
func ListBlueprintsWithMetadata() ([]Blueprint, error) {
	ids, err := ListBlueprints()
	if err != nil {
		return nil, err
	}

	var blueprints []Blueprint
	for _, id := range ids {
		bp, err := LoadBlueprint(id)
		if err != nil {
			continue
		}
		blueprints = append(blueprints, *bp)
	}
	return blueprints, nil
}

// NewSessionFromBlueprint creates a new Session from a Blueprint definition.
// A context-gathering step is prepended so the agent scans the project first.
func NewSessionFromBlueprint(bp *Blueprint, sessionID string, settings, params map[string]string) *Session {
	// Prepend a context-gathering step (auto-confirmed, no human sign-off needed)
	contextStep := SessionStep{
		Key:   "context-step",
		Title: "Project context",
		Nodes: []SessionNode{
			{
				Key:         "scan-project",
				Type:        NodeTestHelper,
				Title:       "Understand the project",
				Description: "Scan the codebase to identify language, framework, dependencies, and existing Stripe code. Report what you find.",
				AutoConfirm: true,
				State:       NodePending,
			},
		},
	}

	steps := make([]SessionStep, 0, len(bp.Steps)+1)
	steps = append(steps, contextStep)

	for _, ch := range bp.Steps {
		nodes := make([]SessionNode, len(ch.Nodes))
		for j, n := range ch.Nodes {
			nodes[j] = SessionNode{
				Key:           n.Key,
				Type:          n.Type,
				Title:         n.Title,
				Description:   n.Description,
				ReviewPrompt:  n.ReviewPrompt,
				ReviewCommand: n.ReviewCommand,
				AutoConfirm:   n.AutoConfirm,
				State:         NodePending,
				Request:       n.Request,
				Events:        n.Events,
			}
		}
		steps = append(steps, SessionStep{
			Key:               ch.Key,
			Title:             ch.Title,
			ReviewGranularity: ch.ReviewGranularity,
			Nodes:             nodes,
		})
	}

	return &Session{
		SchemaVersion: CurrentSessionSchemaVersion,
		ID:            sessionID,
		Blueprint:     bp.ID,
		Status:        SessionActive,
		Settings:      settings,
		Params:        params,
		Steps:         steps,
	}
}
