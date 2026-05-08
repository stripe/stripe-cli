package gen

import (
	"strings"

	"github.com/stripe/stripe-cli/pkg/spec"
)

// FirstSentence returns the first sentence of s using a lightweight heuristic.
// Paragraph breaks ("\n\n") are always a boundary. Otherwise it splits on
// ". " or ".\n" where canEndSentence holds for the character before the period
// and canStartSentence holds for the character after the separator. Known
// abbreviations (e.g., "e.g", "i.e") are never split on.
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
		if !canEndSentence(s[i-1]) {
			continue
		}

		atEnd := i == len(s)-1

		// Mid-string: require ". " or ".\n" followed by a sentence-starting char.
		// Single newlines act as soft line breaks between sentences in many specs.
		if !atEnd {
			if (s[i+1] != ' ' && s[i+1] != '\n') || !canStartSentence(s, i+2) {
				continue
			}
		}

		word := wordBefore(s, i)
		if sentenceAbbrevs[word] {
			continue
		}
		if atEnd && word == "etc" {
			// "etc." at end of string conventionally retains its period.
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

// sentenceAbbrevs lists lowercase word forms that are never sentence boundaries
// even when followed by a period and a space. "etc" is intentionally absent:
// it is a valid boundary mid-string (handled separately above).
var sentenceAbbrevs = map[string]bool{
	"e.g": true,
	"i.e": true,
}

// canEndSentence reports whether c is a valid character immediately before a
// sentence-ending period: a letter, digit, closing bracket, backtick, or
// closing double-quote (for descriptions ending like `"TX".`).
func canEndSentence(c byte) bool {
	return (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') ||
		(c >= '0' && c <= '9') || c == ')' || c == ']' || c == '`' || c == '"'
}

// canStartSentence reports whether s[i] looks like the first character of a
// new sentence: an uppercase letter, backtick, "[", or "(" immediately followed
// by an uppercase letter.
func canStartSentence(s string, i int) bool {
	if i >= len(s) {
		return false
	}
	c := s[i]
	return (c >= 'A' && c <= 'Z') || c == '`' || c == '[' ||
		(c == '(' && i+1 < len(s) && s[i+1] >= 'A' && s[i+1] <= 'Z')
}

// wordBefore returns the lowercase word that ends at s[end] (exclusive),
// scanning back to the nearest whitespace or opening bracket.
func wordBefore(s string, end int) string {
	start := strings.LastIndexAny(s[:end], " \t\n([") + 1
	return strings.ToLower(s[start:end])
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
