package agentsetup

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestClaude_NotDetected(t *testing.T) {
	provider := ClaudeProvider{
		Scanner: Scanner{LookPath: func(string) (string, error) { return "", errors.New("missing") }},
	}

	status := provider.Detect()

	require.Equal(t, ClientClaudeCode, status.Client)
	require.False(t, status.Detected)
	require.Equal(t, StatusNotDetected, status.Status)
}

func TestClaude_DetectedNoPluginSupport(t *testing.T) {
	provider := ClaudeProvider{
		Scanner: Scanner{LookPath: func(string) (string, error) { return "/usr/local/bin/claude", nil }},
		RunOutput: func(_ context.Context, _ string, _ ...string) ([]byte, error) {
			return nil, errors.New("unknown command")
		},
	}

	status := provider.Detect()

	require.True(t, status.Detected)
	require.Equal(t, StatusMissing, status.Status)
	require.Equal(t, "upgrade Claude Code to enable plugin support", status.Error)
}

func TestClaude_DetectedPluginMissing(t *testing.T) {
	provider := ClaudeProvider{
		Scanner:   Scanner{LookPath: func(string) (string, error) { return "/usr/local/bin/claude", nil }},
		RunOutput: func(_ context.Context, _ string, _ ...string) ([]byte, error) { return []byte(`[]`), nil },
	}

	status := provider.Detect()

	require.True(t, status.Detected)
	require.Equal(t, StatusMissing, status.Status)
	require.False(t, status.Plugin.Installed)
}

func TestClaude_OfficialPluginInstalled(t *testing.T) {
	listJSON := mustJSON(t, []claudeInstalledPlugin{
		{ID: "stripe@claude-plugins-official", Version: "2.4.1", Scope: "user", Enabled: true},
	})
	provider := ClaudeProvider{
		Scanner:   Scanner{LookPath: func(string) (string, error) { return "/usr/local/bin/claude", nil }},
		RunOutput: func(_ context.Context, _ string, _ ...string) ([]byte, error) { return listJSON, nil },
	}

	status := provider.Detect()

	require.Equal(t, StatusInstalled, status.Status)
	require.True(t, status.Plugin.Installed)
	require.Equal(t, TargetClaudePlugin, status.Plugin.ID)
	require.Equal(t, "2.4.1", status.Plugin.Version)
	require.Equal(t, "user", status.Plugin.Scope)
}

func TestClaude_MalformedJSON(t *testing.T) {
	provider := ClaudeProvider{
		Scanner:   Scanner{LookPath: func(string) (string, error) { return "/usr/local/bin/claude", nil }},
		RunOutput: func(_ context.Context, _ string, _ ...string) ([]byte, error) { return []byte(`{nope`), nil },
	}

	status := provider.Detect()

	require.Equal(t, StatusMissing, status.Status)
	require.False(t, status.Plugin.Installed)
}

func TestClaude_OtherPluginsIgnored(t *testing.T) {
	listJSON := mustJSON(t, []claudeInstalledPlugin{
		{ID: "other-plugin@marketplace", Version: "1.0.0", Scope: "user", Enabled: true},
	})
	provider := ClaudeProvider{
		Scanner:   Scanner{LookPath: func(string) (string, error) { return "/usr/local/bin/claude", nil }},
		RunOutput: func(_ context.Context, _ string, _ ...string) ([]byte, error) { return listJSON, nil },
	}

	status := provider.Detect()

	require.Equal(t, StatusMissing, status.Status)
	require.False(t, status.Plugin.Installed)
}

func TestClaude_PlanActions(t *testing.T) {
	provider := ClaudeProvider{}
	command := []string{"claude", "plugin", "install", TargetClaudePlugin}

	tests := []struct {
		name   string
		status Status
		force  bool
		want   Plan
	}{
		{name: "not detected", status: Status{}, want: Plan{Action: ActionNone}},
		{name: "error", status: Status{Detected: true, Status: StatusError}, force: true, want: Plan{Action: ActionNone}},
		{name: "missing", status: Status{Detected: true, Status: StatusMissing}, want: Plan{Action: ActionInstall, Command: command}},
		{name: "installed", status: Status{Detected: true, Status: StatusInstalled, Plugin: PluginStatus{Installed: true}}, want: Plan{Action: ActionNone}},
		{name: "forced", status: Status{Detected: true, Status: StatusInstalled, Plugin: PluginStatus{Installed: true}}, force: true, want: Plan{Action: ActionReinstall, Command: command}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require.Equal(t, tt.want, provider.Plan(tt.status, tt.force))
		})
	}
}

func TestClaudeApply_RetriesAfterMarketplaceUpdate(t *testing.T) {
	installErr := errors.New("stale marketplace")
	var calls [][]string
	provider := ClaudeProvider{
		RunCommand: func(_ context.Context, name string, args ...string) error {
			call := append([]string{name}, args...)
			calls = append(calls, call)
			if len(calls) == 1 {
				return installErr
			}
			return nil
		},
	}
	plan := Plan{Action: ActionInstall, Command: []string{"claude", "plugin", "install", TargetClaudePlugin}}

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
