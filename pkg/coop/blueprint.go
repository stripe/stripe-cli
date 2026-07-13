package coop

import (
	"bytes"
	"embed"
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"time"
)

//go:embed blueprints/*.json
var blueprintFS embed.FS

const (
	blueprintNodeCandidatePrefix = "${node"
	blueprintNodeRefPrefix       = "${node."
	blueprintNodeAPIRequests     = NodeType("apiRequests")
)

// BlueprintStep is a step definition within a blueprint.
type BlueprintStep struct {
	StepDefinition
	Description string             `json:"description,omitempty"`
	Required    bool               `json:"required,omitempty"`
	Settings    []BlueprintSetting `json:"settings,omitempty"`
	Params      []BlueprintParam   `json:"params,omitempty"`
	Nodes       []NodeDefinition   `json:"nodes"`
}

// Blueprint is the CLI-friendly representation of a Workbench Blueprint.
type Blueprint struct {
	ID          string             `json:"id"`
	Title       string             `json:"title"`
	Description string             `json:"description,omitempty"`
	Type        string             `json:"type"`
	Products    []string           `json:"products,omitempty"`
	Settings    []BlueprintSetting `json:"settings"`
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
// If no exact match is found, it tries prefix matching (e.g. "one-time" -> "one-time-payment").
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

	bp, err := decodeBlueprint(data)
	if err != nil {
		return nil, fmt.Errorf("parsing blueprint %q: %w", id, err)
	}
	if err := prepareBlueprint(bp); err != nil {
		return nil, fmt.Errorf("validating blueprint %q: %w", id, err)
	}

	return bp, nil
}

// ParseBlueprint parses and validates a blueprint JSON document.
func ParseBlueprint(data []byte) (*Blueprint, error) {
	bp, err := decodeBlueprint(data)
	if err != nil {
		return nil, fmt.Errorf("parsing blueprint: %w", err)
	}
	if err := prepareBlueprint(bp); err != nil {
		return nil, fmt.Errorf("validating blueprint: %w", err)
	}
	return bp, nil
}

func decodeBlueprint(data []byte) (*Blueprint, error) {
	var bp Blueprint
	if err := json.Unmarshal(data, &bp); err != nil {
		return nil, err
	}
	return &bp, nil
}

func prepareBlueprint(bp *Blueprint) error {
	if err := normalizeBlueprint(bp); err != nil {
		return err
	}
	return validateBlueprint(bp)
}

func normalizeBlueprint(bp *Blueprint) error {
	requestRefs := make(map[string]string)
	for stepIndex := range bp.Steps {
		step := &bp.Steps[stepIndex]
		for nodeIndex := range step.Nodes {
			node := &step.Nodes[nodeIndex]
			if node.Type != blueprintNodeAPIRequests {
				continue
			}
			if node.Request != nil || len(node.TestRequests) != 1 {
				return fmt.Errorf("step %q node %q: apiRequests nodes require exactly one request", step.Key, node.Key)
			}

			requestDefinition := node.TestRequests[0]
			request := requestDefinition.APIRequest
			node.Type = NodeAPIRequest
			node.Request = &request
			node.TestRequests = nil
			if step.Key != "" && node.Key != "" && requestDefinition.Key != "" {
				requestRefs[step.Key+"."+node.Key+"."+requestDefinition.Key] = step.Key + "." + node.Key
			}
		}
	}
	return rewriteBlueprintReferences(bp, requestRefs)
}

func rewriteBlueprintReferences(bp *Blueprint, refs map[string]string) error {
	if len(refs) == 0 {
		return nil
	}

	data, err := json.Marshal(bp)
	if err != nil {
		return fmt.Errorf("marshaling blueprint for reference normalization: %w", err)
	}
	var document interface{}
	if err := json.Unmarshal(data, &document); err != nil {
		return fmt.Errorf("decoding blueprint for reference normalization: %w", err)
	}
	document, err = rewriteBlueprintReferenceValues(document, refs)
	if err != nil {
		return err
	}
	data, err = json.Marshal(document)
	if err != nil {
		return fmt.Errorf("marshaling normalized blueprint: %w", err)
	}
	var normalized Blueprint
	if err := json.Unmarshal(data, &normalized); err != nil {
		return fmt.Errorf("decoding normalized blueprint: %w", err)
	}
	*bp = normalized
	return nil
}

