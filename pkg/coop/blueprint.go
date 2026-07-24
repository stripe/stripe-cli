package coop

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"
)

// LoadBlueprint retrieves a Workbench blueprint by its exact key.
func LoadBlueprint(ctx context.Context, repository BlueprintRepository, key string) (*WorkbenchBlueprint, error) {
	if repository == nil {
		return nil, fmt.Errorf("loading blueprints: no blueprint repository configured")
	}
	blueprint, err := repository.Retrieve(ctx, key)
	if err != nil {
		return nil, fmt.Errorf("retrieving blueprint %q: %w", key, err)
	}
	if blueprint == nil {
		return nil, fmt.Errorf("retrieving blueprint %q: empty response", key)
	}
	return blueprint, nil
}

// resolveBlueprint applies the selected Workbench configuration to a detached
// copy. The result retains the Workbench shape used by the session.
func resolveBlueprint(source *WorkbenchBlueprint, selectedSettings, selectedParams map[string]string) (*WorkbenchBlueprint, map[string]string, map[string]string, error) {
	if source == nil {
		return nil, nil, nil, fmt.Errorf("cannot resolve a nil blueprint")
	}
	data, err := json.Marshal(source)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("copying blueprint %q: %w", source.Key, err)
	}
	var resolved WorkbenchBlueprint
	if err := json.Unmarshal(data, &resolved); err != nil {
		return nil, nil, nil, fmt.Errorf("copying blueprint %q: %w", source.Key, err)
	}
	resolved.raw = append(resolved.raw[:0], source.raw...)

	settings := resolveBlueprintSettings(&resolved, selectedSettings)
	params := resolveBlueprintParams(&resolved, selectedParams)
	values := copyStringMap(settings)
	for key, value := range params {
		values[key] = value
	}

	steps := resolved.Steps[:0]
	for stepIndex := range resolved.Steps {
		step := &resolved.Steps[stepIndex]
		included, err := evaluateInclusion(step.IsIncluded, values)
		if err != nil {
			return nil, nil, nil, fmt.Errorf("resolving blueprint %q step %q inclusion: %w", source.Key, step.Key, err)
		}
		if !included {
			continue
		}

		nodes := step.Nodes[:0]
		for nodeIndex := range step.Nodes {
			node := &step.Nodes[nodeIndex]
			included, err := evaluateInclusion(node.IsIncluded, values)
			if err != nil {
				return nil, nil, nil, fmt.Errorf("resolving blueprint %q node %q inclusion: %w", source.Key, node.Key, err)
			}
			if !included {
				continue
			}
			if err := resolveNode(node, settings); err != nil {
				return nil, nil, nil, fmt.Errorf("resolving blueprint %q node %q: %w", source.Key, node.Key, err)
			}
			nodes = append(nodes, *node)
		}
		step.Nodes = nodes
		steps = append(steps, *step)
	}
	resolved.Steps = steps
	return &resolved, settings, params, nil
}

func resolveNode(node *WorkbenchBlueprintNode, settings map[string]string) error {
	switch node.NodeType {
	case NodeAPIRequest:
		if node.APIRequestDetails == nil {
			return fmt.Errorf("apiRequest node is missing api_request_details")
		}
		resolveRequest(&node.APIRequestDetails.Fixture, settings)
	case NodeAsyncHandler:
		if node.AsyncHandlerDetails == nil {
			return fmt.Errorf("asyncHandler node is missing async_handler_details")
		}
	case NodeTestHelper:
		if node.TestHelperDetails == nil {
			return fmt.Errorf("testHelper node is missing test_helper_details")
		}
		for index := range node.TestHelperDetails.Requests {
			request := &node.TestHelperDetails.Requests[index]
			resolveRequest(request, settings)
			if request.Key == "" {
				request.Key = strconv.Itoa(index)
			}
		}
	case NodeUIComponent:
		if node.UIComponentDetails == nil {
			return fmt.Errorf("uiComponent node is missing ui_component_details")
		}
		resolveUIComponent(node.UIComponentDetails, settings)
	default:
		return fmt.Errorf("unsupported node type %q", node.NodeType)
	}
	return nil
}

