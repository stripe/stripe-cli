package useragent

import (
	"errors"
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

func mapEnv(envs map[string]string) func(string) string {
	return func(key string) string {
		return envs[key]
	}
}
