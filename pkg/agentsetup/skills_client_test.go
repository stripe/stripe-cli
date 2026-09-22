package agentsetup

import (
	"context"
	"errors"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestSkillsClientProvider(t *testing.T) {
	tests := []struct {
		name        string
		provider    Provider
		binary      string
		client      string
		displayName string
	}{
		{
			name:        "Grok Build",
			provider:    NewGrokProvider(scannerWithBinary("grok", "/usr/local/bin/grok")),
			binary:      "/usr/local/bin/grok",
			client:      ClientGrok,
			displayName: GrokDisplayName,
		},
		{
			name:        "Kimi Code",
			provider:    NewKimiProvider(scannerWithBinary("kimi", "/usr/local/bin/kimi")),
			binary:      "/usr/local/bin/kimi",
			client:      ClientKimi,
			displayName: KimiDisplayName,
		},
		{
			name:        "GitHub Copilot CLI",
			provider:    NewGitHubCopilotProvider(scannerWithBinary("copilot", "/usr/local/bin/copilot")),
			binary:      "/usr/local/bin/copilot",
			client:      ClientGitHubCopilot,
			displayName: GitHubCopilotDisplayName,
		},
		{
			name:        "Visual Studio Code",
			provider:    NewVSCodeProvider(scannerWithBinary("code", "/usr/local/bin/code")),
			binary:      "/usr/local/bin/code",
			client:      ClientVSCode,
			displayName: VSCodeDisplayName,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			status := tc.provider.Detect()

			require.Equal(t, tc.client, status.Client)
			require.Equal(t, tc.displayName, status.DisplayName)
			require.True(t, status.Detected)
			require.Equal(t, tc.binary, status.ExecutablePath)
			require.Equal(t, SetupSkills, status.Setup)
			require.Equal(t, StatusMissing, status.Status)

			plan := tc.provider.Plan(status, false)
			require.Equal(t, ActionInstallSkills, plan.Action)
			require.Equal(t, []string{"stripe", "agent", "setup"}, plan.Command)
			require.NoError(t, tc.provider.Apply(context.Background(), io.Discard, plan))
		})
	}
}

func TestSkillsClientProviderNotDetected(t *testing.T) {
	provider := NewGrokProvider(Scanner{
		LookPath: func(string) (string, error) { return "", errors.New("not found") },
	})

	status := provider.Detect()

	require.False(t, status.Detected)
	require.Equal(t, StatusNotDetected, status.Status)
	require.Equal(t, ActionNone, provider.Plan(status, false).Action)
}

func TestDefaultProvidersCoverPluginsPackageTargets(t *testing.T) {
	providers := DefaultProviders()

	require.Equal(t,
		"claude-code,codex,cursor,github-copilot,grok,kimi,vscode",
		SupportedProviderIDs(providers),
	)
}

func TestVSCodeProviderDetectsInsiders(t *testing.T) {
	provider := NewVSCodeProvider(Scanner{
		LookPath: func(name string) (string, error) {
			if name == "code-insiders" {
				return "/usr/local/bin/code-insiders", nil
			}
			return "", errors.New("not found")
		},
	})

	status := provider.Detect()

	require.True(t, status.Detected)
	require.Equal(t, "/usr/local/bin/code-insiders", status.ExecutablePath)
}

func TestKimiProviderDetectsConfiguredInstallOutsidePath(t *testing.T) {
	kimiHome := t.TempDir()
	binary := "kimi"
	if runtime.GOOS == "windows" {
		binary += ".exe"
	}
	kimiPath := filepath.Join(kimiHome, "bin", binary)
	require.NoError(t, os.MkdirAll(filepath.Dir(kimiPath), 0755))
	require.NoError(t, os.WriteFile(kimiPath, nil, 0755))

	provider := NewKimiProvider(Scanner{
		LookPath: func(string) (string, error) { return "", errors.New("not found") },
		Getenv: func(name string) string {
			if name == "KIMI_CODE_HOME" {
				return kimiHome
			}
			return ""
		},
	})

	status := provider.Detect()

	require.True(t, status.Detected)
	require.Equal(t, kimiPath, status.ExecutablePath)
}

func scannerWithBinary(wantName, path string) Scanner {
	return Scanner{
		LookPath: func(name string) (string, error) {
			if name == wantName {
				return path, nil
			}
			return "", errors.New("not found")
		},
	}
}
