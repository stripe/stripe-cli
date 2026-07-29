package coop

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"
)

const DefaultOutputSource = "default"

// Selector returns the report-work --output selector for this required value.
func (o RequiredOutput) Selector() string {
	if o.Source == "" {
		return o.Field
	}
	return o.Source + ":" + o.Field
}

// RequiredOutputs returns the values produced by nodeNumber that later nodes
// reference.
func (s *Session) RequiredOutputs(nodeNumber int) ([]RequiredOutput, error) {
	step, _, nodeIndex, err := s.StepByNodeNumber(nodeNumber)
	if err != nil {
		return nil, err
	}
	targetBase := step.Key + "." + step.Nodes[nodeIndex].Key

	seen := map[string]RequiredOutput{}
	current := 0
	for _, candidateStep := range s.Steps {
		for _, candidateNode := range candidateStep.Nodes {
			current++
			if current <= nodeNumber {
				continue
			}
			if err := walkNodeReferences(candidateNode.NodeDefinition, func(reference nodeReference) error {
				if reference.Base != targetBase {
					return nil
				}
				output := RequiredOutput{Source: reference.Source, Field: reference.Field}
				seen[output.Selector()] = output
				return nil
			}); err != nil {
				return nil, fmt.Errorf("scanning node %d while finding required outputs: %w", current, err)
			}
		}
	}

	outputs := make([]RequiredOutput, 0, len(seen))
	for _, output := range seen {
		outputs = append(outputs, output)
	}
	sort.Slice(outputs, func(i, j int) bool {
		return outputs[i].Selector() < outputs[j].Selector()
	})
	return outputs, nil
}

// DependentNodeNumbers returns later nodes that directly or transitively
// reference outputs from nodeNumber.
func (s *Session) DependentNodeNumbers(nodeNumber int) ([]int, error) {
	step, _, nodeIndex, err := s.StepByNodeNumber(nodeNumber)
	if err != nil {
		return nil, err
	}

	dependencyBases := map[string]bool{
		step.Key + "." + step.Nodes[nodeIndex].Key: true,
	}
	var dependents []int
	current := 0
	for _, candidateStep := range s.Steps {
		for _, candidateNode := range candidateStep.Nodes {
			current++
			if current <= nodeNumber {
				continue
			}
			dependent := false
			if err := walkNodeReferences(candidateNode.NodeDefinition, func(reference nodeReference) error {
				if dependencyBases[reference.Base] {
					dependent = true
				}
				return nil
			}); err != nil {
				return nil, fmt.Errorf("scanning node %d while finding dependents: %w", current, err)
			}
			if !dependent {
				continue
			}
			dependents = append(dependents, current)
			dependencyBases[candidateStep.Key+"."+candidateNode.Key] = true
		}
	}
	return dependents, nil
}

// MissingRequiredOutputs returns required values that have not been persisted
// for nodeNumber.
func (s *Session) MissingRequiredOutputs(nodeNumber int) ([]RequiredOutput, error) {
	required, err := s.RequiredOutputs(nodeNumber)
	if err != nil {
		return nil, err
	}
	node, err := s.NodeByNumber(nodeNumber)
	if err != nil {
		return nil, err
	}

	var missing []RequiredOutput
	for _, output := range required {
		source := output.Source
		if source == "" {
			source = DefaultOutputSource
		}
		values := node.Outputs[source]
		if values == nil || len(values[output.Field]) == 0 {
			missing = append(missing, output)
		}
	}
	return missing, nil
}

// ResolvedNodeDefinition returns a copy of a node definition with every
// ${node...} reference replaced from persisted session outputs.
func (s *Session) ResolvedNodeDefinition(nodeNumber int) (NodeDefinition, error) {
	node, err := s.NodeByNumber(nodeNumber)
	if err != nil {
		return NodeDefinition{}, err
	}

	data, err := json.Marshal(node.NodeDefinition)
	if err != nil {
		return NodeDefinition{}, fmt.Errorf("encoding node %d: %w", nodeNumber, err)
	}
	var value any
	if err := json.Unmarshal(data, &value); err != nil {
		return NodeDefinition{}, fmt.Errorf("decoding node %d: %w", nodeNumber, err)
	}
	resolved, err := s.resolveNodeValue(value)
	if err != nil {
		return NodeDefinition{}, fmt.Errorf("resolving node %d: %w", nodeNumber, err)
	}
	data, err = json.Marshal(resolved)
	if err != nil {
		return NodeDefinition{}, fmt.Errorf("encoding resolved node %d: %w", nodeNumber, err)
	}

	var definition NodeDefinition
	if err := json.Unmarshal(data, &definition); err != nil {
		return NodeDefinition{}, fmt.Errorf("decoding resolved node %d: %w", nodeNumber, err)
	}
	return definition, nil
}

