package cmd

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/stripe/stripe-cli/pkg/agentsetup"
	"github.com/stripe/stripe-cli/pkg/stripe"
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

func TestAgentSetupTelemetryDetectsAllSupportedClientsEvenWhenFiltered(t *testing.T) {
	telemetryClient := &recordingTelemetryClient{}
	ctx := telemetryContext(telemetryClient)

	claude := agentsetup.NewClaudeProvider(claudeMissingPluginScanner(t), nil)
	claude.RunOutput = claudeListEmpty
	codex := codexMissingProvider(nil)

	setup := newAgentSetupCmd()
	setup.providers = map[string]agentsetup.Provider{claude.ID(): claude, codex.ID(): codex}
	setup.callingAgent = func() string { return "" }
	setup.cmd.SetContext(ctx)

	_, err := executeCommand(setup.cmd, "--client", "codex", "--status")

	require.NoError(t, err)
	telemetryClient.waitForEventCount(t, 2)
	require.ElementsMatch(t, []telemetryEvent{
		{name: eventAgentSetupDetected, value: agentsetup.ClientClaudeCode},
		{name: eventAgentSetupDetected, value: agentsetup.ClientCodex},
	}, telemetryClient.snapshot())
}

func TestAgentSetupTelemetrySelectionCancel(t *testing.T) {
	telemetryClient := &recordingTelemetryClient{}

	setup := newTestAgentSetupCmd(t, claudeMissingPluginScanner(t), nil)
	setup.isInteractive = func() bool { return true }
	setup.runSelectionTUI = func([]agentsetup.Status) (*Selection, error) { return nil, nil }
	setup.cmd.SetContext(telemetryContext(telemetryClient))

	output, err := executeCommand(setup.cmd)

	require.NoError(t, err)
	require.Contains(t, output, "Canceled. No changes made.")
	telemetryClient.waitForEventCount(t, 2)
	require.ElementsMatch(t, []telemetryEvent{
		{name: eventAgentSetupDetected, value: agentsetup.ClientClaudeCode},
		{name: eventAgentSetupTUI, value: agentSetupTUISelectionCanceled},
	}, telemetryClient.snapshot())
}

func TestAgentSetupTelemetryScopeCancelWhenNoClientsDetected(t *testing.T) {
	telemetryClient := &recordingTelemetryClient{}

	setup := newAgentSetupCmd()
	setup.providers = map[string]agentsetup.Provider{}
	setup.callingAgent = func() string { return "" }
	setup.isInteractive = func() bool { return true }
	setup.runSkillsScopeTUI = func() (string, bool, error) { return "", false, nil }
	setup.cmd.SetContext(telemetryContext(telemetryClient))

	output, err := executeCommand(setup.cmd)

	require.NoError(t, err)
	require.Contains(t, output, "Canceled. No changes made.")
	telemetryClient.waitForEventCount(t, 2)
	require.ElementsMatch(t, []telemetryEvent{
		{name: eventAgentSetupDetected, value: "none"},
		{name: eventAgentSetupTUI, value: agentSetupTUIScopeCanceled},
	}, telemetryClient.snapshot())
}

func TestAgentSetupTelemetryNoSelection(t *testing.T) {
	telemetryClient := &recordingTelemetryClient{}

	setup := newTestAgentSetupCmd(t, claudeMissingPluginScanner(t), nil)
	setup.isInteractive = func() bool { return true }
	setup.runSelectionTUI = func([]agentsetup.Status) (*Selection, error) {
		return &Selection{}, nil
	}
	setup.cmd.SetContext(telemetryContext(telemetryClient))

	output, err := executeCommand(setup.cmd)

	require.NoError(t, err)
	require.Contains(t, output, "Nothing selected. No changes made.")
	telemetryClient.waitForEventCount(t, 2)
	require.ElementsMatch(t, []telemetryEvent{
		{name: eventAgentSetupDetected, value: agentsetup.ClientClaudeCode},
		{name: eventAgentSetupTUI, value: agentSetupTUINoSelection},
	}, telemetryClient.snapshot())
}

func TestAgentSetupTelemetryConfirmedSelectionAndSkills(t *testing.T) {
	telemetryClient := &recordingTelemetryClient{}

	setup := newTestAgentSetupCmd(t, claudeMissingPluginScanner(t), func(context.Context, string, ...string) error { return nil })
	setup.isInteractive = func() bool { return true }
	setup.runSelectionTUI = func(statuses []agentsetup.Status) (*Selection, error) {
		return &Selection{Agents: statuses, InstallSkills: true}, nil
	}
	setup.runSkillsScopeTUI = func() (string, bool, error) { return skillsScopeGlobal, true, nil }
	setup.skillsInstall = func(context.Context, string) ([]string, error) {
		return []string{"stripe-best-practices", "upgrade-stripe"}, nil
	}
	setup.skillsGlobalDir = func() (string, error) { return filepath.Join(t.TempDir(), ".agents", "skills"), nil }
	setup.cmd.SetContext(telemetryContext(telemetryClient))

	_, err := executeCommand(setup.cmd)

	require.NoError(t, err)
	telemetryClient.waitForEventCount(t, 7)
	require.ElementsMatch(t, []telemetryEvent{
		{name: eventAgentSetupDetected, value: agentsetup.ClientClaudeCode},
		{name: eventAgentSetupTUI, value: agentSetupTUIConfirmed},
		{name: eventAgentSetupSelected, value: "client:" + agentsetup.ClientClaudeCode},
		{name: eventAgentSetupSelected, value: "skills:" + skillsScopeGlobal},
		{name: eventAgentSetupClientResult, value: agentsetup.ClientClaudeCode + ":" + agentSetupResultSuccess},
		{name: eventAgentSetupSkillInstalled, value: skillsScopeGlobal + ":stripe-best-practices"},
		{name: eventAgentSetupSkillInstalled, value: skillsScopeGlobal + ":upgrade-stripe"},
	}, telemetryClient.snapshot())
}

