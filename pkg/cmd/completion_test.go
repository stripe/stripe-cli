package cmd

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/stripe/stripe-cli/pkg/ansi"
)

// ---------------------------------------------------------------------------
// Tests from fish-completion branch (preserved)
// ---------------------------------------------------------------------------

func TestDetectShell(t *testing.T) {
	tests := []struct {
		name     string
		envValue string
		expected string
	}{
		{
			name:     "detects bash from /bin/bash",
			envValue: "/bin/bash",
			expected: "bash",
		},
		{
			name:     "detects zsh from /bin/zsh",
			envValue: "/bin/zsh",
			expected: "zsh",
		},
		{
			name:     "detects fish from /usr/bin/fish",
			envValue: "/usr/bin/fish",
			expected: "fish",
		},
		{
			name:     "detects fish from /opt/homebrew/bin/fish",
			envValue: "/opt/homebrew/bin/fish",
			expected: "fish",
		},
		{
			name:     "detects bash even when path contains fish",
			envValue: "/home/fishing/bin/bash",
			expected: "bash",
		},
		{
			name:     "detects zsh even when path contains fish",
			envValue: "/home/shellfish/bin/zsh",
			expected: "zsh",
		},
		{
			name:     "does not false-positive on fish in directory name",
			envValue: "/home/shellfish/bin/csh",
			expected: "",
		},
		{
			name:     "returns empty string for unknown shell",
			envValue: "/bin/csh",
			expected: "",
		},
		{
			name:     "returns empty string when SHELL is empty",
			envValue: "",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv("SHELL", tt.envValue)
			result := detectShell()
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestSelectShellErrors(t *testing.T) {
	t.Run("explicit unsupported shell produces unsupported error", func(t *testing.T) {
		err := selectShell("powershell", true)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "unsupported shell")
		assert.Contains(t, err.Error(), "powershell")
	})

	t.Run("empty shell with no SHELL env produces auto-detect error", func(t *testing.T) {
		t.Setenv("SHELL", "")
		err := selectShell("", true)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "--shell")
	})
}

func TestSelectShellWriteToStdout(t *testing.T) {
	shells := []string{"bash", "zsh", "fish"}
	for _, shell := range shells {
		t.Run(shell, func(t *testing.T) {
			err := selectShell(shell, true)
			assert.NoError(t, err)
		})
	}
}

func TestSelectShellAutoDetectsFish(t *testing.T) {
	t.Setenv("SHELL", "/usr/bin/fish")
	err := selectShell("", true)
	assert.NoError(t, err)
}

// runInTempDir executes fn in a temporary directory, restoring the original
// working directory on cleanup.
func runInTempDir(t *testing.T, fn func()) {
	t.Helper()
	originalWd, err := os.Getwd()
	require.NoError(t, err)

	tmpDir := t.TempDir()
	require.NoError(t, os.Chdir(tmpDir))
	t.Cleanup(func() {
		if err := os.Chdir(originalWd); err != nil {
			t.Errorf("failed to restore working directory: %v", err)
		}
	})

	fn()
}

func TestGenShellCreatesFile(t *testing.T) {
	tests := []struct {
		name         string
		genFunc      func(bool, bool) error
		filename     string
		contentMatch string
	}{
		{
			name:         "bash",
			genFunc:      genBash,
			filename:     "stripe-completion.bash",
			contentMatch: "bash completion",
		},
		{
			name:         "zsh",
			genFunc:      genZsh,
			filename:     "stripe-completion.zsh",
			contentMatch: "zsh completion",
		},
		{
			name:         "fish",
			genFunc:      genFish,
			filename:     "stripe.fish",
			contentMatch: "fish completion for stripe",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			runInTempDir(t, func() {
				err := tt.genFunc(false, false)
				require.NoError(t, err)

				content, err := os.ReadFile(tt.filename)
				require.NoError(t, err)
				assert.NotEmpty(t, content)
				assert.Contains(t, string(content), tt.contentMatch)
			})
		})
	}
}

// ---------------------------------------------------------------------------
// computeAddSentinel (pure function — no I/O)
// ---------------------------------------------------------------------------

func TestComputeAddSentinelToEmptyContent(t *testing.T) {
	result := computeAddSentinel("", "source /home/user/.stripe/stripe-completion.zsh")
	assert.Contains(t, result, sentinelBegin)
	assert.Contains(t, result, "source /home/user/.stripe/stripe-completion.zsh")
	assert.Contains(t, result, sentinelEnd)
}

