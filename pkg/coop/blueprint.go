package coop

import (
	"embed"
	"encoding/json"
	"fmt"
	"strings"
)

//go:embed blueprints/*.json
var blueprintFS embed.FS

// BlueprintNode is a node definition within a blueprint chapter.
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

// BlueprintChapter is a chapter definition within a blueprint.
type BlueprintChapter struct {
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
	Chapters    []BlueprintChapter `json:"chapters"`
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
func NewSessionFromBlueprint(bp *Blueprint, sessionID string, settings map[string]string) *Session {
	// Prepend a context-gathering chapter (auto-confirmed, no human sign-off needed)
	contextChapter := SessionChapter{
		Key:   "context-chapter",
		Title: "Project context",
		Nodes: []SessionNode{
			{
				Key:         "scan-project",
				Type:        NodeTestHelper,
				Title:       "Understand the project",
				Description: "Scan the codebase to identify language, framework, dependencies, and existing Stripe code. Report what you find.",
				AutoConfirm: true,
				State:       StepPending,
			},
		},
	}

	chapters := make([]SessionChapter, 0, len(bp.Chapters)+1)
	chapters = append(chapters, contextChapter)

	for _, ch := range bp.Chapters {
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
				State:         StepPending,
				Request:       n.Request,
				Events:        n.Events,
			}
		}
		chapters = append(chapters, SessionChapter{
			Key:               ch.Key,
			Title:             ch.Title,
			ReviewGranularity: ch.ReviewGranularity,
			Nodes:             nodes,
		})
	}

	return &Session{
		ID:        sessionID,
		Blueprint: bp.ID,
		Status:    SessionActive,
		Settings:  settings,
		Chapters:  chapters,
	}
}
