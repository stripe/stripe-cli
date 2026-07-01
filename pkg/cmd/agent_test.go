package cmd

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
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
	require.Contains(t, output, "AI coding clients")
	require.Contains(t, output, "Claude Code")
	require.Contains(t, output, "Stripe plugin not installed")
}

func TestAgentSetupStatusShowsInstalledDetail(t *testing.T) {
	setup := newTestAgentSetupCmd(t, claudeInstalledPluginScanner(t), nil)

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

func TestAgentSetupJSONPropagatesScanError(t *testing.T) {
	setup := newTestAgentSetupCmd(t, agentsetup.Scanner{
		LookPath: func(string) (string, error) { return "/usr/local/bin/claude", nil },
		HomeDir:  func() (string, error) { return "", errors.New("home failed") },
	}, func(context.Context, string, ...string) error {
		t.Fatal("installer should not run when scan fails")
		return nil
	})

	output, err := executeCommand(setup.cmd, "--json")

	require.Error(t, err)

	var result agentSetupJSON
	require.NoError(t, json.Unmarshal([]byte(output), &result))
	require.Equal(t, agentsetup.StatusError, result.Status)
	require.Len(t, result.Errors, 1)
	require.Contains(t, result.Errors[0], "home failed")
}

func TestAgentSetupForceYesInvokesInstallerWhenInstalled(t *testing.T) {
	var called bool
	setup := newTestAgentSetupCmd(t, claudeInstalledPluginScanner(t), func(ctx context.Context, name string, args ...string) error {
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
	require.Contains(t, output, "No AI coding clients detected on this machine.")
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

func TestAgentSetupSkillsFlagInstallsToLocalDir(t *testing.T) {
	var gotDir string
	setup := newAgentSetupCmd()
	// No agents registered, so only the skills path runs.
	setup.providers = map[string]agentsetup.Provider{}
	setup.cmd.SetContext(context.Background())
	setup.skillsInstall = func(_ context.Context, destDir string) ([]string, error) {
		gotDir = destDir
		return []string{"stripe-best-practices", "upgrade-stripe"}, nil
	}
	localDir := filepath.Join(t.TempDir(), ".agents", "skills")
	setup.skillsLocalDir = func() (string, error) { return localDir, nil }

	output, err := executeCommand(setup.cmd, "--skills")

	require.NoError(t, err)
	require.Equal(t, localDir, gotDir)
	require.Contains(t, output, "Stripe skills (local)")
	require.Contains(t, output, "installed 2 skill(s)")
	require.Contains(t, output, "1 installed, 0 skipped, 0 errors")
}

func TestAgentSetupSkillsGlobalScope(t *testing.T) {
	var gotDir string
	globalDir := filepath.Join(t.TempDir(), ".agents", "skills")

	setup := newAgentSetupCmd()
	setup.providers = map[string]agentsetup.Provider{}
	setup.cmd.SetContext(context.Background())
	setup.skillsInstall = func(_ context.Context, destDir string) ([]string, error) {
		gotDir = destDir
		return []string{"stripe-best-practices"}, nil
	}
	setup.skillsGlobalDir = func() (string, error) { return globalDir, nil }

	output, err := executeCommand(setup.cmd, "--skills", "--skills-scope", "global")

	require.NoError(t, err)
	require.Equal(t, globalDir, gotDir)
	require.Contains(t, output, "Stripe skills (global)")
	require.Contains(t, output, "1 installed, 0 skipped, 0 errors")
}

func TestAgentSetupSkillsInstallFailureReportsError(t *testing.T) {
	setup := newAgentSetupCmd()
	setup.providers = map[string]agentsetup.Provider{}
	setup.cmd.SetContext(context.Background())
	setup.skillsInstall = func(context.Context, string) ([]string, error) {
		return nil, errors.New("index unreachable")
	}
	setup.skillsLocalDir = func() (string, error) { return t.TempDir(), nil }

	output, err := executeCommand(setup.cmd, "--skills")

	require.Error(t, err)
	require.Contains(t, output, "error: index unreachable")
	require.Contains(t, output, "0 installed, 0 skipped, 1 errors")
}

func TestAgentSetupInvalidSkillsScope(t *testing.T) {
	setup := newAgentSetupCmd()
	setup.cmd.SetContext(context.Background())

	_, err := executeCommand(setup.cmd, "--skills", "--skills-scope", "sideways")

	require.Error(t, err)
	require.Contains(t, err.Error(), "invalid --skills-scope")
}

func TestAgentSetupSkillsAlongsideAgent(t *testing.T) {
	var skillsCalled bool
	claude := agentsetup.NewClaudeProvider(claudeMissingPluginScanner(t), func(context.Context, string, ...string) error { return nil })

	setup := newAgentSetupCmd()
	setup.providers = map[string]agentsetup.Provider{claude.ID(): claude}
	setup.cmd.SetContext(context.Background())
	setup.skillsInstall = func(context.Context, string) ([]string, error) {
		skillsCalled = true
		return []string{"stripe-best-practices"}, nil
	}
	setup.skillsLocalDir = func() (string, error) { return filepath.Join(t.TempDir(), ".agents", "skills"), nil }

	// --yes installs the detected agent; --skills adds skills in the same run.
	output, err := executeCommand(setup.cmd, "--yes", "--skills")

	require.NoError(t, err)
	require.True(t, skillsCalled)
	require.Contains(t, output, "Claude Code")
	require.Contains(t, output, "Stripe skills (local)")
	require.Contains(t, output, "2 installed, 0 skipped, 0 errors")
}

func TestAgentSetupCursorIsManualStepNotError(t *testing.T) {
	// Cursor detected, plugin not installed (empty temp home) -> manual step.
	cursor := agentsetup.NewCursorProvider(agentsetup.Scanner{
		LookPath: func(string) (string, error) { return "/usr/local/bin/cursor", nil },
		ReadFile: os.ReadFile,
		HomeDir:  func() (string, error) { return t.TempDir(), nil },
	}, nil)

	setup := newAgentSetupCmd()
	setup.providers = map[string]agentsetup.Provider{cursor.ID(): cursor}
	setup.cmd.SetContext(context.Background())

	output, err := executeCommand(setup.cmd, "--client", "cursor", "--yes")

	require.NoError(t, err) // manual step must not fail the command
	require.Contains(t, output, "manual step: run /add-plugin stripe inside Cursor")
	require.Contains(t, output, "0 installed, 1 skipped, 0 errors")
}

func TestAgentSetupUnsupportedAgentInstallsSkills(t *testing.T) {
	var skillsCalled bool
	// Claude is detected, but we're invoked by an agent with no Stripe plugin.
	claude := agentsetup.NewClaudeProvider(claudeMissingPluginScanner(t), func(context.Context, string, ...string) error {
		t.Fatal("no plugin should be installed for an unsupported agent")
		return nil
	})

	setup := newAgentSetupCmd()
	setup.providers = map[string]agentsetup.Provider{claude.ID(): claude}
	setup.callingAgent = func() string { return "gemini_cli" }
	setup.skillsInstall = func(context.Context, string) ([]string, error) {
		skillsCalled = true
		return []string{"stripe-best-practices"}, nil
	}
	setup.skillsLocalDir = func() (string, error) { return filepath.Join(t.TempDir(), ".agents", "skills"), nil }
	setup.cmd.SetContext(context.Background())

	output, err := executeCommand(setup.cmd)

	require.NoError(t, err)
	require.True(t, skillsCalled)
	require.Contains(t, output, "Detected gemini_cli, which has no Stripe plugin — installing Stripe skills instead.")
	require.Contains(t, output, "Stripe skills (local)")
	require.Contains(t, output, "1 installed, 0 skipped, 0 errors")
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
	// No clients detected, non-interactive: show the full info message with the
	// --skills hint so scripts/CI know what to do.
	setup := newAgentSetupCmd()
	setup.providers = map[string]agentsetup.Provider{} // nothing detected
	setup.callingAgent = func() string { return "" }
	setup.isInteractive = func() bool { return false }
	setup.cmd.SetContext(context.Background())

	output, err := executeCommand(setup.cmd)

	require.NoError(t, err)
	require.Contains(t, output, "No AI coding clients detected on this machine.")
	require.Contains(t, output, "Supported clients for automatic setup:")
	require.Contains(t, output, "Once a client is installed, re-run: stripe agent setup")
	require.Contains(t, output, "stripe agent setup --skills")
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
	setup.providers = map[string]agentsetup.Provider{claude.ID(): claude}
	setup.callingAgent = func() string { return "" }
	setup.cmd.SetContext(context.Background())
	return setup
}

func claudeMissingPluginScanner(t *testing.T) agentsetup.Scanner {
	t.Helper()
	home := t.TempDir()
	return agentsetup.Scanner{
		LookPath: func(string) (string, error) { return "/usr/local/bin/claude", nil },
		ReadFile: os.ReadFile,
		HomeDir:  func() (string, error) { return home, nil },
	}
}

func claudeInstalledPluginScanner(t *testing.T) agentsetup.Scanner {
	t.Helper()
	home := t.TempDir()
	statePath := filepath.Join(home, agentsetup.ClaudePluginStatePath)
	require.NoError(t, os.MkdirAll(filepath.Dir(statePath), 0755))
	require.NoError(t, os.WriteFile(statePath, []byte(`{
		"version": 2,
		"plugins": {
			"stripe@claude-plugins-official": [
				{"scope": "user", "version": "2.4.1", "installPath": "/tmp/stripe"}
			]
		}
	}`), 0600))
	return agentsetup.Scanner{
		LookPath: func(string) (string, error) { return "/usr/local/bin/claude", nil },
		ReadFile: os.ReadFile,
		HomeDir:  func() (string, error) { return home, nil },
	}
}
