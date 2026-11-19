package agentsetup

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func executableFixture(t *testing.T, dir, name string) string {
	t.Helper()
	require.NoError(t, os.MkdirAll(dir, 0755))
	if runtime.GOOS == "windows" {
		name += ".cmd"
		t.Setenv("PATHEXT", ".COM;.EXE;.BAT;.CMD")
	}
	path := filepath.Join(dir, name)
	require.NoError(t, os.WriteFile(path, []byte("fixture"), 0755))
	return path
}

func TestExecutableResolverProjectExclusions(t *testing.T) {
	for _, marker := range []string{".git", "package.json", "go.mod"} {
		t.Run(marker, func(t *testing.T) {
			root := t.TempDir()
			project := filepath.Join(root, "project")
			cwd := filepath.Join(project, "packages", "app", "src")
			require.NoError(t, os.MkdirAll(cwd, 0755))
			require.NoError(t, os.WriteFile(filepath.Join(project, marker), nil, 0644))
			t.Chdir(cwd)

			projectBin := filepath.Join(project, "tools")
			localBin := filepath.Join(cwd, "node_modules", ".bin")
			ancestorBin := filepath.Join(root, "node_modules", ".bin")
			globalBin := filepath.Join(root, "project-global", "bin")
			for _, dir := range []string{projectBin, localBin, ancestorBin, cwd} {
				executableFixture(t, dir, "claude")
			}
			global := executableFixture(t, globalBin, "claude")
			entries := []string{"", ".", "tools", localBin, ancestorBin, projectBin, cwd}
			t.Setenv("PATH", strings.Join(entries, string(os.PathListSeparator)))
			resolver := newExecutableResolver()
			selected, err := resolver.lookPath("claude")
			require.Error(t, err)
			require.Empty(t, selected)
			require.Empty(t, resolver.dirs)

			t.Setenv("PATH", strings.Join(append(entries, globalBin), string(os.PathListSeparator)))
			resolver = newExecutableResolver()
			selected, err = resolver.lookPath("claude")
			require.NoError(t, err)
			require.Equal(t, global, selected)
			require.Equal(t, []string{globalBin}, resolver.dirs)
		})
	}
}

func TestExecutableResolverAliases(t *testing.T) {
	root := t.TempDir()
	project := filepath.Join(root, "project")
	projectBin := filepath.Join(project, "tools")
	projectExe := executableFixture(t, projectBin, "claude")
	globalBin := filepath.Join(root, "global")
	global := executableFixture(t, globalBin, "claude")
	require.NoError(t, os.WriteFile(filepath.Join(project, "package.json"), nil, 0644))
	t.Chdir(project)

	alias := filepath.Join(root, "alias")
	if err := os.Symlink(projectBin, alias); err != nil {
		if runtime.GOOS != "windows" {
			require.NoError(t, err)
		}
		// Directory junctions do not require Windows symlink privileges.
		require.NoError(t, exec.Command("cmd", "/c", "mklink", "/J", alias, projectBin).Run())
	}
	aliasBin := filepath.Join(root, "alias-bin")
	require.NoError(t, os.MkdirAll(aliasBin, 0755))
	if err := os.Symlink(projectExe, filepath.Join(aliasBin, filepath.Base(projectExe))); err != nil {
		if runtime.GOOS != "windows" {
			require.NoError(t, err)
		}
	} else {
		t.Setenv("PATH", aliasBin)
		_, err := newExecutableResolver().lookPath("claude")
		require.Error(t, err)
	}
	// A project path remains excluded even when its target is outside it.
	outsideAlias := filepath.Join(project, "outside")
	if err := os.Symlink(globalBin, outsideAlias); err == nil {
		t.Setenv("PATH", outsideAlias)
		_, err := newExecutableResolver().lookPath("claude")
		require.Error(t, err)
	}
	t.Setenv("PATH", strings.Join([]string{alias, aliasBin, globalBin}, string(os.PathListSeparator)))
	selected, err := newExecutableResolver().lookPath("claude")
	require.NoError(t, err)
	require.Equal(t, global, selected)
	globalAlias := filepath.Join(root, "global-alias")
	if err := os.Symlink(globalBin, globalAlias); err == nil {
		t.Setenv("PATH", globalAlias)
		selected, err := newExecutableResolver().lookPath("claude")
		require.NoError(t, err)
		require.Equal(t, filepath.Join(globalAlias, filepath.Base(global)), selected)
	}
}

