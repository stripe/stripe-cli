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
// Pure function tests — no filesystem needed
// ---------------------------------------------------------------------------

func TestComputeAddSentinel(t *testing.T) {
	tests := []struct {
		name     string
		content  string
		line     string
		wantHas  []string // substrings that must be present
		wantNot  []string // substrings that must be absent
		wantOnce []string // substrings that must appear exactly once
	}{
		{
			name:     "empty content",
			content:  "",
			line:     "source ~/.stripe/stripe-completion.zsh",
			wantHas:  []string{sentinelBegin, "source ~/.stripe/stripe-completion.zsh", sentinelEnd},
			wantOnce: []string{sentinelBegin, sentinelEnd},
		},
		{
			name:     "existing content without sentinel",
			content:  "export PATH=/usr/local/bin:$PATH\n",
			line:     "source ~/.stripe/stripe-completion.zsh",
			wantHas:  []string{"export PATH=/usr/local/bin:$PATH\n", sentinelBegin, sentinelEnd},
			wantOnce: []string{sentinelBegin, sentinelEnd},
		},
		{
			name:     "replace existing block",
			content:  fmt.Sprintf("before\n%s\nold source line\n%s\nafter\n", sentinelBegin, sentinelEnd),
			line:     "new source line",
			wantHas:  []string{"before\n", "new source line", "after\n"},
			wantNot:  []string{"old source line"},
			wantOnce: []string{sentinelBegin, sentinelEnd},
		},
		{
			name:    "orphaned begin only — appends new block",
			content: fmt.Sprintf("before\n%s\norphaned source line\nafter\n", sentinelBegin),
			line:    "new source line",
			wantHas: []string{"new source line", sentinelEnd},
		},
		{
			name:     "orphaned end only — appends new block",
			content:  fmt.Sprintf("before\n%s\nafter\n", sentinelEnd),
			line:     "new source line",
			wantHas:  []string{"new source line", sentinelBegin},
			wantOnce: []string{sentinelBegin},
		},
		{
			name:    "reversed markers — appends new block",
			content: fmt.Sprintf("before\n%s\norphaned\n%s\nafter\n", sentinelEnd, sentinelBegin),
			line:    "new source line",
			wantHas: []string{"new source line"},
		},
		{
			name:    "missing trailing newline — newline inserted before block",
			content: "no trailing newline",
			line:    "source line",
			wantHas: []string{"no trailing newline\n" + sentinelBegin},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := computeAddSentinel(tt.content, tt.line)
			for _, s := range tt.wantHas {
				assert.Contains(t, got, s)
			}
			for _, s := range tt.wantNot {
				assert.NotContains(t, got, s)
			}
			for _, s := range tt.wantOnce {
				assert.Equal(t, 1, strings.Count(got, s), "expected exactly one occurrence of %q", s)
			}
		})
	}
}

