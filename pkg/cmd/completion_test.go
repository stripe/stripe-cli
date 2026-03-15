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
		{name: "detects bash from /bin/bash", envValue: "/bin/bash", expected: "bash"},
		{name: "detects zsh from /bin/zsh", envValue: "/bin/zsh", expected: "zsh"},
		{name: "detects fish from /usr/bin/fish", envValue: "/usr/bin/fish", expected: "fish"},
		{name: "detects fish from /opt/homebrew/bin/fish", envValue: "/opt/homebrew/bin/fish", expected: "fish"},
		{name: "bash takes precedence when path also contains fish", envValue: "/home/fishing/bin/bash", expected: "bash"},
		{name: "zsh takes precedence when path also contains fish", envValue: "/home/shellfish/bin/zsh", expected: "zsh"},
		{name: "returns empty string for unknown shell", envValue: "/bin/csh", expected: ""},
		{name: "returns empty string when SHELL is empty", envValue: "", expected: ""},
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
	tests := []struct {
		name  string
		shell string
	}{
		{name: "unknown shell name produces error", shell: "powershell"},
		{name: "empty shell with no SHELL env produces error", shell: ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.shell == "" {
				t.Setenv("SHELL", "")
			}
			err := selectShell(tt.shell, true)
			require.Error(t, err)
			assert.Contains(t, err.Error(), "--shell")
		})
	}
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

func TestGenFishCreatesFile(t *testing.T) {
	originalWd, err := os.Getwd()
	require.NoError(t, err)

	tmpDir := t.TempDir()
	require.NoError(t, os.Chdir(tmpDir))
	t.Cleanup(func() {
		if err := os.Chdir(originalWd); err != nil {
			t.Errorf("failed to restore working directory: %v", err)
		}
	})

	err = genFish(false)
	require.NoError(t, err)

	content, err := os.ReadFile("stripe.fish")
	require.NoError(t, err)
	assert.NotEmpty(t, content)
	assert.Contains(t, string(content), "fish completion for stripe")
}

// ---------------------------------------------------------------------------
// Sentinel block tests (from sentinel-block-management branch, preserved)
// ---------------------------------------------------------------------------

func TestAddSentinelBlockToNewFile(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, ".zshrc")

	err := addSentinelBlock(configPath, "source /home/user/.stripe/stripe-completion.zsh")
	require.NoError(t, err)

	data, err := os.ReadFile(configPath)
	require.NoError(t, err)
	content := string(data)
	assert.Contains(t, content, sentinelBegin)
	assert.Contains(t, content, "source /home/user/.stripe/stripe-completion.zsh")
	assert.Contains(t, content, sentinelEnd)
}

func TestAddSentinelBlockPreservesExistingContent(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, ".zshrc")

	existing := "export PATH=/usr/local/bin:$PATH\nalias ll='ls -la'\n"
	require.NoError(t, os.WriteFile(configPath, []byte(existing), 0644))

	err := addSentinelBlock(configPath, "source /home/user/.stripe/stripe-completion.zsh")
	require.NoError(t, err)

	data, err := os.ReadFile(configPath)
	require.NoError(t, err)
	content := string(data)
	assert.True(t, strings.HasPrefix(content, existing), "existing content should be preserved at the start")
	assert.Contains(t, content, sentinelBegin)
	assert.Contains(t, content, sentinelEnd)
}

func TestAddSentinelBlockReplaceExisting(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, ".zshrc")

	initial := fmt.Sprintf("before\n%s\nold source line\n%s\nafter\n", sentinelBegin, sentinelEnd)
	require.NoError(t, os.WriteFile(configPath, []byte(initial), 0644))

	err := addSentinelBlock(configPath, "new source line")
	require.NoError(t, err)

	data, err := os.ReadFile(configPath)
	require.NoError(t, err)
	content := string(data)
	assert.Contains(t, content, "before\n")
	assert.Contains(t, content, "new source line")
	assert.NotContains(t, content, "old source line")
	assert.Contains(t, content, "after\n")
	assert.Equal(t, 1, strings.Count(content, sentinelBegin))
	assert.Equal(t, 1, strings.Count(content, sentinelEnd))
}

func TestAddSentinelBlockAppendsNewlineIfMissing(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, ".zshrc")
	require.NoError(t, os.WriteFile(configPath, []byte("no trailing newline"), 0644))

	err := addSentinelBlock(configPath, "source line")
	require.NoError(t, err)

	data, err := os.ReadFile(configPath)
	require.NoError(t, err)
	assert.Contains(t, string(data), "no trailing newline\n"+sentinelBegin)
}

