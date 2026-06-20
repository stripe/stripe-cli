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
)

// BlueprintStep is a step definition within a blueprint.
type BlueprintStep struct {
	StepDefinition
	Description string              `json:"description,omitempty"`
	Required    bool                `json:"required,omitempty"`
	Settings    []BlueprintSetting  `json:"settings,omitempty"`
	Params      []BlueprintParam    `json:"params,omitempty"`
	Nodes       []NodeDefinition    `json:"nodes"`
	Semantics   *BlueprintSemantics `json:"semantics,omitempty"`
	AppRoles    []AppRole           `json:"app_roles,omitempty"`
}

// Blueprint is the CLI-friendly representation of a Workbench Blueprint.
type Blueprint struct {
	ID          string              `json:"id"`
	Title       string              `json:"title"`
	Description string              `json:"description,omitempty"`
	Prompt      string              `json:"prompt,omitempty"`
	Type        string              `json:"type"`
	Products    []string            `json:"products,omitempty"`
	Settings    []BlueprintSetting  `json:"settings"`
	Params      []BlueprintParam    `json:"params,omitempty"`
	Steps       []BlueprintStep     `json:"steps"`
	Semantics   *BlueprintSemantics `json:"semantics,omitempty"`
	AppRoles    []AppRole           `json:"app_roles,omitempty"`
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

	var bp Blueprint
	if err := json.Unmarshal(data, &bp); err != nil {
		return nil, fmt.Errorf("parsing blueprint %q: %w", id, err)
	}
	if err := validateBlueprintReferences(&bp); err != nil {
		return nil, fmt.Errorf("validating blueprint %q: %w", id, err)
	}

	return &bp, nil
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
					Description: "Scan the codebase to identify language, framework, dependencies, existing Stripe code, auth/current-user flow, existing webhook routes, and the app-owned records relevant to this blueprint. Report, only where applicable, the source of truth for money values, customer or user identity, Stripe ID persistence, connected-account mapping, and payment, subscription, or entitlement state before changing code.",
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
			nodeDefinition := n
			nodeDefinition.Semantics = MergeBlueprintSemantics(bp.Semantics, ch.Semantics, n.Semantics)
			nodeDefinition.AppRoles = MergeAppRoles(bp.AppRoles, ch.AppRoles, n.AppRoles)
			nodes[j] = SessionNode{
				NodeDefinition: nodeDefinition,
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

// MergeBlueprintSemantics overlays increasingly specific semantic contracts.
// Blueprint-level values can provide defaults, and chapter/node values can
// refine them for a concrete step.
func MergeBlueprintSemantics(parts ...*BlueprintSemantics) *BlueprintSemantics {
	var merged BlueprintSemantics
	hasValues := false
	for _, part := range parts {
		if part == nil {
			continue
		}
		hasValues = true
		if part.SourceOfTruth != nil {
			merged.SourceOfTruth = mergeSourceOfTruthSemantics(merged.SourceOfTruth, part.SourceOfTruth)
		}
		if part.PaymentLifecycle != nil {
			merged.PaymentLifecycle = mergePaymentLifecycleSemantics(merged.PaymentLifecycle, part.PaymentLifecycle)
		}
		if part.Connect != nil {
			merged.Connect = mergeConnectSemantics(merged.Connect, part.Connect)
		}
		if part.ServerVerification != nil {
			merged.ServerVerification = mergeServerVerificationSemantics(merged.ServerVerification, part.ServerVerification)
		}
		if len(part.EventRoles) > 0 {
			merged.EventRoles = append(merged.EventRoles, part.EventRoles...)
		}
		if len(part.Assertions) > 0 {
			merged.Assertions = append(merged.Assertions, part.Assertions...)
		}
	}
	if !hasValues {
		return nil
	}
	return &merged
}

// MergeAppRoles overlays app role contracts by ID. Later scopes refine earlier
// defaults, while new IDs are appended in declaration order.
func MergeAppRoles(parts ...[]AppRole) []AppRole {
	var merged []AppRole
	indexByID := map[string]int{}
	for _, roles := range parts {
		for _, role := range roles {
			if strings.TrimSpace(role.ID) == "" {
				continue
			}
			if index, ok := indexByID[role.ID]; ok {
				merged[index] = mergeAppRole(merged[index], role)
				continue
			}
			copied := copyAppRole(role)
			indexByID[copied.ID] = len(merged)
			merged = append(merged, copied)
		}
	}
	return merged
}

func mergeAppRole(base, overlay AppRole) AppRole {
	merged := copyAppRole(base)
	if overlay.Kind != "" {
		merged.Kind = overlay.Kind
	}
	if overlay.Required {
		merged.Required = true
	}
	if overlay.Description != "" {
		merged.Description = overlay.Description
	}
	if len(overlay.Examples) > 0 {
		merged.Examples = append([]string(nil), overlay.Examples...)
	}
	if len(overlay.ConsumedBy) > 0 {
		merged.ConsumedBy = append([]string(nil), overlay.ConsumedBy...)
	}
	if len(overlay.Evidence) > 0 {
		merged.Evidence = append([]string(nil), overlay.Evidence...)
	}
	if overlay.MissingBehavior != "" {
		merged.MissingBehavior = overlay.MissingBehavior
	}
	return merged
}

func copyAppRole(role AppRole) AppRole {
	copied := role
	copied.Examples = append([]string(nil), role.Examples...)
	copied.ConsumedBy = append([]string(nil), role.ConsumedBy...)
	copied.Evidence = append([]string(nil), role.Evidence...)
	return copied
}

func mergeSourceOfTruthSemantics(base, overlay *SourceOfTruthSemantics) *SourceOfTruthSemantics {
	if base == nil {
		copied := *overlay
		return &copied
	}
	merged := *base
	if overlay.Amount != "" {
		merged.Amount = overlay.Amount
	}
	if overlay.LineItems != "" {
		merged.LineItems = overlay.LineItems
	}
	if overlay.Catalog != "" {
		merged.Catalog = overlay.Catalog
	}
	if overlay.Customer != "" {
		merged.Customer = overlay.Customer
	}
	if overlay.ConnectedAccount != "" {
		merged.ConnectedAccount = overlay.ConnectedAccount
	}
	return &merged
}

func mergePaymentLifecycleSemantics(base, overlay *PaymentLifecycleSemantics) *PaymentLifecycleSemantics {
	if base == nil {
		copied := *overlay
		copied.FailureEvents = append([]string(nil), overlay.FailureEvents...)
		return &copied
	}
	merged := *base
	merged.FailureEvents = append([]string(nil), base.FailureEvents...)
	if overlay.StartsPayment {
		merged.StartsPayment = true
	}
	if overlay.CompletionEvent != "" {
		merged.CompletionEvent = overlay.CompletionEvent
	}
	if len(overlay.FailureEvents) > 0 {
		merged.FailureEvents = append([]string(nil), overlay.FailureEvents...)
	}
	if overlay.PendingState != "" {
		merged.PendingState = overlay.PendingState
	}
	if overlay.CompletedState != "" {
		merged.CompletedState = overlay.CompletedState
	}
	if overlay.FulfillmentRequiresSignedWebhook {
		merged.FulfillmentRequiresSignedWebhook = true
	}
	return &merged
}

func mergeConnectSemantics(base, overlay *ConnectSemantics) *ConnectSemantics {
	if base == nil {
		copied := *overlay
		return &copied
	}
	merged := *base
	if overlay.RequiresConnectedAccount {
		merged.RequiresConnectedAccount = true
	}
	if overlay.ConnectedAccountOwner != "" {
		merged.ConnectedAccountOwner = overlay.ConnectedAccountOwner
	}
	if overlay.OnboardingRequired {
		merged.OnboardingRequired = true
	}
	if overlay.AccountLinkRequired {
		merged.AccountLinkRequired = true
	}
	if overlay.CapabilityGate != "" {
		merged.CapabilityGate = overlay.CapabilityGate
	}
	if overlay.Source != "" {
		merged.Source = overlay.Source
	}
	return &merged
}

func mergeServerVerificationSemantics(base, overlay *ServerVerificationSemantics) *ServerVerificationSemantics {
	if base == nil {
		copied := *overlay
		return &copied
	}
	merged := *base
	if overlay.Required {
		merged.Required = true
	}
	if overlay.StateSource != "" {
		merged.StateSource = overlay.StateSource
	}
	if overlay.Reason != "" {
		merged.Reason = overlay.Reason
	}
	return &merged
}
