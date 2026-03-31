package gen

import (
	"encoding/json"
	"fmt"
	"os"
	"regexp"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

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
		{"Multi. Sentence.", "Multi", "basic split"},
		{"Single sentence.", "Single sentence", "no split — trailing period stripped"},
		{"Uses e.g., card. Next.", "Uses e.g., card", "e.g. with comma — abbrev list"},
		{"Uses e.g. Card instead.", "Uses e.g. Card instead", "e.g. without comma — no split, trailing period stripped"},
		{"Accepts USD, EUR, etc.", "Accepts USD, EUR, etc.", "etc. at end of string — period kept"},
		{"Accepts USD, EUR, etc. Contact support.", "Accepts USD, EUR, etc", "etc. is a valid sentence boundary"},
		{"Accepts USD, EUR, etc.) and others. Contact support.", "Accepts USD, EUR, etc.) and others", "etc.) — period not followed by space"},
		{"Uses i.e., physical goods. Next.", "Uses i.e., physical goods", "i.e."},
		{"U.S. only. Required.", "U.S. only", "uppercase before period prevents split on U.S."},
		{"Call the Balances API. One of...", "Call the Balances API", "uppercase acronym ending sentence"},
		{"The product code, such as an SKU. Required for L3.", "The product code, such as an SKU", "SKU ends sentence"},
		{"The card's CVC. It is highly recommended.", "The card's CVC", "CVC ends sentence"},
		{"Retrieve the page PDF. Can be set to a4.", "Retrieve the page PDF", "PDF ends sentence"},
		{"Out of scope for SCA. This parameter can only.", "Out of scope for SCA", "SCA ends sentence"},
		{"Destination URL. The URL is activated at onboarding.", "Destination URL", "URL ends sentence"},
		{"An alphanumeric ID. This field is required.", "An alphanumeric ID", "ID ends sentence"},
		{"Amount in cents (e.g., 100 for $1.00). Must be.", "Amount in cents (e.g., 100 for $1.00)", ") triggers split"},
		{"See [docs](https://example.com). Sign in.", "See [docs](https://example.com)", "markdown link ending sentence"},
		{"See [link](url) for details. Next sentence.", "See [link](url) for details", "markdown link mid-sentence"},
		{"First sentence.\nSecond sentence.", "First sentence", "single newline between sentences splits on first"},
		{"First para.\n\nSecond para.", "First para", "paragraph split — trailing period stripped"},
		{"", "", "empty"},
	}

	for _, tc := range cases {
		t.Run(tc.reason, func(t *testing.T) {
			assert.Equal(t, tc.expected, FirstSentence(tc.input))
		})
	}
}

func TestFirstSentenceAudit(t *testing.T) {
	data, _ := os.ReadFile("../../api/openapi-spec/spec3.cli.json")
	var raw interface{}
	json.Unmarshal(data, &raw)

	// [A-Z]\. [A-Z] in result = uppercase word ending a sentence that wasn't split
	re := regexp.MustCompile(`[A-Z]\. [A-Z]`)
	seen := map[string]bool{}
	var violations []string

	var walk func(v interface{})
	walk = func(v interface{}) {
		switch val := v.(type) {
		case map[string]interface{}:
			for k, child := range val {
				if k == "description" {
					if desc, ok := child.(string); ok && !seen[desc] {
						seen[desc] = true
						result := FirstSentence(desc)
						if re.MatchString(result) {
							violations = append(violations, fmt.Sprintf("result=%q  orig=%q", result, desc))
						}
					}
				} else {
					walk(child)
				}
			}
		case []interface{}:
			for _, item := range val {
				walk(item)
			}
		}
	}
	walk(raw)

	t.Logf("%d false negatives (uppercase-ending words before sentence boundary)", len(violations))
	for _, v := range violations {
		t.Log(v)
	}
	// Not a hard failure — audit/inventory only
}

func TestFirstSentenceSpec(t *testing.T) {
	data, err := os.ReadFile("../../api/openapi-spec/spec3.cli.json")
	require.NoError(t, err, "spec file must be readable")

	var raw interface{}
	require.NoError(t, json.Unmarshal(data, &raw), "spec must be valid JSON")

	// Walk all "description" string values in the spec recursively and verify
	// that every truncation is at a valid sentence boundary.
	var walk func(v interface{})
	walk = func(v interface{}) {
		switch val := v.(type) {
		case map[string]interface{}:
			for k, child := range val {
				if k == "description" {
					if desc, ok := child.(string); ok && desc != "" {
						result := FirstSentence(desc)
						if len(result) >= len(desc) {
							continue
						}
						// Determine what immediately follows result in the original desc.
						remainder := desc[len(result):]

						// Case 1: paragraph boundary not ending with '.' (e.g. </p>\n\n,
						// or ". \n\n" where a trailing space precedes the break).
						if strings.HasPrefix(strings.TrimLeft(remainder, " \t"), "\n") {
							continue
						}
						// Case 2: period follows — could be sentence split, paragraph
						// boundary after '.', or single trailing period stripped.
						if strings.HasPrefix(remainder, ".") {
							after := strings.TrimLeft(remainder[1:], " \t")
							if strings.HasPrefix(after, "\n") || after == "" {
								// Paragraph boundary after '.', or end of string — fine.
								continue
							}
							// Sentence split: assert valid sentence start two chars after result.
							if len(remainder) >= 3 {
								next := remainder[2]
								validStart := (next >= 'A' && next <= 'Z') || next == '`' || next == '['
								assert.True(t, validStart,
									"character after sentence split should start a sentence; desc=%q result=%q next=%q",
									desc, result, string(next))
							}
							continue
						}
						// Anything else is unexpected.
						assert.Fail(t, "unexpected truncation",
							"desc=%q result=%q remainder=%q", desc, result, remainder)
					}
				} else {
					walk(child)
				}
			}
		case []interface{}:
			for _, item := range val {
				walk(item)
			}
		}
	}
	walk(raw)
}
