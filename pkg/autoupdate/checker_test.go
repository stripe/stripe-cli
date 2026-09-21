package autoupdate

import (
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestIsMajorVersionChange(t *testing.T) {
	tests := []struct {
		current  string
		latest   string
		expected bool
	}{
		{"1.23.0", "1.24.0", false},
		{"1.23.0", "1.23.1", false},
		{"1.99.0", "2.0.0", true},
		{"2.0.0", "1.99.0", true},
		{"1.0.0", "1.0.0", false},
		{"1.0.0-beta", "1.0.0", false},
		{"invalid", "1.0.0", false},
	}

	for _, tt := range tests {
		t.Run(tt.current+"→"+tt.latest, func(t *testing.T) {
			assert.Equal(t, tt.expected, isMajorVersionChange(tt.current, tt.latest))
		})
	}
}

// The expected names are the ones the .goreleaser archive templates produce. An
// asset name that does not match one of them is not a download that fails
// loudly — fetchLatestRelease finds no matching asset and gives up silently, so
// auto-update simply never happens on that platform.
func TestBinaryAssetNameFor(t *testing.T) {
	tests := []struct {
		goos     string
		goarch   string
		expected string
	}{
		{"darwin", "amd64", "stripe_1.24.0_mac-os_x86_64.tar.gz"},
		{"darwin", "arm64", "stripe_1.24.0_mac-os_arm64.tar.gz"},
		{"linux", "amd64", "stripe_1.24.0_linux_x86_64.tar.gz"},
		{"linux", "arm64", "stripe_1.24.0_linux_arm64.tar.gz"},
		{"linux", "386", "stripe_1.24.0_linux_i386.tar.gz"},
		{"windows", "amd64", "stripe_1.24.0_windows_x86_64.zip"},
		{"windows", "386", "stripe_1.24.0_windows_i386.zip"},
		// No arm64 Windows build is published; that machine runs the x64 one.
		{"windows", "arm64", "stripe_1.24.0_windows_x86_64.zip"},
	}

	for _, tt := range tests {
		t.Run(tt.goos+"/"+tt.goarch, func(t *testing.T) {
			assert.Equal(t, tt.expected, binaryAssetNameFor("1.24.0", tt.goos, tt.goarch))
		})
	}
}

func TestBinaryAssetNameUsesTheRunningPlatform(t *testing.T) {
	assert.Equal(t, binaryAssetNameFor("1.24.0", runtime.GOOS, runtime.GOARCH), binaryAssetName("1.24.0"))
}

func TestChecksumAssetName(t *testing.T) {
	// The checksums file the release publishes for this platform, which is where
	// the archive's expected digest is read from.
	expected := map[string]string{
		"darwin":  "stripe-mac-checksums.txt",
		"linux":   "stripe-linux-checksums.txt",
		"windows": "stripe-windows-checksums.txt",
	}[runtime.GOOS]

	assert.Equal(t, expected, checksumAssetName())
}

func TestMarkerReadWrite(t *testing.T) {
	tmpDir := t.TempDir()
	original := GetStateDirFn
	defer func() { GetStateDirFn = original }()
	GetStateDirFn = func() string { return tmpDir }

	m := UpdateMarker{
		Version:     "1.24.0",
		DownloadURL: "https://example.com/stripe.tar.gz",
		Checksum:    "abc123",
	}

	WriteMarker(m)

	got := ReadMarker()
	require.NotNil(t, got)
	assert.Equal(t, "1.24.0", got.Version)
	assert.Equal(t, "https://example.com/stripe.tar.gz", got.DownloadURL)
	assert.Equal(t, "abc123", got.Checksum)

	ClearMarker()
	assert.Nil(t, ReadMarker())
}

func TestMarkerReadWriteWithReleaseNotes(t *testing.T) {
	tmpDir := t.TempDir()
	original := GetStateDirFn
	defer func() { GetStateDirFn = original }()
	GetStateDirFn = func() string { return tmpDir }

	notes := "## Changes\n\n- Added one thing\n- Fixed another thing"
	WriteMarker(UpdateMarker{
		Version:      "1.24.0",
		DownloadURL:  "https://example.com/stripe.tar.gz",
		Checksum:     "abc123",
		ReleaseNotes: notes,
	})

	got := ReadMarker()
	require.NotNil(t, got)
	assert.Equal(t, notes, got.ReleaseNotes)
}

func TestReadMarkerWithoutReleaseNotes(t *testing.T) {
	tmpDir := t.TempDir()
	original := GetStateDirFn
	defer func() { GetStateDirFn = original }()
	GetStateDirFn = func() string { return tmpDir }

	marker := `{"version":"1.24.0","download_url":"https://example.com/stripe.tar.gz","checksum":"abc123"}`
	require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "update-available"), []byte(marker), 0644))

	got := ReadMarker()
	require.NotNil(t, got)
	assert.Empty(t, got.ReleaseNotes)
}

func TestRecordLastCheck(t *testing.T) {
	tmpDir := t.TempDir()
	original := GetStateDirFn
	defer func() { GetStateDirFn = original }()
	GetStateDirFn = func() string { return tmpDir }

	recordLastCheck()

	data, err := os.ReadFile(filepath.Join(tmpDir, "last_update_check"))
	require.NoError(t, err)

	ts, err := strconv.ParseInt(string(data), 10, 64)
	require.NoError(t, err)
	assert.WithinDuration(t, time.Now(), time.Unix(ts, 0), 2*time.Second)
}

func TestVerifyChecksum(t *testing.T) {
	tmpFile := filepath.Join(t.TempDir(), "testfile")
	os.WriteFile(tmpFile, []byte("hello world\n"), 0644)

	// sha256 of "hello world\n"
	assert.True(t, VerifyChecksum(tmpFile, "a948904f2f0f479b8f8197694b30184b0d2ed1c1cd2a1ec0fb85d299a192a447"))
	assert.False(t, VerifyChecksum(tmpFile, "0000000000000000000000000000000000000000000000000000000000000000"))
	// Empty expected = skip verification
	assert.True(t, VerifyChecksum(tmpFile, ""))
}
