package coopcmd

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/stripe/stripe-cli/pkg/coop"
	"github.com/stripe/stripe-cli/pkg/coop/skill"
)

func testAgent(t *testing.T, id string) *agentInfo {
	t.Helper()
	adapter := harnessByID(id)
	require.NotNil(t, adapter, "unknown harness %q", id)
	return &agentInfo{adapter: adapter, path: "/usr/local/bin/" + adapter.executables[0]}
}

func buildLauncherScript(t *testing.T, id string, autoApprove bool) string {
	t.Helper()
	promptPath := filepath.Join(t.TempDir(), "prompt")
	require.NoError(t, os.WriteFile(promptPath, []byte("prompt"), 0o600))

	launcherPath, err := (&coopRunCmd{}).buildAgentCmd(testAgent(t, id), promptPath, autoApprove)
	require.NoError(t, err)
	launcher, err := os.ReadFile(launcherPath)
	require.NoError(t, err)
	return string(launcher)
}

func TestHarnessRegistryCoversSupportedHarnesses(t *testing.T) {
	want := []string{"claude", "codex", "opencode", "cline", "roo", "openhands", "goose"}
	require.Len(t, harnessRegistry, len(want))
	for _, id := range want {
		require.NotNil(t, harnessByID(id), "missing harness %s", id)
	}
}

func TestHarnessSkillInstallDirectories(t *testing.T) {
	home := "/home/dev"
	tests := map[string]string{
		"claude":    "/home/dev/.claude/skills",
		"codex":     "/home/dev/.agents/skills",
		"opencode":  "/home/dev/.agents/skills",
		"cline":     "/home/dev/.cline/skills",
		"roo":       "/home/dev/.agents/skills",
		"openhands": "/home/dev/.agents/skills",
		"goose":     "/home/dev/.agents/skills",
	}
	for id, wantDir := range tests {
		adapter := harnessByID(id)
		require.NotNil(t, adapter, id)
		require.NotNil(t, adapter.skillDir, "%s must have a managed skill directory", id)
		assert.Equal(t, filepath.FromSlash(wantDir), adapter.skillDir(home), id)
	}
}

func TestHarnessActivationTokens(t *testing.T) {
	tests := map[string]string{
		"claude":    "Use the /stripe-coop skill.",
		"codex":     "Use the $stripe-coop skill.",
		"opencode":  "Use the stripe-coop skill.",
		"cline":     "Use the /stripe-coop skill.",
		"roo":       "Use the stripe-coop skill.",
		"openhands": "Use the stripe-coop skill.",
		"goose":     "Use the stripe-coop skill.",
	}
	for id, want := range tests {
		assert.Equal(t, want, harnessByID(id).activationLine, id)
	}
}

func TestHarnessLaunchArgumentConstruction(t *testing.T) {
	tests := []struct {
		id          string
		autoApprove bool
		contains    []string
		notContains []string
	}{
		{
			id: "opencode", autoApprove: true,
			contains:    []string{`--auto --prompt "$prompt"`},
			notContains: []string{"--dangerously"},
		},
		{
			id: "opencode", autoApprove: false,
			contains:    []string{`--prompt "$prompt"`},
			notContains: []string{"--auto "},
		},
		{
			id: "cline", autoApprove: true,
			contains: []string{`--yolo "$prompt"`},
		},
		{
			id: "cline", autoApprove: false,
			notContains: []string{"--yolo"},
		},
		{
			id: "roo", autoApprove: false,
			contains: []string{`--require-approval "$prompt"`},
		},
		{
			id: "roo", autoApprove: true,
			notContains: []string{"--require-approval"},
		},
		{
			id: "openhands", autoApprove: true,
			contains: []string{`--always-approve -t "$prompt"`},
		},
		{
			id: "openhands", autoApprove: false,
			contains:    []string{`-t "$prompt"`},
			notContains: []string{"--always-approve"},
		},
		{
			id: "goose", autoApprove: true,
			contains: []string{`GOOSE_MODE=auto `, `run -t "$prompt"`},
		},
		{
			id: "goose", autoApprove: false,
			contains: []string{`GOOSE_MODE=approve `, `run -t "$prompt"`},
		},
	}
	for _, tt := range tests {
		t.Run(tt.id, func(t *testing.T) {
			script := buildLauncherScript(t, tt.id, tt.autoApprove)
			assert.Contains(t, script, "exec ")
			assert.Contains(t, script, `"$prompt"`)
			for _, want := range tt.contains {
				assert.Contains(t, script, want)
			}
			for _, unwanted := range tt.notContains {
				assert.NotContains(t, script, unwanted)
			}
		})
	}
}

