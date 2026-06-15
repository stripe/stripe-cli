// Package i18n provides user-facing string lookup for the Stripe CLI.
//
// Strings live in en_us.yaml. Use T for plain lookups and Tf for strings
// with {varName} placeholders — pass alternating name/value pairs as args.
package i18n

import (
	_ "embed"
	"fmt"
	"strings"

	"gopkg.in/yaml.v3"
)

//go:embed en_us.yaml
var rawStrings []byte

var messages map[string]interface{}

func init() {
	if err := yaml.Unmarshal(rawStrings, &messages); err != nil {
		panic("i18n: failed to parse en_us.yaml: " + err.Error())
	}
}

// T returns the string at the given dot-separated key path.
// Panics if the key is missing or is not a leaf string — missing keys are bugs.
func T(key string) string {
	parts := strings.Split(key, ".")
	var current interface{} = messages
	for _, part := range parts {
		m, ok := current.(map[string]interface{})
		if !ok {
			panic(fmt.Sprintf("i18n: key %q: expected map at segment %q", key, part))
		}
		current, ok = m[part]
		if !ok {
			panic(fmt.Sprintf("i18n: key %q not found", key))
		}
	}
	s, ok := current.(string)
	if !ok {
		panic(fmt.Sprintf("i18n: key %q is not a string (got %T)", key, current))
	}
	return s
}

// Args holds named placeholder values for Tf.
type Args map[string]string

// Tf returns the string at the given key with {name} placeholders replaced.
// Example: Tf("errors.not_found", Args{"id": "cus_123"})
func Tf(key string, args Args) string {
	s := T(key)
	for name, val := range args {
		s = strings.ReplaceAll(s, "{"+name+"}", val)
	}
	return s
}
