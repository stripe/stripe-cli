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

func TestExecutableResolverCaseAlias(t *testing.T) {
	root := t.TempDir()
	project := filepath.Join(root, "Project")
	projectBin := filepath.Join(project, "tools")
	executableFixture(t, projectBin, "claude")
	require.NoError(t, os.WriteFile(filepath.Join(project, ".git"), nil, 0644))
	upper := filepath.Join(root, "PROJECT")
	info, err := os.Stat(project)
	require.NoError(t, err)
	if aliasInfo, err := os.Stat(upper); err != nil || !os.SameFile(info, aliasInfo) {
		t.Skip("filesystem is case-sensitive")
	}
	t.Chdir(project)
	t.Setenv("PATH", filepath.Join(upper, "tools"))
	_, err = newExecutableResolver().lookPath("claude")
	require.Error(t, err)
}

func TestExecutableResolverHomeDirectory(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)
	userBin := filepath.Join(home, ".local", "bin")
	user := executableFixture(t, userBin, "claude")
	project := filepath.Join(home, "code", "project")
	projectBin := filepath.Join(project, "bin")
	executableFixture(t, projectBin, "claude")
	require.NoError(t, os.WriteFile(filepath.Join(project, "go.mod"), nil, 0644))

	// Home is not a project root, even when it is a dotfiles repository.
	for _, marker := range []string{"", ".git"} {
		if marker != "" {
			require.NoError(t, os.WriteFile(filepath.Join(home, marker), nil, 0644))
		}
		for cwd, path := range map[string][]string{home: {userBin}, project: {projectBin, userBin}} {
			t.Chdir(cwd)
			t.Setenv("PATH", strings.Join(path, string(os.PathListSeparator)))
			selected, err := newExecutableResolver().lookPath("claude")
			require.NoError(t, err)
			require.Equal(t, user, selected)
		}
	}
}

func TestExecutableResolverCleanedPathEntry(t *testing.T) {
	root := t.TempDir()
	project := filepath.Join(root, "project")
	projectBin := filepath.Join(project, "tools")
	executableFixture(t, projectBin, "claude")
	require.NoError(t, os.WriteFile(filepath.Join(project, ".git"), nil, 0644))
	safe := filepath.Join(root, "safe")
	safeBin := filepath.Join(safe, "b")
	executableFixture(t, safeBin, "claude")
	ext := filepath.Join(root, "ext")
	require.NoError(t, os.MkdirAll(ext, 0755))
	if err := os.Symlink(filepath.Join(safe, "x"), filepath.Join(ext, "a")); err != nil {
		t.Skip("symlinks unavailable")
	}
	require.NoError(t, os.MkdirAll(filepath.Join(safe, "x"), 0755))
	require.NoError(t, os.Symlink(projectBin, filepath.Join(ext, "b")))
	t.Chdir(project)
	// Unresolved, this names safe/b; cleaned, it names ext/b, the project bin.
	sep := string(filepath.Separator)
	// Build the entry by hand: filepath.Join would clean away the "..".
	t.Setenv("PATH", ext+sep+"a"+sep+".."+sep+"b")
	r := newExecutableResolver()
	require.Empty(t, r.dirs)
	_, err := r.lookPath("claude")
	require.Error(t, err)
}

func TestExecutableResolverSymlinkedHome(t *testing.T) {
	root := t.TempDir()
	users := filepath.Join(root, "data", "users")
	physical := filepath.Join(users, "me")
	user := executableFixture(t, filepath.Join(physical, ".local", "bin"), "claude")
	require.NoError(t, os.WriteFile(filepath.Join(users, "package.json"), nil, 0644))
	home := filepath.Join(root, "home", "me")
	require.NoError(t, os.MkdirAll(filepath.Dir(home), 0755))
	if err := os.Symlink(physical, home); err != nil {
		t.Skip("symlinks unavailable")
	}
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)
	t.Chdir(home)
	t.Setenv("PATH", filepath.Join(home, ".local", "bin"))
	selected, err := newExecutableResolver().lookPath("claude")
	require.NoError(t, err)
	require.Equal(t, filepath.Join(home, ".local", "bin", filepath.Base(user)), selected)
}

func TestDefaultProvidersResolveLazily(t *testing.T) {
	globalBin := t.TempDir()
	global := executableFixture(t, globalBin, "claude")
	t.Chdir(t.TempDir())
	t.Setenv("PATH", "")
	providers := DefaultProviders()
	// PATH is first inspected when detection runs, not when providers are built.
	t.Setenv("PATH", globalBin)
	s := providers[ClientClaudeCode].Detect()
	require.True(t, s.Detected)
	require.Equal(t, global, s.ExecutablePath)
}

func TestExecutableResolverQuotedWindowsPathEntry(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("only Windows PATH entries can contain the list separator")
	}
	root := t.TempDir()
	t.Chdir(t.TempDir())
	tools := filepath.Join(root, "Tools;cache", "bin")
	executableFixture(t, tools, "claude")
	other := filepath.Join(root, "other")
	require.NoError(t, os.MkdirAll(other, 0755))
	t.Setenv("PATH", `"`+tools+`";`+other)
	r := newExecutableResolver()
	require.Equal(t, []string{tools, other}, r.dirs)
	var path string
	for _, entry := range r.env {
		if key, value, _ := strings.Cut(entry, "="); strings.EqualFold(key, "PATH") {
			path = value
		}
	}
	require.Equal(t, []string{tools, other}, filepath.SplitList(path))
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
	require.Len(t, got, 2)
	cwd, err := filepath.EvalSymlinks(project)
	require.NoError(t, err)
	got[1], err = filepath.EvalSymlinks(got[1])
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