func TestComputeRemoveSentinel(t *testing.T) {
	tests := []struct {
		name      string
		content   string
		wantFound bool
		wantHas   []string
		wantNot   []string
	}{
		{
			name:      "no block present",
			content:   "export FOO=bar\n",
			wantFound: false,
			wantHas:   []string{"export FOO=bar\n"},
		},
		{
			name:      "empty content",
			content:   "",
			wantFound: false,
		},
		{
			name:      "valid block removed",
			content:   fmt.Sprintf("before\n%s\nsource line\n%s\nafter\n", sentinelBegin, sentinelEnd),
			wantFound: true,
			wantHas:   []string{"before\n", "after\n"},
			wantNot:   []string{sentinelBegin, sentinelEnd, "source line"},
		},
		{
			name:      "orphaned begin only — no-op",
			content:   fmt.Sprintf("before\n%s\norphaned\nafter\n", sentinelBegin),
			wantFound: false,
			wantHas:   []string{sentinelBegin, "orphaned"},
		},
		{
			name:      "orphaned end only — no-op",
			content:   fmt.Sprintf("before\n%s\nafter\n", sentinelEnd),
			wantFound: false,
			wantHas:   []string{sentinelEnd},
		},
		{
			name:      "reversed markers — no-op",
			content:   fmt.Sprintf("before\n%s\norphaned\n%s\nafter\n", sentinelEnd, sentinelBegin),
			wantFound: false,
			wantHas:   []string{sentinelEnd, sentinelBegin},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, found := computeRemoveSentinel(tt.content)
			assert.Equal(t, tt.wantFound, found)
			for _, s := range tt.wantHas {
				assert.Contains(t, got, s)
			}
			for _, s := range tt.wantNot {
				assert.NotContains(t, got, s)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// I/O wrapper tests — addSentinelBlock / removeSentinelBlock
// ---------------------------------------------------------------------------

func TestAddSentinelBlockNewFile(t *testing.T) {
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

func TestAddSentinelBlockPreservesPermissions(t *testing.T) {
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

	// Atomic write creates a new temp file and renames it. To prevent the write,
	// we make the containing directory read-only so CreateTemp fails.
	dir := t.TempDir()
	configPath := filepath.Join(dir, ".zshrc")
	require.NoError(t, os.WriteFile(configPath, []byte("existing\n"), 0644))
	require.NoError(t, os.Chmod(dir, 0555))
	t.Cleanup(func() { os.Chmod(dir, 0755) })

	err := addSentinelBlock(configPath, "source line")
	assert.Error(t, err)
}

func TestRemoveSentinelBlockMissingFile(t *testing.T) {
	err := removeSentinelBlock(filepath.Join(t.TempDir(), "nonexistent"))
	assert.NoError(t, err)
}

func TestRemoveSentinelBlockPreservesPermissions(t *testing.T) {
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

func TestFindManualRemnants(t *testing.T) {
	tests := []struct {
		name           string
		content        string
		scriptFilename string
		wantLen        int
		wantLineNums   []int
	}{
		{
			name:           "detects manual source line",
			content:        "export PATH=/usr/local/bin:$PATH\nsource ~/.stripe/stripe-completion.zsh\nalias ls='ls -G'\n",
			scriptFilename: "stripe-completion.zsh",
			wantLen:        1,
			wantLineNums:   []int{2},
		},
		{
			name:           "detects dot-source syntax",
			content:        ". /some/custom/path/stripe-completion.bash\n",
			scriptFilename: "stripe-completion.bash",
			wantLen:        1,
			wantLineNums:   []int{1},
		},
		{
			name:           "detects line with other commands",
			content:        "[ -f ~/.stripe/stripe-completion.zsh ] && source ~/.stripe/stripe-completion.zsh\n",
			scriptFilename: "stripe-completion.zsh",
			wantLen:        1,
			wantLineNums:   []int{1},
		},
		{
			name:           "detects custom path",
			content:        "source /opt/completions/stripe-completion.zsh\n",
			scriptFilename: "stripe-completion.zsh",
			wantLen:        1,
		},
		{
			name: "ignores lines inside sentinel block",
			content: fmt.Sprintf("before\n%s\nsource ~/.stripe/stripe-completion.zsh\n%s\nafter\n",
				sentinelBegin, sentinelEnd),
			scriptFilename: "stripe-completion.zsh",
			wantLen:        0,
		},
		{
			name:           "ignores comment lines",
			content:        "# source ~/.stripe/stripe-completion.zsh\n",
			scriptFilename: "stripe-completion.zsh",
			wantLen:        0,
		},
		{
			name:           "no match returns nil",
			content:        "export PATH=/usr/local/bin:$PATH\nalias ls='ls -G'\n",
			scriptFilename: "stripe-completion.zsh",
			wantLen:        0,
		},
		{
			name:           "multiple matches",
			content:        "source ~/.stripe/stripe-completion.zsh\nexport FOO=bar\n. /other/stripe-completion.zsh\n",
			scriptFilename: "stripe-completion.zsh",
			wantLen:        2,
			wantLineNums:   []int{1, 3},
		},
		{
			name: "manual line before sentinel block only",
			content: fmt.Sprintf("source ~/my/stripe-completion.zsh\n%s\nsource ~/.stripe/stripe-completion.zsh\n%s\n",
				sentinelBegin, sentinelEnd),
			scriptFilename: "stripe-completion.zsh",
			wantLen:        1,
			wantLineNums:   []int{1},
		},
		{
			name: "manual line after sentinel block only",
			content: fmt.Sprintf("%s\nsource ~/.stripe/stripe-completion.zsh\n%s\nsource ~/custom/stripe-completion.zsh\n",
				sentinelBegin, sentinelEnd),
			scriptFilename: "stripe-completion.zsh",
			wantLen:        1,
			wantLineNums:   []int{4},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			configPath := filepath.Join(dir, "config")
			require.NoError(t, os.WriteFile(configPath, []byte(tt.content), 0644))

			remnants := findManualRemnants(configPath, tt.scriptFilename)

			if tt.wantLen == 0 {
				// nil and empty slice are both acceptable for "no results"
				assert.Len(t, remnants, 0)
			} else {
				require.Len(t, remnants, tt.wantLen)
				for i, wantLine := range tt.wantLineNums {
					assert.Equal(t, wantLine, remnants[i].lineNumber)
				}
			}
		})
	}
}

func TestFindManualRemnantsReturnsNilForMissingFile(t *testing.T) {
	remnants := findManualRemnants(filepath.Join(t.TempDir(), "nonexistent"), "stripe-completion.zsh")
	assert.Nil(t, remnants)
}
