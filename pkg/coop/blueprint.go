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
func resolveBlueprint(source *WorkbenchBlueprint, selectedSettings map[string]string) (*WorkbenchBlueprint, map[string]string, error) {
	if source == nil {
		return nil, nil, fmt.Errorf("cannot resolve a nil blueprint")
	}
	data, err := json.Marshal(source)
	if err != nil {
		return nil, nil, fmt.Errorf("copying blueprint %q: %w", source.Key, err)
	}
	var resolved WorkbenchBlueprint
	if err := json.Unmarshal(data, &resolved); err != nil {
		return nil, nil, fmt.Errorf("copying blueprint %q: %w", source.Key, err)
	}
	resolved.raw = append(resolved.raw[:0], source.raw...)

	settings := resolveBlueprintSettings(&resolved, selectedSettings)
	for stepIndex := range resolved.Steps {
		for nodeIndex := range resolved.Steps[stepIndex].Nodes {
			node := &resolved.Steps[stepIndex].Nodes[nodeIndex]
			if err := resolveNode(node, settings); err != nil {
				return nil, nil, fmt.Errorf("resolving blueprint %q node %q: %w", source.Key, node.Key, err)
			}
		}
	}
	return &resolved, settings, nil
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
	for _, configured := range request.ConfiguredDetails {
		if !configurationMatches(configured.ConfigValue, settings) {
			continue
		}
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

func resolveUIComponent(component *WorkbenchUIComponentDetails, settings map[string]string) {
	if component.StripeElementRef == nil {
		component.StripeElementRef = make(map[string]any)
	}
	for _, configured := range component.ConfiguredDetails {
		if !configurationMatches(configured.ConfigValue, settings) {
			continue
		}
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
	blueprint, resolvedSettings, err := resolveBlueprint(source, settings)
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
		Params:              params,
		Steps:               steps,
		CreatedAt:           now,
		UpdatedAt:           now,
	}, nil
}
