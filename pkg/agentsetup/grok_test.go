package agentsetup

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestGrok_NotDetected(t *testing.T) {
	provider := GrokProvider{
		Scanner: Scanner{LookPath: func(string) (string, error) { return "", errors.New("missing") }},
		RunOutput: func(context.Context, string, ...string) ([]byte, error) {
			t.Fatal("plugin list should not run when Grok is not detected")
			return nil, nil
		},
	}

	status := provider.Detect()

	require.Equal(t, ClientGrok, status.Client)
	require.Equal(t, "Grok", status.DisplayName)
	require.False(t, status.Detected)
	require.Equal(t, StatusNotDetected, status.Status)
	require.Equal(t, Plan{Action: ActionNone}, provider.Plan(status, false))
}

func TestGrok_PluginMissing(t *testing.T) {
	provider := grokTestProvider(`[]`, nil, nil)

	status := provider.Detect()

	require.True(t, status.Detected)
	require.Equal(t, StatusMissing, status.Status)
	require.False(t, status.Plugin.Installed)
	require.Equal(t,
		Plan{Action: ActionInstall, Command: []string{"grok", "plugin", "install", GrokPluginName, "--trust"}},
		provider.Plan(status, false))
}

func TestGrok_PluginInstalled(t *testing.T) {
	provider := grokTestProvider(`[
		{"status":"installed","name":"stripe","repo_key":"plugin-760cfec9","version":"0.7.1",
		 "path":"/Users/x/.grok/installed-plugins/plugin-760cfec9",
		 "source":"https://github.com/stripe/ai.git","marketplace":"xAI Official"}
	]`, nil, nil)

	status := provider.Detect()

	require.Equal(t, StatusInstalled, status.Status)
	require.True(t, status.Plugin.Installed)
	require.Equal(t, "stripe", status.Plugin.ID)
	require.Equal(t, "0.7.1", status.Plugin.Version)
	require.Equal(t, "/Users/x/.grok/installed-plugins/plugin-760cfec9", status.Plugin.StatePath)
	require.Equal(t, Plan{Action: ActionNone}, provider.Plan(status, false))
	require.Equal(t, Plan{Action: ActionReinstall, Command: []string{"grok", "plugin", "update", GrokPluginName}}, provider.Plan(status, true))
}

func TestGrok_OldVersionWithoutPluginSupport(t *testing.T) {
	provider := grokTestProvider("", errors.New("unrecognized subcommand 'plugin'"), nil)

	status := provider.Detect()

	require.True(t, status.Detected)
	require.Equal(t, StatusMissing, status.Status)
	require.Contains(t, status.Error, "upgrade Grok Build")
}

func TestGrokApply_RunsInstallCommand(t *testing.T) {
	var gotName string
	var gotArgs []string

	provider := GrokProvider{
		RunCommand: func(_ context.Context, name string, args ...string) error {
			gotName = name
			gotArgs = args
			return nil
		},
	}

	plan := Plan{Action: ActionInstall, Command: []string{"grok", "plugin", "install", GrokPluginName, "--trust"}}
	err := provider.Apply(context.Background(), nil, plan)

	require.NoError(t, err)
	require.Equal(t, "grok", gotName)
	require.Equal(t, []string{"plugin", "install", GrokPluginName, "--trust"}, gotArgs)
}

func TestGrokApply_NoneIsNoop(t *testing.T) {
	provider := GrokProvider{
		RunCommand: func(context.Context, string, ...string) error {
			t.Fatal("RunCommand should not run for ActionNone")
			return nil
		},
	}

	err := provider.Apply(context.Background(), nil, Plan{Action: ActionNone})

	require.NoError(t, err)
}

func grokTestProvider(listOutput string, listErr error, runCommand RunCommandFunc) GrokProvider {
	return GrokProvider{
		Scanner:    Scanner{LookPath: func(string) (string, error) { return "/usr/local/bin/grok", nil }},
		RunCommand: runCommand,
		RunOutput: func(context.Context, string, ...string) ([]byte, error) {
			if listErr != nil {
				return nil, listErr
			}
			return []byte(listOutput), nil
		},
	}
}
