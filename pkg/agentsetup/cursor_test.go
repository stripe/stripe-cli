package agentsetup

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestCursor_NotDetected(t *testing.T) {
	provider := CursorProvider{
		Scanner: Scanner{LookPath: func(string) (string, error) { return "", errors.New("missing") }},
	}

	status := provider.Detect()

	require.Equal(t, ClientCursor, status.Client)
	require.Equal(t, "Cursor", status.DisplayName)
	require.False(t, status.Detected)
	require.Equal(t, StatusNotDetected, status.Status)
}

func TestCursor_DetectedMissingWhenNoSqlite(t *testing.T) {
	provider := cursorTestProvider(t, nil, nil)

	status := provider.Detect()

	require.True(t, status.Detected)
	require.Equal(t, StatusMissing, status.Status)
	require.False(t, status.Plugin.Installed)
	require.Contains(t, status.Error, "/add-plugin stripe")
}

func TestCursor_DetectedMissingWhenQueryFails(t *testing.T) {
	provider := cursorTestProvider(t, func(name string) (string, error) {
		if name == CursorBinaryName || name == sqlite3BinaryName {
			return "/usr/bin/" + name, nil
		}
		return "", errors.New("missing")
	}, func(context.Context, string, ...string) ([]byte, error) {
		return nil, errors.New("query failed")
	})

	status := provider.Detect()

	require.Equal(t, StatusMissing, status.Status)
	require.False(t, status.Plugin.Installed)
	require.Contains(t, status.Error, "/add-plugin stripe")
}

func TestCursor_DetectedMissingWhenNotInstalled(t *testing.T) {
	workspaceURI := cursorWorkspaceURI(t.TempDir())
	sqliteOut := sqliteInstalledIdsOutput(
		"cursor.plugins.installedIds.123|no-workspace|[]",
		"cursor.plugins.installedIds.123|"+workspaceURI+"|[]",
	)
	provider := cursorTestProviderWithWorkspace(t, sqliteOut)

	status := provider.Detect()

	require.Equal(t, StatusMissing, status.Status)
	require.False(t, status.Plugin.Installed)
}

func TestCursor_InstalledUserScope(t *testing.T) {
	workspaceURI := cursorWorkspaceURI(t.TempDir())
	sqliteOut := sqliteInstalledIdsOutput(
		`cursor.plugins.installedIds.123|no-workspace|[{"id":"408","sources":["user"]}]`,
		"cursor.plugins.installedIds.123|"+workspaceURI+"|[]",
	)
	provider := cursorTestProviderWithWorkspace(t, sqliteOut)

	status := provider.Detect()

	require.Equal(t, StatusInstalled, status.Status)
	require.True(t, status.Plugin.Installed)
	require.Equal(t, TargetCursorPlugin, status.Plugin.ID)
	require.Equal(t, "user", status.Plugin.Scope)
	require.Empty(t, status.Error)
}

func TestCursor_InstalledProjectScope(t *testing.T) {
	home := t.TempDir()
	projectDir := filepath.Join(home, "project")
	require.NoError(t, os.MkdirAll(projectDir, 0o755))

	workspaceURI := cursorWorkspaceURI(projectDir)
	sqliteOut := sqliteInstalledIdsOutput(
		"cursor.plugins.installedIds.123|no-workspace|[]",
		`cursor.plugins.installedIds.123|`+workspaceURI+`|[{"id":"408","sources":["user"]}]`,
	)
	provider := cursorTestProvider(t, func(name string) (string, error) {
		return "/usr/bin/" + name, nil
	}, func(context.Context, string, ...string) ([]byte, error) {
		return []byte(sqliteOut), nil
	})
	provider.Scanner.WorkDir = func() (string, error) { return projectDir, nil }

	status := provider.Detect()

	require.Equal(t, StatusInstalled, status.Status)
	require.True(t, status.Plugin.Installed)
	require.Equal(t, "project", status.Plugin.Scope)
	require.Equal(t, projectDir, status.Plugin.Project)
}

