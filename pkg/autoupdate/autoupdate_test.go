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
	}

	for _, tt := range tests {
		t.Run(tt.current+"→"+tt.latest, func(t *testing.T) {
			assert.Equal(t, tt.expected, isMajorVersionChange(tt.current, tt.latest))
		})
	}
}

func TestMajorVersion(t *testing.T) {
	assert.Equal(t, "1", majorVersion("1.23.0"))
	assert.Equal(t, "2", majorVersion("2.0.0"))
	assert.Equal(t, "0", majorVersion("0.1.0"))
	assert.Equal(t, "", majorVersion(""))
}

func TestMarkerReadWrite(t *testing.T) {
	tmpDir := t.TempDir()
	original := getStateDirFn
	defer func() { getStateDirFn = original }()
	getStateDirFn = func() string { return tmpDir }

	m := updateMarker{
		Version:     "1.24.0",
		DownloadURL: "https://example.com/stripe.tar.gz",
		Checksum:    "abc123",
	}

	writeMarker(m)

	got := readMarker()
	require.NotNil(t, got)
	assert.Equal(t, "1.24.0", got.Version)
	assert.Equal(t, "https://example.com/stripe.tar.gz", got.DownloadURL)
	assert.Equal(t, "abc123", got.Checksum)

	clearMarker()
	assert.Nil(t, readMarker())
}

func TestRecordLastCheck(t *testing.T) {
	tmpDir := t.TempDir()
	original := getStateDirFn
	defer func() { getStateDirFn = original }()
	getStateDirFn = func() string { return tmpDir }

	recordLastCheck()

	data, err := os.ReadFile(filepath.Join(tmpDir, "last_update_check"))
	require.NoError(t, err)

	ts, err := strconv.ParseInt(string(data), 10, 64)
	require.NoError(t, err)
	assert.WithinDuration(t, time.Now(), time.Unix(ts, 0), 2*time.Second)
}