func TestComputeAddSentinelPreservesExistingContent(t *testing.T) {
	existing := "export PATH=/usr/local/bin:$PATH\nalias ll='ls -la'\n"
	result := computeAddSentinel(existing, "source /home/user/.stripe/stripe-completion.zsh")
	assert.True(t, strings.HasPrefix(result, existing), "existing content should be preserved at the start")
	assert.Contains(t, result, sentinelBegin)
	assert.Contains(t, result, sentinelEnd)
}

func TestComputeAddSentinelReplacesExisting(t *testing.T) {
	content := fmt.Sprintf("before\n%s\nold source line\n%s\nafter\n", sentinelBegin, sentinelEnd)
	result := computeAddSentinel(content, "new source line")
	assert.Contains(t, result, "before\n")
	assert.Contains(t, result, "new source line")
	assert.NotContains(t, result, "old source line")
	assert.Contains(t, result, "after\n")
	assert.Equal(t, 1, strings.Count(result, sentinelBegin))
	assert.Equal(t, 1, strings.Count(result, sentinelEnd))
}

func TestComputeAddSentinelAppendsNewlineIfMissing(t *testing.T) {
	result := computeAddSentinel("no trailing newline", "source line")
	assert.Contains(t, result, "no trailing newline\n"+sentinelBegin)
}

func TestComputeAddSentinelOrphanedBeginOnly(t *testing.T) {
	content := fmt.Sprintf("before\n%s\norphaned source line\nafter\n", sentinelBegin)
	result := computeAddSentinel(content, "new source line")
	assert.Contains(t, result, "new source line")
	assert.Contains(t, result, sentinelEnd)
}

func TestComputeAddSentinelOrphanedEndOnly(t *testing.T) {
	content := fmt.Sprintf("before\n%s\nafter\n", sentinelEnd)
	result := computeAddSentinel(content, "new source line")
	assert.Contains(t, result, "new source line")
	assert.Equal(t, 1, strings.Count(result, sentinelBegin))
}

func TestComputeAddSentinelReversedMarkers(t *testing.T) {
	content := fmt.Sprintf("before\n%s\norphaned\n%s\nafter\n", sentinelEnd, sentinelBegin)
	result := computeAddSentinel(content, "new source line")
	assert.Contains(t, result, "new source line")
}

func TestComputeAddSentinelIdempotent(t *testing.T) {
	line := "source /home/user/.stripe/stripe-completion.zsh"
	once := computeAddSentinel("", line)
	twice := computeAddSentinel(once, line)
	assert.Equal(t, once, twice, "applying computeAddSentinel twice should produce the same result")
}

// ---------------------------------------------------------------------------
// computeRemoveSentinel (pure function — no I/O)
// ---------------------------------------------------------------------------

func TestComputeRemoveSentinelOnlyBlock(t *testing.T) {
	content := fmt.Sprintf("%s\nsource line\n%s\n", sentinelBegin, sentinelEnd)
	result, found := computeRemoveSentinel(content)
	require.True(t, found)
	assert.Equal(t, "", result, "removing the only content should yield empty string")
}

func TestComputeRemoveSentinelPreservesOtherContent(t *testing.T) {
	content := fmt.Sprintf("before\n%s\nsource line\n%s\nafter\n", sentinelBegin, sentinelEnd)
	result, found := computeRemoveSentinel(content)
	require.True(t, found)
	assert.Contains(t, result, "before\n")
	assert.Contains(t, result, "after\n")
	assert.NotContains(t, result, sentinelBegin)
	assert.NotContains(t, result, "source line")
}

func TestComputeRemoveSentinelNoBlockPresent(t *testing.T) {
	_, found := computeRemoveSentinel("export FOO=bar\n")
	assert.False(t, found)
}

func TestComputeRemoveSentinelEmptyContent(t *testing.T) {
	_, found := computeRemoveSentinel("")
	assert.False(t, found)
}

func TestComputeRemoveSentinelOrphanedBeginOnly(t *testing.T) {
	content := fmt.Sprintf("before\n%s\norphaned\nafter\n", sentinelBegin)
	_, found := computeRemoveSentinel(content)
	assert.False(t, found)
}

func TestComputeRemoveSentinelOrphanedEndOnly(t *testing.T) {
	content := fmt.Sprintf("before\n%s\nafter\n", sentinelEnd)
	_, found := computeRemoveSentinel(content)
	assert.False(t, found)
}