func TestAddSentinelBlockOrphanedBeginOnly(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, ".zshrc")
	content := fmt.Sprintf("before\n%s\norphaned source line\nafter\n", sentinelBegin)
	require.NoError(t, os.WriteFile(configPath, []byte(content), 0644))

	err := addSentinelBlock(configPath, "new source line")
	require.NoError(t, err)

	data, err := os.ReadFile(configPath)
	require.NoError(t, err)
	result := string(data)
	assert.Contains(t, result, "new source line")
	assert.Contains(t, result, sentinelEnd)
}

func TestAddSentinelBlockOrphanedEndOnly(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, ".zshrc")
	content := fmt.Sprintf("before\n%s\nafter\n", sentinelEnd)
	require.NoError(t, os.WriteFile(configPath, []byte(content), 0644))

	err := addSentinelBlock(configPath, "new source line")
	require.NoError(t, err)

	data, err := os.ReadFile(configPath)
	require.NoError(t, err)
	result := string(data)
	assert.Contains(t, result, "new source line")
	assert.Equal(t, 1, strings.Count(result, sentinelBegin))
}

func TestAddSentinelBlockReversedMarkers(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, ".zshrc")
	content := fmt.Sprintf("before\n%s\norphaned\n%s\nafter\n", sentinelEnd, sentinelBegin)
	require.NoError(t, os.WriteFile(configPath, []byte(content), 0644))

	err := addSentinelBlock(configPath, "new source line")
	require.NoError(t, err)

	data, err := os.ReadFile(configPath)
	require.NoError(t, err)
	assert.Contains(t, string(data), "new source line")
}

func TestRemoveSentinelBlockPreservesOtherContent(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, ".zshrc")
	content := fmt.Sprintf("before\n%s\nsource line\n%s\nafter\n", sentinelBegin, sentinelEnd)
	require.NoError(t, os.WriteFile(configPath, []byte(content), 0644))

	err := removeSentinelBlock(configPath)
	require.NoError(t, err)

	data, err := os.ReadFile(configPath)
	require.NoError(t, err)
	result := string(data)
	assert.Contains(t, result, "before\n")
	assert.Contains(t, result, "after\n")
	assert.NotContains(t, result, sentinelBegin)
	assert.NotContains(t, result, "source line")
}

func TestRemoveSentinelBlockNoBlockPresent(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, ".zshrc")
	original := "export FOO=bar\n"
	require.NoError(t, os.WriteFile(configPath, []byte(original), 0644))

	err := removeSentinelBlock(configPath)
	require.NoError(t, err)

	data, err := os.ReadFile(configPath)
	require.NoError(t, err)
	assert.Equal(t, original, string(data))
}

func TestRemoveSentinelBlockFileMissing(t *testing.T) {
	err := removeSentinelBlock(filepath.Join(t.TempDir(), "nonexistent"))
	assert.NoError(t, err)
}

func TestRemoveSentinelBlockOrphanedBeginOnly(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, ".zshrc")
	content := fmt.Sprintf("before\n%s\norphaned\nafter\n", sentinelBegin)
	require.NoError(t, os.WriteFile(configPath, []byte(content), 0644))

	err := removeSentinelBlock(configPath)
	require.NoError(t, err)

	data, err := os.ReadFile(configPath)
	require.NoError(t, err)
	assert.Equal(t, content, string(data))
}

func TestRemoveSentinelBlockOrphanedEndOnly(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, ".zshrc")
	content := fmt.Sprintf("before\n%s\nafter\n", sentinelEnd)
	require.NoError(t, os.WriteFile(configPath, []byte(content), 0644))

	err := removeSentinelBlock(configPath)
	require.NoError(t, err)

	data, err := os.ReadFile(configPath)
	require.NoError(t, err)
	assert.Equal(t, content, string(data))
}

func TestRemoveSentinelBlockReversedMarkers(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, ".zshrc")

	// End marker appears before begin — should be a no-op
	content := fmt.Sprintf("before\n%s\norphaned\n%s\nafter\n", sentinelEnd, sentinelBegin)
	require.NoError(t, os.WriteFile(configPath, []byte(content), 0644))

	err := removeSentinelBlock(configPath)
	require.NoError(t, err)

	data, err := os.ReadFile(configPath)
	require.NoError(t, err)
	assert.Equal(t, content, string(data), "reversed markers should be left untouched")
}

