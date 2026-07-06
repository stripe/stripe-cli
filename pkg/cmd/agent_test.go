package cmd

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/stripe/stripe-cli/pkg/agentsetup"
)

func TestAgentSetupStatusDoesNotInstall(t *testing.T) {
	setup := newTestAgentSetupCmd(t, claudeMissingPluginScanner(t), func(context.Context, string, ...string) error {
		t.Fatal("installer should not run in --status mode")
		return nil
	})

	output, err := executeCommand(setup.cmd, "--status")

	require.NoError(t, err)
	require.Contains(t, output, "Detected agents with supported Stripe plugins:")
	require.Contains(t, output, "Claude Code")
	require.Contains(t, output, "Stripe plugin not installed")
}

func TestAgentSetupStatusShowsInstalledDetail(t *testing.T) {
	setup := newTestAgentSetupCmdInstalled(t, nil)

	output, err := executeCommand(setup.cmd, "--status")

	require.NoError(t, err)
	require.Contains(t, output, "Claude Code")
	require.Contains(t, output, "Stripe plugin installed")
	require.Contains(t, output, agentsetup.TargetClaudePlugin) // plugin id in the dimmed detail
	require.Contains(t, output, "2.4.1")
}

func TestAgentSetupUnsupportedClient(t *testing.T) {
	setup := newTestAgentSetupCmd(t, claudeMissingPluginScanner(t), nil)

	_, err := executeCommand(setup.cmd, "--client", "cursor")

	require.Error(t, err)
	require.Contains(t, err.Error(), "unsupported agent client")
}

func TestAgentSetupJSONReportsActionWithoutInstalling(t *testing.T) {
	setup := newTestAgentSetupCmd(t, claudeMissingPluginScanner(t), func(context.Context, string, ...string) error {
		t.Fatal("installer should not run in --json mode")
		return nil
	})

	output, err := executeCommand(setup.cmd, "--json")

	require.NoError(t, err)

	var result agentSetupJSON
	require.NoError(t, json.Unmarshal([]byte(output), &result))
	require.Equal(t, agentsetup.StatusMissing, result.Status)
	require.Len(t, result.Clients, 1)
	require.True(t, result.Clients[0].Detected)
	require.False(t, result.Clients[0].Plugin.Installed)
	require.Len(t, result.Actions, 1)
	require.Equal(t, agentsetup.ActionInstall, result.Actions[0].Action)
	require.Equal(t, []string{"claude", "plugin", "install", agentsetup.TargetClaudePlugin}, result.Actions[0].Command)
}

func TestAgentSetupJSONShowsUpgradeHintWhenPluginCommandFails(t *testing.T) {
	setup := newAgentSetupCmd()
	claude := agentsetup.NewClaudeProvider(agentsetup.Scanner{
		LookPath: func(string) (string, error) { return "/usr/local/bin/claude", nil },
	}, nil)
	claude.RunOutput = func(_ context.Context, _ string, _ ...string) ([]byte, error) {
		return nil, errors.New("unknown command")
	}
	setup.providers = map[string]agentsetup.Provider{claude.ID(): claude}
	setup.callingAgent = func() string { return "" }
	setup.cmd.SetContext(context.Background())

	output, err := executeCommand(setup.cmd, "--json")

	require.NoError(t, err)

	var result agentSetupJSON
	require.NoError(t, json.Unmarshal([]byte(output), &result))
	require.Len(t, result.Clients, 1)
	require.True(t, result.Clients[0].Detected)
	require.Contains(t, result.Clients[0].Error, "upgrade Claude Code")
}

func TestAgentSetupForceYesInvokesInstallerWhenInstalled(t *testing.T) {
	var called bool
	setup := newTestAgentSetupCmdInstalled(t, func(ctx context.Context, name string, args ...string) error {
		called = true
		require.Equal(t, "claude", name)
		require.Equal(t, []string{"plugin", "install", agentsetup.TargetClaudePlugin}, args)
		return nil
	})

	output, err := executeCommand(setup.cmd, "--force", "--yes")

	require.NoError(t, err)
	require.True(t, called)
	require.Contains(t, output, "Setting up Stripe agent tooling")
	require.Contains(t, output, "Claude Code")
	require.Contains(t, output, "done")
	require.Contains(t, output, "1 installed, 0 skipped, 0 errors")
}

