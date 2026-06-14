package coopcmd

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/stripe/stripe-cli/pkg/coop"
)

func TestNewCoopSessionAppliesSharedMetadata(t *testing.T) {
	previousOptions := options
	options = Options{SandboxClaimURL: func() string { return "https://dashboard.stripe.com/sandbox/claim_test" }}
	t.Cleanup(func() { options = previousOptions })

	session, err := newCoopSession(
		&coop.Blueprint{ID: "one-time-payment"},
		"coop_123",
		"go",
		[]string{"framework=gin", "framework=chi"},
		[]string{"customer_type=existing", "customer_type=new"},
		"parent_123",
		"deploy",
	)

	require.NoError(t, err)
	require.Equal(t, "coop_123", session.ID)
	assert.Equal(t, "go", session.Settings["language"])
	assert.Equal(t, "chi", session.Settings["framework"])
	assert.Equal(t, "new", session.Params["customer_type"])
	assert.Equal(t, "parent_123", session.ParentSessionID)
	assert.Equal(t, "deploy", session.ParentStepID)
	assert.True(t, session.UsedSandbox)
	assert.False(t, session.CreatedAt.IsZero())
}

func TestNewCoopSessionRejectsMalformedKeyValues(t *testing.T) {
	bp := &coop.Blueprint{ID: "one-time-payment"}

	tests := []struct {
		name     string
		settings []string
		params   []string
		want     string
	}{
		{name: "setting missing equals", settings: []string{"framework"}, want: "--setting must be in key=value format"},
		{name: "setting empty key", settings: []string{"=node"}, want: "--setting key cannot be empty"},
		{name: "setting whitespace key", settings: []string{"  =node"}, want: "--setting key cannot be empty"},
		{name: "param missing equals", params: []string{"customer_type"}, want: "--param must be in key=value format"},
		{name: "param empty key", params: []string{"=existing"}, want: "--param key cannot be empty"},
		{name: "param whitespace key", params: []string{"  =existing"}, want: "--param key cannot be empty"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			session, err := newCoopSession(bp, "coop_123", "go", tt.settings, tt.params, "", "")

			require.Error(t, err)
			assert.Nil(t, session)
			assert.Contains(t, err.Error(), tt.want)
		})
	}
}

func TestCoopRunReturnsStructuredErrorForMalformedSetting(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	cmd := newCoopAgentRunCmd().cmd
	cmd.SilenceErrors = true
	cmd.SilenceUsage = true
	cmd.SetArgs([]string{"one-time-payment", "--setting", "framework"})

	stderr := captureStderr(t, func() {
		err := cmd.Execute()
		require.Error(t, err)
	})

	var resp coop.CommandResponse
	require.NoError(t, json.Unmarshal([]byte(stderr), &resp))
	assert.False(t, resp.OK)
	assert.Contains(t, resp.Error, "--setting must be in key=value format")
	assert.Equal(t, "Use --setting key=value and --param key=value.", resp.Hint)

	store, err := coop.NewStore(coopConfigFolder())
	require.NoError(t, err)
	ids, err := store.List()
	require.NoError(t, err)
	assert.Empty(t, ids)
}

func TestCoopRunReturnsStructuredErrorForMalformedParam(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	cmd := newCoopAgentRunCmd().cmd
	cmd.SilenceErrors = true
	cmd.SilenceUsage = true
	cmd.SetArgs([]string{"one-time-payment", "--param", "=existing"})

	stderr := captureStderr(t, func() {
		err := cmd.Execute()
		require.Error(t, err)
	})

	var resp coop.CommandResponse
	require.NoError(t, json.Unmarshal([]byte(stderr), &resp))
	assert.False(t, resp.OK)
	assert.Contains(t, resp.Error, "--param key cannot be empty")
	assert.Equal(t, "Use --setting key=value and --param key=value.", resp.Hint)

	store, err := coop.NewStore(coopConfigFolder())
	require.NoError(t, err)
	ids, err := store.List()
	require.NoError(t, err)
	assert.Empty(t, ids)
}

