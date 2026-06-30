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
	require.Contains(t, output, "Claude Code: detected")
	require.Contains(t, output, "Stripe plugin: missing")
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
	require.True(t, result.Detected)
	require.False(t, result.PluginInstalled)
	require.Equal(t, "install", result.Action)
	require.Equal(t, []string{"claude", "plugin", "install", agentsetup.TargetClaudePlugin}, result.Command)
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
	require.Contains(t, output, "Installing Stripe agent tooling")
	require.Contains(t, output, "Installed Stripe agent tooling")
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
	require.Contains(t, output, "Installed Stripe agent tooling")
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
	require.Contains(t, output, "Claude Code: not detected")
	require.Contains(t, output, "Nothing to do")
}

func newTestAgentSetupCmd(t *testing.T, scanner agentsetup.Scanner, runInstall agentsetup.RunCommandFunc) *agentSetupCmd {
	t.Helper()
	setup := newAgentSetupCmd()
	setup.scanner = scanner
	if runInstall != nil {
		setup.runInstall = runInstall
	}
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