func TestExecutableResolverWorkingDirectoryAlias(t *testing.T) {
	root := t.TempDir()
	project := filepath.Join(root, "project")
	cwd := filepath.Join(project, "src")
	require.NoError(t, os.MkdirAll(cwd, 0755))
	require.NoError(t, os.WriteFile(filepath.Join(project, ".git"), nil, 0644))
	alias := filepath.Join(root, "work")
	if err := os.Symlink(cwd, alias); err != nil {
		if runtime.GOOS != "windows" {
			require.NoError(t, err)
		}
		require.NoError(t, exec.Command("cmd", "/c", "mklink", "/J", alias, cwd).Run())
	}
	t.Chdir(alias)
	projectBin := filepath.Join(project, "bin")
	executableFixture(t, projectBin, "claude")
	globalBin := filepath.Join(root, "global")
	global := executableFixture(t, globalBin, "claude")
	missing := filepath.Join(root, "missing")
	t.Setenv("PATH", strings.Join([]string{missing, projectBin, globalBin}, string(os.PathListSeparator)))
	selected, err := newExecutableResolver().lookPath("claude")
	require.NoError(t, err)
	require.Equal(t, global, selected)
}

func TestExecutableResolverFailClosed(t *testing.T) {
	r := &executableResolver{err: errors.New("cannot inspect project")}
	_, err := r.lookPath("claude")
	require.Error(t, err)
	_, err = r.run(context.Background(), "claude")
	require.Error(t, err)
}

func TestExecutableResolverWindowsLookup(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("Windows executable lookup")
	}
	root := t.TempDir()
	project := filepath.Join(root, "Project")
	globalBin := filepath.Join(root, "global")
	projectExe := executableFixture(t, project, "claude")
	global := executableFixture(t, globalBin, "claude")
	t.Chdir(project)
	t.Setenv("PATH", strings.ToUpper(project)+string(os.PathListSeparator)+globalBin)
	t.Setenv("PATHEXT", ".CMD;.EXE")
	require.NoError(t, os.WriteFile(filepath.Join(globalBin, "claude.exe"), nil, 0755))
	r := newExecutableResolver()
	t.Setenv("PATHEXT", ".EXE")
	selected, err := r.lookPath("claude")
	require.NoError(t, err)
	require.Equal(t, global, selected)
	require.False(t, r.safe(strings.ToUpper(projectExe)))
	require.True(t, pathContains(`C:\Project`, `c:\project\bin\claude.cmd`))
	require.True(t, pathContains(`C:\Project`, `\\?\C:\Project\bin\claude.cmd`))
	require.True(t, pathContains(`\\server\share\project`, `\\?\UNC\server\share\project\bin`))
	require.False(t, pathContains(`C:\Project`, `D:\Project\bin`))
	require.False(t, pathContains(`C:\Project`, `C:\Project-other\bin`))
	require.False(t, comparableAbsolutePath(`C:relative\bin`))
	require.False(t, comparableAbsolutePath(`\rooted\bin`))
	require.False(t, comparableAbsolutePath(`\\.\C:\Project\bin`))
	require.False(t, comparableAbsolutePath(`\\?\GLOBALROOT\Device\HarddiskVolume1\Project\bin`))
	require.True(t, comparableAbsolutePath(`\\?\C:\Project\bin`))
}

