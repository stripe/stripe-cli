package plugins

import (
	"archive/tar"
	"archive/zip"
	"bytes"
	"compress/gzip"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/spf13/afero"
	"github.com/stretchr/testify/require"
)

func TestGetRuntimesDir(t *testing.T) {
	config := &TestConfig{}
	config.InitConfig()

	runtimesDir := GetRuntimesDir(config)
	require.Contains(t, runtimesDir, "runtimes")
}

func TestGetNodeRuntimePath(t *testing.T) {
	config := &TestConfig{}
	config.InitConfig()

	// Test valid version
	path := GetNodeRuntimePath(config, "20")
	require.NotEmpty(t, path)
	require.Contains(t, path, "node")
	require.Contains(t, path, "20.18.1")

	// Test invalid version
	path = GetNodeRuntimePath(config, "99")
	require.Empty(t, path)
}

func TestIsRuntimeInstalled(t *testing.T) {
	config := &TestConfig{}
	config.InitConfig()
	fs := afero.NewMemMapFs()

	// Runtime should not be installed initially
	require.False(t, IsRuntimeInstalled(config, fs, "20"))

	// Create the runtime directory structure at the platform-correct path
	runtimePath := GetNodeRuntimePath(config, "20")
	var nodeBinary string
	if runtime.GOOS == "windows" {
		nodeBinary = filepath.Join(runtimePath, "node.exe")
	} else {
		nodeBinary = filepath.Join(runtimePath, "bin", "node")
		fs.MkdirAll(filepath.Join(runtimePath, "bin"), 0755)
	}
	afero.WriteFile(fs, nodeBinary, []byte("fake node binary"), 0755)

	// Now runtime should be detected as installed
	require.True(t, IsRuntimeInstalled(config, fs, "20"))
}

func TestGetRuntimeRequirement(t *testing.T) {
	// Release with Node.js runtime requirement
	releaseWithRuntime := Release{
		Version: "1.0.0",
		Runtime: map[string]string{"node": "20"},
	}

	nodeVersion, requiresNode := GetRuntimeRequirement(releaseWithRuntime)
	require.True(t, requiresNode)
	require.Equal(t, "20", nodeVersion)

	// Release without runtime requirement
	releaseWithoutRuntime := Release{
		Version: "1.0.0",
		Runtime: nil,
	}

	_, requiresNode = GetRuntimeRequirement(releaseWithoutRuntime)
	require.False(t, requiresNode)

	// Release with empty runtime
	releaseWithEmptyRuntime := Release{
		Version: "1.0.0",
		Runtime: map[string]string{},
	}

	_, requiresNode = GetRuntimeRequirement(releaseWithEmptyRuntime)
	require.False(t, requiresNode)
}

func TestBuildNodeDownloadURL(t *testing.T) {
	tests := []struct {
		version  string
		os       string
		arch     string
		expected string
	}{
		{
			version:  "20.18.1",
			os:       "darwin",
			arch:     "amd64",
			expected: "https://nodejs.org/dist/v20.18.1/node-v20.18.1-darwin-x64.tar.gz",
		},
		{
			version:  "20.18.1",
			os:       "darwin",
			arch:     "arm64",
			expected: "https://nodejs.org/dist/v20.18.1/node-v20.18.1-darwin-arm64.tar.gz",
		},
		{
			version:  "20.18.1",
			os:       "linux",
			arch:     "amd64",
			expected: "https://nodejs.org/dist/v20.18.1/node-v20.18.1-linux-x64.tar.gz",
		},
		{
			version:  "20.18.1",
			os:       "windows",
			arch:     "amd64",
			expected: "https://nodejs.org/dist/v20.18.1/node-v20.18.1-win-x64.zip",
		},
	}

	for _, tt := range tests {
		t.Run(tt.os+"-"+tt.arch, func(t *testing.T) {
			url, err := buildNodeDownloadURL(tt.version, tt.os, tt.arch)
			require.NoError(t, err)
			require.Equal(t, tt.expected, url)
		})
	}
}

func TestBuildNodeDownloadURLUnsupportedOS(t *testing.T) {
	_, err := buildNodeDownloadURL("20.18.1", "freebsd", "amd64")
	require.Error(t, err)
	require.Contains(t, err.Error(), "unsupported operating system")
}

func TestVerifyChecksum(t *testing.T) {
	data := []byte("test data")
	// SHA256 of "test data"
	validChecksum := "916f0027a575074ce72a331777c3478d6513f786a591bd892da1a577bf2335f9"

	// Valid checksum should pass
	err := verifyChecksum(data, validChecksum)
	require.Nil(t, err)

	// Invalid checksum should fail
	err = verifyChecksum(data, "invalid_checksum")
	require.NotNil(t, err)
	require.Contains(t, err.Error(), "checksum mismatch")
}