func TestAgentSetupRetriesAfterMarketplaceUpdate(t *testing.T) {
	var calls []string
	setup := newTestAgentSetupCmd(t, claudeMissingPluginScanner(t), func(ctx context.Context, name string, args ...string) error {
		call := name
		for _, arg := range args {
			call += " " + arg
		}
		calls = append(calls, call)
		if len(calls) == 1 {
			return fmt.Errorf("plugin not found")
		}
		return nil
	})

	output, err := executeCommand(setup.cmd, "--yes")

	require.NoError(t, err)
	require.Equal(t, []string{
		"claude plugin install " + agentsetup.TargetClaudePlugin,
		"claude plugin marketplace update " + agentsetup.ClaudeMarketplace,
		"claude plugin install " + agentsetup.TargetClaudePlugin,
	}, calls)
	require.Contains(t, output, "Updating Claude plugin marketplace and retrying")
	require.Contains(t, output, "done")
	require.Contains(t, output, "1 installed, 0 skipped, 0 errors")
}

func TestAgentSetupNoClaudeDoesNotFail(t *testing.T) {
	setup := newTestAgentSetupCmd(t, agentsetup.Scanner{
		LookPath: func(string) (string, error) { return "", errors.New("not found") },
	}, func(context.Context, string, ...string) error {
		t.Fatal("installer should not run when Claude Code is not detected")
		return nil
	})

	output, err := executeCommand(setup.cmd)

	require.NoError(t, err)
	require.Contains(t, output, "No supported AI coding clients detected on this machine.")
	require.Contains(t, output, "re-run: stripe agent setup")
}

func TestAgentSetupInstallsAllDetectedClients(t *testing.T) {
	var installed []string
	record := func(_ context.Context, name string, args ...string) error {
		installed = append(installed, name)
		return nil
	}

	claude := agentsetup.NewClaudeProvider(claudeMissingPluginScanner(t), record)
	codex := codexMissingProvider(record)

	setup := newAgentSetupCmd()
	setup.providers = map[string]agentsetup.Provider{claude.ID(): claude, codex.ID(): codex}
	setup.callingAgent = func() string { return "" } // no agent -> install all detected
	setup.cmd.SetContext(context.Background())

	// executeCommand is non-interactive, so all detected clients install without a TUI.
	output, err := executeCommand(setup.cmd)

	require.NoError(t, err)
	require.ElementsMatch(t, []string{"claude", "codex"}, installed)
	require.Contains(t, output, "Claude Code")
	require.Contains(t, output, "Codex CLI")
	require.Contains(t, output, "2 installed, 0 skipped, 0 errors")
}

func TestAgentSetupClientFlagLimitsToOne(t *testing.T) {
	var installed []string
	record := func(_ context.Context, name string, args ...string) error {
		installed = append(installed, name)
		return nil
	}

	claude := agentsetup.NewClaudeProvider(claudeMissingPluginScanner(t), record)
	codex := codexMissingProvider(record)

	setup := newAgentSetupCmd()
	setup.providers = map[string]agentsetup.Provider{claude.ID(): claude, codex.ID(): codex}
	setup.callingAgent = func() string { return "" }
	setup.cmd.SetContext(context.Background())

	output, err := executeCommand(setup.cmd, "--client", "codex")

	require.NoError(t, err)
	require.Equal(t, []string{"codex"}, installed)
	require.Contains(t, output, "1 installed, 0 skipped, 0 errors")
}

func TestAgentSetupAutoInstallsForCallingAgent(t *testing.T) {
	var installed []string
	record := func(_ context.Context, name string, args ...string) error {
		installed = append(installed, name)
		return nil
	}

	claude := agentsetup.NewClaudeProvider(claudeMissingPluginScanner(t), record)
	codex := codexMissingProvider(record)

	setup := newAgentSetupCmd()
	setup.providers = map[string]agentsetup.Provider{claude.ID(): claude, codex.ID(): codex}
	// Simulate being invoked by Codex CLI — only its plugin should install,
	// even though Claude is also detected, and with no --client flag.
	setup.callingAgent = func() string { return "codex_cli" }
	setup.cmd.SetContext(context.Background())

	output, err := executeCommand(setup.cmd)

	require.NoError(t, err)
	require.Equal(t, []string{"codex"}, installed) // Claude NOT installed
	require.Contains(t, output, "Detected Codex CLI — setting up its Stripe plugin.")
	require.Contains(t, output, "1 installed, 0 skipped, 0 errors")
}

