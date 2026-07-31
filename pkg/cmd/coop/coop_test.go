package coopcmd

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestRecommendFollowsAgentJSONContract keeps recommend on the documented
// agent contract: structured JSON with ok/protocol_version on success, and a
// structured recovery error instead of plain text on misuse.
func TestRecommendFollowsAgentJSONContract(t *testing.T) {
	rc := newCoopRecommendCmd()
	rc.all = true
	output := captureStdout(t, func() {
		require.NoError(t, rc.runRecommendCmd(rc.cmd, nil))
	})
	var payload map[string]json.RawMessage
	require.NoError(t, json.Unmarshal([]byte(output), &payload))
	assert.JSONEq(t, "true", string(payload["ok"]))
	assert.JSONEq(t, "1", string(payload["protocol_version"]))
	assert.Contains(t, payload, "blueprints")

	missingAll := newCoopRecommendCmd()
	stderr := captureStderr(t, func() {
		err := missingAll.runRecommendCmd(missingAll.cmd, nil)
		require.ErrorIs(t, err, RenderedError{})
	})
	var failure map[string]json.RawMessage
	require.NoError(t, json.Unmarshal([]byte(stderr), &failure))
	assert.JSONEq(t, "false", string(failure["ok"]))
	assert.Contains(t, failure, "recovery")
}

func TestCoopCommandDoesNotRegisterLegacyAgentCommands(t *testing.T) {
	cmd := newCoopCmd().cmd

	_, _, err := cmd.Find([]string{"step"})
	require.Error(t, err)

	_, _, err = cmd.Find([]string{"next-steps"})
	require.Error(t, err)
}
