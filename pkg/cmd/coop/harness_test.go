package coopcmd

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"charm.land/huh/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mustHarness returns the registry entry for id, failing the test if absent.
func mustHarness(t *testing.T, id string) harness {
	t.Helper()
	h, ok := harnessByID(id)
	require.True(t, ok, "harness %q not in registry", id)
	return h
}

// stubLookPath makes exactly the named binaries appear installed, at a
// synthetic path, for the duration of the test.
func stubLookPath(t *testing.T, installed ...string) {
	t.Helper()
	present := make(map[string]bool, len(installed))
	for _, name := range installed {
		present[name] = true
	}
	original := lookPath
	lookPath = func(file string) (string, error) {
		if present[file] {
			return "/usr/local/bin/" + file, nil
		}
		return "", errors.New("not found")
	}
	t.Cleanup(func() { lookPath = original })
}

// stubSelect answers the harness picker with choice and records the options it
// was offered.
func stubSelect(t *testing.T, choice string) *[]huh.Option[string] {
	t.Helper()
	var offered []huh.Option[string]
	original := selectString
	selectString = func(title string, options []huh.Option[string], value *string) error {
		offered = options
		*value = choice
		return nil
	}
	t.Cleanup(func() { selectString = original })
	return &offered
}

func TestRegistryEntriesAreWellFormed(t *testing.T) {
	seenIDs := map[string]bool{}
	seenBinaries := map[string]bool{}

	for _, h := range supportedHarnesses {
		t.Run(h.id, func(t *testing.T) {
			assert.NotEmpty(t, h.id)
			assert.NotEmpty(t, h.displayName)
			assert.NotEmpty(t, h.binary)
			assert.False(t, seenIDs[h.id], "duplicate harness id %q", h.id)
			assert.False(t, seenBinaries[h.binary], "duplicate harness binary %q", h.binary)
			seenIDs[h.id] = true
			seenBinaries[h.binary] = true
		})
	}
}

func TestHarnessForMatchesIDBinaryAndPath(t *testing.T) {
	tests := []struct {
		name    string
		agent   string
		path    string
		wantID  string
		wantHit bool
	}{
		{name: "by id", agent: "cursor", path: "/usr/local/bin/cursor-agent", wantID: "cursor", wantHit: true},
		{name: "by binary", agent: "cursor-agent", path: "/usr/local/bin/cursor-agent", wantID: "cursor", wantHit: true},
		{name: "by absolute path", agent: "/opt/homebrew/bin/gemini", path: "/opt/homebrew/bin/gemini", wantID: "gemini", wantHit: true},
		{name: "windows extension", agent: "cursor-agent.exe", path: `C:\tools\cursor-agent.exe`, wantID: "cursor", wantHit: true},
		{name: "unknown binary", agent: "mycoder", path: "/usr/local/bin/mycoder", wantHit: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h, ok := harnessFor(tt.agent, tt.path)

			assert.Equal(t, tt.wantHit, ok)
			if tt.wantHit {
				assert.Equal(t, tt.wantID, h.id)
			}
		})
	}
}

// TestHarnessForRejectsLookalikePaths guards the substring-matching bug the
// registry replaced: a custom binary living under a directory whose name merely
// contains "claude" must not inherit Claude Code's flags.
func TestHarnessForRejectsLookalikePaths(t *testing.T) {
	lookalikes := []struct{ agent, path string }{
		{agent: "mycoder", path: "/home/claude-vm/bin/mycoder"},
		{agent: "mycoder", path: "/opt/codex-tools/bin/mycoder"},
		{agent: "claude-wrapper", path: "/usr/local/bin/claude-wrapper"},
		// A wrapper script is a different program from the harness it shares a
		// stem with; only Windows executable extensions are stripped.
		{agent: "claude.sh", path: "/usr/local/bin/claude.sh"},
	}

	for _, tt := range lookalikes {
		t.Run(tt.path, func(t *testing.T) {
			_, ok := harnessFor(tt.agent, tt.path)

			assert.False(t, ok, "%s must not resolve to a registry harness", tt.path)
		})
	}
}

func TestDetectAgentReturnsSoleInstalledHarness(t *testing.T) {
	stubLookPath(t, "gemini")

	agent, err := (&coopRunCmd{}).detectAgent()

	require.NoError(t, err)
	assert.Equal(t, "gemini", agent.harness.id)
	assert.Equal(t, "/usr/local/bin/gemini", agent.path)
}

func TestDetectAgentPromptsWhenMultipleInstalled(t *testing.T) {
	stubLookPath(t, "claude", "opencode", "cursor-agent")
	offered := stubSelect(t, "opencode")

	agent, err := (&coopRunCmd{}).detectAgent()

	require.NoError(t, err)
	assert.Equal(t, "opencode", agent.harness.id)

	// Every installed harness is offered, in registry order, and nothing else.
	var labels, values []string
	for _, option := range *offered {
		labels = append(labels, option.Key)
		values = append(values, option.Value)
	}
	assert.Equal(t, []string{"claude", "cursor", "opencode"}, values)
	assert.Equal(t, []string{"Claude Code", "Cursor CLI", "opencode"}, labels)
}

