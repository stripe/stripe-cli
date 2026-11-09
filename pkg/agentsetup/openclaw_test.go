package agentsetup

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/stripe/stripe-cli/pkg/stripe"
)

const testOpenclawHomeDir = "/home/test-user"

const testOpenclawRepoPath = openclawRepoPlaceholder

func testOpenclawHomeDirFunc() (string, error) {
	return testOpenclawHomeDir, nil
}

func TestOpenclaw_NotDetected(t *testing.T) {
	scanner := Scanner{LookPath: func(string) (string, error) { return "", errors.New("missing") }}
	provider := NewOpenclawProvider(scanner, nil)

	status := provider.Detect()

	require.Equal(t, ClientOpenclaw, status.Client)
	require.Equal(t, "OpenClaw", status.DisplayName)
	require.False(t, status.Detected)
	require.Equal(t, StatusNotDetected, status.Status)
	require.Equal(t, Plan{Action: ActionNone}, provider.Plan(status, false))
}

func TestOpenclaw_HomeDirUnresolvable(t *testing.T) {
	homeDirErr := errors.New("cannot resolve home directory")
	scanner := Scanner{
		LookPath: func(string) (string, error) { return "/usr/local/bin/openclaw", nil },
		HomeDir:  func() (string, error) { return "", homeDirErr },
	}
	testCommand := func(_ context.Context, _ string, args ...string) ([]byte, error) {
		if args[1] == "list" {
			return []byte(`{"plugins": [], "registry": []}`), nil
		}
		t.Fatal("RunCommand should not run when home directory cannot be resolved")
		return nil, nil
	}
	provider := NewOpenclawProvider(scanner, testCommand)
	status := provider.Detect()

	require.Equal(t, StatusError, status.Status)
	require.Equal(t, "Could not resolve home directory for OpenClaw plugin installation", status.Error)
	require.Equal(t, Plan{Action: ActionNone}, provider.Plan(status, false))
	require.Equal(t, Plan{Action: ActionNone}, provider.Plan(status, true))
}

