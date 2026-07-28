package coop

import (
	"bytes"
	"encoding/json"
	"io"
	"regexp"
	"strings"
)

var nodeReferencePattern = regexp.MustCompile(`\$\{node\.([^}:]+):([^}]+)\}`)

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
				continue
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