func rewriteBlueprintReferenceValues(value interface{}, refs map[string]string) (interface{}, error) {
	switch typed := value.(type) {
	case map[string]interface{}:
		rewritten := make(map[string]interface{}, len(typed))
		for key, child := range typed {
			normalizedKey := rewriteBlueprintReferenceString(key, refs)
			if _, exists := rewritten[normalizedKey]; exists {
				return nil, fmt.Errorf("reference normalization creates duplicate object key %q", normalizedKey)
			}
			normalizedChild, err := rewriteBlueprintReferenceValues(child, refs)
			if err != nil {
				return nil, err
			}
			rewritten[normalizedKey] = normalizedChild
		}
		return rewritten, nil
	case []interface{}:
		for i, child := range typed {
			normalizedChild, err := rewriteBlueprintReferenceValues(child, refs)
			if err != nil {
				return nil, err
			}
			typed[i] = normalizedChild
		}
		return typed, nil
	case string:
		return rewriteBlueprintReferenceString(typed, refs), nil
	default:
		return value, nil
	}
}

func rewriteBlueprintReferenceString(value string, refs map[string]string) string {
	for requestRef, nodeRef := range refs {
		value = strings.ReplaceAll(value, "${node."+requestRef+":", "${node."+nodeRef+":")
	}
	return value
}

func validateBlueprint(bp *Blueprint) error {
	if strings.TrimSpace(bp.ID) == "" {
		return fmt.Errorf("blueprint id is required")
	}
	if strings.TrimSpace(bp.Title) == "" {
		return fmt.Errorf("blueprint title is required")
	}
	if len(bp.Steps) == 0 {
		return fmt.Errorf("blueprint steps are required")
	}

	stepKeys := make(map[string]bool, len(bp.Steps))
	for stepIndex, step := range bp.Steps {
		if strings.TrimSpace(step.Key) == "" {
			return fmt.Errorf("blueprint step %d key is required", stepIndex)
		}
		if stepKeys[step.Key] {
			return fmt.Errorf("duplicate blueprint step key %q", step.Key)
		}
		stepKeys[step.Key] = true
		if strings.TrimSpace(step.Title) == "" {
			return fmt.Errorf("blueprint step %q title is required", step.Key)
		}
		if len(step.Nodes) == 0 {
			return fmt.Errorf("blueprint step %q nodes are required", step.Key)
		}

		nodeKeys := make(map[string]bool, len(step.Nodes))
		for nodeIndex, node := range step.Nodes {
			if strings.TrimSpace(node.Key) == "" {
				return fmt.Errorf("blueprint step %q node %d key is required", step.Key, nodeIndex)
			}
			if nodeKeys[node.Key] {
				return fmt.Errorf("blueprint step %q has duplicate node key %q", step.Key, node.Key)
			}
			nodeKeys[node.Key] = true
			if strings.TrimSpace(node.Title) == "" {
				return fmt.Errorf("blueprint step %q node %q title is required", step.Key, node.Key)
			}
			if !supportedBlueprintNodeType(node.Type) {
				return fmt.Errorf("blueprint step %q node %q has unsupported type %q for contract version %d", step.Key, node.Key, node.Type, CurrentBlueprintContractVersion)
			}
			if node.Type == NodeAPIRequest {
				if node.Request == nil {
					return fmt.Errorf("blueprint step %q apiRequest node %q request is required", step.Key, node.Key)
				}
				if strings.TrimSpace(node.Request.Path) == "" || strings.TrimSpace(node.Request.Method) == "" {
					return fmt.Errorf("blueprint step %q apiRequest node %q request path and method are required", step.Key, node.Key)
				}
			}
			if node.Type == NodeAsyncHandler && len(node.Events) == 0 {
				return fmt.Errorf("blueprint step %q asyncHandler node %q events are required", step.Key, node.Key)
			}
			if node.ExpectedNumberOfEvents < 0 {
				return fmt.Errorf("blueprint step %q node %q expectedNumberOfEvents must not be negative", step.Key, node.Key)
			}
			requestKeys := make(map[string]bool, len(node.TestRequests))
			for _, request := range node.TestRequests {
				if strings.TrimSpace(request.Key) == "" || strings.TrimSpace(request.Path) == "" || strings.TrimSpace(request.Method) == "" {
					return fmt.Errorf("blueprint step %q node %q requests require key, path, and method", step.Key, node.Key)
				}
				if requestKeys[request.Key] {
					return fmt.Errorf("blueprint step %q node %q has duplicate request key %q", step.Key, node.Key, request.Key)
				}
				requestKeys[request.Key] = true
			}
		}
	}
	return validateBlueprintReferences(bp)
}

func supportedBlueprintNodeType(nodeType NodeType) bool {
	switch nodeType {
	case NodeAPIRequest, NodeAsyncHandler, NodeUIComponent, NodeTestHelper, NodeCLICommand, NodeDashboard, NodeSetUpWebhooks:
		return true
	default:
		return false
	}
}

