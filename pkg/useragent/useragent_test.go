package useragent

import (
	"errors"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func noStat(string) error     { return errors.New("not found") }
func yesStat(string) error    { return nil }
func errExe() (string, error) { return "", errors.New("error") }

func exe(path string) func() (string, error) {
	return func() (string, error) { return path, nil }
}

func env(val string) func(string) string {
	return func(string) string { return val }
}

func noEnv(string) string { return "" }

func TestDetectInstallMethod(t *testing.T) {
	tests := []struct {
		name     string
		envVal   string
		exePath  string
		exeErr   bool
		hasStat  bool
		expected string
	}{
		{"npm_global via env", "npm_global", "/any/path", false, false, "npm_global"},
		{"npm_run via env", "npm_run", "/any/path", false, false, "npm_run"},
		{"npx via env", "npx", "/any/path", false, false, "npx"},
		{"homebrew cellar", "", "/opt/homebrew/Cellar/stripe/1.0/bin/stripe", false, false, "homebrew"},
		{"homebrew usr local cellar", "", "/usr/local/Cellar/stripe/1.0/bin/stripe", false, false, "homebrew"},
		{"scoop", "", "C:/Users/foo/scoop/apps/stripe/current/stripe.exe", false, false, "scoop"},
		{"apt with dpkg file", "", "/usr/bin/stripe", false, true, "apt"},
		{"unknown no dpkg file", "", "/usr/bin/stripe", false, false, "unknown"},
		{"unknown exe error", "", "", true, false, "unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			getEnv := noEnv
			if tt.envVal != "" {
				getEnv = env(tt.envVal)
			}

			getExe := exe(tt.exePath)
			if tt.exeErr {
				getExe = errExe
			}

			statFn := noStat
			if tt.hasStat {
				statFn = yesStat
			}

			result := DetectInstallMethod(getEnv, getExe, statFn)
			require.Equal(t, tt.expected, result)
		})
	}
}

func TestDetectInTmux(t *testing.T) {
	tests := []struct {
		name     string
		envs     map[string]string
		expected bool
	}{
		{"tmux", map[string]string{"TMUX": "/tmp/tmux-501/default,123,0"}, true},
		{"not tmux", map[string]string{}, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := DetectInTmux(mapEnv(tt.envs))
			require.Equal(t, tt.expected, result)
		})
	}
}

func TestDetectInScreen(t *testing.T) {
	tests := []struct {
		name     string
		envs     map[string]string
		expected bool
	}{
		{"screen", map[string]string{"STY": "1234.pts-0.host"}, true},
		{"not screen", map[string]string{}, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := DetectInScreen(mapEnv(tt.envs))
			require.Equal(t, tt.expected, result)
		})
	}
}

func TestDetectTerminalProgram(t *testing.T) {
	tests := []struct {
		name     string
		envs     map[string]string
		expected string
	}{
		{"lc terminal", map[string]string{"LC_TERMINAL": "iTerm2"}, "iTerm2"},
		{"warp", map[string]string{"WARP_CLIENT_VERSION": "v0.2026.06.01"}, "warp"},
		{"windows terminal", map[string]string{"WT_SESSION": "abc"}, "windows_terminal"},
		{"kitty", map[string]string{"KITTY_WINDOW_ID": "1"}, "kitty"},
		{"alacritty window id", map[string]string{"ALACRITTY_WINDOW_ID": "123"}, "alacritty"},
		{"alacritty log", map[string]string{"ALACRITTY_LOG": "/tmp/alacritty.log"}, "alacritty"},
		{"wezterm executable", map[string]string{"WEZTERM_EXECUTABLE": "/Applications/WezTerm.app"}, "wezterm"},
		{"wezterm pane", map[string]string{"WEZTERM_PANE": "1"}, "wezterm"},
		{"ghostty", map[string]string{"GHOSTTY_RESOURCES_DIR": "/Applications/Ghostty.app/Contents/Resources"}, "ghostty"},
		{"term program fallback", map[string]string{"TERM_PROGRAM": "Apple_Terminal"}, "Apple_Terminal"},
		{"specific env wins over term program", map[string]string{"TERM_PROGRAM": "tmux", "LC_TERMINAL": "iTerm2"}, "iTerm2"},
		{"unknown", map[string]string{}, ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := DetectTerminalProgram(mapEnv(tt.envs))
			require.Equal(t, tt.expected, result)
		})
	}
}

