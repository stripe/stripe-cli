package agentsetup

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestScanCursor_NotDetected(t *testing.T) {
	scanner := Scanner{
		LookPath: func(string) (string, error) { return "", errors.New("missing") },
	}

	status := scanner.ScanCursor()

	require.Equal(t, ClientCursor, status.Client)
	require.False(t, status.Detected)
	require.Equal(t, StatusNotDetected, status.Status)
}

func TestScanCursor_NoPluginsDir(t *testing.T) {
	home := t.TempDir()
	status := cursorTestScanner(home).ScanCursor()

	require.True(t, status.Detected)
	require.False(t, status.Plugin.Installed)
	require.Equal(t, StatusMissing, status.Status)
	require.Equal(t, filepath.Join(home, CursorPluginsDir), status.Plugin.StatePath)
}

func TestScanCursor_PluginInstalled(t *testing.T) {
	home := t.TempDir()
	hashPath := writeCursorPlugin(t, home, "cursor-public", "abc123", `{"name":"stripe","version":"0.1.0"}`, true)

	status := cursorTestScanner(home).ScanCursor()

	require.Equal(t, StatusInstalled, status.Status)
	require.True(t, status.Plugin.Installed)
	require.Equal(t, "stripe@cursor-public", status.Plugin.ID)
	require.Equal(t, "0.1.0", status.Plugin.Version)
	require.Equal(t, "user", status.Plugin.Scope)
	require.Equal(t, hashPath, status.Plugin.StatePath)
}

func TestScanCursor_IncompleteInstallIsMissing(t *testing.T) {
	home := t.TempDir()
	// No .cache-complete marker => install did not finish.
	writeCursorPlugin(t, home, "cursor-public", "abc123", `{"name":"stripe","version":"0.1.0"}`, false)

	status := cursorTestScanner(home).ScanCursor()

	require.Equal(t, StatusMissing, status.Status)
	require.False(t, status.Plugin.Installed)
}

func TestScanCursor_InstalledWithUnreadableMetadataStillDetected(t *testing.T) {
	home := t.TempDir()
	writeCursorPlugin(t, home, "cursor-public", "abc123", `{nope`, true)

	status := cursorTestScanner(home).ScanCursor()

	require.Equal(t, StatusInstalled, status.Status)
	require.True(t, status.Plugin.Installed)
	require.Equal(t, "stripe@cursor-public", status.Plugin.ID)
	require.Empty(t, status.Plugin.Version)
}

func cursorTestScanner(home string) Scanner {
	return Scanner{
		LookPath: func(string) (string, error) { return "/usr/local/bin/cursor", nil },
		ReadFile: os.ReadFile,
		HomeDir:  func() (string, error) { return home, nil },
	}
}

// writeCursorPlugin creates a Cursor plugin cache entry and returns the hash
// directory path. When complete is true it also writes the .cache-complete
// marker that ScanCursor treats as "install finished".
func writeCursorPlugin(t *testing.T, home, marketplace, hash, pluginJSON string, complete bool) string {
	t.Helper()
	hashPath := filepath.Join(home, CursorPluginsDir, "cache", marketplace, CursorPluginName, hash)
	require.NoError(t, os.MkdirAll(filepath.Join(hashPath, ".cursor-plugin"), 0755))
	require.NoError(t, os.WriteFile(filepath.Join(hashPath, ".cursor-plugin", "plugin.json"), []byte(pluginJSON), 0600))
	if complete {
		require.NoError(t, os.WriteFile(filepath.Join(hashPath, ".cache-complete"), nil, 0600))
	}
	return hashPath
}