func TestComputeRemoveSentinelReversedMarkers(t *testing.T) {
	content := fmt.Sprintf("before\n%s\norphaned\n%s\nafter\n", sentinelEnd, sentinelBegin)
	_, found := computeRemoveSentinel(content)
	assert.False(t, found, "reversed markers should not be treated as a valid block")
}

// ---------------------------------------------------------------------------
// readConfigFile
// ---------------------------------------------------------------------------

func TestReadConfigFileMissing(t *testing.T) {
	content, perm, err := readConfigFile(filepath.Join(t.TempDir(), "nonexistent"))
	require.NoError(t, err)
	assert.Equal(t, "", content)
	assert.Equal(t, os.FileMode(0644), perm)
}

func TestReadConfigFilePreservesPermissions(t *testing.T) {
	if runtime.GOOS == "windows" || os.Getuid() == 0 {
		t.Skip("Cannot test Unix file permissions on Windows or as root")
	}

	dir := t.TempDir()
	path := filepath.Join(dir, ".zshrc")
	require.NoError(t, os.WriteFile(path, []byte("content\n"), 0600))

	content, perm, err := readConfigFile(path)
	require.NoError(t, err)
	assert.Equal(t, "content\n", content)
	assert.Equal(t, os.FileMode(0600), perm)
}

func TestReadConfigFilePermissionDenied(t *testing.T) {
	if runtime.GOOS == "windows" || os.Getuid() == 0 {
		t.Skip("Cannot test Unix file permissions on Windows or as root")
	}

	dir := t.TempDir()
	path := filepath.Join(dir, ".zshrc")
	require.NoError(t, os.WriteFile(path, []byte("content"), 0644))
	require.NoError(t, os.Chmod(path, 0000))
	t.Cleanup(func() { os.Chmod(path, 0644) })

	_, _, err := readConfigFile(path)
	assert.Error(t, err)
}

// ---------------------------------------------------------------------------
// findManualRemnants (from sentinel-block-management branch, preserved)
// ---------------------------------------------------------------------------

func TestFindManualRemnantsDetectsManualSourceLine(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, ".zshrc")
	content := "export PATH=/usr/local/bin:$PATH\nsource ~/.stripe/stripe-completion.zsh\nalias ls='ls -G'\n"
	require.NoError(t, os.WriteFile(configPath, []byte(content), 0644))

	remnants := findManualRemnants(configPath, "stripe-completion.zsh")
	require.Len(t, remnants, 1)
	assert.Equal(t, 2, remnants[0].lineNumber)
	assert.Contains(t, remnants[0].lineText, "stripe-completion.zsh")
}

func TestFindManualRemnantsDetectsDotSourceSyntax(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, ".bashrc")
	require.NoError(t, os.WriteFile(configPath, []byte(". /some/custom/path/stripe-completion.bash\n"), 0644))

	remnants := findManualRemnants(configPath, "stripe-completion.bash")
	require.Len(t, remnants, 1)
	assert.Equal(t, 1, remnants[0].lineNumber)
}

func TestFindManualRemnantsDetectsLineWithOtherCommands(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, ".zshrc")
	require.NoError(t, os.WriteFile(configPath, []byte("[ -f ~/.stripe/stripe-completion.zsh ] && source ~/.stripe/stripe-completion.zsh\n"), 0644))

	remnants := findManualRemnants(configPath, "stripe-completion.zsh")
	require.Len(t, remnants, 1)
}

func TestFindManualRemnantsDetectsCustomPath(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, ".zshrc")
	require.NoError(t, os.WriteFile(configPath, []byte("source /opt/completions/stripe-completion.zsh\n"), 0644))

	remnants := findManualRemnants(configPath, "stripe-completion.zsh")
	require.Len(t, remnants, 1)
}

func TestFindManualRemnantsIgnoresSentinelBlock(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, ".zshrc")
	content := fmt.Sprintf("before\n%s\nsource ~/.stripe/stripe-completion.zsh\n%s\nafter\n", sentinelBegin, sentinelEnd)
	require.NoError(t, os.WriteFile(configPath, []byte(content), 0644))

	remnants := findManualRemnants(configPath, "stripe-completion.zsh")
	assert.Empty(t, remnants)
}

