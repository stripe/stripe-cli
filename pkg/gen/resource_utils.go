package gen

import "github.com/stripe/stripe-cli/pkg/spec"

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