func TestOpenclaw_PluginMissing(t *testing.T) {
	provider := openclawTestProvider(`{"plugins": [], "registry": []}`, nil, nil)

	status := provider.Detect()

	require.True(t, status.Detected)
	require.Equal(t, StatusMissing, status.Status)
	require.False(t, status.Plugin.Installed)
	provider.Scanner.HomeDir = func() (string, error) {
		t.Fatal("planning should not resolve the home directory")
		return "", nil
	}
	provider.mkdirTemp = func(string, string) (string, error) {
		t.Fatal("planning should not allocate a workspace")
		return "", nil
	}
	provider.removeAll = func(string) error {
		t.Fatal("planning should not remove files")
		return nil
	}
	require.Equal(t,
		Plan{Action: ActionInstall, Commands: [][]string{
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
		Plan{Action: ActionUpdate, Commands: [][]string{
			{"git", "clone", "--branch", "plugins/agent-plugin", "--depth", "1", "https://github.com/stripe/ai.git", testOpenclawRepoPath},
			{"openclaw", "plugins", "install", "--force", "--accept-capabilities", testOpenclawRepoPath},
		}},
		provider.Plan(status, true))
}

func TestOpenclaw_OldVersionWithoutPluginSupport(t *testing.T) {
	provider := openclawTestProvider("", errors.New("unrecognized subcommand 'plugins'"), nil)

	status := provider.Detect()

	require.True(t, status.Detected)
	require.Equal(t, StatusMissing, status.Status)
	require.Contains(t, status.Error, "upgrade OpenClaw")
}

func TestOpenclawApply_NoneIsNoop(t *testing.T) {
	testCommand := func(context.Context, string, ...string) ([]byte, error) {
		t.Fatal("RunCommand should not run for ActionNone")
		return nil, nil
	}
	provider := NewOpenclawProvider(Scanner{HomeDir: testOpenclawHomeDirFunc}, testCommand)

	err := provider.Apply(context.Background(), nil, Plan{Action: ActionNone})

	require.NoError(t, err)
}

func TestOpenclawApply_ClonesThenInstalls(t *testing.T) {
	for _, action := range []string{ActionInstall, ActionUpdate} {
		t.Run(action, func(t *testing.T) {
			home := t.TempDir()
			require.NoError(t, os.Mkdir(filepath.Join(home, ".openclaw"), 0700))
			var commands [][]string
			var repoPath string
			provider := NewOpenclawProvider(Scanner{HomeDir: func() (string, error) { return home, nil }}, func(_ context.Context, name string, args ...string) ([]byte, error) {
				commands = append(commands, append([]string{name}, args...))
				path := args[len(args)-1]
				require.NotEqual(t, openclawRepoPlaceholder, path)
				require.DirExists(t, filepath.Dir(path))
				if name == "git" {
					repoPath = path
					require.NoDirExists(t, path)
					require.NoError(t, os.Mkdir(path, 0700))
				} else {
					require.Equal(t, repoPath, path)
					require.DirExists(t, path)
				}
				return nil, nil
			})
			status := Status{Detected: true, Plugin: PluginStatus{Installed: action == ActionUpdate}}
			plan := provider.Plan(status, true)
			out := new(bytes.Buffer)
			telemetry := &openclawTelemetryClient{events: make(chan [2]string, 10)}
			ctx := stripe.WithTelemetryClient(context.Background(), telemetry)

			require.NoError(t, provider.Apply(ctx, out, plan))

			require.Equal(t, [][]string{
				{"git", "clone", "--branch", "plugins/agent-plugin", "--depth", "1", "https://github.com/stripe/ai.git", repoPath},
				{"openclaw", "plugins", "install", "--force", "--accept-capabilities", repoPath},
			}, commands)
			require.Equal(t, provider.Plan(status, true), plan, "Apply must not mutate the plan")
			require.Equal(t, filepath.Join(home, ".openclaw"), filepath.Dir(filepath.Dir(repoPath)))
			require.DirExists(t, filepath.Join(home, ".openclaw"))
			require.NoDirExists(t, filepath.Dir(repoPath))
			require.Empty(t, out.String())
			assertNoOpenclawCleanupEvent(t, telemetry)
		})
	}
}

func TestOpenclawApply_CleansUpEvenIfInstallFails(t *testing.T) {
	for _, failStage := range []string{"git", "openclaw"} {
		t.Run(failStage, func(t *testing.T) {
			home := t.TempDir()
			require.NoError(t, os.Mkdir(filepath.Join(home, ".openclaw"), 0700))
			installErr := errors.New("installation failed")
			var repoPath string
			var commands []string
			provider := NewOpenclawProvider(Scanner{HomeDir: func() (string, error) { return home, nil }}, func(_ context.Context, name string, args ...string) ([]byte, error) {
				commands = append(commands, name)
				repoPath = args[len(args)-1]
				if name == "git" {
					require.NoError(t, os.Mkdir(repoPath, 0700))
				}
				if name == failStage {
					return nil, installErr
				}
				return nil, nil
			})
			out := new(bytes.Buffer)

			err := provider.Apply(context.Background(), out, provider.Plan(Status{Detected: true}, false))

			require.Same(t, installErr, err)
			require.Contains(t, out.String(), failStage+" step failed")
			if failStage == "git" {
				require.Equal(t, []string{"git"}, commands)
			} else {
				require.Equal(t, []string{"git", "openclaw"}, commands)
			}
			require.NoDirExists(t, filepath.Dir(repoPath))
			require.DirExists(t, filepath.Join(home, ".openclaw"))
		})
	}
}

func TestOpenclawApply_CleanupFailureIsOnlyAWarning(t *testing.T) {
	for _, failStage := range []string{"", "git", "openclaw"} {
		t.Run("stage="+failStage, func(t *testing.T) {
			home := t.TempDir()
			require.NoError(t, os.Mkdir(filepath.Join(home, ".openclaw"), 0700))
			installErr := errors.New("installation failed")
			provider := NewOpenclawProvider(Scanner{HomeDir: func() (string, error) { return home, nil }}, func(_ context.Context, name string, _ ...string) ([]byte, error) {
				if name == failStage {
					return nil, installErr
				}
				return nil, nil
			}).(OpenclawProvider)
			var workspace string
			cleanupCalls := 0
			provider.removeAll = func(path string) error {
				workspace = path
				cleanupCalls++
				return errors.New("removal failed")
			}
			out := new(bytes.Buffer)
			telemetry := &openclawTelemetryClient{events: make(chan [2]string, 10)}
			ctx := stripe.WithTelemetryClient(context.Background(), telemetry)

			err := provider.Apply(ctx, out, provider.Plan(Status{Detected: true}, false))

			if failStage == "" {
				require.NoError(t, err)
			} else {
				require.Same(t, installErr, err)
			}
			require.Equal(t, 1, cleanupCalls)
			require.DirExists(t, workspace)
			require.Contains(t, out.String(), "Warning: OpenClaw plugin cleanup failed")
			require.Contains(t, out.String(), fmt.Sprintf("%q", workspace))
			select {
			case event := <-telemetry.events:
				require.Equal(t, [2]string{"Agent Setup: Plugin Cleanup", "openclaw:error"}, event)
			case <-time.After(time.Second):
				t.Fatal("missing cleanup event")
			}
			assertNoOpenclawCleanupEvent(t, telemetry)
		})
	}
}

func TestOpenclawApply_WorkspaceAllocationFails(t *testing.T) {
	allocationErr := errors.New("allocation failed")
	for _, tc := range []struct {
		stage, message string
		cause          error
	}{
		{"home", "resolving home directory", allocationErr},
		{"parent", "creating OpenClaw plugin workspace", os.ErrNotExist},
		{"workspace", "creating OpenClaw plugin workspace", allocationErr},
	} {
		t.Run(tc.stage, func(t *testing.T) {
			home := t.TempDir()
			provider := NewOpenclawProvider(Scanner{HomeDir: func() (string, error) {
				if tc.stage == "home" {
					return "", allocationErr
				}
				return home, nil
			}}, func(context.Context, string, ...string) ([]byte, error) {
				t.Fatal("no subprocess should run without a workspace")
				return nil, nil
			}).(OpenclawProvider)
			if tc.stage != "parent" {
				provider.mkdirTemp = func(dir, _ string) (string, error) {
					require.Equal(t, "workspace", tc.stage, "must not allocate a workspace after a prerequisite fails")
					require.Equal(t, filepath.Join(home, ".openclaw"), dir)
					return "", allocationErr
				}
			}
			provider.removeAll = func(string) error {
				t.Fatal("no workspace to clean up")
				return nil
			}

			err := provider.Apply(context.Background(), nil, provider.Plan(Status{Detected: true, Status: StatusMissing}, false))

			require.ErrorIs(t, err, tc.cause)
			require.ErrorContains(t, err, tc.message)
			if tc.stage == "parent" {
				require.NoDirExists(t, filepath.Join(home, ".openclaw"))
			}
		})
	}
}

func TestOpenclawApply_RetryLeavesLegacyDirectoryUntouched(t *testing.T) {
	for _, cleanupFails := range []bool{false, true} {
		t.Run(fmt.Sprintf("cleanupFails=%t", cleanupFails), func(t *testing.T) {
			home := t.TempDir()
			legacy := filepath.Join(home, ".openclaw", "stripe")
			require.NoError(t, os.MkdirAll(legacy, 0700))
			marker := filepath.Join(legacy, "keep.txt")
			require.NoError(t, os.WriteFile(marker, []byte("keep"), 0600))
			var paths []string
			provider := NewOpenclawProvider(Scanner{HomeDir: func() (string, error) { return home, nil }}, func(_ context.Context, name string, args ...string) ([]byte, error) {
				if name == "git" {
					path := args[len(args)-1]
					paths = append(paths, path)
					require.NoDirExists(t, path)
					require.NoError(t, os.Mkdir(path, 0700))
					if len(paths) == 1 {
						return nil, errors.New("clone interrupted")
					}
				}
				return nil, nil
			}).(OpenclawProvider)
			cleanupCalls := 0
			provider.removeAll = func(path string) error {
				cleanupCalls++
				if cleanupFails && cleanupCalls == 1 {
					return errors.New("removal failed")
				}
				return os.RemoveAll(path)
			}
			plan := provider.Plan(Status{Detected: true, Status: StatusMissing}, false)

			require.Error(t, provider.Apply(context.Background(), nil, plan))
			require.NoError(t, provider.Apply(context.Background(), nil, plan))

			require.Len(t, paths, 2)
			require.NotEqual(t, paths[0], paths[1])
			for _, path := range paths {
				require.Equal(t, filepath.Join(home, ".openclaw"), filepath.Dir(filepath.Dir(path)))
			}
			require.Equal(t, 2, cleanupCalls)
			require.NoDirExists(t, filepath.Dir(paths[1]))
			contents, err := os.ReadFile(marker)
			require.NoError(t, err)
			require.Equal(t, "keep", string(contents))
		})
	}
}

type openclawTelemetryClient struct {
	events chan [2]string
}

func (c *openclawTelemetryClient) SendAPIRequestEvent(context.Context, string, bool) (*http.Response, error) {
	return nil, nil
}

func (c *openclawTelemetryClient) SendEvent(_ context.Context, name, value string) {
	c.events <- [2]string{name, value}
}

func assertNoOpenclawCleanupEvent(t *testing.T, client *openclawTelemetryClient) {
	t.Helper()
	select {
	case event := <-client.events:
		t.Fatalf("unexpected cleanup event: %v", event)
	case <-time.After(25 * time.Millisecond):
	}
}

func TestOpenclawApply_MissingCommand(t *testing.T) {
	provider := NewOpenclawProvider(Scanner{HomeDir: testOpenclawHomeDirFunc}, func(context.Context, string, ...string) ([]byte, error) {
		t.Fatal("RunCommand should not run without a command")
		return nil, nil
	}).(OpenclawProvider)

	err := provider.Apply(context.Background(), nil, Plan{Action: ActionInstall})

	require.Error(t, err)
}

func openclawTestProvider(listOutput string, listErr error, installCommand RunCommandFunc) OpenclawProvider {
	scanner := Scanner{
		LookPath: func(string) (string, error) { return "/usr/local/bin/openclaw", nil },
		HomeDir:  testOpenclawHomeDirFunc,
	}
	testCommand := func(ctx context.Context, name string, args ...string) ([]byte, error) {
		if args[1] == "list" {
			if listErr != nil {
				return nil, listErr
			}
			return []byte(listOutput), nil
		}
		if installCommand != nil {
			return installCommand(ctx, name, args...)
		}
		return nil, nil
	}
	provider := NewOpenclawProvider(scanner, testCommand).(OpenclawProvider)
	return provider
}
