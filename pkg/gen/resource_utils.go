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

// DenormalizeObject accepts a schema and returns a map of its properties,
// fully expanded to the lowest level. Note that the one exception to this is
// for arrays of objects. The `GetType` function explicitly does not support
// arrays of objects, so we don't expand those (this is due to there being no
// way to specify an array of objects in a shell).
//
// We denormalize into a dot-notation since that has the most compatibility
// across shells. This is not following proper conventions (as per
// https://www.gnu.org/software/libc/manual/html_node/Argument-Syntax.html),
// but our API is becoming increasingly complex and dot-notation is the most
// compatible way to handle it (brackets are not supported in all shells).
func DenormalizeObject(name string, schema *spec.Schema) map[string]string {
	tmpProperties := denormalizedObject(schema)
	properties := make(map[string]string)
	for propName, propType := range tmpProperties {
		properties[name+"."+propName] = propType
	}
	return properties
}

// denormalizeObject is a recursive function that handles the actual unfolding
// of objects into a dot notation.
func denormalizedObject(schema *spec.Schema) map[string]string {
	properties := make(map[string]string)

	for propName, propSchema := range schema.Properties {
		if propSchema.Type == "object" {
			subProperties := denormalizedObject(propSchema)
			for subPropName, subPropSchema := range subProperties {
				properties[propName+"."+subPropName] = subPropSchema
			}
		} else {
			scalarType := GetType(propSchema)

			if scalarType == nil {
				continue
			}

			properties[propName] = *scalarType
		}
	}

	return properties
}
