package autoupdate

import (
	"archive/tar"
	"archive/zip"
	"bytes"
	"compress/gzip"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestIsNewer(t *testing.T) {
	assert.True(t, isNewer("1.2.3", "1.2.4"))
	assert.True(t, isNewer("v1.2.3", "v1.3.0"))
	assert.True(t, isNewer("1.9.0", "1.10.0"), "versions compare numerically, not lexically")
	assert.False(t, isNewer("1.2.3", "1.2.3"))
	assert.False(t, isNewer("1.2.4", "1.2.3"), "never downgrade")
	assert.False(t, isNewer("master", "1.2.3"), "a build from source has no version to compare")
	assert.False(t, isNewer("1.2.3", "not-a-version"))
}

func TestAssetNames(t *testing.T) {
	tests := []struct {
		goos, goarch       string
		archive, checksums string
	}{
		{"darwin", "amd64", "stripe_1.2.3_mac-os_x86_64.tar.gz", "stripe-mac-checksums.txt"},
		{"darwin", "arm64", "stripe_1.2.3_mac-os_arm64.tar.gz", "stripe-mac-checksums.txt"},
		{"linux", "amd64", "stripe_1.2.3_linux_x86_64.tar.gz", "stripe-linux-checksums.txt"},
		{"linux", "arm64", "stripe_1.2.3_linux_arm64.tar.gz", "stripe-linux-checksums.txt"},
		{"windows", "amd64", "stripe_1.2.3_windows_x86_64.zip", "stripe-windows-checksums.txt"},
		{"windows", "386", "stripe_1.2.3_windows_i386.zip", "stripe-windows-checksums.txt"},
		// No Windows arm64 build is published, so ARM machines take the emulated
		// x64 archive rather than 404ing.
		{"windows", "arm64", "stripe_1.2.3_windows_x86_64.zip", "stripe-windows-checksums.txt"},
	}

	for _, test := range tests {
		archive, checksums, err := assetNames("1.2.3", test.goos, test.goarch)
		require.NoError(t, err, "%s/%s", test.goos, test.goarch)
		assert.Equal(t, test.archive, archive)
		assert.Equal(t, test.checksums, checksums)
	}
}

func TestAssetNamesRejectsUnpublishedPlatforms(t *testing.T) {
	_, _, err := assetNames("1.2.3", "plan9", "amd64")
	assert.ErrorContains(t, err, "unsupported operating system")

	// The mac and linux releases build amd64 and arm64 only.
	_, _, err = assetNames("1.2.3", "linux", "386")
	assert.ErrorContains(t, err, "unsupported architecture")

	_, _, err = assetNames("1.2.3", "darwin", "386")
	assert.ErrorContains(t, err, "unsupported architecture")
}

func TestLookupChecksum(t *testing.T) {
	contents := strings.Join([]string{
		"aaaa  stripe_1.2.3_windows_i386.zip",
		"bbbb  stripe_1.2.3_windows_x86_64.zip",
		"cccc *stripe_1.2.3_linux_x86_64.tar.gz",
		"",
	}, "\n")

	sum, err := lookupChecksum(contents, "stripe_1.2.3_windows_x86_64.zip")
	require.NoError(t, err)
	assert.Equal(t, "bbbb", sum)

	sum, err = lookupChecksum(contents, "stripe_1.2.3_linux_x86_64.tar.gz")
	require.NoError(t, err)
	assert.Equal(t, "cccc", sum)

	// A partial name must not match a longer one, and "." must not act as a regex
	// wildcard: either would verify a different archive than the one downloaded.
	_, err = lookupChecksum(contents, "stripe_1.2.3_windows_x86_64.zi")
	assert.ErrorContains(t, err, "no checksum published")

	_, err = lookupChecksum(contents, "stripe_1_2_3_windows_x86_64.zip")
	assert.ErrorContains(t, err, "no checksum published")

	_, err = lookupChecksum(contents, "stripe_9.9.9_windows_x86_64.zip")
	assert.ErrorContains(t, err, "no checksum published")
}

func TestVerifySHA256(t *testing.T) {
	path := filepath.Join(t.TempDir(), "archive")
	require.NoError(t, os.WriteFile(path, []byte("payload"), 0600))

	digest := sha256.Sum256([]byte("payload"))
	expected := hex.EncodeToString(digest[:])

	require.NoError(t, verifySHA256(path, expected))
	require.NoError(t, verifySHA256(path, strings.ToUpper(expected)), "digest case is not significant")

	err := verifySHA256(path, strings.Repeat("0", 64))
	assert.ErrorContains(t, err, "checksum mismatch")
}

func TestExtractBinary(t *testing.T) {
	dir := t.TempDir()

	tarball := filepath.Join(dir, "release.tar.gz")
	require.NoError(t, os.WriteFile(tarball, buildTarGz(t, binaryName(), "tar payload"), 0600))

	extracted, err := extractBinary(tarball, dir)
	require.NoError(t, err)
	assertFileContents(t, extracted, "tar payload")

	zipped := filepath.Join(dir, "release.zip")
	require.NoError(t, os.WriteFile(zipped, buildZip(t, binaryName(), "zip payload"), 0600))

	extracted, err = extractBinary(zipped, dir)
	require.NoError(t, err)
	assertFileContents(t, extracted, "zip payload")
}

func TestExtractBinaryRejectsAnArchiveWithoutTheBinary(t *testing.T) {
	dir := t.TempDir()

	tarball := filepath.Join(dir, "release.tar.gz")
	require.NoError(t, os.WriteFile(tarball, buildTarGz(t, "README.md", "not a binary"), 0600))

	_, err := extractBinary(tarball, dir)
	assert.ErrorContains(t, err, "does not contain")

	zipped := filepath.Join(dir, "release.zip")
	require.NoError(t, os.WriteFile(zipped, buildZip(t, "README.md", "not a binary"), 0600))

	_, err = extractBinary(zipped, dir)
	assert.ErrorContains(t, err, "does not contain")
}

func TestDownloadVerifiesAndExtracts(t *testing.T) {
	server, archive, checksums := serveRelease(t, "9.9.9", "the new binary", true)
	defer server.Close()

	dir := t.TempDir()

	binary, err := download(t.Context(), "9.9.9", dir)
	require.NoError(t, err)
	assertFileContents(t, binary, "the new binary")

	// Both assets were fetched under the names the release publishes.
	assert.FileExists(t, filepath.Join(dir, archive))
	assert.NotEmpty(t, checksums)
}

func TestDownloadRejectsATamperedArchive(t *testing.T) {
	server, _, _ := serveRelease(t, "9.9.9", "the new binary", false)
	defer server.Close()

	_, err := download(t.Context(), "9.9.9", t.TempDir())
	assert.ErrorContains(t, err, "checksum mismatch")
}

func TestDownloadReportsAMissingRelease(t *testing.T) {
	server := httptest.NewServer(http.NotFoundHandler())
	defer server.Close()

	swapReleaseBaseURL(t, server.URL)

	_, err := download(t.Context(), "9.9.9", t.TempDir())
	assert.ErrorContains(t, err, "could not download")
}

// serveRelease stands up a stand-in for the GitHub release download host, serving
// an archive built for the platform the test is running on. When honest is false
// the published checksum does not match the archive.
func serveRelease(t *testing.T, release, payload string, honest bool) (*httptest.Server, string, string) {
	t.Helper()

	archive, checksums, err := assetNames(release, runtime.GOOS, runtime.GOARCH)
	require.NoError(t, err)

	var body []byte
	if strings.HasSuffix(archive, ".zip") {
		body = buildZip(t, binaryName(), payload)
	} else {
		body = buildTarGz(t, binaryName(), payload)
	}

	digest := sha256.Sum256(body)
	published := hex.EncodeToString(digest[:])

	if !honest {
		published = strings.Repeat("0", 64)
	}

	sums := fmt.Sprintf("%s  %s\n", published, archive)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case strings.HasSuffix(r.URL.Path, archive):
			_, _ = w.Write(body)
		case strings.HasSuffix(r.URL.Path, checksums):
			_, _ = w.Write([]byte(sums))
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))

	swapReleaseBaseURL(t, server.URL)

	return server, archive, sums
}