func TestFindManualRemnantsIgnoresComments(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, ".zshrc")
	require.NoError(t, os.WriteFile(configPath, []byte("# source ~/.stripe/stripe-completion.zsh\n"), 0644))

	remnants := findManualRemnants(configPath, "stripe-completion.zsh")
	assert.Empty(t, remnants)
}

func TestFindManualRemnantsReturnsNilForMissingFile(t *testing.T) {
	remnants := findManualRemnants(filepath.Join(t.TempDir(), "nonexistent"), "stripe-completion.zsh")
	assert.Nil(t, remnants)
}

func TestFindManualRemnantsNoMatchReturnsNil(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, ".zshrc")
	require.NoError(t, os.WriteFile(configPath, []byte("export PATH=/usr/local/bin:$PATH\n"), 0644))

	remnants := findManualRemnants(configPath, "stripe-completion.zsh")
	assert.Nil(t, remnants)
}

func TestFindManualRemnantsMultipleMatches(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, ".zshrc")
	require.NoError(t, os.WriteFile(configPath, []byte("source ~/.stripe/stripe-completion.zsh\nexport FOO=bar\n. /other/stripe-completion.zsh\n"), 0644))

	remnants := findManualRemnants(configPath, "stripe-completion.zsh")
	require.Len(t, remnants, 2)
	assert.Equal(t, 1, remnants[0].lineNumber)
	assert.Equal(t, 3, remnants[1].lineNumber)
}

func TestFindManualRemnantsOutsideSentinelWithManualBefore(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, ".zshrc")
	content := fmt.Sprintf("source ~/my/stripe-completion.zsh\n%s\nsource ~/.stripe/stripe-completion.zsh\n%s\n", sentinelBegin, sentinelEnd)
	require.NoError(t, os.WriteFile(configPath, []byte(content), 0644))

	remnants := findManualRemnants(configPath, "stripe-completion.zsh")
	require.Len(t, remnants, 1)
	assert.Equal(t, 1, remnants[0].lineNumber)
}

func TestFindManualRemnantsOutsideSentinelWithManualAfter(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, ".zshrc")
	content := fmt.Sprintf("%s\nsource ~/.stripe/stripe-completion.zsh\n%s\nsource ~/custom/stripe-completion.zsh\n", sentinelBegin, sentinelEnd)
	require.NoError(t, os.WriteFile(configPath, []byte(content), 0644))

	remnants := findManualRemnants(configPath, "stripe-completion.zsh")
	require.Len(t, remnants, 1)
	assert.Equal(t, 4, remnants[0].lineNumber)
}

// ---------------------------------------------------------------------------
// Test helpers for install/uninstall
// ---------------------------------------------------------------------------

func fakeHomeDir(dir string) homeDirFunc {
	return func() (string, error) { return dir, nil }
}

func failingHomeDir() homeDirFunc {
	return func() (string, error) { return "", fmt.Errorf("no home directory") }
}

// disableColors suppresses ANSI color output during tests.
func disableColors(t *testing.T) {
	t.Helper()
	ansi.DisableColors = true
	t.Cleanup(func() { ansi.DisableColors = false })
}

// alwaysConfirm overrides installConfirmFn and uninstallConfirmFn to always
// return true (auto-accept). Restores originals on cleanup.
func alwaysConfirm(t *testing.T) {
	t.Helper()
	disableColors(t)
	origInstall := installConfirmFn
	origUninstall := uninstallConfirmFn
	accept := func(_ string) bool { return true }
	installConfirmFn = accept
	uninstallConfirmFn = accept
	t.Cleanup(func() {
		installConfirmFn = origInstall
		uninstallConfirmFn = origUninstall
	})
}

// neverConfirm overrides installConfirmFn and uninstallConfirmFn to always
// return false (auto-decline).
func neverConfirm(t *testing.T) {
	t.Helper()
	disableColors(t)
	origInstall := installConfirmFn
	origUninstall := uninstallConfirmFn
	decline := func(_ string) bool { return false }
	installConfirmFn = decline
	uninstallConfirmFn = decline
	t.Cleanup(func() {
		installConfirmFn = origInstall
		uninstallConfirmFn = origUninstall
	})
}

// ---------------------------------------------------------------------------
// generateCompletionScript
// ---------------------------------------------------------------------------

func TestGenerateCompletionScriptBash(t *testing.T) {
	var buf bytes.Buffer
	err := generateCompletionScript("bash", &buf)
	require.NoError(t, err)
	assert.Contains(t, buf.String(), "__complete")
}

