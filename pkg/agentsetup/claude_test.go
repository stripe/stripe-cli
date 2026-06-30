package agentsetup

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestScanClaude_NotDetected(t *testing.T) {
	scanner := Scanner{
		LookPath: func(string) (string, error) { return "", errors.New("missing") },
	}

	status := scanner.ScanClaude()

	require.Equal(t, ClientClaudeCode, status.Client)
	require.False(t, status.Detected)
	require.Equal(t, StatusNotDetected, status.Status)
}

func TestScanClaude_MissingPluginState(t *testing.T) {
	home := t.TempDir()
	scanner := testScanner(home)

	status := scanner.ScanClaude()

	require.True(t, status.Detected)
	require.False(t, status.Plugin.Installed)
	require.Equal(t, StatusMissing, status.Status)
	require.Equal(t, filepath.Join(home, ClaudePluginStatePath), status.Plugin.StatePath)
}

func TestScanClaude_OfficialPluginInstalled(t *testing.T) {
	home := t.TempDir()
	writeClaudePluginState(t, home, `{
		"version": 2,
		"plugins": {
			"stripe@claude-plugins-official": [
				{"scope": "user", "version": "2.4.1", "installPath": "/tmp/stripe"}
			]
		}
	}`)

	status := testScanner(home).ScanClaude()

	require.Equal(t, StatusInstalled, status.Status)
	require.True(t, status.Plugin.Installed)
	require.Equal(t, TargetClaudePlugin, status.Plugin.ID)
	require.Equal(t, "2.4.1", status.Plugin.Version)
	require.Equal(t, "user", status.Plugin.Scope)
}

func TestScanClaude_LocalStripePluginInstalled(t *testing.T) {
	home := t.TempDir()
	projectPath := filepath.Join(home, "project")
	writeClaudePluginState(t, home, `{
		"version": 2,
		"plugins": {
			"stripe@stripe": [
				{"scope": "local", "version": "0.1.0", "installPath": "/tmp/stripe", "projectPath": "`+projectPath+`"}
			]
		}
	}`)

	status := testScannerWithWorkDir(home, filepath.Join(projectPath, "subdir")).ScanClaude()

	require.Equal(t, StatusInstalled, status.Status)
	require.True(t, status.Plugin.Installed)
	require.Equal(t, LocalClaudePlugin, status.Plugin.ID)
	require.Equal(t, "0.1.0", status.Plugin.Version)
	require.Equal(t, "local", status.Plugin.Scope)
	require.Equal(t, projectPath, status.Plugin.Project)
}

func TestScanClaude_LocalStripePluginForDifferentProjectIsMissing(t *testing.T) {
	home := t.TempDir()
	projectPath := filepath.Join(home, "other-project")
	writeClaudePluginState(t, home, `{
		"version": 2,
		"plugins": {
			"stripe@stripe": [
				{"scope": "local", "version": "0.1.0", "installPath": "/tmp/stripe", "projectPath": "`+projectPath+`"}
			]
		}
	}`)

	status := testScannerWithWorkDir(home, filepath.Join(home, "current-project")).ScanClaude()

	require.Equal(t, StatusMissing, status.Status)
	require.False(t, status.Plugin.Installed)
}

func TestScanClaude_MalformedPluginState(t *testing.T) {
	home := t.TempDir()
	writeClaudePluginState(t, home, `{nope`)

	status := testScanner(home).ScanClaude()

	require.Equal(t, StatusError, status.Status)
	require.Contains(t, status.Error, "parsing Claude plugin state")
}

func testScanner(home string) Scanner {
	return testScannerWithWorkDir(home, home)
}

func testScannerWithWorkDir(home, workDir string) Scanner {
	return Scanner{
		LookPath: func(string) (string, error) { return "/usr/local/bin/claude", nil },
		ReadFile: os.ReadFile,
		HomeDir:  func() (string, error) { return home, nil },
		WorkDir:  func() (string, error) { return workDir, nil },
	}
}

func writeClaudePluginState(t *testing.T, home string, body string) {
	t.Helper()
	path := filepath.Join(home, ClaudePluginStatePath)
	require.NoError(t, os.MkdirAll(filepath.Dir(path), 0755))
	require.NoError(t, os.WriteFile(path, []byte(body), 0600))
}
