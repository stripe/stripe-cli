package coop

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"strconv"
	"strings"
	"time"
)

const (
	blueprintNodeCandidatePrefix = "${node"
	blueprintNodeRefPrefix       = "${node."
)

// BlueprintStep is a step definition within a compiled blueprint.
type BlueprintStep struct {
	StepDefinition
	Description string           `json:"description,omitempty"`
	Required    bool             `json:"required,omitempty"`
	Nodes       []NodeDefinition `json:"nodes"`
}

// Blueprint is the internal, session-ready representation of a Workbench blueprint.
type Blueprint struct {
	ID               string            `json:"id"`
	Title            string            `json:"title"`
	Description      string            `json:"description,omitempty"`
	Type             string            `json:"type"`
	Products         []string          `json:"products,omitempty"`
	Steps            []BlueprintStep   `json:"steps"`
	Pin              BlueprintPin      `json:"-"`
	ResolvedSettings map[string]string `json:"-"`
}

// LoadBlueprint retrieves and compiles a Workbench blueprint by its exact key.
func LoadBlueprint(ctx context.Context, repository BlueprintRepository, key string, settings map[string]string) (*Blueprint, error) {
	if repository == nil {
		return nil, fmt.Errorf("loading blueprints: no blueprint repository configured")
	}
	source, err := repository.Retrieve(ctx, key)
	if err != nil {
		return nil, fmt.Errorf("retrieving blueprint %q: %w", key, err)
	}
	return CompileBlueprint(source, settings)
}

// CompileBlueprint normalizes an API blueprint into the model pinned in a session.
func CompileBlueprint(source *WorkbenchBlueprint, selectedSettings map[string]string) (*Blueprint, error) {
	if source == nil {
		return nil, fmt.Errorf("cannot compile a nil blueprint")
	}
	resolvedSettings := resolveBlueprintSettings(source, selectedSettings)
	blueprint := &Blueprint{
		ID:               source.Key,
		Title:            source.Title.DefaultMessage,
		Description:      source.Description.DefaultMessage,
		Type:             source.BlueprintType,
		Products:         append([]string(nil), source.Metadata.Products...),
		ResolvedSettings: resolvedSettings,
		Pin: BlueprintPin{
			ID:               source.ID,
			Key:              source.Key,
			Title:            source.Title.DefaultMessage,
			BlueprintVersion: source.BlueprintVersion,
			TemplateVersion:  source.TemplateVersion,
			Digest:           blueprintDigest(source),
		},
	}

	for _, sourceStep := range source.Steps {
		blueprint.Pin.Steps = append(blueprint.Pin.Steps, BlueprintStepPin{
			Key:             sourceStep.Key,
			StepVersion:     sourceStep.StepVersion,
			TemplateVersion: sourceStep.TemplateVersion,
		})
		included, err := evaluateInclusion(sourceStep.IsIncluded, resolvedSettings)
		if err != nil {
			return nil, fmt.Errorf("evaluating blueprint %q step %q inclusion: %w", source.Key, sourceStep.Key, err)
		}
		if !included {
			continue
		}
		step := BlueprintStep{
			StepDefinition: StepDefinition{
				Key:     strings.TrimPrefix(sourceStep.Key, source.Key+"--"),
				Title:   sourceStep.Title.DefaultMessage,
				Outputs: compileOutputs(sourceStep.Outputs),
			},
			Description: sourceStep.Description.DefaultMessage,
			Required:    sourceStep.Required,
		}
		for _, sourceNode := range sourceStep.Nodes {
			included, err := evaluateInclusion(sourceNode.IsIncluded, resolvedSettings)
			if err != nil {
				return nil, fmt.Errorf("evaluating blueprint %q node %q inclusion: %w", source.Key, sourceNode.Key, err)
			}
			if !included {
				continue
			}
			node, err := compileNode(sourceNode, resolvedSettings)
			if err != nil {
				return nil, fmt.Errorf("compiling blueprint %q node %q: %w", source.Key, sourceNode.Key, err)
			}
			step.Nodes = append(step.Nodes, node)
		}
		blueprint.Steps = append(blueprint.Steps, step)
	}
	if err := validateBlueprintReferences(blueprint); err != nil {
		return nil, fmt.Errorf("validating blueprint %q: %w", source.Key, err)
	}
	return blueprint, nil
}

