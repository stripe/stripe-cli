package coop

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"regexp"
	"strings"
)

var nodeReferencePattern = regexp.MustCompile(`\$\{node\.([^}:]+):([^}]+)\}`)

const nodeReferenceCandidatePrefix = "${node"

type nodeReference struct {
	Ref    string
	Base   string
	Source string
	Field  string
}

func walkNodeReferences(value any, visit func(nodeReference) error) error {
	return visitJSONStrings(value, func(text string) error {
		for _, match := range nodeReferencePattern.FindAllStringSubmatch(text, -1) {
			base, source, ok := splitNodeReference(match[1])
			if !ok {
				base = match[1]
			}
			if err := visit(nodeReference{
				Ref:    match[1],
				Base:   base,
				Source: source,
				Field:  match[2],
			}); err != nil {
				return err
			}
		}
		return nil
	})
}

func visitJSONStrings(value any, visit func(string) error) error {
	data, err := json.Marshal(value)
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
		text, ok := token.(string)
		if !ok {
			continue
		}
		if err := visit(text); err != nil {
			return err
		}
	}
}

func splitNodeReference(ref string) (base, source string, ok bool) {
	parts := strings.Split(ref, ".")
	if len(parts) < 2 || parts[0] == "" || parts[1] == "" {
		return "", "", false
	}
	base = parts[0] + "." + parts[1]
	if len(parts) > 2 {
		source = strings.Join(parts[2:], ".")
	}
	return base, source, true
}

func (s *Session) validateNodeReferences() error {
	nodeNumber := 0
	for _, step := range s.Steps {
		for _, node := range step.Nodes {
			nodeNumber++
			if err := visitJSONStrings(node.BlueprintNode, validateNodeReferenceSyntax); err != nil {
				return fmt.Errorf("validating node %q: %w", node.Key, err)
			}
			if err := walkNodeReferences(node.BlueprintNode, func(reference nodeReference) error {
				location, source, err := s.resolveNodeReference(reference.Ref)
				if err != nil {
					return err
				}
				if location.nodeNumber > nodeNumber || (location.nodeNumber == nodeNumber && source == "") {
					return fmt.Errorf("node %q references %q before it has completed", node.Key, reference.Ref)
				}
				return nil
			}); err != nil {
				return fmt.Errorf("validating node %q: %w", node.Key, err)
			}
		}
	}
	return nil
}

func validateNodeReferenceSyntax(value string) error {
	for {
		start := findNodeReferenceCandidate(value)
		if start == -1 {
			return nil
		}
		value = value[start:]
		end := strings.IndexByte(value, '}')
		next := findNodeReferenceCandidate(value[len(nodeReferenceCandidatePrefix):])
		if next != -1 {
			next += len(nodeReferenceCandidatePrefix)
		}
		if end == -1 || (next != -1 && next < end) {
			candidate := value
			if next != -1 {
				candidate = value[:next]
			}
			return fmt.Errorf("malformed node reference %q: missing closing brace", candidate)
		}
		placeholder := value[:end+1]
		if !nodeReferencePattern.MatchString(placeholder) {
			return fmt.Errorf("malformed node reference %q: expected ${node.<ref>:<field>}", placeholder)
		}
		value = value[end+1:]
	}
}

func findNodeReferenceCandidate(value string) int {
	offset := 0
	for {
		start := strings.Index(value[offset:], nodeReferenceCandidatePrefix)
		if start == -1 {
			return -1
		}
		start += offset
		after := start + len(nodeReferenceCandidatePrefix)
		if after == len(value) || value[after] == '.' || value[after] == ':' || value[after] == '}' {
			return start
		}
		offset = after
	}
}