func TestGenerateCompletionScriptZsh(t *testing.T) {
	var buf bytes.Buffer
	err := generateCompletionScript("zsh", &buf)
	require.NoError(t, err)
	assert.NotEmpty(t, buf.String())
}

func TestGenerateCompletionScriptFish(t *testing.T) {
	var buf bytes.Buffer
	err := generateCompletionScript("fish", &buf)
	require.NoError(t, err)
	assert.NotEmpty(t, buf.String())
}

func TestGenerateCompletionScriptUnsupported(t *testing.T) {
	var buf bytes.Buffer
	err := generateCompletionScript("powershell", &buf)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unsupported shell")
}

// ---------------------------------------------------------------------------
// installCompletion
// ---------------------------------------------------------------------------

func TestInstallCompletionZsh(t *testing.T) {
	alwaysConfirm(t)
	home := t.TempDir()
	err := installCompletion("zsh", fakeHomeDir(home))
	require.NoError(t, err)

	scriptPath := filepath.Join(home, ".stripe", "stripe-completion.zsh")
	data, err := os.ReadFile(scriptPath)
	require.NoError(t, err)
	assert.NotEmpty(t, data)

	configPath := filepath.Join(home, ".zshrc")
	configData, err := os.ReadFile(configPath)
	require.NoError(t, err)
	content := string(configData)
	assert.Contains(t, content, sentinelBegin)
	assert.Contains(t, content, fmt.Sprintf("source \"%s\"", scriptPath))
	assert.Contains(t, content, sentinelEnd)
}

func TestInstallCompletionBash(t *testing.T) {
	alwaysConfirm(t)
	home := t.TempDir()
	err := installCompletion("bash", fakeHomeDir(home))
	require.NoError(t, err)

	scriptPath := filepath.Join(home, ".stripe", "stripe-completion.bash")
	data, err := os.ReadFile(scriptPath)
	require.NoError(t, err)
	assert.NotEmpty(t, data)

	var configPath string
	if runtime.GOOS == "darwin" {
		configPath = filepath.Join(home, ".bash_profile")
	} else {
		configPath = filepath.Join(home, ".bashrc")
	}
	configData, err := os.ReadFile(configPath)
	require.NoError(t, err)
	assert.Contains(t, string(configData), sentinelBegin)
	assert.Contains(t, string(configData), fmt.Sprintf("source \"%s\"", scriptPath))
}

func TestInstallCompletionFish(t *testing.T) {
	home := t.TempDir()
	err := installCompletion("fish", fakeHomeDir(home))
	require.NoError(t, err)

	scriptPath := filepath.Join(home, ".config", "fish", "completions", "stripe.fish")
	data, err := os.ReadFile(scriptPath)
	require.NoError(t, err)
	assert.NotEmpty(t, data)

	// Fish should not modify any shell config files
	for _, f := range []string{".zshrc", ".bashrc", ".bash_profile"} {
		_, err := os.Stat(filepath.Join(home, f))
		assert.True(t, os.IsNotExist(err), "fish install should not create %s", f)
	}
}

func TestInstallCompletionIdempotent(t *testing.T) {
	alwaysConfirm(t)
	home := t.TempDir()
	require.NoError(t, installCompletion("zsh", fakeHomeDir(home)))
	require.NoError(t, installCompletion("zsh", fakeHomeDir(home)))

	configPath := filepath.Join(home, ".zshrc")
	data, err := os.ReadFile(configPath)
	require.NoError(t, err)
	content := string(data)
	assert.Equal(t, 1, strings.Count(content, sentinelBegin), "sentinel begin should appear exactly once")
	assert.Equal(t, 1, strings.Count(content, sentinelEnd), "sentinel end should appear exactly once")
}

func TestInstallCompletionHomeDirError(t *testing.T) {
	err := installCompletion("zsh", failingHomeDir())
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "could not determine home directory")
}

func TestInstallCompletionWritePermissionDenied(t *testing.T) {
	if os.Getuid() == 0 {
		t.Skip("Cannot test permission errors as root")
	}

	home := t.TempDir()
	scriptDir := filepath.Join(home, ".stripe")
	require.NoError(t, os.MkdirAll(scriptDir, 0755))
	require.NoError(t, os.Chmod(scriptDir, 0555))
	t.Cleanup(func() { os.Chmod(scriptDir, 0755) })

	err := installCompletion("zsh", fakeHomeDir(home))
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "could not write completion script")
}

