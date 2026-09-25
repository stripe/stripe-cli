package agentsetup

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestOpenclaw_NotDetected(t *testing.T) {
	scanner := Scanner{LookPath: func(string) (string, error) { return "", errors.New("missing") }}
	provider := NewOpenclawProvider(scanner, nil)

	status := provider.Detect()

	require.Equal(t, ClientOpenclaw, status.Client)
	require.Equal(t, "Openclaw", status.DisplayName)
	require.False(t, status.Detected)
	require.Equal(t, StatusNotDetected, status.Status)
	require.Equal(t, Plan{Action: ActionNone}, provider.Plan(status, false))
}

func TestOpenclaw_PluginMissing(t *testing.T) {
	provider := openclawTestProvider(`[]`, nil, nil)

	status := provider.Detect()

	require.True(t, status.Detected)
	require.Equal(t, StatusMissing, status.Status)
	require.False(t, status.Plugin.Installed)
	require.Equal(t,
		Plan{Action: ActionInstall, Command: []string{
			"git", "clone", "--branch", "plugins/agent-plugin", "--depth", "1",
			"https://github.com/stripe/ai.git", openclawClonedRepoPath,
		}},
		provider.Plan(status, false))
}

func TestOpenclaw_PluginInstalled(t *testing.T) {
	provider := openclawTestProvider(`[
		{"id":"plugin-1","name":"stripe","version":"0.7.1",
		 "source":"https://github.com/stripe/ai.git","format":"bundle"}
	]`, nil, nil)

	status := provider.Detect()

	require.Equal(t, StatusInstalled, status.Status)
	require.True(t, status.Plugin.Installed)
	require.Equal(t, "stripe", status.Plugin.ID)
	require.Equal(t, "0.7.1", status.Plugin.Version)
	require.Equal(t, "https://github.com/stripe/ai.git", status.Plugin.StatePath)
	require.Equal(t, "user", status.Plugin.Scope)
	require.Equal(t, Plan{Action: ActionNone}, provider.Plan(status, false))
	require.Equal(t,
		Plan{Action: ActionReinstall, Command: []string{"git", "pull", "-C", openclawClonedRepoPath}},
		provider.Plan(status, true))
}

func TestOpenclaw_OldVersionWithoutPluginSupport(t *testing.T) {
	provider := openclawTestProvider("", errors.New("unrecognized subcommand 'plugins'"), nil)

	status := provider.Detect()

	require.True(t, status.Detected)
	require.Equal(t, StatusMissing, status.Status)
	require.Contains(t, status.Error, "upgrade Openclaw Build")
}

func TestOpenclawApply_ClonesThenInstalls(t *testing.T) {
	var commandNames []string
	var commandArgs [][]string
	runCommand := func(_ context.Context, name string, args ...string) error {
		commandNames = append(commandNames, name)
		commandArgs = append(commandArgs, args)
		return nil
	}
	provider := NewOpenclawProvider(Scanner{}, runCommand).(OpenclawProvider)

	plan := Plan{Action: ActionInstall, Command: []string{
		"git", "clone", "--branch", "plugins/agent-plugin", "--depth", "1",
		"https://github.com/stripe/ai.git", openclawClonedRepoPath,
	}}
	err := provider.Apply(context.Background(), nil, plan)

	require.NoError(t, err)
	require.Equal(t, []string{"git", "openclaw"}, commandNames)
	require.Equal(t,
		[]string{"clone", "--branch", "plugins/agent-plugin", "--depth", "1", "https://github.com/stripe/ai.git", openclawClonedRepoPath},
		commandArgs[0])
	require.Equal(t, []string{"plugins", "install", openclawClonedRepoPath}, commandArgs[1])
}

func TestOpenclawApply_ReinstallPullsThenInstalls(t *testing.T) {
	var commandArgs [][]string
	runCommand := func(_ context.Context, _ string, args ...string) error {
		commandArgs = append(commandArgs, args)
		return nil
	}
	provider := NewOpenclawProvider(Scanner{}, runCommand).(OpenclawProvider)

	plan := Plan{Action: ActionReinstall, Command: []string{"git", "pull", "-C", openclawClonedRepoPath}}
	err := provider.Apply(context.Background(), nil, plan)

	require.NoError(t, err)
	require.Equal(t, []string{"pull", "-C", openclawClonedRepoPath}, commandArgs[0])
	require.Equal(t, []string{"plugins", "install", openclawClonedRepoPath}, commandArgs[1])
}

func TestOpenclawApply_NoneIsNoop(t *testing.T) {
	runCommand := func(context.Context, string, ...string) error {
		t.Fatal("RunCommand should not run for ActionNone")
		return nil
	}
	provider := NewOpenclawProvider(Scanner{}, runCommand).(OpenclawProvider)

	err := provider.Apply(context.Background(), nil, Plan{Action: ActionNone})

	require.NoError(t, err)
}

func TestOpenclawApply_StopsWhenCloneFails(t *testing.T) {
	cloneErr := errors.New("clone failed")
	var commandsRan []string
	runCommand := func(_ context.Context, name string, _ ...string) error {
		commandsRan = append(commandsRan, name)
		return cloneErr
	}
	provider := NewOpenclawProvider(Scanner{}, runCommand).(OpenclawProvider)

	plan := Plan{Action: ActionInstall, Command: []string{
		"git", "clone", "--branch", "plugins/agent-plugin", "--depth", "1",
		"https://github.com/stripe/ai.git", openclawClonedRepoPath,
	}}
	err := provider.Apply(context.Background(), nil, plan)

	require.ErrorIs(t, err, cloneErr)
	require.Equal(t, []string{"git"}, commandsRan, "install must not run when the git step fails")
}

func TestOpenclawApply_MissingCommand(t *testing.T) {
	provider := NewOpenclawProvider(Scanner{}, func(context.Context, string, ...string) error {
		t.Fatal("RunCommand should not run without a command")
		return nil
	}).(OpenclawProvider)

	err := provider.Apply(context.Background(), nil, Plan{Action: ActionInstall})

	require.Error(t, err)
}

func openclawTestProvider(listOutput string, listErr error, runCommand RunCommandFunc) OpenclawProvider {
	scanner := Scanner{LookPath: func(string) (string, error) { return "/usr/local/bin/openclaw", nil }}
	runOutput := func(context.Context, string, ...string) ([]byte, error) {
		if listErr != nil {
			return nil, listErr
		}
		return []byte(listOutput), nil
	}
	provider := NewOpenclawProvider(scanner, runCommand).(OpenclawProvider)
	provider.RunOutput = runOutput
	return provider
}
