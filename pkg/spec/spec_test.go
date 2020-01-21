package spec

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestLoadSpec(t *testing.T) {
	data, err := LoadSpec("../../api/openapi-spec/spec3.sdk.json")
	require.NoError(t, err)
	require.NotEmpty(t, data)
}

func TestUnmarshal_Simple(t *testing.T) {
	var schema Schema

	data := []byte(`{"type": "string"}`)
	err := json.Unmarshal(data, &schema)
	require.NoError(t, err)
	require.Equal(t, "string", schema.Type)
}

func TestUnmarshal_UnsupportedField(t *testing.T) {
	var schema Schema

	// We don't support 'const'
	data := []byte(`{const: "hello"}`)
	err := json.Unmarshal(data, &schema)
	require.Error(t, err)
}