func TestAddSentinelBlockReadPermissionDenied(t *testing.T) {
	if runtime.GOOS == "windows" || os.Getuid() == 0 {
		t.Skip("Cannot test Unix file permissions on Windows or as root")
	}

	dir := t.TempDir()
	configPath := filepath.Join(dir, ".zshrc")
	require.NoError(t, os.WriteFile(configPath, []byte("content"), 0644))
	require.NoError(t, os.Chmod(configPath, 0000))
	t.Cleanup(func() { os.Chmod(configPath, 0644) })

	err := addSentinelBlock(configPath, "source line")
	assert.Error(t, err)
}

func TestAddSentinelBlockWritePermissionDenied(t *testing.T) {
	if runtime.GOOS == "windows" || os.Getuid() == 0 {
		t.Skip("Cannot test Unix file permissions on Windows or as root")
	}

	dir := t.TempDir()
	configPath := filepath.Join(dir, ".zshrc")
	require.NoError(t, os.WriteFile(configPath, []byte("existing\n"), 0644))
	require.NoError(t, os.Chmod(configPath, 0444))
	t.Cleanup(func() { os.Chmod(configPath, 0644) })

	err := addSentinelBlock(configPath, "source line")
	assert.Error(t, err)
}

func TestRemoveSentinelBlockReadPermissionDenied(t *testing.T) {
	if runtime.GOOS == "windows" || os.Getuid() == 0 {
		t.Skip("Cannot test Unix file permissions on Windows or as root")
	}

	dir := t.TempDir()
	configPath := filepath.Join(dir, ".zshrc")
	content := fmt.Sprintf("%s\nline\n%s\n", sentinelBegin, sentinelEnd)
	require.NoError(t, os.WriteFile(configPath, []byte(content), 0644))
	require.NoError(t, os.Chmod(configPath, 0000))
	t.Cleanup(func() { os.Chmod(configPath, 0644) })

	err := removeSentinelBlock(configPath)
	assert.Error(t, err)
}

func TestRemoveSentinelBlockWritePermissionDenied(t *testing.T) {
	if runtime.GOOS == "windows" || os.Getuid() == 0 {
		t.Skip("Cannot test Unix file permissions on Windows or as root")
	}

	dir := t.TempDir()
	configPath := filepath.Join(dir, ".zshrc")
	content := fmt.Sprintf("%s\nline\n%s\n", sentinelBegin, sentinelEnd)
	require.NoError(t, os.WriteFile(configPath, []byte(content), 0644))
	require.NoError(t, os.Chmod(configPath, 0444))
	t.Cleanup(func() { os.Chmod(configPath, 0644) })

	err := removeSentinelBlock(configPath)
	assert.Error(t, err)
}

func TestAddSentinelBlockPreservesFilePermissions(t *testing.T) {
	if runtime.GOOS == "windows" || os.Getuid() == 0 {
		t.Skip("Cannot test Unix file permissions on Windows or as root")
	}

	dir := t.TempDir()
	configPath := filepath.Join(dir, ".zshrc")
	require.NoError(t, os.WriteFile(configPath, []byte("existing\n"), 0600))

	err := addSentinelBlock(configPath, "source line")
	require.NoError(t, err)

	info, err := os.Stat(configPath)
	require.NoError(t, err)
	assert.Equal(t, os.FileMode(0600), info.Mode().Perm(), "file permissions should be preserved")
}

func TestRemoveSentinelBlockPreservesFilePermissions(t *testing.T) {
	if runtime.GOOS == "windows" || os.Getuid() == 0 {
		t.Skip("Cannot test Unix file permissions on Windows or as root")
	}

	dir := t.TempDir()
	configPath := filepath.Join(dir, ".zshrc")
	content := fmt.Sprintf("before\n%s\nline\n%s\nafter\n", sentinelBegin, sentinelEnd)
	require.NoError(t, os.WriteFile(configPath, []byte(content), 0600))

	err := removeSentinelBlock(configPath)
	require.NoError(t, err)

	info, err := os.Stat(configPath)
	require.NoError(t, err)
	assert.Equal(t, os.FileMode(0600), info.Mode().Perm(), "file permissions should be preserved")
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

// alwaysConfirm overrides confirmFunc to always return true (auto-accept).
// Returns a cleanup function that restores the original.
func alwaysConfirm(t *testing.T) {
	t.Helper()
	original := confirmFunc
	confirmFunc = func(_ string) bool { return true }
	t.Cleanup(func() { confirmFunc = original })
}

// neverConfirm overrides confirmFunc to always return false (auto-decline).
func neverConfirm(t *testing.T) {
	t.Helper()
	original := confirmFunc
	confirmFunc = func(_ string) bool { return false }
	t.Cleanup(func() { confirmFunc = original })
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
	assert.Contains(t, content, "source "+scriptPath)
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
	assert.Contains(t, string(configData), "source "+scriptPath)
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
