package cmd

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestLoginRequiredAgentHint(t *testing.T) {
	msg := loginRequiredAgentHint()
	require.Contains(t, msg, "stripe sandbox create --from-git")
	require.Contains(t, msg, "stripe sandbox create --email")
	require.Contains(t, msg, "STRIPE_API_KEY")
	require.Contains(t, msg, "--api-key")
}