func resolveRequest(request *WorkbenchRequestFixture, settings map[string]string) {
	if request.Params == nil {
		request.Params = make(map[string]any)
	}
	if request.HiddenParams == nil {
		request.HiddenParams = make(map[string]any)
	}
	if request.Headers == nil {
		request.Headers = make(map[string]string)
	}
	matches := matchingRequestDetails(request.ConfiguredDetails, settings)
	for _, configured := range matches {
		mergeMap(request.Params, configured.Params)
		mergeMap(request.HiddenParams, configured.HiddenParams)
		for key, value := range configured.Headers {
			request.Headers[key] = value
		}
		if configured.ExpectedErrorType != "" {
			request.ExpectedErrorType = configured.ExpectedErrorType
		}
	}
	request.ConfiguredDetails = nil
}

func matchingRequestDetails(details []WorkbenchConfiguredDetails, settings map[string]string) []WorkbenchConfiguredDetails {
	var matches []WorkbenchConfiguredDetails
	for _, configured := range details {
		if configurationMatches(configured.ConfigValue, settings) {
			matches = append(matches, configured)
		}
	}
	if len(matches) == 0 && len(details) == 1 && hasOnlyStaticSelectors(details[0].ConfigValue) {
		return details
	}
	return matches
}

func resolveUIComponent(component *WorkbenchUIComponentDetails, settings map[string]string) {
	if component.StripeElementRef == nil {
		component.StripeElementRef = make(map[string]any)
	}
	matches := matchingUIComponentDetails(component.ConfiguredDetails, settings)
	for _, configured := range matches {
		if configured.Display != "" {
			component.Display = configured.Display
		}
		if configured.DisplayComponentRef != nil {
			component.DisplayComponentRef = configured.DisplayComponentRef
		}
		mergeMap(component.StripeElementRef, configured.StripeElementRef)
		if configured.Options != nil {
			component.Options = configured.Options
		}
	}
	component.ConfiguredDetails = nil
	for optionIndex := range component.Options {
		for requestIndex := range component.Options[optionIndex].Requests {
			resolveRequest(&component.Options[optionIndex].Requests[requestIndex], settings)
		}
	}
}

func matchingUIComponentDetails(details []WorkbenchUIConfiguredDetails, settings map[string]string) []WorkbenchUIConfiguredDetails {
	var matches []WorkbenchUIConfiguredDetails
	for _, configured := range details {
		if configurationMatches(configured.ConfigValue, settings) {
			matches = append(matches, configured)
		}
	}
	if len(matches) == 0 && len(details) == 1 && hasOnlyStaticSelectors(details[0].ConfigValue) {
		return details
	}
	return matches
}

func hasOnlyStaticSelectors(selectors map[string]string) bool {
	for selector := range selectors {
		if strings.HasPrefix(selector, "${settings:") {
			return false
		}
	}
	return len(selectors) > 0
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
	defaults, resolved := settingDefaults(source.BlueprintSettings)
	for _, step := range source.Steps {
		for _, group := range step.SettingsSchema {
			addFieldDefaults(resolved, group.Settings)
		}
		for target, sourceRef := range step.Settings {
			if value, exists := resolveMappedValue(target, sourceRef, "blueprint_settings.", selected, defaults, resolved); exists {
				resolved[target] = value
			}
		}
	}
	for key, value := range selected {
		resolved[key] = value
	}
	return resolved
}

func resolveBlueprintParams(source *WorkbenchBlueprint, selected map[string]string) map[string]string {
	defaults, resolved := paramDefaults(source.BlueprintParams)
	for _, step := range source.Steps {
		for _, group := range step.ParamsSchema {
			addFieldDefaults(resolved, group.Params)
		}
		for target, sourceRef := range step.Params {
			if sourceRef == "${env:livemode}" {
				resolved[target] = "false"
				continue
			}
			if value, exists := resolveMappedValue(target, sourceRef, "blueprint_params.", selected, defaults, resolved); exists {
				resolved[target] = value
			}
		}
	}
	for key, value := range selected {
		resolved[key] = value
	}
	return resolved
}