func TestCoopRunPreservesBlueprintLoadError(t *testing.T) {
	cmd := newCoopAgentRunCmd().cmd
	cmd.SilenceErrors = true
	cmd.SilenceUsage = true
	cmd.SetArgs([]string{"flat"})

	stderr := captureStderr(t, func() {
		err := cmd.Execute()
		require.Error(t, err)
	})

	var resp coop.CommandResponse
	require.NoError(t, json.Unmarshal([]byte(stderr), &resp))
	assert.False(t, resp.OK)
	assert.Contains(t, resp.Error, "ambiguous blueprint prefix")
	assert.NotContains(t, resp.Error, "not found")
	assert.Equal(t, "stripe coop recommend", resp.Hint)
}

func TestCoopRunKeepsNotFoundGuidance(t *testing.T) {
	cmd := newCoopAgentRunCmd().cmd
	cmd.SilenceErrors = true
	cmd.SilenceUsage = true
	cmd.SetArgs([]string{"nonexistent-blueprint"})

	stderr := captureStderr(t, func() {
		err := cmd.Execute()
		require.Error(t, err)
	})

	var resp coop.CommandResponse
	require.NoError(t, json.Unmarshal([]byte(stderr), &resp))
	assert.Contains(t, resp.Error, "not found")
	assert.Equal(t, "stripe coop recommend", resp.Hint)
}

func TestCoopStartPreservesBlueprintLoadError(t *testing.T) {
	err := newCoopRunCmd().runCmd(nil, []string{"flat"})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "ambiguous blueprint prefix")
	assert.NotContains(t, err.Error(), "not found")
	assert.Contains(t, err.Error(), "stripe coop recommend")
}

func TestCoopStartKeepsNotFoundGuidance(t *testing.T) {
	err := newCoopRunCmd().runCmd(nil, []string{"nonexistent-blueprint"})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
	assert.Contains(t, err.Error(), "stripe coop recommend")
}

func TestAgentInstructionsFrameBlueprintAsAppImplementation(t *testing.T) {
	bp := &coop.Blueprint{Title: "Metered subscription"}
	session := &coop.Session{ID: "coop_123"}

	instructions := agentInstructions(bp, session)

	assert.Contains(t, instructions, "The blueprint describes the Stripe flow the developer wants in their app")
	assert.Contains(t, instructions, "The initial steps array and each start-work blueprint_step object are the concrete blueprint contract")
	assert.Contains(t, instructions, "Preserve the step order, type, api_request, events, review_prompt, and review_command")
	assert.Contains(t, instructions, "Stripe CLI commands are useful for setup and verification, but they are not the implementation")
	assert.Contains(t, instructions, `"apiRequest": Implement app code that calls this Stripe API`)
	assert.Contains(t, instructions, `"asyncHandler": Implement the app's webhook or async event handler for every event listed on the step`)
	assert.Contains(t, instructions, "Do not hardcode port 4242 unless the app is actually listening there")
	assert.Contains(t, instructions, "When start-work returns agent_guidance")
	assert.Contains(t, instructions, "blueprint_step.api_request.path")
	assert.Contains(t, instructions, "blueprint_step.api_request.params")
	assert.Contains(t, instructions, "sdk_example is present")
	assert.Contains(t, instructions, "generated SDK translation")
	assert.Contains(t, instructions, "do not treat an empty SDK call as complete")
	assert.Contains(t, instructions, "blueprint_step.events")
	assert.Contains(t, instructions, "webhook_example is present")
	assert.Contains(t, instructions, "generated handler translation")
	assert.Contains(t, instructions, "Thin event notifications are lightweight")
	assert.Contains(t, instructions, "treat v1.<event> thin migration aliases as the same logical event as <event>")
	assert.Contains(t, instructions, "Verification exercises the app code, not only a direct Stripe CLI/API call")
	assert.Contains(t, instructions, "Every non-skipped reviewable step needs at least one passed report-check before report-work")
	assert.Contains(t, instructions, "report-work points to the app file/function/route you changed")
	assert.Contains(t, instructions, "Never pass full card numbers to Stripe APIs or CLI commands")
}
