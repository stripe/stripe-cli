package gen

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/stripe/stripe-cli/pkg/spec"
)

func TestGetTypeAnyOf(t *testing.T) {
	subSchema1 := &spec.Schema{
		Type: "string",
	}
	subSchema2 := &spec.Schema{
		Type: "integer",
	}
	s := &spec.Schema{
		AnyOf: []*spec.Schema{subSchema1, subSchema2},
	}

	result := *GetType(s)

	assert.Equal(t, "string", result)
}

func TestGetTypeNilEnum(t *testing.T) {
	s := &spec.Schema{
		Enum: []interface{}{""},
	}

	result := GetType(s)

	assert.Nil(t, result)
}

func TestGetTypeInScalarString(t *testing.T) {
	s := &spec.Schema{
		Type: "string",
	}

	result := *GetType(s)

	assert.Equal(t, "string", result)
}

func TestGetTypeInScalarBoolean(t *testing.T) {
	s := &spec.Schema{
		Type: "boolean",
	}

	result := *GetType(s)

	assert.Equal(t, "boolean", result)
}

func TestGetTypeInScalarInteger(t *testing.T) {
	s := &spec.Schema{
		Type: "integer",
	}

	result := *GetType(s)

	assert.Equal(t, "integer", result)
}

func TestGetTypeInScalarNumber(t *testing.T) {
	s := &spec.Schema{
		Type: "number",
	}

	result := *GetType(s)

	assert.Equal(t, "number", result)
}

func TestGetTypeSimpleArray(t *testing.T) {
	s := &spec.Schema{
		Type: "array",
		Items: &spec.Schema{
			Type: "string",
		},
	}

	result := *GetType(s)

	assert.Equal(t, "array", result)
}

func TestGetTypeComplexArray(t *testing.T) {
	s := &spec.Schema{
		Type: "array",
		Items: &spec.Schema{
			Type: "object",
		},
	}

	result := GetType(s)

	assert.Nil(t, result)
}

func TestDenormalizeObject(t *testing.T) {
	s := &spec.Schema{
		Properties: map[string]*spec.Schema{
			"foo": {
				Type: "string",
			},
			"bar": {
				Type: "integer",
			},
		},
	}

	result := DenormalizeObject("test", s)

	assert.Equal(t, "string", result["test.foo"])
	assert.Equal(t, "integer", result["test.bar"])
}
