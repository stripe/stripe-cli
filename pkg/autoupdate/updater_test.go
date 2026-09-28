package autoupdate

import (
	"archive/tar"
	"archive/zip"
	"compress/gzip"
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func createTestTarGz(t *testing.T, filename string, content []byte) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "test.tar.gz")
	f, err := os.Create(path)
	require.NoError(t, err)

	gw := gzip.NewWriter(f)
	tw := tar.NewWriter(gw)

	require.NoError(t, tw.WriteHeader(&tar.Header{
		Name:     filename,
		Size:     int64(len(content)),
		Mode:     0755,
		Typeflag: tar.TypeReg,
	}))
	_, err = tw.Write(content)
	require.NoError(t, err)

	require.NoError(t, tw.Close())
	require.NoError(t, gw.Close())
	require.NoError(t, f.Close())
	return path
}

// createTestZip writes the archive format the Windows release publishes.
func createTestZip(t *testing.T, filename string, content []byte) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "test.zip")
	f, err := os.Create(path)
	require.NoError(t, err)

	zw := zip.NewWriter(f)
	entry, err := zw.Create(filename)
	require.NoError(t, err)
	_, err = entry.Write(content)
	require.NoError(t, err)

	require.NoError(t, zw.Close())
	require.NoError(t, f.Close())
	return path
}

func sha256sum(path string) string {
	data, _ := os.ReadFile(path)
	h := sha256.Sum256(data)
	return hex.EncodeToString(h[:])
}

func TestExtractFromTarGz(t *testing.T) {
	content := []byte("#!/bin/sh\necho hello\n")
	archivePath := createTestTarGz(t, binaryName(), content)

	destPath := filepath.Join(t.TempDir(), "stripe")
	err := extractFromTarGz(archivePath, destPath)
	require.NoError(t, err)

	got, err := os.ReadFile(destPath)
	require.NoError(t, err)
	assert.Equal(t, content, got)
}

func TestExtractFromTarGz_NestedPath(t *testing.T) {
	content := []byte("binary content")
	archivePath := createTestTarGz(t, "stripe_1.43.8_linux_arm64/"+binaryName(), content)

	destPath := filepath.Join(t.TempDir(), "stripe")
	err := extractFromTarGz(archivePath, destPath)
	require.NoError(t, err)

	got, err := os.ReadFile(destPath)
	require.NoError(t, err)
	assert.Equal(t, content, got)
}

func TestExtractFromTarGz_NoBinary(t *testing.T) {
	archivePath := createTestTarGz(t, "not-stripe", []byte("nope"))

	destPath := filepath.Join(t.TempDir(), "stripe")
	err := extractFromTarGz(archivePath, destPath)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not found in archive")
}

func TestExtractFromZip(t *testing.T) {
	content := []byte("MZ windows binary")
	archivePath := createTestZip(t, binaryName(), content)

	destPath := filepath.Join(t.TempDir(), binaryName())
	err := extractFromZip(archivePath, destPath)
	require.NoError(t, err)

	got, err := os.ReadFile(destPath)
	require.NoError(t, err)
	assert.Equal(t, content, got)
}

func TestExtractFromZip_NestedPath(t *testing.T) {
	content := []byte("MZ windows binary")
	// Zip entry names use forward slashes whatever the platform reading them.
	archivePath := createTestZip(t, "stripe_1.43.8_windows_x86_64/"+binaryName(), content)

	destPath := filepath.Join(t.TempDir(), binaryName())
	err := extractFromZip(archivePath, destPath)
	require.NoError(t, err)

	got, err := os.ReadFile(destPath)
	require.NoError(t, err)
	assert.Equal(t, content, got)
}

func TestExtractFromZip_NoBinary(t *testing.T) {
	archivePath := createTestZip(t, "not-stripe", []byte("nope"))

	destPath := filepath.Join(t.TempDir(), binaryName())
	err := extractFromZip(archivePath, destPath)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not found in archive")
}

