package plugins

import (
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

	// Create the runtime directory structure
	runtimePath := GetNodeRuntimePath(config, "20")

	// Use the correct path based on OS
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
			url := buildNodeDownloadURL(tt.version, tt.os, tt.arch)
			require.Equal(t, tt.expected, url)
		})
	}
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

func TestGetNodeBinaryPath(t *testing.T) {
	config := &TestConfig{}
	config.InitConfig()

	// Valid version should return a path
	path := GetNodeBinaryPath(config, "20")
	require.NotEmpty(t, path)
	require.Contains(t, path, "node")
	require.Contains(t, path, "20.18.1")

	// Should use bin/node on Unix-like systems
	if runtime.GOOS != "windows" {
		require.Contains(t, path, "bin/node")
	}

	// Invalid version should return empty string
	path = GetNodeBinaryPath(config, "99")
	require.Empty(t, path)
}

func TestIsWithinDirectory(t *testing.T) {
	tests := []struct {
		name       string
		destPath   string
		targetPath string
		expected   bool
	}{
		{
			name:       "valid subdirectory",
			destPath:   "/tmp/runtime",
			targetPath: "/tmp/runtime/bin/node",
			expected:   true,
		},
		{
			name:       "valid same directory",
			destPath:   "/tmp/runtime",
			targetPath: "/tmp/runtime",
			expected:   true,
		},
		{
			name:       "path traversal attempt with ..",
			destPath:   "/tmp/runtime",
			targetPath: "/tmp/runtime/../../../etc/passwd",
			expected:   false,
		},
		{
			name:       "direct parent escape",
			destPath:   "/tmp/runtime",
			targetPath: "/tmp/other",
			expected:   false,
		},
		{
			name:       "path traversal in middle",
			destPath:   "/tmp/runtime",
			targetPath: "/tmp/runtime/subdir/../../outside/file",
			expected:   false,
		},
		{
			name:       "clean path with dots",
			destPath:   "/tmp/runtime",
			targetPath: "/tmp/runtime/./bin/node",
			expected:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isWithinDirectory(tt.destPath, tt.targetPath)
			require.Equal(t, tt.expected, result, "isWithinDirectory(%s, %s)", tt.destPath, tt.targetPath)
		})
	}
}