func TestDetectAgentPropagatesPickerError(t *testing.T) {
	stubLookPath(t, "claude", "codex")
	pickerErr := errors.New("picker canceled")
	original := selectString
	selectString = func(string, []huh.Option[string], *string) error { return pickerErr }
	t.Cleanup(func() { selectString = original })

	agent, err := (&coopRunCmd{}).detectAgent()

	require.ErrorIs(t, err, pickerErr)
	assert.Nil(t, agent)
}

func TestDetectAgentErrorListsSupportedHarnesses(t *testing.T) {
	stubLookPath(t)

	_, err := (&coopRunCmd{}).detectAgent()

	require.Error(t, err)
	for _, h := range supportedHarnesses {
		assert.Contains(t, err.Error(), h.id)
	}
}

func TestDetectAgentResolvesExplicitAgentFlag(t *testing.T) {
	// --agent names a harness whose binary is installed but which is not the
	// first in registry order.
	stubLookPath(t, "claude", "cursor-agent")

	agent, err := (&coopRunCmd{agent: "cursor-agent"}).detectAgent()

	require.NoError(t, err)
	assert.Equal(t, "cursor", agent.harness.id)
	assert.Equal(t, "--force", agent.harness.autoApproveFlag)
}

func TestDetectAgentFallsBackToCustomHarness(t *testing.T) {
	stubLookPath(t, "mycoder")

	agent, err := (&coopRunCmd{agent: "mycoder"}).detectAgent()

	require.NoError(t, err)
	assert.Equal(t, "mycoder", agent.harness.id)
	// A custom binary gets no flags: co-op cannot know its permission model or
	// whether it seeds an interactive session from a flag.
	assert.Empty(t, agent.harness.autoApproveFlag)
	assert.Empty(t, agent.harness.promptFlag)
}

func TestDetectAgentErrorsWhenExplicitAgentMissing(t *testing.T) {
	stubLookPath(t)

	_, err := (&coopRunCmd{agent: "mycoder"}).detectAgent()

	require.Error(t, err)
	assert.Contains(t, err.Error(), `agent "mycoder" not found in PATH`)
}

func TestPromptAutoApproveSkippedWithoutAutoApproveFlag(t *testing.T) {
	original := selectString
	selectString = func(string, []huh.Option[string], *string) error {
		t.Fatal("permission prompt shown for a harness with no auto-approve flag")
		return nil
	}
	t.Cleanup(func() { selectString = original })

	autoApprove, err := (&coopRunCmd{}).promptAutoApprove(&agentInfo{harness: customHarness("mycoder")})

	require.NoError(t, err)
	assert.False(t, autoApprove)
}

func TestPromptAutoApproveTitleNamesHarness(t *testing.T) {
	var title string
	original := selectString
	selectString = func(t string, _ []huh.Option[string], value *string) error {
		title = t
		*value = "auto"
		return nil
	}
	t.Cleanup(func() { selectString = original })

	autoApprove, err := (&coopRunCmd{}).promptAutoApprove(&agentInfo{harness: mustHarness(t, "opencode")})

	require.NoError(t, err)
	assert.True(t, autoApprove)
	assert.Equal(t, "Permission mode for opencode:", title)
}

func TestBuildAgentCmdRendersHarnessInvocation(t *testing.T) {
	tests := []struct {
		name        string
		harness     harness
		autoApprove bool
		wantExec    string
	}{
		{
			name:     "positional prompt",
			harness:  mustHarness(t, "claude"),
			wantExec: `exec '/usr/local/bin/claude' "$prompt"`,
		},
		{
			name:        "positional prompt with auto approve",
			harness:     mustHarness(t, "claude"),
			autoApprove: true,
			wantExec:    `exec '/usr/local/bin/claude' --dangerously-skip-permissions "$prompt"`,
		},
		{
			name:     "interactive prompt flag",
			harness:  mustHarness(t, "gemini"),
			wantExec: `exec '/usr/local/bin/gemini' -i "$prompt"`,
		},
		{
			name:        "interactive prompt flag with auto approve",
			harness:     mustHarness(t, "gemini"),
			autoApprove: true,
			wantExec:    `exec '/usr/local/bin/gemini' --approval-mode=yolo -i "$prompt"`,
		},
		{
			name:        "opencode seeds the tui with --prompt",
			harness:     mustHarness(t, "opencode"),
			autoApprove: true,
			wantExec:    `exec '/usr/local/bin/opencode' --auto --prompt "$prompt"`,
		},
		{
			name:     "subcommand args precede flags",
			harness:  mustHarness(t, "goose"),
			wantExec: `exec '/usr/local/bin/goose' run -s -t "$prompt"`,
		},
		{
			name:     "pi takes a bare positional prompt",
			harness:  mustHarness(t, "pi"),
			wantExec: `exec '/usr/local/bin/pi' "$prompt"`,
		},
		{
			name:        "custom agent never gets flags",
			harness:     customHarness("mycoder"),
			autoApprove: true,
			wantExec:    `exec '/usr/local/bin/mycoder' "$prompt"`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			promptPath := filepath.Join(t.TempDir(), "prompt.txt")
			require.NoError(t, os.WriteFile(promptPath, []byte("do the thing"), 0o600))
			agent := &agentInfo{harness: tt.harness, path: "/usr/local/bin/" + tt.harness.binary}

			launcherPath, err := (&coopRunCmd{}).buildAgentCmd(agent, promptPath, tt.autoApprove)
			require.NoError(t, err)

			script, err := os.ReadFile(launcherPath)
			require.NoError(t, err)
			assert.Contains(t, string(script), tt.wantExec+"\n")
			// The launcher removes the prompt file and itself before exec'ing,
			// so a stale prompt is never left behind in TMPDIR.
			assert.Contains(t, string(script), "rm -f "+shellQuote(promptPath)+" "+shellQuote(launcherPath))
		})
	}
}

