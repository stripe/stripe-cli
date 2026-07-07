package helpers

import (
	"errors"
	"fmt"
	"testing"

	"charm.land/huh/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAccessible(t *testing.T) {
	t.Setenv("STRIPE_COOP_ACCESSIBLE_PROMPTS", "true")
	assert.True(t, Accessible())

	t.Setenv("STRIPE_COOP_ACCESSIBLE_PROMPTS", "0")
	assert.False(t, Accessible())
}

func TestNormalizePromptErrorPreservesUserAbort(t *testing.T) {
	err := normalizePromptError(fmt.Errorf("prompt failed: %w", huh.ErrUserAborted))

	require.Error(t, err)
	assert.Contains(t, err.Error(), "canceled")
	assert.True(t, errors.Is(err, huh.ErrUserAborted))
}