func TestAgentCommandEnvironment(t *testing.T) {
	if os.Getenv("AGENT_SETUP_ENV_HELPER") == "1" {
		cwd, err := os.Getwd()
		if err != nil {
			os.Exit(1)
		}
		_ = json.NewEncoder(os.Stdout).Encode([]string{os.Getenv("PATH"), cwd})
		os.Exit(0)
	}
	project := t.TempDir()
	globalBin := t.TempDir()
	projectBin := filepath.Join(project, "node_modules", ".bin")
	executableFixture(t, projectBin, "claude")
	global := executableFixture(t, globalBin, "claude")
	t.Chdir(project)
	t.Setenv("AGENT_SETUP_ENV_HELPER", "1")
	t.Setenv("PATH", projectBin+string(os.PathListSeparator)+globalBin)
	p := DefaultProviders()[ClientClaudeCode].(ClaudeProvider)
	selected, err := p.Scanner.LookPath("claude")
	require.NoError(t, err)
	require.Equal(t, global, selected)
	// Changing PATH cannot change the selected executable or child environment.
	t.Setenv("PATH", projectBin)
	exe, err := os.Executable()
	require.NoError(t, err)
	out, err := p.RunCommand(context.Background(), exe, "-test.run=^TestAgentCommandEnvironment$")
	require.NoError(t, err, string(out))
	var got []string
	require.NoError(t, json.Unmarshal(out, &got))
	cwd, err := filepath.EvalSymlinks(project)
	require.NoError(t, err)
	require.Equal(t, []string{globalBin, cwd}, got)
	_, err = p.RunCommand(context.Background(), filepath.Join(projectBin, filepath.Base(global)))
	require.Error(t, err)
	require.Equal(t, projectBin, os.Getenv("PATH"))
}

func TestProvidersUseSelectedExecutable(t *testing.T) {
	constructors := map[string]func(Scanner, RunCommandFunc) Provider{
		"claude":   NewClaudeProvider,
		"codex":    NewCodexProvider,
		"grok":     NewGrokProvider,
		"openclaw": NewOpenclawProvider,
	}
	for binary, newProvider := range constructors {
		t.Run(binary, func(t *testing.T) {
			project := t.TempDir()
			globalBin := t.TempDir()
			t.Chdir(project)
			projectBin := filepath.Join(project, "node_modules", ".bin")
			executableFixture(t, projectBin, binary)
			selected := executableFixture(t, globalBin, binary)
			calls := 0
			installed := false
			run := func(_ context.Context, name string, args ...string) ([]byte, error) {
				calls++
				switch name {
				case "mkdir", "git", "rm":
				default:
					require.Equal(t, selected, name)
				}
				if len(args) > 1 && args[1] == "marketplace" {
					return []byte(codexTestMarketplaceList), nil
				}
				if len(args) > 1 && args[1] == "add" {
					installed = true
				}
				if installed {
					return []byte(`{"installed":[{"pluginId":"stripe@openai-curated"}]}`), nil
				}
				return []byte(`[]`), nil
			}
			t.Setenv("PATH", projectBin)
			p := newProvider(DefaultScanner(), run)
			s := p.Detect()
			require.False(t, s.Detected)
			require.Empty(t, s.ExecutablePath)
			require.Equal(t, ActionNone, p.Plan(s, true).Action)
			require.Zero(t, calls)

			t.Setenv("PATH", projectBin+string(os.PathListSeparator)+globalBin)
			p = newProvider(DefaultScanner(), run)
			s = p.Detect()
			require.True(t, s.Detected)
			require.Equal(t, selected, s.ExecutablePath)
			t.Setenv("PATH", projectBin)
			plan := p.Plan(s, false)
			require.Equal(t, ActionInstall, plan.Action)
			for _, command := range plan.Commands {
				if command[0] != "mkdir" && command[0] != "git" && command[0] != "rm" {
					require.Equal(t, selected, command[0])
				}
			}
			require.NoError(t, p.Apply(context.Background(), nil, plan))
			s.ExecutablePath = ""
			require.Equal(t, ActionNone, p.Plan(s, false).Action)
		})
	}
}
