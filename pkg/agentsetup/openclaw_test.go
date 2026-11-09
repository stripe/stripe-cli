package agentsetup

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"
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

func TestOpenclawApply_WithoutUnixUtilities(t *testing.T) {
	binary, err := os.Executable()
	require.NoError(t, err)
	contents, err := os.ReadFile(binary)
	require.NoError(t, err)
	binDir := t.TempDir()
	suffix := ""
	if runtime.GOOS == "windows" {
		suffix = ".exe"
		t.Setenv("PATHEXT", ".EXE")
	}
	for _, name := range []string{"git", "openclaw"} {
		require.NoError(t, os.WriteFile(filepath.Join(binDir, name+suffix), contents, 0700))
	}
	t.Setenv("PATH", binDir)
	t.Setenv("STRIPE_OPENCLAW_TEST_HELPER", "1")
	var repoPath string
	home := t.TempDir()
	provider := NewOpenclawProvider(Scanner{HomeDir: func() (string, error) { return home, nil }}, func(ctx context.Context, name string, args ...string) ([]byte, error) {
		repoPath = args[len(args)-1]
		return RunCommand(ctx, name, append([]string{"-test.run=^TestOpenclawSubprocessHelper$", "--"}, args...)...)
	})

	err = provider.Apply(context.Background(), nil, provider.Plan(Status{Detected: true, Status: StatusMissing}, false))

	require.NoError(t, err)
	require.NotEmpty(t, repoPath)
	require.Equal(t, filepath.Join(home, ".openclaw"), filepath.Dir(filepath.Dir(repoPath)))
	require.NoDirExists(t, filepath.Dir(repoPath))
	require.DirExists(t, filepath.Join(home, ".openclaw"))
}

