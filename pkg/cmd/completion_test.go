package cmd

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

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