func TestInstallUnsupportedShell(t *testing.T) {
	cc := newCompletionCmd()
	cc.cmd.SetArgs([]string{"--install", "--shell", "powershell"})

	err := cc.cmd.Execute()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unsupported shell")
	assert.Contains(t, err.Error(), "powershell")
}

func TestInstallMutuallyExclusiveFlags(t *testing.T) {
	cc := newCompletionCmd()
	cc.cmd.SetArgs([]string{"--install", "--uninstall"})

	err := cc.cmd.Execute()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "if any flags in the group [install uninstall] are set none of the others can be")
}

// ---------------------------------------------------------------------------
// confirmation prompt
// ---------------------------------------------------------------------------

func TestInstallDeclinedDoesNotModifyConfig(t *testing.T) {
	neverConfirm(t)
	home := t.TempDir()

	err := installCompletion("zsh", fakeHomeDir(home))
	require.NoError(t, err) // declining is not an error

	// Script file should still be written (it's written before the prompt)
	scriptPath := filepath.Join(home, ".stripe", "stripe-completion.zsh")
	_, err = os.Stat(scriptPath)
	assert.NoError(t, err, "script file should exist even when config change is declined")

	// Config file should NOT have been created or modified
	configPath := filepath.Join(home, ".zshrc")
	_, err = os.Stat(configPath)
	assert.True(t, os.IsNotExist(err), "config file should not exist when user declines")
}

func TestUninstallDeclinedDoesNotModifyConfig(t *testing.T) {
	// First install with confirmation
	alwaysConfirm(t)
	home := t.TempDir()
	require.NoError(t, installCompletion("zsh", fakeHomeDir(home)))

	// Now decline the uninstall
	neverConfirm(t)
	require.NoError(t, uninstallCompletion("zsh", fakeHomeDir(home)))

	// Script file should be removed (removed before the prompt)
	scriptPath := filepath.Join(home, ".stripe", "stripe-completion.zsh")
	_, err := os.Stat(scriptPath)
	assert.True(t, os.IsNotExist(err), "script file should be removed regardless of config prompt")

	// Config file should still have the sentinel block
	configPath := filepath.Join(home, ".zshrc")
	data, err := os.ReadFile(configPath)
	require.NoError(t, err)
	assert.Contains(t, string(data), sentinelBegin, "config should be unchanged when user declines")
}

func TestInstallFishSkipsConfirmation(t *testing.T) {
	// Use neverConfirm to prove fish doesn't hit the prompt
	neverConfirm(t)
	home := t.TempDir()

	err := installCompletion("fish", fakeHomeDir(home))
	require.NoError(t, err)

	scriptPath := filepath.Join(home, ".config", "fish", "completions", "stripe.fish")
	_, err = os.Stat(scriptPath)
	assert.NoError(t, err, "fish install should succeed without confirmation")
}

// ---------------------------------------------------------------------------
// uninstallCompletion
// ---------------------------------------------------------------------------

func TestUninstallCompletion(t *testing.T) {
	alwaysConfirm(t)
	home := t.TempDir()
	require.NoError(t, installCompletion("zsh", fakeHomeDir(home)))

	scriptPath := filepath.Join(home, ".stripe", "stripe-completion.zsh")
	_, err := os.Stat(scriptPath)
	require.NoError(t, err)

	require.NoError(t, uninstallCompletion("zsh", fakeHomeDir(home)))

	_, err = os.Stat(scriptPath)
	assert.True(t, os.IsNotExist(err))

	configPath := filepath.Join(home, ".zshrc")
	data, err := os.ReadFile(configPath)
	require.NoError(t, err)
	content := string(data)
	assert.NotContains(t, content, sentinelBegin)
	assert.NotContains(t, content, sentinelEnd)
}

func TestUninstallCompletionFish(t *testing.T) {
	home := t.TempDir()
	require.NoError(t, installCompletion("fish", fakeHomeDir(home)))

	scriptPath := filepath.Join(home, ".config", "fish", "completions", "stripe.fish")
	_, err := os.Stat(scriptPath)
	require.NoError(t, err)

	require.NoError(t, uninstallCompletion("fish", fakeHomeDir(home)))
	_, err = os.Stat(scriptPath)
	assert.True(t, os.IsNotExist(err))
}

func TestUninstallWhenNotInstalled(t *testing.T) {
	alwaysConfirm(t)
	home := t.TempDir()
	err := uninstallCompletion("zsh", fakeHomeDir(home))
	assert.NoError(t, err)
}