func compileNode(source WorkbenchBlueprintNode, settings map[string]string) (NodeDefinition, error) {
	node := NodeDefinition{
		Type:        source.NodeType,
		Key:         source.Key,
		Title:       source.Title.DefaultMessage,
		Description: source.Description.DefaultMessage,
		AutoConfirm: source.IsInformationalNode,
	}
	switch source.NodeType {
	case NodeAPIRequest:
		if source.APIRequestDetails == nil {
			return node, fmt.Errorf("apiRequest node is missing api_request_details")
		}
		request := compileRequest(source.APIRequestDetails.Fixture, settings)
		node.Request = &request
	case NodeAsyncHandler:
		if source.AsyncHandlerDetails == nil {
			return node, fmt.Errorf("asyncHandler node is missing async_handler_details")
		}
		node.Events = append([]AsyncEvent(nil), source.AsyncHandlerDetails.Events...)
	case NodeTestHelper:
		if source.TestHelperDetails == nil {
			return node, fmt.Errorf("testHelper node is missing test_helper_details")
		}
		for i, request := range source.TestHelperDetails.Requests {
			compiled := compileRequest(request, settings)
			key := request.Key
			if key == "" {
				key = strconv.Itoa(i)
			}
			node.TestRequests = append(node.TestRequests, TestHelperRequest{Key: key, APIRequest: compiled})
		}
	case NodeUIComponent:
		if source.UIComponentDetails == nil {
			return node, fmt.Errorf("uiComponent node is missing ui_component_details")
		}
		node.UIComponent = compileUIComponent(*source.UIComponentDetails, settings)
	default:
		return node, fmt.Errorf("unsupported node type %q", source.NodeType)
	}
	node.ReviewPrompt = deriveReviewPrompt(node)
	node.ReviewCommand = deriveReviewCommand(node)
	return node, nil
}

func compileOutputs(source []WorkbenchStepOutput) []BlueprintOutput {
	outputs := make([]BlueprintOutput, 0, len(source))
	for _, output := range source {
		outputs = append(outputs, BlueprintOutput{
			Name:   output.Name,
			Source: output.Source,
			Schema: cloneMap(output.Schema),
		})
	}
	return outputs
}

func compileRequest(source WorkbenchRequestFixture, settings map[string]string) APIRequest {
	params := cloneMap(source.Params)
	hiddenParams := cloneMap(source.HiddenParams)
	headers := cloneStringMap(source.Headers)
	if params == nil {
		params = make(map[string]any)
	}
	if hiddenParams == nil {
		hiddenParams = make(map[string]any)
	}
	if headers == nil {
		headers = make(map[string]string)
	}
	expectedErrorType := ""
	for _, configured := range source.ConfiguredDetails {
		if !configurationMatches(configured.ConfigValue, settings) {
			continue
		}
		deepMerge(params, configured.Params)
		deepMerge(hiddenParams, configured.HiddenParams)
		for key, value := range configured.Headers {
			headers[key] = value
		}
		if configured.ExpectedErrorType != "" {
			expectedErrorType = configured.ExpectedErrorType
		}
	}
	return APIRequest{
		Key:               source.Key,
		Path:              source.Path,
		Method:            source.Method,
		Headers:           headers,
		Params:            params,
		HiddenParams:      hiddenParams,
		ExpectedErrorType: expectedErrorType,
		ProcessingDetails: source.ProcessingDetails,
		RegenerateEnv:     source.RegenerateEnv,
	}
}

