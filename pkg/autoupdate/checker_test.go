package autoupdate

import (
	"os"
	"path/filepath"
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