func TestDetectAgentPrefersExplicitFlagAndKnownAdapters(t *testing.T) {
	originalLookPath := lookPath
	lookPath = func(file string) (string, error) {
		if file == "opencode" {
			return "/opt/bin/opencode", nil
		}
		return "", errors.New("not found")
	}
	t.Cleanup(func() { lookPath = originalLookPath })

	agent, err := (&coopRunCmd{agent: "opencode"}).detectAgent()
	require.NoError(t, err)
	assert.Equal(t, "opencode", agent.adapter.id)
	assert.Equal(t, "/opt/bin/opencode", agent.path)

	// Auto-detection finds the single installed harness without prompting.
	agent, err = (&coopRunCmd{}).detectAgent()
	require.NoError(t, err)
	assert.Equal(t, "opencode", agent.adapter.id)
}

func TestDetectAgentFallsBackToGenericAdapterForUnknownCommand(t *testing.T) {
	originalLookPath := lookPath
	lookPath = func(file string) (string, error) {
		if file == "my-agent" {
			return "/opt/bin/my-agent", nil
		}
		return "", errors.New("not found")
	}
	t.Cleanup(func() { lookPath = originalLookPath })

	agent, err := (&coopRunCmd{agent: "my-agent"}).detectAgent()
	require.NoError(t, err)
	assert.Equal(t, "my-agent", agent.adapter.id)
	assert.Nil(t, agent.adapter.skillDir, "unknown agents receive no managed skill install")
	assert.True(t, agent.adapter.interactiveLaunch)

	script := func() string {
		promptPath := filepath.Join(t.TempDir(), "prompt")
		require.NoError(t, os.WriteFile(promptPath, []byte("prompt"), 0o600))
		launcherPath, err := (&coopRunCmd{}).buildAgentCmd(agent, promptPath, true)
		require.NoError(t, err)
		launcher, err := os.ReadFile(launcherPath)
		require.NoError(t, err)
		return string(launcher)
	}()
	assert.Contains(t, script, `'/opt/bin/my-agent' "$prompt"`)
}

func TestEnsureHarnessCoopSkillInstallsIntoHarnessDirectory(t *testing.T) {
	home := t.TempDir()
	originalHome := userHomeDir
	userHomeDir = func() (string, error) { return home, nil }
	t.Cleanup(func() { userHomeDir = originalHome })

	for _, id := range []string{"codex", "claude"} {
		installed, err := ensureHarnessCoopSkill(harnessByID(id))
		require.NoError(t, err, id)
		assert.True(t, installed, id)
	}

	for _, dir := range []string{
		filepath.Join(home, ".agents", "skills", skill.Name),
		filepath.Join(home, ".claude", "skills", skill.Name),
	} {
		_, err := os.Stat(filepath.Join(dir, "SKILL.md"))
		require.NoError(t, err)
		_, err = os.Stat(filepath.Join(dir, skill.ManifestFileName))
		require.NoError(t, err)
	}

	// Second run is an idempotent no-op that still reports the skill ready.
	installed, err := ensureHarnessCoopSkill(harnessByID("codex"))
	require.NoError(t, err)
	assert.True(t, installed)
}

