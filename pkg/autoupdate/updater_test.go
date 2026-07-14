package autoupdate

import (
	"archive/tar"
	"compress/gzip"
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"testing"

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

func sha256sum(path string) string {
	data, _ := os.ReadFile(path)
	h := sha256.Sum256(data)
	return hex.EncodeToString(h[:])
}

func TestExtractFromTarGz(t *testing.T) {
	content := []byte("#!/bin/sh\necho hello\n")
	archivePath := createTestTarGz(t, "stripe", content)

	destPath := filepath.Join(t.TempDir(), "stripe")
	err := extractFromTarGz(archivePath, destPath)
	require.NoError(t, err)

	got, err := os.ReadFile(destPath)
	require.NoError(t, err)
	assert.Equal(t, content, got)
}

func TestExtractFromTarGz_NestedPath(t *testing.T) {
	content := []byte("binary content")
	archivePath := createTestTarGz(t, "stripe_1.43.8_linux_arm64/stripe", content)

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

func TestDownloadAndReplace(t *testing.T) {
	content := []byte("#!/bin/sh\necho updated\n")
	archivePath := createTestTarGz(t, "stripe", content)
	archiveData, err := os.ReadFile(archivePath)
	require.NoError(t, err)

	checksum := sha256sum(archivePath)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write(archiveData)
	}))
	defer server.Close()

	dir := t.TempDir()
	exePath := filepath.Join(dir, "stripe")
	require.NoError(t, os.WriteFile(exePath, []byte("old binary"), 0755))

	marker := &UpdateMarker{
		Version:     "1.43.8",
		DownloadURL: server.URL + "/stripe.tar.gz",
		Checksum:    checksum,
	}

	err = downloadAndReplace(marker, exePath)
	require.NoError(t, err)

	got, err := os.ReadFile(exePath)
	require.NoError(t, err)
	assert.Equal(t, content, got)

	if runtime.GOOS != "windows" {
		info, err := os.Stat(exePath)
		require.NoError(t, err)
		assert.Equal(t, os.FileMode(0755), info.Mode().Perm())
	}
}

func TestDownloadAndReplace_BadChecksum(t *testing.T) {
	content := []byte("#!/bin/sh\necho updated\n")
	archivePath := createTestTarGz(t, "stripe", content)
	archiveData, err := os.ReadFile(archivePath)
	require.NoError(t, err)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write(archiveData)
	}))
	defer server.Close()

	dir := t.TempDir()
	exePath := filepath.Join(dir, "stripe")
	require.NoError(t, os.WriteFile(exePath, []byte("old binary"), 0755))

	marker := &UpdateMarker{
		Version:     "1.43.8",
		DownloadURL: server.URL + "/stripe.tar.gz",
		Checksum:    "0000000000000000000000000000000000000000000000000000000000000000",
	}

	err = downloadAndReplace(marker, exePath)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "checksum verification failed")

	got, _ := os.ReadFile(exePath)
	assert.Equal(t, []byte("old binary"), got, "original binary should be unchanged")
}

func TestDownloadAndReplace_ServerError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	dir := t.TempDir()
	exePath := filepath.Join(dir, "stripe")
	require.NoError(t, os.WriteFile(exePath, []byte("old binary"), 0755))

	marker := &UpdateMarker{
		Version:     "1.43.8",
		DownloadURL: server.URL + "/stripe.tar.gz",
	}

	err := downloadAndReplace(marker, exePath)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "status 500")
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
