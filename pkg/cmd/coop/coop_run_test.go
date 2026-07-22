package coopcmd

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"charm.land/huh/v2"
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

func TestAgentInstructionsIncludeOptionalStripeDocsGuidanceExactlyOnce(t *testing.T) {
	bp, err := coop.LoadBlueprint("one-time-payment")
	require.NoError(t, err)
	normalSession := coop.NewSessionFromBlueprint(bp, "coop_normal", nil, nil)

	guidedAction := &coop.GuidedAction{
		ID:           "guided-action",
		Title:        "Guided action",
		AgentContext: "Use the existing project.",
		Steps: []coop.SessionStep{
			{
				StepDefinition: coop.StepDefinition{Key: "guided-step", Title: "Guided step"},
				Nodes: []coop.SessionNode{
					{
						NodeDefinition: coop.NodeDefinition{Key: "guided-node", Title: "Guided node"},
						State:          coop.NodePending,
					},
				},
			},
		},
	}
	guidedSession := coop.NewSessionFromGuidedAction(guidedAction, "coop_guided", coop.GuidedActionSessionOptions{})

	tests := []struct {
		name         string
		instructions string
	}{
		{name: "normal blueprint", instructions: newCoopAgentRunResponse(bp, normalSession).AgentInstructions},
		{name: "guided follow-up", instructions: newCoopAgentGuidedActionResponse(guidedAction, guidedSession).AgentInstructions},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Contains(t, tt.instructions, "We will handle coordination and orchestration with you")
			assert.Contains(t, tt.instructions, "Prefer delegating well-bounded, independent tasks to cheaper subagents wherever possible, using your best judgment")
			assert.Contains(t, tt.instructions, "The main agent remains responsible for integration decisions and Co-op lifecycle commands")
			assert.Equal(t, 1, strings.Count(tt.instructions, stripeAgentGuidanceStart))
			assert.Equal(t, 1, strings.Count(tt.instructions, stripeAgentGuidanceEnd))
			assert.Contains(t, tt.instructions, "Co-op is responsible for selecting the integration and API family through its recommender and blueprint")
			assert.Contains(t, tt.instructions, "Do not use documentation or the repo-scoped Stripe skill to choose or switch integrations or API families")
			assert.Contains(t, tt.instructions, ".agents/skills/stripe-best-practices for Codex")
			assert.Contains(t, tt.instructions, ".claude/skills/stripe-best-practices for Claude Code")
			assert.Contains(t, tt.instructions, "Codex detects newly installed repo skills automatically")
			assert.Contains(t, tt.instructions, "Co-op prepares Claude Code's empty repo skill root before discovery")
			assert.Contains(t, tt.instructions, "Do not invoke this skill before Co-op selects the blueprint")
			assert.Contains(t, tt.instructions, "optional, supplemental implementation guidance")
			assert.Contains(t, tt.instructions, "neither skill nor documentation lookup is mandatory")
			assert.Contains(t, tt.instructions, "ambiguous or need clarification, proactively consult current official Stripe documentation")
			assert.Contains(t, tt.instructions, "Documentation lookup is optional, not a mandatory preflight or ceremony")
			assert.Contains(t, tt.instructions, "STRONGLY PREFER CURRENT OFFICIAL STRIPE CLI DOCUMENTATION OVER MODEL MEMORY")
			assert.Contains(t, tt.instructions, "if they conflict, follow the CLI documentation")
			assert.Contains(t, tt.instructions, `stripe docs search "<specific Stripe question>" --non-interactive --no-pager`)
			assert.Contains(t, tt.instructions, "stripe docs <result-path> --non-interactive --no-pager")
			assert.Contains(t, tt.instructions, "stripe docs api <resource-or-event> --non-interactive --no-pager")
			assert.Contains(t, tt.instructions, "stripe docs api <HTTP-method> <endpoint> --non-interactive --no-pager")
			assert.NotContains(t, tt.instructions, "Use the embedded Stripe best-practices guidance below to identify the correct integration and API family")
			assert.NotContains(t, tt.instructions, "Before writing Stripe-facing integration code")
			assert.NotContains(t, tt.instructions, "Open at least one relevant result")
			assert.NotContains(t, tt.instructions, "Search again when moving into a materially different Stripe domain")
			assert.NotContains(t, tt.instructions, "Latest Stripe API version: **2026-06-24.dahlia**")
			assert.Contains(t, tt.instructions, "BEFORE YOU START — ensure you have API access")
			assert.Contains(t, tt.instructions, "Agent lifecycle commands")
			assert.Contains(t, tt.instructions, `The "await" command is critical at step boundaries`)
		})
	}
}

