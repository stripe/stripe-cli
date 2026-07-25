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
	if _, err := s.NodeByNumber(nodeNumber); err != nil {
		return nil, err
	}

	seen := map[string]RequiredOutput{}
	current := 0
	for _, candidateStep := range s.Steps {
		for _, candidateNode := range candidateStep.Nodes {
			current++
			if current <= nodeNumber {
				continue
			}
			if err := walkNodeReferences(candidateNode.BlueprintNode, func(reference nodeReference) error {
				location, source, err := s.resolveNodeReference(reference.Ref)
				if err != nil {
					return err
				}
				if location.nodeNumber != nodeNumber {
					return nil
				}
				output := RequiredOutput{Source: source, Field: reference.Field}
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
	if _, err := s.NodeByNumber(nodeNumber); err != nil {
		return nil, err
	}

	dependencyNumbers := map[int]bool{nodeNumber: true}
	var dependents []int
	current := 0
	for _, candidateStep := range s.Steps {
		for _, candidateNode := range candidateStep.Nodes {
			current++
			if current <= nodeNumber {
				continue
			}
			dependent := false
			if err := walkNodeReferences(candidateNode.BlueprintNode, func(reference nodeReference) error {
				location, _, err := s.resolveNodeReference(reference.Ref)
				if err != nil {
					return err
				}
				if dependencyNumbers[location.nodeNumber] {
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
			dependencyNumbers[current] = true
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
func (s *Session) ResolvedNodeDefinition(nodeNumber int) (BlueprintNode, error) {
	node, err := s.NodeByNumber(nodeNumber)
	if err != nil {
		return BlueprintNode{}, err
	}

	data, err := json.Marshal(node.BlueprintNode)
	if err != nil {
		return BlueprintNode{}, fmt.Errorf("encoding node %d: %w", nodeNumber, err)
	}
	var value any
	if err := json.Unmarshal(data, &value); err != nil {
		return BlueprintNode{}, fmt.Errorf("decoding node %d: %w", nodeNumber, err)
	}
	resolved, err := s.resolveNodeValue(value, nodeNumber)
	if err != nil {
		return BlueprintNode{}, fmt.Errorf("resolving node %d: %w", nodeNumber, err)
	}
	data, err = json.Marshal(resolved)
	if err != nil {
		return BlueprintNode{}, fmt.Errorf("encoding resolved node %d: %w", nodeNumber, err)
	}

	var definition BlueprintNode
	if err := json.Unmarshal(data, &definition); err != nil {
		return BlueprintNode{}, fmt.Errorf("decoding resolved node %d: %w", nodeNumber, err)
	}
	return definition, nil
}

func (s *Session) resolveNodeValue(value any, nodeNumber int) (any, error) {
	switch typed := value.(type) {
	case string:
		return s.resolveNodeString(typed, nodeNumber)
	case []any:
		resolved := make([]any, len(typed))
		for i, item := range typed {
			value, err := s.resolveNodeValue(item, nodeNumber)
			if err != nil {
				return nil, err
			}
			resolved[i] = value
		}
		return resolved, nil
	case map[string]any:
		resolved := make(map[string]any, len(typed))
		for key, item := range typed {
			resolvedKeyValue, err := s.resolveNodeString(key, nodeNumber)
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
			resolvedValue, err := s.resolveNodeValue(item, nodeNumber)
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

func (s *Session) resolveNodeString(value string, nodeNumber int) (any, error) {
	matches := nodeReferencePattern.FindAllStringSubmatchIndex(value, -1)
	if len(matches) == 0 {
		return value, nil
	}

	if len(matches) == 1 && matches[0][0] == 0 && matches[0][1] == len(value) {
		ref := value[matches[0][2]:matches[0][3]]
		location, _, err := s.resolveNodeReference(ref)
		if err != nil {
			return nil, err
		}
		if location.nodeNumber == nodeNumber {
			return value, nil
		}
		raw, err := s.lookupNodeOutput(ref, value[matches[0][4]:matches[0][5]])
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
		location, _, err := s.resolveNodeReference(ref)
		if err != nil {
			return nil, err
		}
		if location.nodeNumber == nodeNumber {
			resolved.WriteString(value[match[0]:match[1]])
			last = match[1]
			continue
		}
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
	location, source, err := s.resolveNodeReference(ref)
	if err != nil {
		return nil, err
	}
	outputSource := source
	if outputSource == "" {
		outputSource = DefaultOutputSource
	}
	if values := location.node.Outputs[outputSource]; values != nil && len(values[field]) > 0 {
		return values[field], nil
	}
	return nil, fmt.Errorf("missing output %q for ${node.%s:%s}", RequiredOutput{Source: source, Field: field}.Selector(), ref, field)
}

type sessionNodeLocation struct {
	nodeNumber int
	node       *SessionNode
}

func (s *Session) resolveNodeReference(ref string) (sessionNodeLocation, string, error) {
	if ref == "" {
		return sessionNodeLocation{}, "", fmt.Errorf("invalid empty node output reference")
	}

	if location, source, matches := s.matchNodeReference(ref, true); matches == 1 {
		return location, source, nil
	} else if matches > 1 {
		return sessionNodeLocation{}, "", fmt.Errorf("ambiguous node output reference %q", ref)
	}
	if location, source, matches := s.matchNodeReference(ref, false); matches == 1 {
		return location, source, nil
	} else if matches > 1 {
		return sessionNodeLocation{}, "", fmt.Errorf("ambiguous node output reference %q", ref)
	}
	return sessionNodeLocation{}, "", fmt.Errorf("unknown output source ${node.%s}", ref)
}

func (s *Session) matchNodeReference(ref string, qualified bool) (sessionNodeLocation, string, int) {
	var found sessionNodeLocation
	foundSource := ""
	matches := 0
	nodeNumber := 0
	for stepIndex := range s.Steps {
		step := &s.Steps[stepIndex]
		for nodeIndex := range step.Nodes {
			nodeNumber++
			node := &step.Nodes[nodeIndex]
			base := node.Key
			if qualified {
				base = step.Key + "." + node.Key
			}
			source, ok := referenceSuffix(ref, base)
			if !ok {
				continue
			}
			matches++
			found = sessionNodeLocation{nodeNumber: nodeNumber, node: node}
			foundSource = source
		}
	}
	return found, foundSource, matches
}

func referenceSuffix(ref, base string) (string, bool) {
	if ref == base {
		return "", true
	}
	prefix := base + "."
	if strings.HasPrefix(ref, prefix) {
		return strings.TrimPrefix(ref, prefix), true
	}
	return "", false
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