// TestBuildAgentCmdExportsAutoApproveEnv covers Goose, whose approval policy is
// an environment variable rather than a flag.
func TestBuildAgentCmdExportsAutoApproveEnv(t *testing.T) {
	tests := []struct {
		name        string
		autoApprove bool
		wantEnv     bool
	}{
		{name: "auto approve exports GOOSE_MODE", autoApprove: true, wantEnv: true},
		// Declining must not export anything: GOOSE_MODE already defaults to
		// auto, and overriding it would silently discard a developer's
		// configured approval mode.
		{name: "normal mode leaves env untouched", autoApprove: false, wantEnv: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			promptPath := filepath.Join(t.TempDir(), "prompt.txt")
			require.NoError(t, os.WriteFile(promptPath, []byte("prompt"), 0o600))
			agent := &agentInfo{harness: mustHarness(t, "goose"), path: "/usr/local/bin/goose"}

			launcherPath, err := (&coopRunCmd{}).buildAgentCmd(agent, promptPath, tt.autoApprove)
			require.NoError(t, err)

			script, err := os.ReadFile(launcherPath)
			require.NoError(t, err)
			if tt.wantEnv {
				assert.Contains(t, string(script), "export GOOSE_MODE=auto\n")
			} else {
				assert.NotContains(t, string(script), "GOOSE_MODE")
			}
			// The export must precede exec, or it never reaches the agent.
			assert.Contains(t, string(script), `exec '/usr/local/bin/goose' run -s -t "$prompt"`)
		})
	}
}

// TestHarnessesWithoutApprovalControlSkipThePrompt documents the harnesses where
// co-op deliberately shows no permission-mode choice.
func TestHarnessesWithoutApprovalControlSkipThePrompt(t *testing.T) {
	// Pi has no permission system at all, so there is nothing to auto-approve.
	assert.False(t, mustHarness(t, "pi").offersAutoApprove())
	// Goose controls approvals via the environment, which still counts.
	assert.True(t, mustHarness(t, "goose").offersAutoApprove())
	assert.True(t, mustHarness(t, "claude").offersAutoApprove())
}

// TestPermissionNoticeWarnsWhenNoGateExists guards against the missing
// permission-mode prompt reading as "the safe default applied". Every harness
// that offers no permission choice must either say why, or be a custom binary
// whose permission model co-op cannot know.
func TestPermissionNoticeWarnsWhenNoGateExists(t *testing.T) {
	for _, h := range supportedHarnesses {
		t.Run(h.id, func(t *testing.T) {
			if h.offersAutoApprove() {
				assert.Empty(t, h.permissionNotice(), "harness with a permission gate must not warn")
				return
			}
			assert.NotEmpty(t, h.permissionNotice(),
				"harness %q shows no permission prompt and no warning, so users cannot tell it is ungated", h.id)
		})
	}
}

// TestCustomHarnessDoesNotClaimToBeUngated separates "known to have no gate"
// from "co-op does not know", which must not produce the same warning.
func TestCustomHarnessDoesNotClaimToBeUngated(t *testing.T) {
	custom := customHarness("mycoder")

	assert.False(t, custom.offersAutoApprove())
	assert.Empty(t, custom.permissionNotice())
}

// TestBuildAgentCmdQuotesAgentPath keeps the shell-injection guard in place for
// the registry-driven argv construction.
func TestBuildAgentCmdQuotesAgentPath(t *testing.T) {
	promptPath := filepath.Join(t.TempDir(), "prompt.txt")
	require.NoError(t, os.WriteFile(promptPath, []byte("prompt"), 0o600))
	agent := &agentInfo{harness: mustHarness(t, "claude"), path: "/tmp/a$(touch pwned)/claude"}

	launcherPath, err := (&coopRunCmd{}).buildAgentCmd(agent, promptPath, false)
	require.NoError(t, err)

	script, err := os.ReadFile(launcherPath)
	require.NoError(t, err)
	assert.Contains(t, string(script), `exec '/tmp/a$(touch pwned)/claude' "$prompt"`)
}