func settingDefaults(groups []WorkbenchSettingGroup) (map[string]string, map[string]string) {
	defaults := make(map[string]string)
	resolved := make(map[string]string)
	for _, group := range groups {
		for _, field := range group.Settings {
			if field.Schema.DefaultValue == nil {
				continue
			}
			value := stringifyDefault(field.Schema.DefaultValue)
			defaults[group.Key+":"+field.Name] = value
			if _, exists := resolved[field.Name]; !exists {
				resolved[field.Name] = value
			}
		}
	}
	return defaults, resolved
}

func paramDefaults(groups []WorkbenchParamGroup) (map[string]string, map[string]string) {
	defaults := make(map[string]string)
	resolved := make(map[string]string)
	for _, group := range groups {
		for _, field := range group.Params {
			if field.Schema.DefaultValue == nil {
				continue
			}
			value := stringifyDefault(field.Schema.DefaultValue)
			defaults[group.Key+":"+field.Name] = value
			if _, exists := resolved[field.Name]; !exists {
				resolved[field.Name] = value
			}
		}
	}
	return defaults, resolved
}

func addFieldDefaults(resolved map[string]string, fields []WorkbenchField) {
	for _, field := range fields {
		if field.Schema.DefaultValue == nil {
			continue
		}
		if _, exists := resolved[field.Name]; !exists {
			resolved[field.Name] = stringifyDefault(field.Schema.DefaultValue)
		}
	}
}

func resolveMappedValue(target, sourceRef, prefix string, selected, defaults, resolved map[string]string) (string, bool) {
	if value, exists := selected[target]; exists {
		return value, true
	}
	group, name, ok := parseGroupedReference(sourceRef, prefix)
	if !ok {
		return "", false
	}
	if value, exists := selected[name]; exists {
		return value, true
	}
	if value, exists := defaults[group+":"+name]; exists {
		return value, true
	}
	value, exists := resolved[name]
	return value, exists
}

func parseGroupedReference(value, referenceType string) (string, string, bool) {
	prefix := "${" + referenceType
	if !strings.HasPrefix(value, prefix) || !strings.HasSuffix(value, "}") {
		return "", "", false
	}
	group, name, ok := strings.Cut(strings.TrimSuffix(strings.TrimPrefix(value, prefix), "}"), ":")
	return group, name, ok && group != "" && name != ""
}

// evaluateInclusion handles the equality expression currently returned by
// Workbench. Unknown shapes fail instead of silently changing the workflow.
func evaluateInclusion(condition any, values map[string]string) (bool, error) {
	switch condition := condition.(type) {
	case nil:
		return true, nil
	case bool:
		return condition, nil
	case map[string]any:
		if len(condition) != 1 {
			return false, fmt.Errorf("expected one inclusion operator, got %d", len(condition))
		}
		rawOperands, ok := condition["=="]
		if !ok {
			for operator := range condition {
				return false, fmt.Errorf("unsupported inclusion operator %q", operator)
			}
		}
		operands, ok := rawOperands.([]any)
		if !ok || len(operands) != 2 {
			return false, fmt.Errorf("== requires two operands")
		}
		left, err := resolveInclusionOperand(operands[0], values)
		if err != nil {
			return false, err
		}
		right, err := resolveInclusionOperand(operands[1], values)
		if err != nil {
			return false, err
		}
		return stringifyDefault(left) == stringifyDefault(right), nil
	default:
		return false, fmt.Errorf("unsupported inclusion value %T", condition)
	}
}