func TestExtractTarGzZipSlipProtection(t *testing.T) {
	fs := afero.NewMemMapFs()
	destPath := "/tmp/test-extract"

	// Create a malicious tar.gz archive with path traversal attempts
	var buf bytes.Buffer
	gzw := gzip.NewWriter(&buf)
	tw := tar.NewWriter(gzw)

	// Test case 1: Path with ".." trying to escape
	maliciousContent := []byte("malicious content")
	err := tw.WriteHeader(&tar.Header{
		Name:     "node-v20.0.0-darwin-arm64/../../../etc/passwd",
		Mode:     0644,
		Size:     int64(len(maliciousContent)),
		Typeflag: tar.TypeReg,
	})
	require.NoError(t, err)
	_, err = tw.Write(maliciousContent)
	require.NoError(t, err)

	tw.Close()
	gzw.Close()

	// Attempt to extract should fail due to path traversal
	err = extractTarGz(fs, buf.Bytes(), destPath)
	require.Error(t, err)
	require.Contains(t, err.Error(), "illegal file path")

	// Verify the malicious file was not created outside destPath
	exists, _ := afero.Exists(fs, "/etc/passwd")
	require.False(t, exists)
}

func TestExtractTarGzValidArchive(t *testing.T) {
	fs := afero.NewMemMapFs()
	destPath := "/tmp/test-extract"

	// Create a valid tar.gz archive
	var buf bytes.Buffer
	gzw := gzip.NewWriter(&buf)
	tw := tar.NewWriter(gzw)

	// Add a valid file
	validContent := []byte("valid content")
	err := tw.WriteHeader(&tar.Header{
		Name:     "node-v20.0.0-darwin-arm64/bin/node",
		Mode:     0755,
		Size:     int64(len(validContent)),
		Typeflag: tar.TypeReg,
	})
	require.NoError(t, err)
	_, err = tw.Write(validContent)
	require.NoError(t, err)

	tw.Close()
	gzw.Close()

	// Extraction should succeed
	err = extractTarGz(fs, buf.Bytes(), destPath)
	require.NoError(t, err)

	// Verify the file was created in the correct location
	exists, _ := afero.Exists(fs, filepath.Join(destPath, "bin", "node"))
	require.True(t, exists)

	// Verify the content
	content, err := afero.ReadFile(fs, filepath.Join(destPath, "bin", "node"))
	require.NoError(t, err)
	require.Equal(t, validContent, content)
}

func TestExtractZipZipSlipProtection(t *testing.T) {
	fs := afero.NewMemMapFs()
	destPath := "/tmp/test-extract"

	// Create a malicious zip archive with path traversal attempts
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)

	// Test case: Path with ".." trying to escape
	maliciousContent := []byte("malicious content")
	fw, err := zw.Create("node-v20.0.0-win-x64/../../../windows/system32/evil.dll")
	require.NoError(t, err)
	_, err = fw.Write(maliciousContent)
	require.NoError(t, err)

	zw.Close()

	// Attempt to extract should fail due to path traversal
	err = extractZip(fs, buf.Bytes(), destPath)
	require.Error(t, err)
	require.Contains(t, err.Error(), "illegal file path")

	// Verify the malicious file was not created outside destPath
	exists, _ := afero.Exists(fs, "/windows/system32/evil.dll")
	require.False(t, exists)
}

func TestExtractZipValidArchive(t *testing.T) {
	fs := afero.NewMemMapFs()
	destPath := "/tmp/test-extract"

	// Create a valid zip archive
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)

	// Add a directory
	_, err := zw.Create("node-v20.0.0-win-x64/bin/")
	require.NoError(t, err)

	// Add a valid file
	validContent := []byte("valid windows node.exe content")
	fw, err := zw.Create("node-v20.0.0-win-x64/node.exe")
	require.NoError(t, err)
	_, err = fw.Write(validContent)
	require.NoError(t, err)

	zw.Close()

	// Extraction should succeed
	err = extractZip(fs, buf.Bytes(), destPath)
	require.NoError(t, err)

	// Verify the file was created in the correct location
	exists, _ := afero.Exists(fs, filepath.Join(destPath, "node.exe"))
	require.True(t, exists)

	// Verify the content
	content, err := afero.ReadFile(fs, filepath.Join(destPath, "node.exe"))
	require.NoError(t, err)
	require.Equal(t, validContent, content)
}

func TestGetReleaseForVersion(t *testing.T) {
	plugin := Plugin{
		Shortname: "test-plugin",
		Releases: []Release{
			{
				Arch:    "amd64",
				OS:      "darwin",
				Version: "1.0.0",
				Sum:     "abc123",
				Runtime: map[string]string{"node": "20"},
			},
			{
				Arch:    "arm64",
				OS:      "darwin",
				Version: "1.0.0",
				Sum:     "def456",
				Runtime: map[string]string{"node": "20"},
			},
			{
				Arch:    "amd64",
				OS:      "linux",
				Version: "1.0.0",
				Sum:     "ghi789",
				Runtime: map[string]string{"node": "20"},
			},
			{
				Arch:    "amd64",
				OS:      "windows",
				Version: "1.0.0",
				Sum:     "jkl012",
				Runtime: map[string]string{"node": "20"},
			},
		},
	}

	// Should find release for current platform and version
	release := plugin.getReleaseForVersion("1.0.0")
	require.NotNil(t, release)
	require.Equal(t, "1.0.0", release.Version)

	// Should return nil for non-existent version
	release = plugin.getReleaseForVersion("2.0.0")
	require.Nil(t, release)
}
