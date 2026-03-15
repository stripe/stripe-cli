package cmd

import (
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
	// rootCmd must be initialized for Cobra's completion generation to work.
	// The init() function in root.go sets this up.
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
// addSentinelBlock / removeSentinelBlock
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

	content := string(data)
	assert.True(t, strings.Contains(content, "no trailing newline\n"+sentinelBegin))
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
	assert.Equal(t, 1, strings.Count(result, sentinelBegin), "should have exactly one begin marker")
	assert.GreaterOrEqual(t, strings.Count(result, sentinelEnd), 1, "should have at least one end marker")
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
	result := string(data)
	assert.Contains(t, result, "new source line")
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
	assert.NotContains(t, result, sentinelEnd)
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
// findManualRemnants
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

	content := ". /some/custom/path/stripe-completion.bash\n"
	require.NoError(t, os.WriteFile(configPath, []byte(content), 0644))

	remnants := findManualRemnants(configPath, "stripe-completion.bash")
	require.Len(t, remnants, 1)
	assert.Equal(t, 1, remnants[0].lineNumber)
}

func TestFindManualRemnantsDetectsLineWithOtherCommands(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, ".zshrc")

	content := "[ -f ~/.stripe/stripe-completion.zsh ] && source ~/.stripe/stripe-completion.zsh\n"
	require.NoError(t, os.WriteFile(configPath, []byte(content), 0644))

	remnants := findManualRemnants(configPath, "stripe-completion.zsh")
	require.Len(t, remnants, 1)
	assert.Equal(t, 1, remnants[0].lineNumber)
}

func TestFindManualRemnantsDetectsCustomPath(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, ".zshrc")

	content := "source /opt/completions/stripe-completion.zsh\n"
	require.NoError(t, os.WriteFile(configPath, []byte(content), 0644))

	remnants := findManualRemnants(configPath, "stripe-completion.zsh")
	require.Len(t, remnants, 1)
}

func TestFindManualRemnantsIgnoresSentinelBlock(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, ".zshrc")

	content := fmt.Sprintf("before\n%s\nsource ~/.stripe/stripe-completion.zsh\n%s\nafter\n",
		sentinelBegin, sentinelEnd)
	require.NoError(t, os.WriteFile(configPath, []byte(content), 0644))

	remnants := findManualRemnants(configPath, "stripe-completion.zsh")
	assert.Empty(t, remnants)
}

func TestFindManualRemnantsIgnoresComments(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, ".zshrc")

	content := "# source ~/.stripe/stripe-completion.zsh\n"
	require.NoError(t, os.WriteFile(configPath, []byte(content), 0644))

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

	content := "export PATH=/usr/local/bin:$PATH\nalias ls='ls -G'\n"
	require.NoError(t, os.WriteFile(configPath, []byte(content), 0644))

	remnants := findManualRemnants(configPath, "stripe-completion.zsh")
	assert.Nil(t, remnants)
}

func TestFindManualRemnantsMultipleMatches(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, ".zshrc")

	content := "source ~/.stripe/stripe-completion.zsh\nexport FOO=bar\n. /other/stripe-completion.zsh\n"
	require.NoError(t, os.WriteFile(configPath, []byte(content), 0644))

	remnants := findManualRemnants(configPath, "stripe-completion.zsh")
	require.Len(t, remnants, 2)
	assert.Equal(t, 1, remnants[0].lineNumber)
	assert.Equal(t, 3, remnants[1].lineNumber)
}

func TestFindManualRemnantsOutsideSentinelWithManualBefore(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, ".zshrc")

	content := fmt.Sprintf("source ~/my/stripe-completion.zsh\n%s\nsource ~/.stripe/stripe-completion.zsh\n%s\n",
		sentinelBegin, sentinelEnd)
	require.NoError(t, os.WriteFile(configPath, []byte(content), 0644))

	remnants := findManualRemnants(configPath, "stripe-completion.zsh")
	require.Len(t, remnants, 1)
	assert.Equal(t, 1, remnants[0].lineNumber)
}

func TestFindManualRemnantsOutsideSentinelWithManualAfter(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, ".zshrc")

	content := fmt.Sprintf("%s\nsource ~/.stripe/stripe-completion.zsh\n%s\nsource ~/custom/stripe-completion.zsh\n",
		sentinelBegin, sentinelEnd)
	require.NoError(t, os.WriteFile(configPath, []byte(content), 0644))

	remnants := findManualRemnants(configPath, "stripe-completion.zsh")
	require.Len(t, remnants, 1)
	assert.Equal(t, 4, remnants[0].lineNumber)
}