func compileUIComponent(source WorkbenchUIComponentDetails, settings map[string]string) *UIComponentDetails {
	display := source.Display
	displayComponentRef := source.DisplayComponentRef
	stripeElementRef := cloneMap(source.StripeElementRef)
	sourceOptions := source.Options
	if stripeElementRef == nil {
		stripeElementRef = make(map[string]any)
	}
	for _, configured := range source.ConfiguredDetails {
		if !configurationMatches(configured.ConfigValue, settings) {
			continue
		}
		if configured.Display != "" {
			display = configured.Display
		}
		if configured.DisplayComponentRef != nil {
			displayComponentRef = configured.DisplayComponentRef
		}
		deepMerge(stripeElementRef, configured.StripeElementRef)
		if configured.Options != nil {
			sourceOptions = configured.Options
		}
	}
	component := &UIComponentDetails{
		Display:             display,
		DisplayComponentRef: displayComponentRef,
		StripeElementRef:    stripeElementRef,
	}
	for _, sourceOption := range sourceOptions {
		option := UIComponentOption{
			Type:  sourceOption.Type,
			Title: sourceOption.Title.DefaultMessage,
			Link:  sourceOption.Link,
		}
		for _, sourceRequest := range sourceOption.Requests {
			option.Requests = append(option.Requests, compileRequest(sourceRequest, settings))
		}
		component.Options = append(component.Options, option)
	}
	return component
}

func configurationMatches(selectors map[string]string, settings map[string]string) bool {
	for selector, expected := range selectors {
		name := selector
		if strings.HasPrefix(selector, "${settings:") && strings.HasSuffix(selector, "}") {
			name = strings.TrimSuffix(strings.TrimPrefix(selector, "${settings:"), "}")
		}
		if settings[name] != expected {
			return false
		}
	}
	return true
}

func resolveBlueprintSettings(source *WorkbenchBlueprint, selected map[string]string) map[string]string {
	defaults := make(map[string]string)
	resolved := make(map[string]string)
	for _, group := range source.BlueprintSettings {
		for _, field := range group.Settings {
			value := stringifyDefault(field.Schema.DefaultValue)
			defaults[group.Key+":"+field.Name] = value
			if _, exists := resolved[field.Name]; !exists {
				resolved[field.Name] = value
			}
		}
	}
	for _, step := range source.Steps {
		for _, group := range step.Settings {
			for _, field := range group.Settings {
				if _, exists := resolved[field.Name]; !exists {
					resolved[field.Name] = stringifyDefault(field.Schema.DefaultValue)
				}
			}
		}
		for target, sourceRef := range step.Config.Settings {
			group, name, ok := parseBlueprintSettingReference(sourceRef)
			if !ok {
				continue
			}
			value, exists := selected[target]
			if !exists {
				value, exists = selected[name]
			}
			if !exists {
				value, exists = defaults[group+":"+name]
			}
			if !exists {
				value, exists = resolved[name]
			}
			if exists {
				resolved[target] = value
			}
		}
	}
	for key, value := range selected {
		resolved[key] = value
	}
	// Workbench blueprints can branch on the mode of the API credential through
	// a step param. Co-op deliberately loads blueprints with the configured
	// test-mode key, so these references are always false.
	for _, step := range source.Steps {
		for target, sourceRef := range step.Config.Params {
			if sourceRef == "${env:livemode}" {
				resolved[target] = "false"
			}
		}
	}
	return resolved
}

func parseBlueprintSettingReference(value string) (string, string, bool) {
	const prefix = "${blueprint_settings."
	if !strings.HasPrefix(value, prefix) || !strings.HasSuffix(value, "}") {
		return "", "", false
	}
	group, name, ok := strings.Cut(strings.TrimSuffix(strings.TrimPrefix(value, prefix), "}"), ":")
	return group, name, ok && group != "" && name != ""
}

func stringifyDefault(value any) string {
	switch value := value.(type) {
	case nil:
		return ""
	case string:
		return value
	case bool:
		return strconv.FormatBool(value)
	case float64:
		return strconv.FormatFloat(value, 'f', -1, 64)
	default:
		encoded, _ := json.Marshal(value)
		return string(encoded)
	}
}

func cloneMap(source map[string]any) map[string]any {
	if source == nil {
		return nil
	}
	encoded, _ := json.Marshal(source)
	var clone map[string]any
	_ = json.Unmarshal(encoded, &clone)
	return clone
}

func cloneStringMap(source map[string]string) map[string]string {
	if source == nil {
		return nil
	}
	clone := make(map[string]string, len(source))
	for key, value := range source {
		clone[key] = value
	}
	return clone
}