func swapReleaseBaseURL(t *testing.T, url string) {
	t.Helper()

	original := releaseBaseURL
	releaseBaseURL = url

	t.Cleanup(func() { releaseBaseURL = original })
}

func buildTarGz(t *testing.T, name, contents string) []byte {
	t.Helper()

	var buffer bytes.Buffer

	gzipWriter := gzip.NewWriter(&buffer)
	tarWriter := tar.NewWriter(gzipWriter)

	require.NoError(t, tarWriter.WriteHeader(&tar.Header{
		Name:     name,
		Mode:     0755,
		Size:     int64(len(contents)),
		Typeflag: tar.TypeReg,
	}))

	_, err := tarWriter.Write([]byte(contents))
	require.NoError(t, err)
	require.NoError(t, tarWriter.Close())
	require.NoError(t, gzipWriter.Close())

	return buffer.Bytes()
}

func buildZip(t *testing.T, name, contents string) []byte {
	t.Helper()

	var buffer bytes.Buffer

	zipWriter := zip.NewWriter(&buffer)

	entry, err := zipWriter.Create(name)
	require.NoError(t, err)

	_, err = entry.Write([]byte(contents))
	require.NoError(t, err)
	require.NoError(t, zipWriter.Close())

	return buffer.Bytes()
}

func assertFileContents(t *testing.T, path, expected string) {
	t.Helper()

	actual, err := os.ReadFile(path)
	require.NoError(t, err)
	assert.Equal(t, expected, string(actual))
}