func TestDetectAgentHost(t *testing.T) {
	tests := []struct {
		name string
		envs map[string]string
		kind string
		raw  string
	}{
		{"claude desktop", map[string]string{"CLAUDE_CODE_ENTRYPOINT": "claude-desktop"}, "desktop", "claude-desktop"},
		// Both are desktop, and raw is the only thing that tells them apart.
		{"claude desktop 3p", map[string]string{"CLAUDE_CODE_ENTRYPOINT": "claude-desktop-3p"}, "desktop", "claude-desktop-3p"},
		{"codex desktop, normalized from \"Codex Desktop\"", map[string]string{"CODEX_INTERNAL_ORIGINATOR_OVERRIDE": "Codex Desktop"}, "desktop", "codex-desktop"},
		{"codex typescript sdk", map[string]string{"CODEX_INTERNAL_ORIGINATOR_OVERRIDE": "codex_sdk_ts"}, "sdk", "codex-sdk-ts"},
		{"terminal", map[string]string{"CLAUDE_CODE_ENTRYPOINT": "cli"}, "terminal", "cli"},
		{"ide", map[string]string{"CLAUDE_CODE_ENTRYPOINT": "claude-vscode"}, "ide", "claude-vscode"},
		{"sdk ts", map[string]string{"CLAUDE_CODE_ENTRYPOINT": "sdk-ts"}, "sdk", "sdk-ts"},
		{"sdk py", map[string]string{"CLAUDE_CODE_ENTRYPOINT": "sdk-py"}, "sdk", "sdk-py"},
		// remote_desktop is Claude Desktop driving a remotely executing session. The CLI
		// runs on the remote host, so it belongs to remote rather than desktop.
		{"remote desktop is remote", map[string]string{"CLAUDE_CODE_ENTRYPOINT": "remote_desktop"}, "remote", "remote-desktop"},
		{"agent env wins over codex", map[string]string{"CLAUDE_CODE_ENTRYPOINT": "cli", "CODEX_INTERNAL_ORIGINATOR_OVERRIDE": "Codex Desktop"}, "terminal", "cli"},
		{"uncategorized claude host", map[string]string{"CLAUDE_CODE_ENTRYPOINT": "claude-security"}, "other", "claude-security"},
		{"uncategorized codex host", map[string]string{"CODEX_INTERNAL_ORIGINATOR_OVERRIDE": "Codex Web"}, "other", "codex-web"},
		{"no host", map[string]string{}, "", ""},

		// Normalization runs before reporting, so one host cannot arrive under several
		// spellings -- including from different platforms.
		{"raw is normalized, not verbatim", map[string]string{"CLAUDE_CODE_ENTRYPOINT": " Some_New HOST "}, "other", "some-new-host"},
		{"raw is bounded", map[string]string{"CLAUDE_CODE_ENTRYPOINT": strings.Repeat("x", 100)}, "other", strings.Repeat("x", 32)},
		{"unprintable stripped", map[string]string{"CLAUDE_CODE_ENTRYPOINT": "we\x00ird\x7f"}, "other", "weird"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			kind, raw := DetectAgentHost(mapEnv(tt.envs))
			require.Equal(t, tt.kind, kind)
			require.Equal(t, tt.raw, raw)
		})
	}
}

func TestDetectAgentVersion(t *testing.T) {
	tests := []struct {
		name     string
		envs     map[string]string
		expected string
	}{
		{"claude code underscore form", map[string]string{"AI_AGENT": "claude-code_2-1-222_agent"}, "2.1.222"},
		{"claude code harness role", map[string]string{"AI_AGENT": "claude-code_2-1-222_harness"}, "2.1.222"},
		{"claude code slash form", map[string]string{"AI_AGENT": "claude-code/2.1.222"}, "2.1.222"},
		{"agent fallback var", map[string]string{"AGENT": "goose_1-2-3"}, "1.2.3"},
		{"no version reported", map[string]string{"AI_AGENT": "goose"}, ""},
		{"unset", map[string]string{}, ""},
		// A wrong version is worse than a missing one, so anything that is not a plain
		// dot-separated number is rejected rather than guessed at.
		{"non numeric segment rejected", map[string]string{"AI_AGENT": "claude-code_beta_agent"}, ""},
		{"slash form with junk rejected", map[string]string{"AI_AGENT": "claude-code/v2.1.222"}, ""},
		{"trailing dot rejected", map[string]string{"AI_AGENT": "claude-code/2.1."}, ""},
		{"double dot rejected", map[string]string{"AI_AGENT": "claude-code/2..1"}, ""},
		{"overlong rejected", map[string]string{"AI_AGENT": "claude-code/" + strings.Repeat("1.", 40)}, ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := DetectAgentVersion(mapEnv(tt.envs))
			require.Equal(t, tt.expected, result)
		})
	}
}

func mapEnv(envs map[string]string) func(string) string {
	return func(key string) string {
		return envs[key]
	}
}

