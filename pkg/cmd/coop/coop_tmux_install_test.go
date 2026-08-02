package coopcmd

import (
	"errors"
	"io/fs"
	"os"
	"runtime"
	"testing"

	"charm.land/huh/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func stubTmuxInstallEnv(t *testing.T) {
	t.Helper()
	originalCanOffer := canOfferBrewInstall
	originalInstall := runTmuxInstall
	originalStat := statFile
	originalSelect := selectString
	t.Cleanup(func() {
		canOfferBrewInstall = originalCanOffer
		runTmuxInstall = originalInstall
		statFile = originalStat
		selectString = originalSelect
	})
}

func TestOfferTmuxSetupRespectsPersistedDecline(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("windows never offers")
	}
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	stubTmuxInstallEnv(t)
	persistTmuxInstallDecline()
	canOfferBrewInstall = func() bool { return true }
	selectString = func(title string, options []huh.Option[string], value *string) error {
		t.Fatal("declined install must never prompt again")
		return nil
	}

	assert.False(t, (&coopRunCmd{}).offerTmuxSetup())
}

func TestOfferTmuxSetupPersistsNever(t *testing.T) {
	if runtime.GOOS != "darwin" {
		t.Skip("interactive offer is brew/macOS only")
	}
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	stubTmuxInstallEnv(t)
	canOfferBrewInstall = func() bool { return true }
	installCalled := false
	runTmuxInstall = func() error { installCalled = true; return nil }
	selectString = func(title string, options []huh.Option[string], value *string) error {
		require.Len(t, options, 3)
		*value = "never"
		return nil
	}

	assert.False(t, (&coopRunCmd{}).offerTmuxSetup())
	assert.False(t, installCalled)
	assert.True(t, declinedTmuxInstall(), "never must persist across launches")
}

func TestOfferTmuxSetupRunsBrewInstall(t *testing.T) {
	if runtime.GOOS != "darwin" {
		t.Skip("interactive offer is brew/macOS only")
	}
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	stubTmuxInstallEnv(t)
	canOfferBrewInstall = func() bool { return true }
	installCalled := false
	runTmuxInstall = func() error { installCalled = true; return nil }
	selectString = func(title string, options []huh.Option[string], value *string) error {
		*value = "install"
		return nil
	}

	(&coopRunCmd{}).offerTmuxSetup()

	assert.True(t, installCalled)
	assert.False(t, declinedTmuxInstall())
}

func TestTmuxInstallHintPrefersImageFixInContainers(t *testing.T) {
	stubTmuxInstallEnv(t)
	statFile = func(name string) (os.FileInfo, error) {
		if name == "/.dockerenv" {
			return nil, nil
		}
		return nil, fs.ErrNotExist
	}

	hint := tmuxInstallHint()
	assert.Contains(t, hint, "your image", "container installs evaporate on rebuild; the image is the fix")
}

func TestTmuxInstallHintErrsQuietWhenNothingDetected(t *testing.T) {
	stubTmuxInstallEnv(t)
	statFile = func(name string) (os.FileInfo, error) { return nil, fs.ErrNotExist }
	t.Setenv("PATH", t.TempDir()) // no package managers resolvable

	assert.Empty(t, tmuxInstallHint())
}

func TestTmuxInstallHintFailsClosedOnStatError(t *testing.T) {
	stubTmuxInstallEnv(t)
	statFile = func(name string) (os.FileInfo, error) { return nil, errors.New("permission denied") }
	t.Setenv("PATH", t.TempDir())

	assert.Empty(t, tmuxInstallHint())
}

func TestNativeSplitHintPerTerminal(t *testing.T) {
	tests := []struct {
		name     string
		env      map[string]string
		contains string
	}{
		{name: "windows terminal", env: map[string]string{"WT_SESSION": "x"}, contains: "Windows Terminal"},
		{name: "vscode", env: map[string]string{"TERM_PROGRAM": "vscode"}, contains: "VS Code"},
		{name: "iterm2", env: map[string]string{"TERM_PROGRAM": "iTerm.app"}, contains: "iTerm2"},
		{name: "ghostty", env: map[string]string{"TERM_PROGRAM": "ghostty"}, contains: "Ghostty"},
		{name: "warp", env: map[string]string{"TERM_PROGRAM": "WarpTerminal"}, contains: "Warp"},
		{name: "wezterm", env: map[string]string{"TERM_PROGRAM": "WezTerm"}, contains: "WezTerm"},
		{name: "apple terminal", env: map[string]string{"TERM_PROGRAM": "Apple_Terminal"}, contains: "Terminal tab"},
		{name: "kitty", env: map[string]string{"KITTY_WINDOW_ID": "1"}, contains: "kitty"},
		{name: "unknown", env: nil, contains: ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv("WT_SESSION", "")
			t.Setenv("TERM_PROGRAM", "")
			t.Setenv("KITTY_WINDOW_ID", "")
			for k, v := range tt.env {
				t.Setenv(k, v)
			}
			hint := nativeSplitHint()
			if tt.contains == "" {
				assert.Empty(t, hint)
				return
			}
			assert.Contains(t, hint, tt.contains)
		})
	}
}
