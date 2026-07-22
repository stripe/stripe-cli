package agentsetup

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestScanCodex_NotDetected(t *testing.T) {
	provider := CodexProvider{
		Scanner: Scanner{LookPath: func(string) (string, error) { return "", errors.New("missing") }},
		RunOutput: func(context.Context, string, ...string) ([]byte, error) {
			t.Fatal("plugin list should not run when Codex is not detected")
			return nil, nil
		},
	}

	status := provider.Detect()

	require.Equal(t, ClientCodex, status.Client)
	require.False(t, status.Detected)
	require.Equal(t, StatusNotDetected, status.Status)
}

func TestScanCodex_PluginMissing(t *testing.T) {
	provider := codexTestProvider(`{"installed":[],"available":[]}`, nil, nil)

	status := provider.Detect()

	require.True(t, status.Detected)
	require.Equal(t, StatusMissing, status.Status)
	require.False(t, status.Plugin.Installed)
	require.Equal(t, Plan{Action: ActionInstall, Command: []string{"codex", "plugin", "add", TargetCodexPlugin}}, provider.Plan(status, false))
}

func TestScanCodex_PluginInstalled(t *testing.T) {
	// Real codex-cli schema: pluginId + marketplaceName.
	provider := codexTestProvider(`{"installed":[{"pluginId":"stripe@openai-curated","name":"stripe","marketplaceName":"openai-curated","version":"3fdeeb49"}]}`, nil, nil)

	status := provider.Detect()

	require.Equal(t, StatusInstalled, status.Status)
	require.True(t, status.Plugin.Installed)
	require.Equal(t, TargetCodexPlugin, status.Plugin.ID)
	require.Equal(t, "3fdeeb49", status.Plugin.Version)
	require.Equal(t, Plan{Action: ActionNone}, provider.Plan(status, false))
}

func TestScanCodex_PluginInstalledByNameAndMarketplace(t *testing.T) {
	// Entry without pluginId still matches on name + marketplaceName.
	provider := codexTestProvider(`{"installed":[{"name":"stripe","marketplaceName":"openai-curated","version":"2.0.0"}]}`, nil, nil)

	status := provider.Detect()

	require.Equal(t, StatusInstalled, status.Status)
	require.Equal(t, "2.0.0", status.Plugin.Version)
}

func TestScanCodex_OldVersionWithoutPluginSupport(t *testing.T) {
	provider := codexTestProvider("", errors.New("unrecognized subcommand 'plugin'"), nil)

	status := provider.Detect()

	// Old Codex shows as detected but with an error hint — the TUI renders
	// it as disabled (visible but not selectable).
	require.True(t, status.Detected)
	require.Equal(t, StatusMissing, status.Status)
	require.Contains(t, status.Error, "upgrade Codex")
}

func TestCodexApply_RunsAddCommandAndVerifies(t *testing.T) {
	var gotName string
	var gotArgs []string
	installed := false

	provider := CodexProvider{
		Scanner: Scanner{LookPath: func(string) (string, error) { return "/usr/local/bin/codex", nil }},
		RunCommand: func(_ context.Context, name string, args ...string) error {
			gotName = name
			gotArgs = args
			installed = true // simulate a successful add
			return nil
		},
		RunOutput: func(context.Context, string, ...string) ([]byte, error) {
			if installed {
				return []byte(`{"installed":[{"pluginId":"stripe@openai-curated","name":"stripe","marketplaceName":"openai-curated","version":"1.0.0"}]}`), nil
			}
			return []byte(`{"installed":[]}`), nil
		},
	}

	status := provider.Detect()
	plan := provider.Plan(status, false)
	err := provider.Apply(context.Background(), nil, plan)

	require.NoError(t, err)
	require.Equal(t, "codex", gotName)
	require.Equal(t, []string{"plugin", "add", TargetCodexPlugin}, gotArgs)
}

// TestCodexApply_FailsWhenExitZeroButNotInstalled covers the real-world case
// where `codex plugin add` prints an error but exits 0. Apply must not report
// success when the plugin is still not present afterward.
func TestCodexApply_FailsWhenExitZeroButNotInstalled(t *testing.T) {
	provider := codexTestProvider(`{"installed":[]}`, nil, func(context.Context, string, ...string) error {
		return nil // add "succeeds" (exit 0) but installs nothing
	})

	status := provider.Detect()
	plan := provider.Plan(status, false)
	err := provider.Apply(context.Background(), nil, plan)

	require.Error(t, err)
	require.Contains(t, err.Error(), "is not installed")
}

func codexTestProvider(listOutput string, listErr error, runCommand RunCommandFunc) CodexProvider {
	return CodexProvider{
		Scanner:    Scanner{LookPath: func(string) (string, error) { return "/usr/local/bin/codex", nil }},
		RunCommand: runCommand,
		RunOutput: func(context.Context, string, ...string) ([]byte, error) {
			if listErr != nil {
				return nil, listErr
			}
			return []byte(listOutput), nil
		},
	}
}
