package useragent

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
)

func noStat(string) error { return errors.New("not found") }
func yesStat(string) error { return nil }
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
		{"npm via env", "npm", "/any/path", false, false, "npm"},
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