func TestCursor_OtherPluginIDsIgnored(t *testing.T) {
	workspaceURI := cursorWorkspaceURI(t.TempDir())
	sqliteOut := sqliteInstalledIdsOutput(
		`cursor.plugins.installedIds.123|no-workspace|[{"id":"999","sources":["user"]}]`,
		"cursor.plugins.installedIds.123|"+workspaceURI+"|[]",
	)
	provider := cursorTestProviderWithWorkspace(t, sqliteOut)

	status := provider.Detect()

	require.Equal(t, StatusMissing, status.Status)
	require.False(t, status.Plugin.Installed)
}

func TestCursor_MalformedJSONFallsBackToMissing(t *testing.T) {
	workspaceURI := cursorWorkspaceURI(t.TempDir())
	sqliteOut := sqliteInstalledIdsOutput(
		"cursor.plugins.installedIds.123|no-workspace|{nope",
		"cursor.plugins.installedIds.123|"+workspaceURI+"|[]",
	)
	provider := cursorTestProviderWithWorkspace(t, sqliteOut)

	status := provider.Detect()

	require.Equal(t, StatusMissing, status.Status)
	require.False(t, status.Plugin.Installed)
}

func TestCursor_InstalledReadsVersionFromCache(t *testing.T) {
	home := t.TempDir()
	projectDir := filepath.Join(home, "project")
	require.NoError(t, os.MkdirAll(projectDir, 0o755))
	writeCursorPluginCache(t, home, "abc123", `{"name":"stripe","version":"0.1.0"}`)

	sqliteOut := sqliteInstalledIdsOutput(
		`cursor.plugins.installedIds.123|no-workspace|[{"id":"408","sources":["user"]}]`,
	)
	provider := cursorTestProvider(t, func(name string) (string, error) {
		return "/usr/bin/" + name, nil
	}, func(context.Context, string, ...string) ([]byte, error) {
		return []byte(sqliteOut), nil
	})
	provider.Scanner.HomeDir = func() (string, error) { return home, nil }
	provider.Scanner.WorkDir = func() (string, error) { return projectDir, nil }
	dbPath, err := cursorStateDBPath(home, runtime.GOOS)
	require.NoError(t, err)
	provider.Scanner.Stat = func(path string) (os.FileInfo, error) {
		if path == dbPath {
			return fileInfoStub{}, nil
		}
		return os.Stat(path)
	}

	status := provider.Detect()

	require.Equal(t, StatusInstalled, status.Status)
	require.Equal(t, "0.1.0", status.Plugin.Version)
}

func TestCursor_PlanManualWhenNotInstalled(t *testing.T) {
	provider := cursorTestProvider(t, nil, nil)

	status := provider.Detect()
	plan := provider.Plan(status, false)

	require.Equal(t, ActionManual, plan.Action)
	require.Contains(t, plan.Manual, "/add-plugin stripe")
}

func TestCursor_PlanNoneWhenNotDetected(t *testing.T) {
	provider := CursorProvider{
		Scanner: Scanner{LookPath: func(string) (string, error) { return "", errors.New("missing") }},
	}

	status := provider.Detect()
	plan := provider.Plan(status, false)

	require.Equal(t, ActionNone, plan.Action)
}

func TestCursor_PlanNoneWhenInstalled(t *testing.T) {
	provider := CursorProvider{
		Scanner: Scanner{LookPath: func(string) (string, error) { return "/usr/local/bin/cursor", nil }},
	}

	status := provider.Detect()
	status.Plugin.Installed = true
	plan := provider.Plan(status, false)

	require.Equal(t, ActionNone, plan.Action)
}

func TestCursor_ErrorHintForTUIDisable(t *testing.T) {
	provider := cursorTestProvider(t, nil, nil)

	status := provider.Detect()

	require.Contains(t, status.Error, "/add-plugin stripe")
}