func TestCoopAgentRunContinuesWhenOptionalRepoSkillInstallFails(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	rc := newCoopAgentRunCmd()
	ensureErr := fmt.Errorf("skill install failed")
	rc.ensureSkill = func() error { return ensureErr }
	rc.cmd.SilenceErrors = true
	rc.cmd.SilenceUsage = true
	rc.cmd.SetArgs([]string{"one-time-payment"})

	var runErr error
	stderr := captureStderr(t, func() {
		_ = captureStdout(t, func() {
			runErr = rc.cmd.Execute()
		})
	})

	require.NoError(t, runErr)
	assert.Contains(t, stderr, "unable to install the optional repo-scoped Stripe skill; continuing without it")
	assert.Contains(t, stderr, ensureErr.Error())
	store, storeErr := coop.NewStore(coopConfigFolder())
	require.NoError(t, storeErr)
	ids, listErr := store.List()
	require.NoError(t, listErr)
	assert.Len(t, ids, 1)
}

func TestCoopStartDoesNotInstallSkillWhenAgentDetectionFails(t *testing.T) {
	rc := newCoopRunCmd()
	ensureCalled := false
	rc.ensureSkill = func() error {
		ensureCalled = true
		return nil
	}
	rc.agent = "definitely-missing-coop-test-agent"

	runErr := rc.runCmd(nil, []string{"one-time-payment"})

	require.Error(t, runErr)
	assert.Contains(t, runErr.Error(), `agent "definitely-missing-coop-test-agent" not found`)
	assert.False(t, ensureCalled)
}

func TestCoopStartDiscoveryDoesNotInstallSkillBeforeRecommendation(t *testing.T) {
	rc := newCoopRunCmd()
	ensureCalled := false
	rc.ensureSkill = func() error {
		ensureCalled = true
		return nil
	}
	rc.agent = "definitely-missing-coop-test-agent"

	err := rc.runCmd(nil, nil)

	require.Error(t, err)
	assert.False(t, ensureCalled)
}

func TestCoopStartPreparesOnlyTheNeededAgentSkillState(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("launcher integration requires a POSIX shell")
	}

	tests := []struct {
		name           string
		agent          string
		blueprint      string
		wantSkill      bool
		wantClaudeRoot bool
	}{
		{name: "Claude discovery prepares empty root", agent: "claude", wantClaudeRoot: true},
		{name: "Codex discovery leaves skill absent", agent: "codex"},
		{name: "Claude explicit blueprint installs skill", agent: "claude", blueprint: "one-time-payment", wantSkill: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv("XDG_CONFIG_HOME", t.TempDir())
			t.Setenv("PATH", launcherFallbackTestPath(t))
			t.Setenv("TMUX", "")
			_, tmuxErr := exec.LookPath("tmux")
			require.Error(t, tmuxErr, "the test PATH must keep an installed tmux isolated")
			fakeAgent := filepath.Join(t.TempDir(), tt.agent)
			require.NoError(t, os.WriteFile(fakeAgent, []byte("#!/bin/sh\nexit 0\n"), 0o700))

			originalSelectString := selectString
			selectString = func(_ string, _ []huh.Option[string], value *string) error {
				*value = "normal"
				return nil
			}
			t.Cleanup(func() { selectString = originalSelectString })

			rc := newCoopRunCmd()
			rc.agent = fakeAgent
			skillCalled := false
			claudeRootCalled := false
			rc.ensureSkill = func() error {
				skillCalled = true
				return nil
			}
			rc.ensureClaudeSkills = func() error {
				claudeRootCalled = true
				return nil
			}
			args := []string(nil)
			if tt.blueprint != "" {
				args = []string{tt.blueprint}
			}

			var runErr error
			_ = captureStdout(t, func() { runErr = rc.runCmd(nil, args) })

			require.NoError(t, runErr)
			assert.Equal(t, tt.wantSkill, skillCalled)
			assert.Equal(t, tt.wantClaudeRoot, claudeRootCalled)
		})
	}
}

func launcherFallbackTestPath(t *testing.T) string {
	t.Helper()

	path := t.TempDir()
	for _, name := range []string{"bash", "cat", "rm"} {
		target, err := exec.LookPath(name)
		require.NoError(t, err)
		require.NoError(t, os.Symlink(target, filepath.Join(path, name)))
	}
	return path
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