func TestUninstallPreservesExistingConfigContent(t *testing.T) {
	alwaysConfirm(t)
	home := t.TempDir()
	configPath := filepath.Join(home, ".zshrc")
	existing := "export PATH=/usr/local/bin:$PATH\nalias ll='ls -la'\n"
	require.NoError(t, os.WriteFile(configPath, []byte(existing), 0644))

	require.NoError(t, installCompletion("zsh", fakeHomeDir(home)))
	require.NoError(t, uninstallCompletion("zsh", fakeHomeDir(home)))

	data, err := os.ReadFile(configPath)
	require.NoError(t, err)
	content := string(data)
	assert.Contains(t, content, "export PATH=/usr/local/bin:$PATH")
	assert.Contains(t, content, "alias ll='ls -la'")
	assert.NotContains(t, content, sentinelBegin)
}

func TestUninstallCompletionHomeDirError(t *testing.T) {
	err := uninstallCompletion("zsh", failingHomeDir())
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "could not determine home directory")
}

// ---------------------------------------------------------------------------
// install/uninstall with manual remnant warnings
// ---------------------------------------------------------------------------

func TestInstallWarnsAboutManualRemnants(t *testing.T) {
	alwaysConfirm(t)
	home := t.TempDir()
	configPath := filepath.Join(home, ".zshrc")
	existing := "export PATH=/usr/local/bin:$PATH\nsource ~/.stripe/stripe-completion.zsh\nalias ll='ls -la'\n"
	require.NoError(t, os.WriteFile(configPath, []byte(existing), 0644))

	err := installCompletion("zsh", fakeHomeDir(home))
	require.NoError(t, err)

	data, err := os.ReadFile(configPath)
	require.NoError(t, err)
	content := string(data)
	assert.Contains(t, content, sentinelBegin)
	assert.Contains(t, content, "source ~/.stripe/stripe-completion.zsh")

	remnants := findManualRemnants(configPath, "stripe-completion.zsh")
	require.Len(t, remnants, 1, "the pre-existing manual line should be detected as a remnant")
	assert.Equal(t, 2, remnants[0].lineNumber)
}

func TestUninstallWarnsAboutManualRemnants(t *testing.T) {
	alwaysConfirm(t)
	home := t.TempDir()
	configPath := filepath.Join(home, ".zshrc")
	require.NoError(t, os.WriteFile(configPath, []byte("source ~/.stripe/stripe-completion.zsh\n"), 0644))
	require.NoError(t, installCompletion("zsh", fakeHomeDir(home)))
	require.NoError(t, uninstallCompletion("zsh", fakeHomeDir(home)))

	data, err := os.ReadFile(configPath)
	require.NoError(t, err)
	content := string(data)
	assert.NotContains(t, content, sentinelBegin)
	assert.Contains(t, content, "source ~/.stripe/stripe-completion.zsh")

	remnants := findManualRemnants(configPath, "stripe-completion.zsh")
	require.Len(t, remnants, 1, "the pre-existing manual line should survive uninstall")
}

func TestInstallNoWarningWhenClean(t *testing.T) {
	alwaysConfirm(t)
	home := t.TempDir()
	require.NoError(t, installCompletion("zsh", fakeHomeDir(home)))

	configPath := filepath.Join(home, ".zshrc")
	remnants := findManualRemnants(configPath, "stripe-completion.zsh")
	assert.Empty(t, remnants)
}

func TestUninstallNoWarningWhenClean(t *testing.T) {
	alwaysConfirm(t)
	home := t.TempDir()
	require.NoError(t, installCompletion("zsh", fakeHomeDir(home)))
	require.NoError(t, uninstallCompletion("zsh", fakeHomeDir(home)))

	configPath := filepath.Join(home, ".zshrc")
	remnants := findManualRemnants(configPath, "stripe-completion.zsh")
	assert.Empty(t, remnants)
}

// ---------------------------------------------------------------------------
// completionScriptFilename
// ---------------------------------------------------------------------------

func TestCompletionScriptFilename(t *testing.T) {
	assert.Equal(t, "stripe-completion.bash", completionScriptFilename("bash"))
	assert.Equal(t, "stripe-completion.zsh", completionScriptFilename("zsh"))
	assert.Equal(t, "stripe.fish", completionScriptFilename("fish"))
	assert.Equal(t, "", completionScriptFilename("powershell"))
}
