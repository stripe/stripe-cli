package gen

import "github.com/stripe/stripe-cli/pkg/spec"

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