func (s *Session) resolveNodeValue(value any) (any, error) {
	switch typed := value.(type) {
	case string:
		return s.resolveNodeString(typed)
	case []any:
		resolved := make([]any, len(typed))
		for i, item := range typed {
			value, err := s.resolveNodeValue(item)
			if err != nil {
				return nil, err
			}
			resolved[i] = value
		}
		return resolved, nil
	case map[string]any:
		resolved := make(map[string]any, len(typed))
		for key, item := range typed {
			resolvedKeyValue, err := s.resolveNodeString(key)
			if err != nil {
				return nil, err
			}
			resolvedKey, ok := resolvedKeyValue.(string)
			if !ok {
				return nil, fmt.Errorf("node reference used as an object key must resolve to a string")
			}
			if _, exists := resolved[resolvedKey]; exists {
				return nil, fmt.Errorf("node reference produced duplicate object key %q", resolvedKey)
			}
			resolvedValue, err := s.resolveNodeValue(item)
			if err != nil {
				return nil, err
			}
			resolved[resolvedKey] = resolvedValue
		}
		return resolved, nil
	default:
		return value, nil
	}
}

func (s *Session) resolveNodeString(value string) (any, error) {
	matches := nodeReferencePattern.FindAllStringSubmatchIndex(value, -1)
	if len(matches) == 0 {
		return value, nil
	}

	if len(matches) == 1 && matches[0][0] == 0 && matches[0][1] == len(value) {
		raw, err := s.lookupNodeOutput(value[matches[0][2]:matches[0][3]], value[matches[0][4]:matches[0][5]])
		if err != nil {
			return nil, err
		}
		var decoded any
		if err := json.Unmarshal(raw, &decoded); err != nil {
			return nil, fmt.Errorf("decoding output for %q: %w", value, err)
		}
		return decoded, nil
	}

	var resolved strings.Builder
	last := 0
	for _, match := range matches {
		resolved.WriteString(value[last:match[0]])
		ref := value[match[2]:match[3]]
		field := value[match[4]:match[5]]
		raw, err := s.lookupNodeOutput(ref, field)
		if err != nil {
			return nil, err
		}
		text, err := outputAsString(raw)
		if err != nil {
			return nil, fmt.Errorf("using ${node.%s:%s} inside a string: %w", ref, field, err)
		}
		resolved.WriteString(text)
		last = match[1]
	}
	resolved.WriteString(value[last:])
	return resolved.String(), nil
}

func (s *Session) lookupNodeOutput(ref, field string) (json.RawMessage, error) {
	base, source, ok := splitNodeReference(ref)
	if !ok {
		return nil, fmt.Errorf("invalid node output reference ${node.%s:%s}", ref, field)
	}
	stepKey, nodeKey, _ := strings.Cut(base, ".")
	for i := range s.Steps {
		if s.Steps[i].Key != stepKey {
			continue
		}
		for j := range s.Steps[i].Nodes {
			node := &s.Steps[i].Nodes[j]
			if node.Key != nodeKey {
				continue
			}
			outputSource := source
			if outputSource == "" {
				outputSource = DefaultOutputSource
			}
			if values := node.Outputs[outputSource]; values != nil && len(values[field]) > 0 {
				return values[field], nil
			}
			return nil, fmt.Errorf("missing output %q for ${node.%s:%s}", RequiredOutput{Source: source, Field: field}.Selector(), ref, field)
		}
	}
	return nil, fmt.Errorf("unknown output source ${node.%s:%s}", ref, field)
}

func outputAsString(raw json.RawMessage) (string, error) {
	var decoded any
	if err := json.Unmarshal(raw, &decoded); err != nil {
		return "", err
	}
	switch value := decoded.(type) {
	case string:
		return value, nil
	case float64, bool:
		return fmt.Sprint(value), nil
	case nil:
		return "", nil
	default:
		return "", fmt.Errorf("objects and arrays cannot be interpolated into strings")
	}
}
