package prompt

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestAccessible(t *testing.T) {
	t.Setenv("STRIPE_COOP_ACCESSIBLE_PROMPTS", "true")
	assert.True(t, Accessible())

	t.Setenv("STRIPE_COOP_ACCESSIBLE_PROMPTS", "0")
	assert.False(t, Accessible())
}