// TestOpenclawSubprocessHelper runs as a mock Git or OpenClaw executable.
func TestOpenclawSubprocessHelper(t *testing.T) {
	if os.Getenv("STRIPE_OPENCLAW_TEST_HELPER") != "1" {
		return
	}
	args := os.Args[3:]
	path := args[len(args)-1]
	if strings.TrimSuffix(filepath.Base(os.Args[0]), ".exe") == "git" {
		if err := os.Mkdir(path, 0700); err != nil {
			os.Exit(1)
		}
		if err := os.WriteFile(filepath.Join(path, "checkout"), []byte("ready"), 0600); err != nil {
			os.Exit(1)
		}
	} else {
		if _, err := os.Stat(filepath.Join(path, "checkout")); err != nil {
			os.Exit(1)
		}
	}
	os.Exit(0)
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

func TestOpenclawApply_CleansUpAfterEveryOutcome(t *testing.T) {
	for _, tc := range []struct{ action, failStage string }{
		{ActionInstall, ""},
		{ActionUpdate, ""},
		{ActionInstall, "git"},
		{ActionInstall, "openclaw"},
	} {
		for _, cleanupFails := range []bool{false, true} {
			t.Run(fmt.Sprintf("%s/stage=%s/cleanupFails=%t", tc.action, tc.failStage, cleanupFails), func(t *testing.T) {
				primaryErr := errors.New("subprocess failed")
				var commands [][]string
				var repoPath string
				home := t.TempDir()
				provider := NewOpenclawProvider(Scanner{HomeDir: func() (string, error) { return home, nil }}, func(_ context.Context, name string, args ...string) ([]byte, error) {
					commands = append(commands, append([]string{name}, args...))
					path := args[len(args)-1]
					require.NotEqual(t, openclawRepoPlaceholder, path)
					require.DirExists(t, filepath.Dir(path))
					if name == "git" {
						repoPath = path
						require.NoDirExists(t, path)
						require.NoError(t, os.Mkdir(path, 0700))
						t.Cleanup(func() { require.NoError(t, os.RemoveAll(filepath.Dir(path))) })
					} else {
						require.Equal(t, repoPath, path)
						require.DirExists(t, path)
					}
					if name == tc.failStage {
						return nil, primaryErr
					}
					return nil, nil
				}).(OpenclawProvider)
				cleanupCalls := 0
				provider.removeAll = func(path string) error {
					cleanupCalls++
					require.Equal(t, filepath.Dir(repoPath), path)
					if cleanupFails {
						return errors.New("removal failed")
					}
					return os.RemoveAll(path)
				}
				out := new(bytes.Buffer)
				telemetry := newOpenclawTelemetryClient()
				ctx := stripe.WithTelemetryClient(context.Background(), telemetry)
				status := Status{Detected: true, Plugin: PluginStatus{Installed: tc.action == ActionUpdate}}
				plan := provider.Plan(status, true)

				err := provider.Apply(ctx, out, plan)

				if tc.failStage == "" {
					require.NoError(t, err)
					if !cleanupFails {
						require.Empty(t, out.String())
					}
				} else {
					require.Same(t, primaryErr, err)
					require.Contains(t, out.String(), tc.failStage+" step failed")
				}
				expected := [][]string{
					{"git", "clone", "--branch", "plugins/agent-plugin", "--depth", "1", "https://github.com/stripe/ai.git", repoPath},
					{"openclaw", "plugins", "install", "--force", "--accept-capabilities", repoPath},
				}
				if tc.failStage == "git" {
					expected = expected[:1]
				}
				require.Equal(t, expected, commands)
				require.Equal(t, provider.Plan(status, true), plan, "Apply must not mutate the plan")
				require.Equal(t, 1, cleanupCalls)
				workspace := filepath.Dir(repoPath)
				require.Equal(t, filepath.Join(home, ".openclaw"), filepath.Dir(workspace))
				require.DirExists(t, filepath.Join(home, ".openclaw"))
				if cleanupFails {
					require.DirExists(t, workspace)
					require.Contains(t, out.String(), "Warning: OpenClaw plugin cleanup failed")
					require.Contains(t, out.String(), fmt.Sprintf("%q", workspace))
					select {
					case event := <-telemetry.events:
						require.Equal(t, [2]string{"Agent Setup: Plugin Cleanup", "openclaw:error"}, event)
					case <-time.After(time.Second):
						t.Fatal("missing cleanup event")
					}
				} else {
					require.NoDirExists(t, workspace)
					require.NotContains(t, out.String(), "Warning:")
				}
				assertNoOpenclawCleanupEvent(t, telemetry)
			})
		}
	}
}

func TestOpenclawApply_WorkspaceAllocationFails(t *testing.T) {
	for _, stage := range []string{"home", "parent", "workspace"} {
		t.Run(stage, func(t *testing.T) {
			home := t.TempDir()
			allocationErr := errors.New("allocation failed")
			provider := NewOpenclawProvider(Scanner{HomeDir: func() (string, error) {
				if stage == "home" {
					return "", allocationErr
				}
				return home, nil
			}}, func(context.Context, string, ...string) ([]byte, error) {
				t.Fatal("no subprocess should run without a workspace")
				return nil, nil
			}).(OpenclawProvider)
			if stage == "parent" {
				require.NoError(t, os.WriteFile(filepath.Join(home, ".openclaw"), nil, 0600))
			}
			provider.mkdirTemp = func(dir, _ string) (string, error) {
				require.Equal(t, "workspace", stage, "must not allocate a workspace after a prerequisite fails")
				require.Equal(t, filepath.Join(home, ".openclaw"), dir)
				return "", allocationErr
			}
			provider.removeAll = func(string) error {
				t.Fatal("no workspace to clean up")
				return nil
			}

			err := provider.Apply(context.Background(), nil, provider.Plan(Status{Detected: true, Status: StatusMissing}, false))

			switch stage {
			case "home":
				require.ErrorIs(t, err, allocationErr)
				require.ErrorContains(t, err, "resolving home directory")
			case "parent":
				require.ErrorContains(t, err, "creating OpenClaw directory")
			case "workspace":
				require.ErrorIs(t, err, allocationErr)
				require.ErrorContains(t, err, "creating OpenClaw plugin workspace")
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
			t.Cleanup(func() {
				for _, path := range paths {
					require.NoError(t, os.RemoveAll(filepath.Dir(path)))
				}
			})
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

func newOpenclawTelemetryClient() *openclawTelemetryClient {
	return &openclawTelemetryClient{events: make(chan [2]string, 10)}
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