// codexMissingProvider returns a Codex provider that detects the binary, starts
// with the Stripe plugin not installed, records install commands via record, and
// reports the plugin as installed once the add command has run (mirroring the
// post-install verification the provider performs).
func codexMissingProvider(record agentsetup.RunCommandFunc) agentsetup.CodexProvider {
	installed := false
	return agentsetup.CodexProvider{
		Scanner: agentsetup.Scanner{LookPath: func(string) (string, error) { return "/usr/local/bin/codex", nil }},
		RunCommand: func(ctx context.Context, name string, args ...string) error {
			installed = true
			if record != nil {
				return record(ctx, name, args...)
			}
			return nil
		},
		RunOutput: func(context.Context, string, ...string) ([]byte, error) {
			if installed {
				return []byte(`{"installed":[{"pluginId":"stripe@openai-curated","name":"stripe","marketplaceName":"openai-curated","version":"1.0.0"}]}`), nil
			}
			return []byte(`{"installed":[]}`), nil
		},
	}
}

func TestAgentSetupUnsupportedAgentInstallsSkillsToLocal(t *testing.T) {
	var gotDir string
	claude := agentsetup.NewClaudeProvider(claudeMissingPluginScanner(t), func(context.Context, string, ...string) error {
		t.Fatal("no plugin should be installed for an unsupported agent")
		return nil
	})

	setup := newAgentSetupCmd()
	setup.providers = map[string]agentsetup.Provider{claude.ID(): claude}
	setup.callingAgent = func() string { return "gemini_cli" }
	localDir := filepath.Join(t.TempDir(), ".agents", "skills")
	setup.skillsInstall = func(_ context.Context, destDir string) ([]string, error) {
		gotDir = destDir
		return []string{"stripe-best-practices"}, nil
	}
	setup.skillsLocalDir = func() (string, error) { return localDir, nil }
	setup.cmd.SetContext(context.Background())

	output, err := executeCommand(setup.cmd)

	require.NoError(t, err)
	require.Equal(t, localDir, gotDir)
	require.Contains(t, output, "Stripe skills (local)")
	require.Contains(t, output, "1 installed, 0 skipped, 0 errors")
}

func TestAgentSetupCursorIsSkippedNotInstalled(t *testing.T) {
	// Cursor detected but plugin not installed — shows manual step hint.
	cursor := agentsetup.NewCursorProvider(agentsetup.Scanner{
		LookPath: func(string) (string, error) { return "/usr/local/bin/cursor", nil },
	}, nil)

	setup := newAgentSetupCmd()
	setup.providers = map[string]agentsetup.Provider{cursor.ID(): cursor}
	setup.callingAgent = func() string { return "" }
	setup.cmd.SetContext(context.Background())

	output, err := executeCommand(setup.cmd, "--client", "cursor", "--yes")

	require.NoError(t, err)
	require.Contains(t, output, "manual step")
	require.Contains(t, output, "/add-plugin stripe")
	require.Contains(t, output, "0 installed, 1 skipped, 0 errors")
}


func TestAgentSetupStatusHidesUndetectedClients(t *testing.T) {
	// Claude detected, Cursor not — --status should list only Claude.
	claude := agentsetup.NewClaudeProvider(claudeMissingPluginScanner(t), nil)
	cursor := agentsetup.NewCursorProvider(agentsetup.Scanner{
		LookPath: func(string) (string, error) { return "", errors.New("not found") },
	}, nil)

	setup := newAgentSetupCmd()
	setup.providers = map[string]agentsetup.Provider{claude.ID(): claude, cursor.ID(): cursor}
	setup.callingAgent = func() string { return "" }
	setup.cmd.SetContext(context.Background())

	output, err := executeCommand(setup.cmd, "--status")

	require.NoError(t, err)
	require.Contains(t, output, "Claude Code")
	require.NotContains(t, output, "Cursor")
	require.NotContains(t, output, "not detected")
}

func TestAgentSetupNoClientsNonInteractiveShowsHint(t *testing.T) {
	// No clients detected, non-interactive: show info message.
	setup := newAgentSetupCmd()
	setup.providers = map[string]agentsetup.Provider{} // nothing detected
	setup.callingAgent = func() string { return "" }
	setup.isInteractive = func() bool { return false }
	setup.cmd.SetContext(context.Background())

	output, err := executeCommand(setup.cmd)

	require.NoError(t, err)
	require.Contains(t, output, "No supported AI coding clients detected on this machine.")
	require.Contains(t, output, "Supported clients for automatic setup:")
	require.Contains(t, output, "Once a client is installed, re-run: stripe agent setup")
}

