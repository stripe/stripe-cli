package agentsetup

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestClaude_NotDetected(t *testing.T) {
	scanner := Scanner{LookPath: func(string) (string, error) { return "", errors.New("missing") }}
	provider := NewClaudeProvider(scanner, nil)

	status := provider.Detect()

	require.Equal(t, ClientClaudeCode, status.Client)
	require.False(t, status.Detected)
	require.Equal(t, StatusNotDetected, status.Status)
	require.Equal(t, Plan{Action: ActionNone}, provider.Plan(status, false))
}

func TestClaude_DetectedNoPluginSupport(t *testing.T) {
	scanner := Scanner{LookPath: func(string) (string, error) { return "/usr/local/bin/claude", nil }}
	runOutput := func(_ context.Context, _ string, _ ...string) ([]byte, error) {
		return nil, errors.New("unknown command")
	}
	provider := NewClaudeProvider(scanner, nil).(ClaudeProvider)
	provider.RunOutput = runOutput

	status := provider.Detect()

	require.True(t, status.Detected)
	require.Equal(t, StatusMissing, status.Status)
	require.Equal(t, "upgrade Claude Code to enable plugin support", status.Error)
}

func TestClaude_DetectedPluginMissing(t *testing.T) {
	scanner := Scanner{LookPath: func(string) (string, error) { return "/usr/local/bin/claude", nil }}
	runOutput := func(_ context.Context, _ string, _ ...string) ([]byte, error) { return []byte(`[]`), nil }
	provider := NewClaudeProvider(scanner, nil).(ClaudeProvider)
	provider.RunOutput = runOutput

	status := provider.Detect()

	require.True(t, status.Detected)
	require.Equal(t, StatusMissing, status.Status)
	require.False(t, status.Plugin.Installed)
	require.Equal(t, Plan{Action: ActionInstall, Commands: [][]string{{"claude", "plugin", "install", "stripe@claude-plugins-official"}}}, provider.Plan(status, false))
}

func TestClaude_OfficialPluginInstalled(t *testing.T) {
	listJSON := mustJSON(t, []claudeInstalledPlugin{
		{ID: "stripe@claude-plugins-official", Version: "2.4.1", Scope: "user", Enabled: true},
	})
	scanner := Scanner{LookPath: func(string) (string, error) { return "/usr/local/bin/claude", nil }}
	runOutput := func(_ context.Context, _ string, _ ...string) ([]byte, error) { return listJSON, nil }
	provider := NewClaudeProvider(scanner, nil).(ClaudeProvider)
	provider.RunOutput = runOutput

	status := provider.Detect()

	require.Equal(t, StatusInstalled, status.Status)
	require.True(t, status.Plugin.Installed)
	require.Equal(t, TargetClaudePlugin, status.Plugin.ID)
	require.Equal(t, "2.4.1", status.Plugin.Version)
	require.Equal(t, "user", status.Plugin.Scope)
	require.Equal(t, Plan{Action: ActionNone}, provider.Plan(status, false))
	require.Equal(t, Plan{Action: ActionReinstall, Commands: [][]string{{"claude", "plugin", "install", "stripe@claude-plugins-official"}}}, provider.Plan(status, true))
}

func TestClaude_MalformedJSON(t *testing.T) {
	scanner := Scanner{LookPath: func(string) (string, error) { return "/usr/local/bin/claude", nil }}
	runOutput := func(_ context.Context, _ string, _ ...string) ([]byte, error) { return []byte(`{nope`), nil }
	provider := NewClaudeProvider(scanner, nil).(ClaudeProvider)
	provider.RunOutput = runOutput

	status := provider.Detect()

	require.Equal(t, StatusMissing, status.Status)
	require.False(t, status.Plugin.Installed)
}

func TestClaude_OtherPluginsIgnored(t *testing.T) {
	listJSON := mustJSON(t, []claudeInstalledPlugin{
		{ID: "other-plugin@marketplace", Version: "1.0.0", Scope: "user", Enabled: true},
	})
	scanner := Scanner{LookPath: func(string) (string, error) { return "/usr/local/bin/claude", nil }}
	runOutput := func(_ context.Context, _ string, _ ...string) ([]byte, error) { return listJSON, nil }
	provider := NewClaudeProvider(scanner, nil).(ClaudeProvider)
	provider.RunOutput = runOutput

	status := provider.Detect()

	require.Equal(t, StatusMissing, status.Status)
	require.False(t, status.Plugin.Installed)
	require.Equal(t, Plan{Action: ActionInstall, Commands: [][]string{{"claude", "plugin", "install", "stripe@claude-plugins-official"}}}, provider.Plan(status, false))
}

func TestClaudeApply_RetriesAfterMarketplaceRefresh(t *testing.T) {
	installErr := errors.New("stale marketplace")
	var calls [][]string
	runCommand := func(_ context.Context, name string, args ...string) error {
		call := append([]string{name}, args...)
		calls = append(calls, call)
		if len(calls) == 1 {
			return installErr
		}
		return nil
	}
	provider := NewClaudeProvider(Scanner{}, runCommand)
	plan := Plan{Action: ActionInstall, Commands: [][]string{{"claude", "plugin", "install", TargetClaudePlugin}}}

	err := provider.Apply(context.Background(), nil, plan)

	require.NoError(t, err)
	require.Equal(t, [][]string{
		{"claude", "plugin", "install", TargetClaudePlugin},
		{"claude", "plugin", "marketplace", "update", ClaudeMarketplace},
		{"claude", "plugin", "install", TargetClaudePlugin},
	}, calls)
}

func mustJSON(t *testing.T, v interface{}) []byte {
	t.Helper()
	b, err := json.Marshal(v)
	require.NoError(t, err)
	return b
}
