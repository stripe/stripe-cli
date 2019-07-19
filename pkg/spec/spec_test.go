package spec

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
)

func BenchmarkLoadSpec(b *testing.B) {
	for n := 0; n < b.N; n++ {
		LoadSpec("")
	}
}

func TestLoadSpec(t *testing.T) {
	data, err := LoadSpec("")
	assert.NoError(t, err)
	assert.NotEmpty(t, data)
}

func TestUnmarshal_Simple(t *testing.T) {
	data := []byte(`{"type": "string"}`)
	var schema Schema
	err := json.Unmarshal(data, &schema)
	assert.NoError(t, err)
	assert.Equal(t, "string", schema.Type)
}

func TestUnmarshal_UnsupportedField(t *testing.T) {
	// We don't support 'const'
	data := []byte(`{const: "hello"}`)
	var schema Schema
	err := json.Unmarshal(data, &schema)
	assert.Error(t, err)
}