// TestObservedAgentSessions pins the detectors against environments captured from real
// agent sessions, so a vendor changing what it exports shows up here as a failure
// rather than as a silent gap in reporting.
func TestObservedAgentSessions(t *testing.T) {
	tests := []struct {
		name        string
		envs        map[string]string
		agent       string
		hostKind    string
		hostRaw     string
		version     string
		terminal    string
		description string
	}{
		{
			name: "claude code in claude desktop, macos",
			envs: map[string]string{
				"CLAUDECODE":                  "1",
				"AI_AGENT":                    "claude-code_2-1-222_agent",
				"CLAUDE_CODE_ENTRYPOINT":      "claude-desktop",
				"CLAUDE_AGENT_SDK_VERSION":    "0.3.222",
				"CLAUDE_CODE_SESSION_ID":      sensitiveSessionID,
				"CLAUDE_CODE_HOST_SESSION_ID": sensitiveHostID,
			},
			agent:    "claude_code",
			hostKind: "desktop",
			hostRaw:  "claude-desktop",
			version:  "2.1.222",
		},
		{
			name: "claude code in claude desktop, windows",
			envs: map[string]string{
				"CLAUDECODE":               "1",
				"AI_AGENT":                 "claude-code_2-1-227_agent",
				"CLAUDE_CODE_ENTRYPOINT":   "claude-desktop",
				"CLAUDE_AGENT_SDK_VERSION": "0.3.227",
				"CLAUDE_CODE_SESSION_ID":   sensitiveSessionID,
				"CLAUDE_CODE_OAUTH_SCOPES": sensitiveScopes,
			},
			agent:       "claude_code",
			hostKind:    "desktop",
			hostRaw:     "claude-desktop",
			version:     "2.1.227",
			description: "no __CFBundleIdentifier on Windows; the env vars carry it",
		},
		{
			name: "codex desktop, windows",
			envs: map[string]string{
				"CODEX_CI":                           "1",
				"CODEX_INTERNAL_ORIGINATOR_OVERRIDE": "Codex Desktop",
				"CODEX_THREAD_ID":                    sensitiveThreadID,
				"CODEX_PERMISSION_PROFILE":           ":read-only",
				"CODEX_SANDBOX_NETWORK_DISABLED":     "1",
			},
			agent:       "codex_cli",
			hostKind:    "desktop",
			hostRaw:     "codex-desktop",
			version:     "",
			description: "Desktop also sets the generic Codex signals; the host is what distinguishes it",
		},
		{
			name: "codex typescript sdk",
			envs: map[string]string{
				"CODEX_CI":                           "1",
				"CODEX_INTERNAL_ORIGINATOR_OVERRIDE": "codex_sdk_ts",
				"CODEX_THREAD_ID":                    sensitiveThreadID,
				"TERM_PROGRAM":                       "vscode",
			},
			agent:    "codex_cli",
			hostKind: "sdk",
			hostRaw:  "codex-sdk-ts",
			version:  "",
			terminal: "vscode",
			description: "The TypeScript SDK identifies itself through the Codex originator override " +
				"and forwards inherited terminal metadata, but does not export its version",
		},
		{
			name: "codex cli",
			envs: map[string]string{
				"CODEX_THREAD_ID": sensitiveThreadID,
				"CODEX_SANDBOX":   "1",
			},
			agent:    "codex_cli",
			hostKind: "",
			version:  "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			getEnv := mapEnv(tt.envs)
			require.Equal(t, tt.agent, DetectAIAgent(getEnv), tt.description)
			kind, raw := DetectAgentHost(getEnv)
			require.Equal(t, tt.hostKind, kind, tt.description)
			require.Equal(t, tt.hostRaw, raw, tt.description)
			require.Equal(t, tt.version, DetectAgentVersion(getEnv), tt.description)
			require.Equal(t, tt.terminal, DetectTerminalProgram(getEnv), tt.description)
		})
	}
}

// Distinctive stand-ins for the per-session values these products export, so
// TestObservedAgentSessions_NoSensitiveValuesReported can assert none of them reach a
// reported field.
const (
	sensitiveSessionID = "SESSIONID-1111"
	sensitiveHostID    = "HOSTID-2222"
	sensitiveThreadID  = "THREADID-3333"
	sensitiveScopes    = "SCOPES-4444"
)

func TestObservedAgentSessions_NoSensitiveValuesReported(t *testing.T) {
	// Session, thread, host and scope values must never leave the machine. Several are
	// read as presence checks during detection, so assert on what is reported rather
	// than on what is read.
	envs := map[string]string{
		"CLAUDECODE":                         "1",
		"AI_AGENT":                           "claude-code_2-1-227_agent",
		"CLAUDE_CODE_ENTRYPOINT":             "claude-desktop",
		"CLAUDE_CODE_SESSION_ID":             sensitiveSessionID,
		"CLAUDE_CODE_HOST_SESSION_ID":        sensitiveHostID,
		"CLAUDE_CODE_OAUTH_SCOPES":           sensitiveScopes,
		"CODEX_THREAD_ID":                    sensitiveThreadID,
		"CODEX_INTERNAL_ORIGINATOR_OVERRIDE": "Codex Desktop",
		"CODEX_PERMISSION_PROFILE":           ":read-only",
	}
	getEnv := mapEnv(envs)

	hostKind, hostRaw := DetectAgentHost(getEnv)
	reported := []string{
		hostRaw,
		DetectAIAgent(getEnv),
		hostKind,
		DetectAgentVersion(getEnv),
		DetectInstallMethod(getEnv, errExe, noStat),
		DetectTerminalProgram(getEnv),
	}

	for _, secret := range []string{sensitiveSessionID, sensitiveHostID, sensitiveThreadID, sensitiveScopes} {
		for _, value := range reported {
			require.NotContains(t, value, secret)
		}
	}
}