// The archive is downloaded to a temporary name, so its format has to be read off
// the bytes rather than an extension.
func TestExtractBinary_PicksTheFormatFromTheContents(t *testing.T) {
	for _, tt := range []struct {
		format  string
		archive func(*testing.T, string, []byte) string
	}{
		{"tar.gz", createTestTarGz},
		{"zip", createTestZip},
	} {
		t.Run(tt.format, func(t *testing.T) {
			content := []byte("binary for " + tt.format)
			// A name that gives nothing away, as os.CreateTemp produces.
			archivePath := tt.archive(t, binaryName(), content)
			anonymous := filepath.Join(t.TempDir(), "stripe-update-archive-1234")
			require.NoError(t, os.Rename(archivePath, anonymous))

			destPath := filepath.Join(t.TempDir(), binaryName())
			require.NoError(t, extractBinary(anonymous, destPath))

			got, err := os.ReadFile(destPath)
			require.NoError(t, err)
			assert.Equal(t, content, got)
		})
	}
}

func TestExtractBinary_ShortFileIsAnError(t *testing.T) {
	archivePath := filepath.Join(t.TempDir(), "stripe-update-archive-1234")
	require.NoError(t, os.WriteFile(archivePath, []byte("PK"), 0644))

	assert.Error(t, extractBinary(archivePath, filepath.Join(t.TempDir(), binaryName())))
}

func TestDownloadAndReplace(t *testing.T) {
	content := []byte("#!/bin/sh\necho updated\n")
	archivePath := createTestTarGz(t, binaryName(), content)
	archiveData, err := os.ReadFile(archivePath)
	require.NoError(t, err)

	checksum := sha256sum(archivePath)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write(archiveData)
	}))
	defer server.Close()

	dir := t.TempDir()
	exePath := filepath.Join(dir, binaryName())
	require.NoError(t, os.WriteFile(exePath, []byte("old binary"), 0755))

	marker := &UpdateMarker{
		Version:     "1.43.8",
		DownloadURL: server.URL + "/stripe.tar.gz",
		Checksum:    checksum,
	}

	reason, err := downloadAndReplace(marker, exePath)
	require.NoError(t, err)
	assert.Empty(t, reason, "a successful update reports no failure reason")

	got, err := os.ReadFile(exePath)
	require.NoError(t, err)
	assert.Equal(t, content, got)

	if runtime.GOOS != "windows" {
		info, err := os.Stat(exePath)
		require.NoError(t, err)
		assert.Equal(t, os.FileMode(0755), info.Mode().Perm())
	}

	// Nothing staged is left in the install directory.
	assert.NoFileExists(t, exePath+oldSuffix)
	entries, err := os.ReadDir(dir)
	require.NoError(t, err)
	assert.Len(t, entries, 1)
}

// The Windows release ships a zip rather than a tar.gz, and the binary inside it
// is named stripe.exe.
func TestDownloadAndReplace_Zip(t *testing.T) {
	content := []byte("MZ updated windows binary")
	archivePath := createTestZip(t, binaryName(), content)
	archiveData, err := os.ReadFile(archivePath)
	require.NoError(t, err)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write(archiveData)
	}))
	defer server.Close()

	dir := t.TempDir()
	exePath := filepath.Join(dir, binaryName())
	require.NoError(t, os.WriteFile(exePath, []byte("old binary"), 0755))

	marker := &UpdateMarker{
		Version:     "1.43.8",
		DownloadURL: server.URL + "/stripe.zip",
		Checksum:    sha256sum(archivePath),
	}

	reason, err := downloadAndReplace(marker, exePath)
	require.NoError(t, err)
	assert.Empty(t, reason)

	got, err := os.ReadFile(exePath)
	require.NoError(t, err)
	assert.Equal(t, content, got)
}