func TestCursorWorkspaceURI(t *testing.T) {
	require.Equal(t, "file:///Users/foo/project", cursorWorkspaceURI("/Users/foo/project"))
	require.Equal(t, "file:///C:/Users/foo/project", cursorWorkspaceURI(`C:\Users\foo\project`))
}

func TestParseCursorInstalledIdsOutput(t *testing.T) {
	workspaceURI := "file:///tmp/project"
	output := sqliteInstalledIdsOutput(
		`cursor.plugins.installedIds.1|no-workspace|[{"id":"408","sources":["user"]}]`,
		"cursor.plugins.installedIds.1|"+workspaceURI+"|[]",
	)

	user, project, path := parseCursorInstalledIdsOutput(output, workspaceURI)
	require.True(t, user)
	require.False(t, project)
	require.Empty(t, path)
}

func TestCursorInstalledIdsContainsStripe(t *testing.T) {
	require.True(t, cursorInstalledIdsContainsStripe(`[{"id":"408","sources":["user"]}]`))
	require.False(t, cursorInstalledIdsContainsStripe(`[{"id":"999"}]`))
	require.False(t, cursorInstalledIdsContainsStripe(`{nope`))
}

func cursorTestProviderWithWorkspace(t *testing.T, sqliteOut string) CursorProvider {
	t.Helper()
	workDir := t.TempDir()
	return cursorTestProvider(t, func(name string) (string, error) {
		if name == CursorBinaryName || name == sqlite3BinaryName {
			return "/usr/bin/" + name, nil
		}
		return "", errors.New("missing")
	}, func(context.Context, string, ...string) ([]byte, error) {
		return []byte(sqliteOut), nil
	}, func() (string, error) { return workDir, nil })
}

func cursorTestProvider(t *testing.T, lookPath LookPathFunc, runOutput RunOutputFunc, workDir ...func() (string, error)) CursorProvider {
	t.Helper()

	if lookPath == nil {
		lookPath = func(name string) (string, error) {
			if name == CursorBinaryName {
				return "/usr/local/bin/cursor", nil
			}
			return "", errors.New("missing")
		}
	}

	home := t.TempDir()
	dbPath, err := cursorStateDBPath(home, runtime.GOOS)
	require.NoError(t, err)
	require.NoError(t, os.MkdirAll(filepath.Dir(dbPath), 0o755))
	require.NoError(t, os.WriteFile(dbPath, []byte{}, 0o644))

	scanner := Scanner{
		LookPath: lookPath,
		HomeDir:  func() (string, error) { return home, nil },
		WorkDir:  func() (string, error) { return t.TempDir(), nil },
		ReadFile: os.ReadFile,
		ReadDir:  os.ReadDir,
		Stat:     os.Stat,
	}
	if len(workDir) > 0 {
		scanner.WorkDir = workDir[0]
	}

	return CursorProvider{
		Scanner:   scanner,
		RunOutput: runOutput,
	}
}

func sqliteInstalledIdsOutput(rows ...string) string {
	return strings.Join(rows, "\n")
}

func writeCursorPluginCache(t *testing.T, home, hash, pluginJSON string) {
	t.Helper()
	hashPath := filepath.Join(home, CursorPluginsDir, "cache", CursorMarketplace, CursorPluginName, hash)
	require.NoError(t, os.MkdirAll(filepath.Join(hashPath, ".cursor-plugin"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(hashPath, ".cursor-plugin", "plugin.json"), []byte(pluginJSON), 0o644))
}

type fileInfoStub struct{}

func (fileInfoStub) Name() string       { return "state.vscdb" }
func (fileInfoStub) Size() int64        { return 0 }
func (fileInfoStub) Mode() os.FileMode  { return 0 }
func (fileInfoStub) ModTime() time.Time { return time.Time{} }
func (fileInfoStub) IsDir() bool        { return false }
func (fileInfoStub) Sys() any { return nil }
