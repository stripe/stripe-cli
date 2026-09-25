package agentsetup

import (
	"context"
	"errors"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

const testOpenclawHomeDir = "/home/test-user"

var testOpenclawRepoPath = filepath.Join(testOpenclawHomeDir, openclawRepoDir)

func testOpenclawHomeDirFunc() (string, error) {
	return testOpenclawHomeDir, nil
}

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
	provider := openclawTestProvider(`{"plugins": [], "registry": []}`, nil, nil)

	status := provider.Detect()

	require.True(t, status.Detected)
	require.Equal(t, StatusMissing, status.Status)
	require.False(t, status.Plugin.Installed)
	require.Equal(t,
		Plan{Action: ActionInstall, Commands: [][]string{
			{"mkdir", "-p", testOpenclawRepoPath},
			{"git", "clone", "--branch", "plugins/agent-plugin", "--depth", "1", "https://github.com/stripe/ai.git", testOpenclawRepoPath},
			{"openclaw", "plugins", "install", "--force", "--accept-capabilities", testOpenclawRepoPath},
		}},
		provider.Plan(status, false))
}

func TestOpenclaw_PluginInstalled(t *testing.T) {
	provider := openclawTestProvider(`{
		"plugins": [
			{"id":"xai","name":"@openclaw/xai-plugin","version":"2026.9.6",
			 "source":"/opt/homebrew/lib/node_modules/openclaw/dist/extensions/xai/index.js","format":"openclaw"},
			{"id":"plugin-1","name":"stripe","version":"0.7.1",
			 "source":"https://github.com/stripe/ai.git","format":"bundle"}
		],
		"registry": []
	}`, nil, nil)

	status := provider.Detect()

	require.Equal(t, StatusInstalled, status.Status)
	require.True(t, status.Plugin.Installed)
	require.Equal(t, "stripe", status.Plugin.ID)
	require.Equal(t, "0.7.1", status.Plugin.Version)
	require.Equal(t, "https://github.com/stripe/ai.git", status.Plugin.StatePath)
	require.Equal(t, "user", status.Plugin.Scope)
	require.Equal(t, Plan{Action: ActionNone}, provider.Plan(status, false))
	require.Equal(t,
		Plan{Action: ActionReinstall, Commands: [][]string{
			{"git", "pull", "-C", testOpenclawRepoPath},
			{"openclaw", "plugins", "install", "--force", "--accept-capabilities", testOpenclawRepoPath},
		}},
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
	provider := NewOpenclawProvider(Scanner{HomeDir: testOpenclawHomeDirFunc}, runCommand).(OpenclawProvider)

	preInstallCommands := [][]string{
		{"mkdir", "-p", testOpenclawRepoPath},
		{"git", "clone", "--branch", "plugins/agent-plugin", "--depth", "1", "https://github.com/stripe/ai.git", testOpenclawRepoPath},
		{"openclaw", "plugins", "install", "--force", "--accept-capabilities", testOpenclawRepoPath},
	}

	plan := Plan{Action: ActionInstall, Commands: preInstallCommands}
	err := provider.Apply(context.Background(), nil, plan)

	require.NoError(t, err)
	require.Equal(t, []string{"mkdir", "git", "openclaw"}, commandNames)
	require.Equal(t, []string{"-p", testOpenclawRepoPath}, commandArgs[0])
	require.Equal(t,
		[]string{"clone", "--branch", "plugins/agent-plugin", "--depth", "1", "https://github.com/stripe/ai.git", testOpenclawRepoPath},
		commandArgs[1])
	require.Equal(t, []string{"plugins", "install", "--force", "--accept-capabilities", testOpenclawRepoPath}, commandArgs[2])
}

func TestOpenclawApply_ReinstallPullsThenInstalls(t *testing.T) {
	var commandArgs [][]string
	runCommand := func(_ context.Context, _ string, args ...string) error {
		commandArgs = append(commandArgs, args)
		return nil
	}
	provider := NewOpenclawProvider(Scanner{HomeDir: testOpenclawHomeDirFunc}, runCommand).(OpenclawProvider)

	plan := Plan{Action: ActionReinstall, Commands: [][]string{
		{"git", "pull", "-C", testOpenclawRepoPath},
		{"openclaw", "plugins", "install", "--force", "--accept-capabilities", testOpenclawRepoPath},
	}}
	err := provider.Apply(context.Background(), nil, plan)

	require.NoError(t, err)
	require.Equal(t, []string{"pull", "-C", testOpenclawRepoPath}, commandArgs[0])
	require.Equal(t, []string{"plugins", "install", "--force", "--accept-capabilities", testOpenclawRepoPath}, commandArgs[1])
}

func TestOpenclawApply_NoneIsNoop(t *testing.T) {
	runCommand := func(context.Context, string, ...string) error {
		t.Fatal("RunCommand should not run for ActionNone")
		return nil
	}
	provider := NewOpenclawProvider(Scanner{HomeDir: testOpenclawHomeDirFunc}, runCommand).(OpenclawProvider)

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
	provider := NewOpenclawProvider(Scanner{HomeDir: testOpenclawHomeDirFunc}, runCommand).(OpenclawProvider)

	plan := Plan{Action: ActionInstall, Commands: [][]string{{
		"git", "clone", "--branch", "plugins/agent-plugin", "--depth", "1",
		"https://github.com/stripe/ai.git", testOpenclawRepoPath,
	}}}
	err := provider.Apply(context.Background(), nil, plan)

	require.ErrorIs(t, err, cloneErr)
	require.Equal(t, []string{"git"}, commandsRan, "install must not run when the git step fails")
}

func TestOpenclawApply_MissingCommand(t *testing.T) {
	provider := NewOpenclawProvider(Scanner{HomeDir: testOpenclawHomeDirFunc}, func(context.Context, string, ...string) error {
		t.Fatal("RunCommand should not run without a command")
		return nil
	}).(OpenclawProvider)

	err := provider.Apply(context.Background(), nil, Plan{Action: ActionInstall})

	require.Error(t, err)
}

func openclawTestProvider(listOutput string, listErr error, runCommand RunCommandFunc) OpenclawProvider {
	scanner := Scanner{
		LookPath: func(string) (string, error) { return "/usr/local/bin/openclaw", nil },
		HomeDir:  testOpenclawHomeDirFunc,
	}
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