func TestDownloadAndReplace_BadChecksum(t *testing.T) {
	content := []byte("#!/bin/sh\necho updated\n")
	archivePath := createTestTarGz(t, binaryName(), content)
	archiveData, err := os.ReadFile(archivePath)
	require.NoError(t, err)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write(archiveData)
	}))
	defer server.Close()

	dir := t.TempDir()
	exePath := filepath.Join(dir, binaryName())
	require.NoError(t, os.WriteFile(exePath, []byte("old binary"), 0755))

	marker := &UpdateMarker{
		Version:     "1.43.8",
		DownloadURL: server.URL + "/stripe.tar.gz",
		Checksum:    "0000000000000000000000000000000000000000000000000000000000000000",
	}

	reason, err := downloadAndReplace(marker, exePath)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "checksum verification failed")
	assert.Equal(t, reasonChecksum, reason)

	got, _ := os.ReadFile(exePath)
	assert.Equal(t, []byte("old binary"), got, "original binary should be unchanged")
}

func TestDownloadAndReplace_ServerError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	dir := t.TempDir()
	exePath := filepath.Join(dir, binaryName())
	require.NoError(t, os.WriteFile(exePath, []byte("old binary"), 0755))

	marker := &UpdateMarker{
		Version:     "1.43.8",
		DownloadURL: server.URL + "/stripe.tar.gz",
	}

	reason, err := downloadAndReplace(marker, exePath)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "status 500")
	assert.Equal(t, reasonStatus, reason)
}

func TestApplyIfPending_NoMarker(t *testing.T) {
	tmpDir := t.TempDir()
	original := GetStateDirFn
	defer func() { GetStateDirFn = original }()
	GetStateDirFn = func() string { return tmpDir }

	// Should return without panic when no marker exists
	ApplyIfPending()
}

func TestApplyIfPending_SameVersion(t *testing.T) {
	tmpDir := t.TempDir()
	original := GetStateDirFn
	defer func() { GetStateDirFn = original }()
	GetStateDirFn = func() string { return tmpDir }

	// Write a marker with the current version — should be cleared without action
	WriteMarker(UpdateMarker{
		Version:     "master",
		DownloadURL: "https://example.com/stripe.tar.gz",
	})

	ApplyIfPending()

	// Marker should still exist since version.Version is "master" and we return early
	// (the "master" check happens before reading the marker)
}

// A server that accepts the request and then stalls is the case net/http's
// default transport does not cover: it bounds the dial and the TLS handshake, but
// not reading the body. Before the deadline, this hung the user's command.
func TestDownloadAndReplace_StalledDownloadTimesOut(t *testing.T) {
	release := make(chan struct{})
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Length", "1000000")
		w.WriteHeader(http.StatusOK)
		if f, ok := w.(http.Flusher); ok {
			f.Flush()
		}
		_, _ = w.Write([]byte("partial"))
		if f, ok := w.(http.Flusher); ok {
			f.Flush()
		}
		<-release // never send the rest until the test is done
	}))
	defer func() { close(release); server.Close() }()

	original := downloadTimeout
	downloadTimeout = 300 * time.Millisecond
	defer func() { downloadTimeout = original }()

	dir := t.TempDir()
	exePath := filepath.Join(dir, binaryName())
	require.NoError(t, os.WriteFile(exePath, []byte("old binary"), 0755))

	marker := &UpdateMarker{Version: "1.43.8", DownloadURL: server.URL + "/stripe.tar.gz"}

	start := time.Now()
	reason, err := downloadAndReplace(marker, exePath)

	assert.Error(t, err)
	assert.Equal(t, reasonDownload, reason, "a timed-out transfer is a download failure")
	assert.Less(t, time.Since(start), 30*time.Second, "must give up on the deadline, not hang")

	// The binary the user is running is untouched, and nothing staged is left.
	got, readErr := os.ReadFile(exePath)
	require.NoError(t, readErr)
	assert.Equal(t, []byte("old binary"), got)
	entries, readErr := os.ReadDir(dir)
	require.NoError(t, readErr)
	assert.Len(t, entries, 1)
}