func deepMerge(destination, source map[string]any) {
	if source == nil {
		return
	}
	if destination == nil {
		return
	}
	for key, value := range source {
		sourceMap, sourceIsMap := value.(map[string]any)
		destinationMap, destinationIsMap := destination[key].(map[string]any)
		if sourceIsMap && destinationIsMap {
			deepMerge(destinationMap, sourceMap)
			continue
		}
		destination[key] = cloneValue(value)
	}
}

func cloneValue(value any) any {
	encoded, _ := json.Marshal(value)
	var clone any
	_ = json.Unmarshal(encoded, &clone)
	return clone
}

func blueprintDigest(source *WorkbenchBlueprint) string {
	raw := source.raw
	if len(raw) == 0 {
		raw, _ = json.Marshal(source)
	}
	sum := sha256.Sum256(raw)
	return "sha256:" + hex.EncodeToString(sum[:])
}

func validateBlueprintReferences(bp *Blueprint) error {
	validRefs := map[string]bool{}
	for _, step := range bp.Steps {
		for _, node := range step.Nodes {
			validRefs[node.Key] = true
			validRefs[step.Key+"."+node.Key] = true
			for index, request := range node.TestRequests {
				validRefs[node.Key+"."+strconv.Itoa(index)] = true
				validRefs[step.Key+"."+node.Key+"."+strconv.Itoa(index)] = true
				if request.Key != "" {
					validRefs[node.Key+"."+request.Key] = true
					validRefs[step.Key+"."+node.Key+"."+request.Key] = true
				}
			}
			if node.UIComponent != nil {
				for _, option := range node.UIComponent.Options {
					for index := range option.Requests {
						validRefs[node.Key+"."+strconv.Itoa(index)] = true
						validRefs[step.Key+"."+node.Key+"."+strconv.Itoa(index)] = true
					}
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
		if !validRefs[ref] && !validNumericNodeReference(ref, validRefs) {
			return fmt.Errorf("unknown node reference %q", ref)
		}
		value = value[end+1:]
	}
}

func validNumericNodeReference(ref string, validRefs map[string]bool) bool {
	parts := strings.Split(ref, ".")
	if len(parts) < 2 || !isIntegerRefSegment(parts[len(parts)-1]) {
		return false
	}
	return validRefs[strings.Join(parts[:len(parts)-1], ".")]
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

// NewSessionFromBlueprint creates a pinned Session from a compiled blueprint.
func NewSessionFromBlueprint(bp *Blueprint, sessionID string, settings, params map[string]string) *Session {
	now := time.Now().UTC()
	contextStep := SessionStep{
		StepDefinition: StepDefinition{Key: "context-step", Title: "Project context"},
		Nodes: []SessionNode{{
			NodeDefinition: NodeDefinition{
				Key:         "scan-project",
				Type:        NodeTestHelper,
				Title:       "Understand the project",
				Description: "Scan the codebase to identify language, framework, dependencies, and existing Stripe code. Report what you find.",
				AutoConfirm: true,
			},
			State: NodePending,
		}},
	}

	steps := make([]SessionStep, 0, len(bp.Steps)+1)
	steps = append(steps, contextStep)
	for _, sourceStep := range bp.Steps {
		nodes := make([]SessionNode, len(sourceStep.Nodes))
		for index, node := range sourceStep.Nodes {
			nodes[index] = SessionNode{NodeDefinition: node, State: NodePending}
		}
		steps = append(steps, SessionStep{StepDefinition: sourceStep.StepDefinition, Nodes: nodes})
	}

	pinnedSettings := make(map[string]string, len(bp.ResolvedSettings)+len(settings))
	for key, value := range bp.ResolvedSettings {
		pinnedSettings[key] = value
	}
	for key, value := range settings {
		pinnedSettings[key] = value
	}
	pin := bp.Pin
	return &Session{
		SchemaVersion: CurrentSessionSchemaVersion,
		ID:            sessionID,
		Blueprint:     bp.ID,
		BlueprintPin:  &pin,
		Status:        SessionActive,
		Settings:      pinnedSettings,
		Params:        params,
		Steps:         steps,
		CreatedAt:     now,
		UpdatedAt:     now,
	}
}