func TestAgentSetupAgentScopingWinsOverYes(t *testing.T) {
	// Inside a coding agent, --yes must NOT broaden to all clients — it still
	// only sets up the calling agent.
	var installed []string
	record := func(_ context.Context, name string, args ...string) error {
		installed = append(installed, name)
		return nil
	}
	claude := agentsetup.NewClaudeProvider(claudeMissingPluginScanner(t), func(context.Context, string, ...string) error {
		t.Fatal("Claude must not be installed when the calling agent is Codex")
		return nil
	})
	codex := codexMissingProvider(record)

	setup := newAgentSetupCmd()
	setup.providers = map[string]agentsetup.Provider{claude.ID(): claude, codex.ID(): codex}
	setup.callingAgent = func() string { return "codex_cli" }
	setup.cmd.SetContext(context.Background())

	output, err := executeCommand(setup.cmd, "--yes")

	require.NoError(t, err)
	require.Equal(t, []string{"codex"}, installed) // only codex, despite --yes and Claude detected
	require.Contains(t, output, "Detected Codex CLI — setting up its Stripe plugin.")
	require.Contains(t, output, "1 installed, 0 skipped, 0 errors")
}

func TestAgentSetupYesInstallsAllWhenNoAgent(t *testing.T) {
	// In a plain CLI (no calling agent), --yes still installs every detected client.
	var installed []string
	record := func(_ context.Context, name string, args ...string) error {
		installed = append(installed, name)
		return nil
	}
	claude := agentsetup.NewClaudeProvider(claudeMissingPluginScanner(t), record)
	codex := codexMissingProvider(record)

	setup := newAgentSetupCmd()
	setup.providers = map[string]agentsetup.Provider{claude.ID(): claude, codex.ID(): codex}
	setup.callingAgent = func() string { return "" }
	setup.cmd.SetContext(context.Background())

	_, err := executeCommand(setup.cmd, "--yes")

	require.NoError(t, err)
	require.ElementsMatch(t, []string{"claude", "codex"}, installed)
}

func TestAgentSetupAgentNeverUsesInteractivePicker(t *testing.T) {
	// Gemini CLI allocates a PTY, so isInteractive() reports true even though no
	// human is present. When an agent is detected we must skip the picker and
	// fall back to non-interactive behavior (skills for an unsupported agent).
	var skillsCalled bool
	claude := agentsetup.NewClaudeProvider(claudeMissingPluginScanner(t), func(context.Context, string, ...string) error {
		t.Fatal("no plugin should install; and the picker must not run")
		return nil
	})

	setup := newAgentSetupCmd()
	setup.providers = map[string]agentsetup.Provider{claude.ID(): claude}
	setup.callingAgent = func() string { return "gemini_cli" }
	setup.isInteractive = func() bool { return true } // simulate a PTY
	setup.skillsInstall = func(context.Context, string) ([]string, error) {
		skillsCalled = true
		return []string{"stripe-best-practices"}, nil
	}
	setup.skillsLocalDir = func() (string, error) { return filepath.Join(t.TempDir(), ".agents", "skills"), nil }
	setup.cmd.SetContext(context.Background())

	output, err := executeCommand(setup.cmd)

	require.NoError(t, err)
	require.True(t, skillsCalled)
	require.Contains(t, output, "installing Stripe skills instead")
	require.Contains(t, output, "1 installed, 0 skipped, 0 errors")
}

func newTestAgentSetupCmd(t *testing.T, scanner agentsetup.Scanner, runInstall agentsetup.RunCommandFunc) *agentSetupCmd {
	t.Helper()
	setup := newAgentSetupCmd()
	claude := agentsetup.NewClaudeProvider(scanner, runInstall)
	claude.RunOutput = claudeListEmpty
	setup.providers = map[string]agentsetup.Provider{claude.ID(): claude}
	setup.callingAgent = func() string { return "" }
	setup.cmd.SetContext(context.Background())
	return setup
}

func newTestAgentSetupCmdInstalled(t *testing.T, runInstall agentsetup.RunCommandFunc) *agentSetupCmd {
	t.Helper()
	setup := newAgentSetupCmd()
	claude := agentsetup.NewClaudeProvider(agentsetup.Scanner{
		LookPath: func(string) (string, error) { return "/usr/local/bin/claude", nil },
	}, runInstall)
	claude.RunOutput = claudeListInstalled
	setup.providers = map[string]agentsetup.Provider{claude.ID(): claude}
	setup.callingAgent = func() string { return "" }
	setup.cmd.SetContext(context.Background())
	return setup
}

func claudeMissingPluginScanner(t *testing.T) agentsetup.Scanner {
	t.Helper()
	return agentsetup.Scanner{
		LookPath: func(string) (string, error) { return "/usr/local/bin/claude", nil },
	}
}

func claudeListEmpty(_ context.Context, _ string, _ ...string) ([]byte, error) {
	return []byte(`[]`), nil
}

func claudeListInstalled(_ context.Context, _ string, _ ...string) ([]byte, error) {
	return []byte(`[{"id":"stripe@claude-plugins-official","version":"2.4.1","scope":"user","enabled":true}]`), nil
}
