package gen

import (
	"strings"

	"github.com/stripe/stripe-cli/pkg/spec"
)

// sentenceAbbrevs lists dot-terminated tokens that never split a sentence
// mid-string. "e.g" and "i.e" always introduce continuation text.
// "etc" is intentionally absent: it IS a valid sentence boundary mid-string,
// but its trailing period is preserved when it ends the string (see below).
var sentenceAbbrevs = map[string]bool{
	"e.g": true,
	"i.e": true,
}

// FirstSentence returns the first sentence of s using a lightweight heuristic.
// It splits on ". " where the character before the period is a letter, digit,
// ")", or "]", and the character after the space starts with an uppercase
// letter, backtick, or "[". Paragraph breaks ("\n\n") are always a boundary.
// Known abbreviations (e.g., "e.g", "i.e") are never split.
func FirstSentence(s string) string {
	if s == "" {
		return ""
	}
	// Paragraph break is always a sentence boundary.
	if i := strings.Index(s, "\n\n"); i >= 0 {
		s = strings.TrimRight(s[:i], " \t")
	}
	for i := 1; i < len(s); i++ {
		if s[i] != '.' {
			continue
		}
		atEnd := i == len(s)-1
		// For a mid-string period, require ". " or ".\n" followed by an uppercase
		// letter, backtick, or "[". Single newlines are used as soft line breaks
		// between sentences in many spec descriptions.
		if !atEnd {
			if (s[i+1] != ' ' && s[i+1] != '\n') || i+2 >= len(s) {
				continue
			}
			next := s[i+2]
			if !((next >= 'A' && next <= 'Z') || next == '`' || next == '[' ||
				(next == '(' && i+3 < len(s) && s[i+3] >= 'A' && s[i+3] <= 'Z')) {
				continue
			}
		}
		prev := s[i-1]
		if !((prev >= 'a' && prev <= 'z') || (prev >= 'A' && prev <= 'Z') ||
			(prev >= '0' && prev <= '9') || prev == ')' || prev == ']' || prev == '`') {
			continue
		}
		// Walk back to find the word before the period.
		wordStart := i - 1
		for wordStart > 0 && s[wordStart-1] != ' ' && s[wordStart-1] != '\t' &&
			s[wordStart-1] != '\n' && s[wordStart-1] != '(' && s[wordStart-1] != '[' {
			wordStart--
		}
		word := strings.ToLower(s[wordStart:i])
		if sentenceAbbrevs[word] {
			continue
		}
		if atEnd && word == "etc" {
			// "etc." ends a list and conventionally retains its period.
			return s
		}
		return s[:i]
	}
	return s
}

// ResolveObjectSchema returns s if it is a plain object schema (type "object" or has
// Properties), or the first anyOf/oneOf branch that is an object schema. Returns nil if
// no object branch is found.
func ResolveObjectSchema(s *spec.Schema) *spec.Schema {
	if s == nil {
		return nil
	}
	if s.Type == "object" || len(s.Properties) > 0 {
		return s
	}
	for _, sub := range s.AnyOf {
		if obj := ResolveObjectSchema(sub); obj != nil {
			return obj
		}
	}
	for _, sub := range s.OneOf {
		if obj := ResolveObjectSchema(sub); obj != nil {
			return obj
		}
	}
	return nil
}

// IsClearableObject reports whether s uses the anyOf clearable-object pattern:
// one object branch and one empty-string-only branch. This is the Stripe v1 API
// convention for optional nested objects that can be removed by passing "".
func IsClearableObject(s *spec.Schema) bool {
	if len(s.AnyOf) == 0 {
		return false
	}
	hasObject, hasEmptyString := false, false
	for _, sub := range s.AnyOf {
		if sub.Type == "object" {
			hasObject = true
		}
		if sub.Type == "string" && len(sub.Enum) == 1 && sub.Enum[0] == "" {
			hasEmptyString = true
		}
	}
	return hasObject && hasEmptyString
}

var scalarTypes = map[string]bool{
	"boolean": true,
	"integer": true,
	"number":  true,
	"string":  true,
}

// GetType accepts a schema and returns its scalar type, if it has one.
//
// If the schema is monomorphic, it returns its type if it's scalar.
//
// If the schema is polymorphic, it returns the first scalar type for the
// schema, if there is any.
func GetType(schema *spec.Schema) *string {
	switch {
	case len(schema.AnyOf) > 0:
		for _, subSchema := range schema.AnyOf {
			scalarType := GetType(subSchema)
			if scalarType != nil {
				return scalarType
			}
		}
	case scalarTypes[schema.Type]:
		// Special case for string types that only support the "" (empty
		// string) value: we consider these to be non-scalar so we don't
		// generate a flag for those.
		if schema.Type == "string" {
			if len(schema.Enum) == 1 && schema.Enum[0] == "" {
				return nil
			}
		}
		return &schema.Type
	case schema.Type == "array" && schema.Items.Type != "object":
		arr := "array"
		return &arr
	}

	return nil
}
