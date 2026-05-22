package cmd

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestAPIKeyExpiredMessage_DefaultProfile(t *testing.T) {
	msg := apiKeyExpiredMessage("default")
	require.Contains(t, msg, "default profile")
	require.Contains(t, msg, "`stripe login`")
	require.Contains(t, msg, "`stripe whoami`")
	require.NotContains(t, msg, "--project-name")
}

func TestAPIKeyExpiredMessage_NamedProfile(t *testing.T) {
	msg := apiKeyExpiredMessage("work")
	require.Contains(t, msg, `"work"`)
	require.Contains(t, msg, "--project-name=work")
}