func resolveInclusionOperand(value any, values map[string]string) (any, error) {
	text, ok := value.(string)
	if !ok {
		return value, nil
	}
	for _, prefix := range []string{"${params:", "${settings:"} {
		if strings.HasPrefix(text, prefix) && strings.HasSuffix(text, "}") {
			key := strings.TrimSuffix(strings.TrimPrefix(text, prefix), "}")
			resolved, exists := values[key]
			if !exists {
				return nil, fmt.Errorf("inclusion references unknown value %q", key)
			}
			return resolved, nil
		}
	}
	return text, nil
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

func mergeMap(destination, source map[string]any) {
	for key, value := range source {
		sourceMap, sourceIsMap := value.(map[string]any)
		destinationMap, destinationIsMap := destination[key].(map[string]any)
		if sourceIsMap && destinationIsMap {
			mergeMap(destinationMap, sourceMap)
			continue
		}
		destination[key] = value
	}
}

func blueprintDigest(source *WorkbenchBlueprint) string {
	raw := source.raw
	if len(raw) == 0 {
		raw, _ = json.Marshal(source)
	}
	sum := sha256.Sum256(raw)
	return "sha256:" + hex.EncodeToString(sum[:])
}

func blueprintPin(source *WorkbenchBlueprint) BlueprintPin {
	pin := BlueprintPin{
		ID:               source.ID,
		Key:              source.Key,
		Title:            source.Title.DefaultMessage,
		BlueprintVersion: source.BlueprintVersion,
		TemplateVersion:  source.TemplateVersion,
		Digest:           blueprintDigest(source),
	}
	for _, step := range source.Steps {
		pin.Steps = append(pin.Steps, BlueprintStepPin{
			Key:             step.Key,
			StepVersion:     step.StepVersion,
			TemplateVersion: step.TemplateVersion,
		})
	}
	return pin
}

// NewSessionFromBlueprint pins an effective Workbench definition with co-op progress.
func NewSessionFromBlueprint(source *WorkbenchBlueprint, sessionID string, settings, params map[string]string) (*Session, error) {
	blueprint, resolvedSettings, resolvedParams, err := resolveBlueprint(source, settings, params)
	if err != nil {
		return nil, err
	}
	now := time.Now().UTC()
	contextStep := SessionStep{
		WorkbenchStepDefinition: WorkbenchStepDefinition{
			Key:   "context-step",
			Title: MessageDescriptor{DefaultMessage: "Project context"},
		},
		Nodes: []SessionNode{{
			WorkbenchBlueprintNode: WorkbenchBlueprintNode{
				NodeType:            NodeTestHelper,
				Key:                 "scan-project",
				Title:               MessageDescriptor{DefaultMessage: "Understand the project"},
				Description:         MessageDescriptor{DefaultMessage: "Scan the codebase to identify language, framework, dependencies, and existing Stripe code. Report what you find."},
				IsInformationalNode: true,
				TestHelperDetails:   &WorkbenchTestHelperDetails{},
			},
			State: NodePending,
		}},
	}

	steps := make([]SessionStep, 0, len(blueprint.Steps)+1)
	steps = append(steps, contextStep)
	for _, step := range blueprint.Steps {
		nodes := make([]SessionNode, len(step.Nodes))
		for index, node := range step.Nodes {
			nodes[index] = SessionNode{
				WorkbenchBlueprintNode: node,
				ReviewPrompt:           deriveReviewPrompt(node),
				ReviewCommand:          deriveReviewCommand(node),
				State:                  NodePending,
			}
		}
		steps = append(steps, SessionStep{
			WorkbenchStepDefinition: step.WorkbenchStepDefinition,
			Nodes:                   nodes,
		})
	}

	definition := blueprint.WorkbenchBlueprintDefinition
	pin := blueprintPin(source)
	return &Session{
		SchemaVersion:       CurrentSessionSchemaVersion,
		ID:                  sessionID,
		Blueprint:           blueprint.Key,
		BlueprintDefinition: &definition,
		BlueprintPin:        &pin,
		Status:              SessionActive,
		Settings:            resolvedSettings,
		Params:              resolvedParams,
		Steps:               steps,
		CreatedAt:           now,
		UpdatedAt:           now,
	}, nil
}
