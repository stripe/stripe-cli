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

func TestResolveObjectSchemaDirect(t *testing.T) {
	s := &spec.Schema{Type: "object"}
	assert.Equal(t, s, ResolveObjectSchema(s))
}

func TestResolveObjectSchemaAnyOf(t *testing.T) {
	obj := &spec.Schema{Type: "object"}
	s := &spec.Schema{
		AnyOf: []*spec.Schema{
			obj,
			{Type: "string", Enum: []interface{}{""}},
		},
	}
	assert.Equal(t, obj, ResolveObjectSchema(s))
}

func TestResolveObjectSchemaNone(t *testing.T) {
	s := &spec.Schema{Type: "string"}
	assert.Nil(t, ResolveObjectSchema(s))
}

func TestIsClearableObjectTrue(t *testing.T) {
	s := &spec.Schema{
		AnyOf: []*spec.Schema{
			{Type: "object"},
			{Type: "string", Enum: []interface{}{""}},
		},
	}
	assert.True(t, IsClearableObject(s))
}

func TestIsClearableObjectFalseNoAnyOf(t *testing.T) {
	s := &spec.Schema{Type: "object"}
	assert.False(t, IsClearableObject(s))
}

func TestIsClearableObjectFalseNoEmptyString(t *testing.T) {
	s := &spec.Schema{
		AnyOf: []*spec.Schema{
			{Type: "object"},
			{Type: "string"},
		},
	}
	assert.False(t, IsClearableObject(s))
}

func TestFirstSentence(t *testing.T) {
	cases := []struct {
		input    string
		expected string
		reason   string
	}{
		// Core mechanics
		{"Multi. Sentence.", "Multi", "basic split"},
		{"Single sentence.", "Single sentence", "trailing period stripped"},
		{"First sentence.\nSecond sentence.", "First sentence", "single newline as sentence separator"},
		{"First para.\n\nSecond para.", "First para", "paragraph break"},
		{"", "", "empty"},

		// canEndSentence: uppercase letter (acronym before period)
		{"Destination URL. The URL is activated at onboarding.", "Destination URL", "uppercase letter ends word before period"},

		// canEndSentence: closing bracket / backtick / double-quote
		{"Amount in cents (e.g., 100 for $1.00). Must be.", "Amount in cents (e.g., 100 for $1.00)", "closing paren before period"},
		{"Set `enabled`. Next sentence.", "Set `enabled`", "backtick before period"},
		{"See [docs](https://example.com). Sign in.", "See [docs](https://example.com)", "markdown link — closing paren before period"},
		{`State code, such as "NY" or "TX".`, `State code, such as "NY" or "TX"`, "closing double-quote before period — trailing period stripped"},
		{`State code, such as "NY" or "TX". Required.`, `State code, such as "NY" or "TX"`, "closing double-quote before period — mid-string split"},

		// canStartSentence
		{"U.S. only. Required.", "U.S. only", "lowercase after period does not start sentence — prevents U.S. split"},
		{"Apply taxes. (Examples include VAT and GST.)", "Apply taxes", "open paren followed by uppercase starts sentence"},

		// sentenceAbbrevs
		{"Uses e.g., card. Next.", "Uses e.g., card", "e.g. — period not followed by space, separator check prevents split"},
		{"Uses e.g. Card instead.", "Uses e.g. Card instead", "e.g. — sentenceAbbrevs prevents split even before uppercase"},
		{"Uses i.e., physical goods. Next.", "Uses i.e., physical goods", "i.e. abbreviation"},

		// etc. special-cased separately from sentenceAbbrevs
		{"Accepts USD, EUR, etc.", "Accepts USD, EUR, etc.", "etc. at end of string retains period"},
		{"Accepts USD, EUR, etc. Contact support.", "Accepts USD, EUR, etc", "etc. mid-string is a valid sentence boundary"},
	}

	for _, tc := range cases {
		t.Run(tc.reason, func(t *testing.T) {
			assert.Equal(t, tc.expected, FirstSentence(tc.input))
		})
	}
}