func TestEnsureCoopSkillForFallsBackOnUnmanagedSkill(t *testing.T) {
	rc := &coopRunCmd{installCoopSkill: func(*harnessAdapter) (bool, error) {
		return false, skill.ErrUnmanagedSkill
	}}
	ready := rc.ensureCoopSkillFor(nil, testAgent(t, "codex"))
	assert.False(t, ready)

	prompt := rc.buildAgentPrompt(testAgent(t, "codex"), "")
	assert.NotContains(t, prompt, "$stripe-coop", "fallback prompt must not rely on the skill")
	assert.Contains(t, prompt, "production-grade Stripe integration")
}

func TestCompactSessionPromptCarriesOnlyDynamicContext(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())

	rc := &coopRunCmd{language: "python", coopSkillReady: true}
	session, err := rc.startSessionQuietly(commandTestBlueprint(t))
	require.NoError(t, err)

	prompt, err := rc.buildAgentPromptForSession(testAgent(t, "codex"), session)
	require.NoError(t, err)

	assert.Contains(t, prompt, "Use the $stripe-coop skill.")
	assert.Contains(t, prompt, "Session: "+session.ID)
	assert.Contains(t, prompt, "Integration: ")
	assert.Contains(t, prompt, "Language: python")
	assert.Contains(t, prompt, "stripe coop agent start-work --session="+session.ID+" --step=1")
	assert.NotContains(t, prompt, "<") // no unresolved placeholders

	// The stable lifecycle contract lives in the skill, not the prompt.
	assert.NotContains(t, prompt, "await-review")
	assert.NotContains(t, prompt, "report-check")
	assert.NotContains(t, prompt, stripeAgentGuidanceStart)
	assert.NotContains(t, prompt, "pm_card_visa")
	assert.NotContains(t, prompt, "sandbox create")
	assert.Less(t, len(prompt), 800)

	claudePrompt, err := rc.buildAgentPromptForSession(testAgent(t, "claude"), session)
	require.NoError(t, err)
	assert.Contains(t, claudePrompt, "Use the /stripe-coop skill.")
}

func TestCompactDiscoveryPromptCarriesOnlyDynamicContext(t *testing.T) {
	rc := &coopRunCmd{language: "python", coopSkillReady: true}
	prompt := rc.buildAgentPrompt(testAgent(t, "codex"), "")

	assert.Contains(t, prompt, "Use the $stripe-coop skill.")
	assert.Contains(t, prompt, "No blueprint is selected.")
	assert.Contains(t, prompt, "gather only developer intent that cannot be inferred")
	assert.Contains(t, prompt, "stripe coop recommend --all")
	assert.Contains(t, prompt, "Language hint: python.")
	assert.NotContains(t, prompt, "await-review")
	assert.NotContains(t, prompt, stripeAgentGuidanceStart)
	assert.Less(t, len(prompt), 600)
}

// TestAgentResponsesCarryProtocolVersion pins the wire-visible protocol
// version stamp on both success and failure output paths.
func TestAgentResponsesCarryProtocolVersion(t *testing.T) {
	output := captureStdout(t, func() {
		require.NoError(t, outputAgentResponse(coop.CommandResponse{
			OK:      true,
			Message: "done",
		}, nil))
	})
	assert.Contains(t, output, `"protocol_version": 1`)

	stderr := captureStderr(t, func() {
		err := outputAgentResponse(coop.CommandResponse{}, errors.New("boom"))
		require.ErrorIs(t, err, RenderedError{})
	})
	assert.Contains(t, stderr, `"protocol_version": 1`)
}

// TestSkillDocumentsEveryAgentSubcommand keeps the bundled skill's command
// reference in lockstep with the registered `stripe coop agent` subcommands.
func TestSkillDocumentsEveryAgentSubcommand(t *testing.T) {
	files, err := skill.Files()
	require.NoError(t, err)
	commandAPI := string(files["references/command-api.md"])

	agentCmd := newCoopAgentCmd().cmd
	subcommands := agentCmd.Commands()
	require.NotEmpty(t, subcommands)
	for _, sub := range subcommands {
		name := strings.Fields(sub.Use)[0]
		assert.Contains(t, commandAPI, "stripe coop agent "+name,
			"references/command-api.md must document %q", name)
	}
}
