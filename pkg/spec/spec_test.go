package spec

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestLoadSpec(t *testing.T) {
	data, err := LoadSpec()
	assert.NoError(t, err)
	assert.NotEmpty(t, data)
}