func validateBlueprintReferences(bp *Blueprint) error {
	validRefs := map[string]bool{}
	for _, step := range bp.Steps {
		for _, node := range step.Nodes {
			validRefs[step.Key+"."+node.Key] = true
			for _, request := range node.TestRequests {
				if request.Key != "" {
					validRefs[step.Key+"."+node.Key+"."+request.Key] = true
				}
			}
		}
	}

	data, err := json.Marshal(bp)
	if err != nil {
		return err
	}
	decoder := json.NewDecoder(bytes.NewReader(data))
	for {
		token, err := decoder.Token()
		if err == io.EOF {
			return nil
		}
		if err != nil {
			return err
		}
		value, ok := token.(string)
		if !ok {
			continue
		}
		if err := validateBlueprintReferenceString(value, validRefs); err != nil {
			return err
		}
	}
}

func validateBlueprintReferenceString(value string, validRefs map[string]bool) error {
	for {
		start := findBlueprintNodeCandidate(value)
		if start == -1 {
			return nil
		}
		value = value[start:]

		end := strings.IndexByte(value, '}')
		next := findBlueprintNodeCandidate(value[len(blueprintNodeCandidatePrefix):])
		if next != -1 {
			next += len(blueprintNodeCandidatePrefix)
		}
		if end == -1 || (next != -1 && next < end) {
			candidate := value
			if next != -1 {
				candidate = value[:next]
			}
			return fmt.Errorf("malformed node reference %q: missing closing brace", candidate)
		}

		placeholder := value[:end+1]
		if !strings.HasPrefix(placeholder, blueprintNodeRefPrefix) {
			return fmt.Errorf("malformed node reference %q: expected ${node.<ref>:<field>}", placeholder)
		}
		body := placeholder[len(blueprintNodeRefPrefix) : len(placeholder)-1]
		ref, field, ok := strings.Cut(body, ":")
		if !ok || ref == "" || field == "" {
			return fmt.Errorf("malformed node reference %q: expected ${node.<ref>:<field>}", placeholder)
		}

		if validRefs[ref] {
			value = value[end+1:]
			continue
		}
		parts := strings.Split(ref, ".")
		if len(parts) == 3 && validRefs[parts[0]+"."+parts[1]] && isIntegerRefSegment(parts[2]) {
			value = value[end+1:]
			continue
		}
		return fmt.Errorf("unknown node reference %q", ref)
	}
}

func findBlueprintNodeCandidate(value string) int {
	offset := 0
	for {
		start := strings.Index(value[offset:], blueprintNodeCandidatePrefix)
		if start == -1 {
			return -1
		}
		start += offset
		after := start + len(blueprintNodeCandidatePrefix)
		if after == len(value) || value[after] == '.' || value[after] == ':' || value[after] == '}' {
			return start
		}
		offset = after
	}
}

func isIntegerRefSegment(value string) bool {
	if value == "" {
		return false
	}
	for _, r := range value {
		if r < '0' || r > '9' {
			return false
		}
	}
	return true
}

func prefixMatchBlueprint(prefix string) (string, error) {
	ids, err := ListBlueprints()
	if err != nil {
		return "", fmt.Errorf("loading blueprints: %w", err)
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
			return nil, fmt.Errorf("loading blueprint metadata for %q: %w", id, err)
		}
		blueprints = append(blueprints, *bp)
	}
	return blueprints, nil
}

// NewSessionFromBlueprint creates a new Session from a Blueprint definition.
// A context-gathering step is prepended so the agent scans the project first.
func NewSessionFromBlueprint(bp *Blueprint, sessionID string, settings, params map[string]string) *Session {
	now := time.Now().UTC()

	// Prepend a context-gathering step (auto-confirmed, no human sign-off needed)
	contextStep := SessionStep{
		StepDefinition: StepDefinition{
			Key:   "context-step",
			Title: "Project context",
		},
		Nodes: []SessionNode{
			{
				NodeDefinition: NodeDefinition{
					Key:         "scan-project",
					Type:        NodeTestHelper,
					Title:       "Understand the project",
					Description: "Scan the codebase to identify language, framework, dependencies, and existing Stripe code. Report what you find.",
					AutoConfirm: true,
				},
				State: NodePending,
			},
		},
	}

	steps := make([]SessionStep, 0, len(bp.Steps)+1)
	steps = append(steps, contextStep)

	for _, ch := range bp.Steps {
		nodes := make([]SessionNode, len(ch.Nodes))
		for j, n := range ch.Nodes {
			nodes[j] = SessionNode{
				NodeDefinition: n,
				State:          NodePending,
			}
		}
		steps = append(steps, SessionStep{
			StepDefinition: ch.StepDefinition,
			Nodes:          nodes,
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
		CreatedAt:     now,
		UpdatedAt:     now,
	}
}