func TestAgentSetupTelemetryClientResultStates(t *testing.T) {
	telemetryClient := &recordingTelemetryClient{}

	skipped := testAgentProvider{
		id: "skipped",
		status: agentsetup.Status{
			Client:      "skipped",
			DisplayName: "Skipped",
			Detected:    true,
			Status:      agentsetup.StatusInstalled,
		},
		plan: agentsetup.Plan{Action: agentsetup.ActionNone},
	}
	manual := testAgentProvider{
		id: "manual",
		status: agentsetup.Status{
			Client:      "manual",
			DisplayName: "Manual",
			Detected:    true,
			Status:      agentsetup.StatusMissing,
		},
		plan: agentsetup.Plan{Action: agentsetup.ActionManual, Manual: "manual step"},
	}
	failed := testAgentProvider{
		id: "failed",
		status: agentsetup.Status{
			Client:      "failed",
			DisplayName: "Failed",
			Detected:    true,
			Status:      agentsetup.StatusMissing,
		},
		plan:     agentsetup.Plan{Action: agentsetup.ActionInstall, Command: []string{"failed", "install"}},
		applyErr: errors.New("boom"),
	}

	setup := newAgentSetupCmd()
	setup.providers = map[string]agentsetup.Provider{
		skipped.ID(): skipped,
		manual.ID():  manual,
		failed.ID():  failed,
	}
	setup.callingAgent = func() string { return "" }
	setup.cmd.SetContext(telemetryContext(telemetryClient))

	_, err := executeCommand(setup.cmd, "--yes")

	require.Error(t, err)
	telemetryClient.waitForEventCount(t, 6)
	require.ElementsMatch(t, []telemetryEvent{
		{name: eventAgentSetupDetected, value: "skipped"},
		{name: eventAgentSetupDetected, value: "manual"},
		{name: eventAgentSetupDetected, value: "failed"},
		{name: eventAgentSetupClientResult, value: "skipped:" + agentSetupResultSkipped},
		{name: eventAgentSetupClientResult, value: "manual:" + agentSetupResultManual},
		{name: eventAgentSetupClientResult, value: "failed:" + agentSetupResultFailed},
	}, telemetryClient.snapshot())
}

func TestAgentSetupTelemetrySkillsFailure(t *testing.T) {
	telemetryClient := &recordingTelemetryClient{}

	setup := newAgentSetupCmd()
	setup.providers = map[string]agentsetup.Provider{}
	setup.skillsInstall = func(context.Context, string) ([]string, error) {
		return nil, errors.New("index unreachable")
	}
	setup.skillsLocalDir = func() (string, error) { return t.TempDir(), nil }
	setup.cmd.SetContext(telemetryContext(telemetryClient))

	_, err := executeCommand(setup.cmd, "--skills")

	require.Error(t, err)
	telemetryClient.waitForEventCount(t, 2)
	require.ElementsMatch(t, []telemetryEvent{
		{name: eventAgentSetupDetected, value: "none"},
		{name: eventAgentSetupSkillsResult, value: skillsScopeLocal + ":" + agentSetupResultFailed},
	}, telemetryClient.snapshot())
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
	require.Contains(t, output, "No supported AI coding clients detected on this machine.")
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

type telemetryEvent struct {
	name  string
	value string
}

type recordingTelemetryClient struct {
	mu     sync.Mutex
	events []telemetryEvent
}

func (c *recordingTelemetryClient) SendAPIRequestEvent(context.Context, string, bool) (*http.Response, error) {
	return nil, nil
}

func (c *recordingTelemetryClient) SendEvent(_ context.Context, eventName string, eventValue string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.events = append(c.events, telemetryEvent{name: eventName, value: eventValue})
}

func (c *recordingTelemetryClient) snapshot() []telemetryEvent {
	c.mu.Lock()
	defer c.mu.Unlock()

	events := make([]telemetryEvent, len(c.events))
	copy(events, c.events)
	return events
}

func (c *recordingTelemetryClient) waitForEventCount(t *testing.T, count int) {
	t.Helper()

	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if len(c.snapshot()) >= count {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}

	t.Fatalf("timed out waiting for %d telemetry events; saw %v", count, c.snapshot())
}

func telemetryContext(client stripe.TelemetryClient) context.Context {
	ctx := stripe.WithTelemetryClient(context.Background(), client)
	return stripe.WithEventMetadata(ctx, stripe.NewEventMetadata())
}

type testAgentProvider struct {
	id       string
	status   agentsetup.Status
	plan     agentsetup.Plan
	applyErr error
}

func (p testAgentProvider) ID() string {
	return p.id
}

func (p testAgentProvider) Detect() agentsetup.Status {
	return p.status
}

func (p testAgentProvider) Plan(agentsetup.Status, bool) agentsetup.Plan {
	return p.plan
}

func (p testAgentProvider) Apply(context.Context, io.Writer, agentsetup.Plan) error {
	return p.applyErr
}
