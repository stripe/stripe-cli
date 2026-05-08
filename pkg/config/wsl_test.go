package config

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestIsWSLFromVersion(t *testing.T) {
	tests := []struct {
		name    string
		content string
		want    bool
	}{
		{
			name:    "microsoft keyword",
			content: "Linux version 5.15.0-microsoft-standard-WSL2",
			want:    true,
		},
		{
			name:    "Microsoft capitalised",
			content: "Linux version 5.15.0-Microsoft-standard",
			want:    true,
		},
		{
			name:    "wsl keyword",
			content: "Linux version 5.15.0 (wsl@build)",
			want:    true,
		},
		{
			name:    "WSL uppercase",
			content: "Linux version 5.15.0 (WSL2)",
			want:    true,
		},
		{
			name:    "plain linux",
			content: "Linux version 6.1.0-28-amd64 (debian-kernel@lists.debian.org)",
			want:    false,
		},
		{
			name:    "empty",
			content: "",
			want:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require.Equal(t, tt.want, isWSLFromVersion(tt.content))
		})
	}
}

func TestIsWSL_UnreadableProcVersion(t *testing.T) {
	// isWSL() returns false when /proc/version cannot be read; we can verify
	// the helper directly covers that path via isWSLFromVersion with empty input.
	require.False(t, isWSLFromVersion(""))
}

func wslExpectedPassword(t *testing.T, machineID, bootID string) string {
	t.Helper()
	const appKey = "stripe-cli-keyring-v1"
	mac := hmac.New(sha256.New, []byte(appKey))
	mac.Write([]byte(machineID))
	mac.Write([]byte(bootID))
	return hex.EncodeToString(mac.Sum(nil))
}

func TestWslFilePasswordFromPaths_BothFiles(t *testing.T) {
	dir := t.TempDir()
	machineIDPath := filepath.Join(dir, "machine-id")
	bootIDPath := filepath.Join(dir, "boot_id")

	require.NoError(t, os.WriteFile(machineIDPath, []byte("abc123\n"), 0600))
	require.NoError(t, os.WriteFile(bootIDPath, []byte("def456\n"), 0600))

	got, err := wslFilePasswordFromPaths(machineIDPath, bootIDPath)
	require.NoError(t, err)
	require.Equal(t, wslExpectedPassword(t, "abc123", "def456"), got)
}

func TestWslFilePasswordFromPaths_MachineIDMissing(t *testing.T) {
	dir := t.TempDir()
	machineIDPath := filepath.Join(dir, "machine-id") // does not exist
	bootIDPath := filepath.Join(dir, "boot_id")

	require.NoError(t, os.WriteFile(bootIDPath, []byte("def456\n"), 0600))

	_, err := wslFilePasswordFromPaths(machineIDPath, bootIDPath)
	require.Error(t, err)
	require.Contains(t, err.Error(), machineIDPath)
}

func TestWslFilePasswordFromPaths_BootIDMissing(t *testing.T) {
	dir := t.TempDir()
	machineIDPath := filepath.Join(dir, "machine-id")
	bootIDPath := filepath.Join(dir, "boot_id") // does not exist

	require.NoError(t, os.WriteFile(machineIDPath, []byte("abc123\n"), 0600))

	_, err := wslFilePasswordFromPaths(machineIDPath, bootIDPath)
	require.Error(t, err)
	require.Contains(t, err.Error(), bootIDPath)
}

func TestWslFilePasswordFromPaths_BothMissing(t *testing.T) {
	dir := t.TempDir()
	machineIDPath := filepath.Join(dir, "machine-id")
	bootIDPath := filepath.Join(dir, "boot_id")

	_, err := wslFilePasswordFromPaths(machineIDPath, bootIDPath)
	require.Error(t, err)
}

func TestWslFilePasswordFromPaths_Deterministic(t *testing.T) {
	dir := t.TempDir()
	machineIDPath := filepath.Join(dir, "machine-id")
	bootIDPath := filepath.Join(dir, "boot_id")

	require.NoError(t, os.WriteFile(machineIDPath, []byte("stable-id\n"), 0600))
	require.NoError(t, os.WriteFile(bootIDPath, []byte("stable-boot\n"), 0600))

	first, err := wslFilePasswordFromPaths(machineIDPath, bootIDPath)
	require.NoError(t, err)
	second, err := wslFilePasswordFromPaths(machineIDPath, bootIDPath)
	require.NoError(t, err)

	require.Equal(t, first, second)
}

func TestWslFilePasswordFromPaths_DifferentIDsDifferentPasswords(t *testing.T) {
	dir := t.TempDir()
	bootIDPath := filepath.Join(dir, "boot_id")
	require.NoError(t, os.WriteFile(bootIDPath, []byte("same-boot\n"), 0600))

	pathA := filepath.Join(dir, "machine-id-a")
	pathB := filepath.Join(dir, "machine-id-b")
	require.NoError(t, os.WriteFile(pathA, []byte("id-aaa\n"), 0600))
	require.NoError(t, os.WriteFile(pathB, []byte("id-bbb\n"), 0600))

	pwA, err := wslFilePasswordFromPaths(pathA, bootIDPath)
	require.NoError(t, err)
	pwB, err := wslFilePasswordFromPaths(pathB, bootIDPath)
	require.NoError(t, err)

	require.NotEqual(t, pwA, pwB)
}

func TestWslFilePasswordFromPaths_TrimsWhitespace(t *testing.T) {
	dir := t.TempDir()
	machineIDPath := filepath.Join(dir, "machine-id")
	bootIDPath := filepath.Join(dir, "boot_id")

	require.NoError(t, os.WriteFile(machineIDPath, []byte("  trimmed-id  \n"), 0600))
	require.NoError(t, os.WriteFile(bootIDPath, []byte("  trimmed-boot  \n"), 0600))

	got, err := wslFilePasswordFromPaths(machineIDPath, bootIDPath)
	require.NoError(t, err)
	require.Equal(t, wslExpectedPassword(t, "trimmed-id", "trimmed-boot"), got)
}
