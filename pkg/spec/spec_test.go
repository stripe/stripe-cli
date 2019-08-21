package spec

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestLoadSpec(t *testing.T) {
	data, err := LoadSpec("")
	require.NoError(t, err)
	require.NotEmpty(t, data)
}

func TestUnmarshal_Simple(t *testing.T) {
	data := []byte(`{"type": "string"}`)
	var schema Schema
	err := json.Unmarshal(data, &schema)
	require.NoError(t, err)
	require.Equal(t, "string", schema.Type)
}

func TestUnmarshal_UnsupportedField(t *testing.T) {
	// We don't support 'const'
	data := []byte(`{const: "hello"}`)
	var schema Schema
	err := json.Unmarshal(data, &schema)
	require.Error(t, err)
}
