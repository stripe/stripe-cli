package plugins

import (
	"archive/tar"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/spf13/afero"

	"github.com/stripe/stripe-cli/pkg/ansi"
	"github.com/stripe/stripe-cli/pkg/config"
)

// NodeRuntimeConfig holds configuration for a specific Node.js runtime version
type NodeRuntimeConfig struct {
	Version   string
	Checksums map[string]map[string]string // OS -> Arch -> Checksum
}

// Hardcoded Node.js LTS runtime configurations
// Using Node.js 20.x LTS (Iron) as the default runtime
//
// Checksums are verified from official Node.js distribution over HTTPS.
// For maximum security, checksums can also be verified against GPG signatures.
//
// To update checksums for a new Node.js version:
// 1. Download checksums:
//    curl -fsO "https://nodejs.org/dist/vX.Y.Z/SHASUMS256.txt"
//
// 2. (Optional) Verify GPG signature:
//    curl -fsO "https://nodejs.org/dist/vX.Y.Z/SHASUMS256.txt.asc"
//    curl -fsLo "nodejs-keyring.kbx" "https://github.com/nodejs/release-keys/raw/HEAD/gpg/pubring.kbx"
//    gpgv --keyring="nodejs-keyring.kbx" --output SHASUMS256.txt < SHASUMS256.txt.asc
//
// 3. Extract checksums for each platform:
//    grep "darwin-x64.tar.gz" SHASUMS256.txt      # macOS Intel
//    grep "darwin-arm64.tar.gz" SHASUMS256.txt    # macOS Apple Silicon
//    grep "linux-x64.tar.gz" SHASUMS256.txt       # Linux Intel
//    grep "linux-arm64.tar.gz" SHASUMS256.txt     # Linux ARM
//    grep "win-x64.zip" SHASUMS256.txt            # Windows
//
// 4. Update the checksums in this file and document the verification date
var nodeRuntimeConfigs = map[string]NodeRuntimeConfig{
	"20": {
		Version: "20.18.1",
		// Checksums verified from https://nodejs.org/dist/v20.18.1/SHASUMS256.txt
		// Verified on 2026-02-11
		Checksums: map[string]map[string]string{
			"darwin": {
				"amd64": "c5497dd17c8875b53712edaf99052f961013cedc203964583fc0cfc0aaf93581", // node-v20.18.1-darwin-x64.tar.gz
				"arm64": "9e92ce1032455a9cc419fe71e908b27ae477799371b45a0844eedb02279922a4", // node-v20.18.1-darwin-arm64.tar.gz
			},
			"linux": {
				"amd64": "259e5a8bf2e15ecece65bd2a47153262eda71c0b2c9700d5e703ce4951572784", // node-v20.18.1-linux-x64.tar.gz
				"arm64": "73cd297378572e0bc9dfc187c5ec8cca8d43aee6a596c10ebea1ed5f9ec682b6", // node-v20.18.1-linux-arm64.tar.gz
			},
			"windows": {
				"amd64": "56e5aacdeee7168871721b75819ccacf2367de8761b78eaceacdecd41e04ca03", // node-v20.18.1-win-x64.zip
			},
		},
	},
	// Note: Node.js 22 and 24 configurations below use placeholder checksums
	// These should be updated with real checksums when these LTS versions are released
	"22": {
		Version: "22.13.0",
		// TODO: Replace with verified checksums when Node.js 22 LTS is released
		Checksums: map[string]map[string]string{
			"darwin": {
				"amd64": "cd4b101bf5edeef5fe85bf4bca12b6d5da79e4f3efb0bc0c6b3c28b9a1b7ee2f",
				"arm64": "6f15c8a8f4f23cb5a7a8c8b67c91da6e9f1f3e2f8e8c8a7f6f5f4f3f2f1f0e0",
			},
			"linux": {
				"amd64": "c3c7ebae9dad0df8e91e2e1b8e6fb8e1c9a7f6e5d4c3b2a1f0e9d8c7b6a5f4e3",
				"arm64": "d4d8f9eafbecfe1f3e4e5f6f7f8f9f0f1f2f3f4f5f6f7f8f9f0f1f2f3f4f5f6",
			},
			"windows": {
				"amd64": "e5e9fafbfcfdfe0f1f2f3f4f5f6f7f8f9f0f1f2f3f4f5f6f7f8f9f0f1f2f3f4",
			},
		},
	},
	"24": {
		Version: "24.0.0",
		// TODO: Replace with verified checksums when Node.js 24 LTS is released
		Checksums: map[string]map[string]string{
			"darwin": {
				"amd64": "f6fafbfcfdfe0f1f2f3f4f5f6f7f8f9f0f1f2f3f4f5f6f7f8f9f0f1f2f3f4f5",
				"arm64": "a0a1a2a3a4a5a6a7a8a9b0b1b2b3b4b5c0c1c2c3c4c5c6c7c8c9d0d1d2d3d4d5",
			},
			"linux": {
				"amd64": "b1b2b3b4b5b6b7b8b9c0c1c2c3c4c5c6c7c8c9d0d1d2d3d4d5d6d7d8d9e0e1e2",
				"arm64": "c2c3c4c5c6c7c8c9d0d1d2d3d4d5d6d7d8d9e0e1e2e3e4e5e6e7e8e9f0f1f2f3",
			},
			"windows": {
				"amd64": "d3d4d5d6d7d8d9e0e1e2e3e4e5e6e7e8e9f0f1f2f3f4f5f6f7f8f9a0a1a2a3a4",
			},
		},
	},
}

// GetRuntimesDir returns the directory where runtimes are installed
func GetRuntimesDir(cfg config.IConfig) string {
	configPath := cfg.GetConfigFolder(os.Getenv("XDG_CONFIG_HOME"))
	return filepath.Join(configPath, "runtimes")
}

// GetNodeRuntimePath returns the path to a specific Node.js runtime installation
func GetNodeRuntimePath(cfg config.IConfig, majorVersion string) string {
	runtimeConfig, exists := nodeRuntimeConfigs[majorVersion]
	if !exists {
		return ""
	}
	return filepath.Join(GetRuntimesDir(cfg), "node", runtimeConfig.Version)
}

// GetNodeBinaryPath returns the path to the node executable for a specific version
func GetNodeBinaryPath(cfg config.IConfig, majorVersion string) string {
	runtimePath := GetNodeRuntimePath(cfg, majorVersion)
	if runtimePath == "" {
		return ""
	}

	// Construct path to node binary
	if runtime.GOOS == "windows" {
		return filepath.Join(runtimePath, "node.exe")
	}
	return filepath.Join(runtimePath, "bin", "node")
}

// IsRuntimeInstalled checks if a specific Node.js runtime is already installed
func IsRuntimeInstalled(cfg config.IConfig, fs afero.Fs, majorVersion string) bool {
	runtimePath := GetNodeRuntimePath(cfg, majorVersion)
	if runtimePath == "" {
		return false
	}

	// Check if the node binary exists
	nodeBinary := filepath.Join(runtimePath, "bin", "node")
	if runtime.GOOS == "windows" {
		nodeBinary = filepath.Join(runtimePath, "node.exe")
	}

	exists, err := afero.Exists(fs, nodeBinary)
	return err == nil && exists
}

// InstallNodeRuntime downloads and installs a specific Node.js runtime version
func InstallNodeRuntime(ctx context.Context, cfg config.IConfig, fs afero.Fs, majorVersion string) error {
	runtimeConfig, exists := nodeRuntimeConfigs[majorVersion]
	if !exists {
		return fmt.Errorf("unsupported Node.js version: %s", majorVersion)
	}

	// Check if runtime is already installed
	if IsRuntimeInstalled(cfg, fs, majorVersion) {
		return nil
	}

	opsys := runtime.GOOS
	arch := runtime.GOARCH

	checksum, ok := runtimeConfig.Checksums[opsys][arch]
	if !ok {
		return fmt.Errorf("Node.js %s is not available for %s/%s", majorVersion, opsys, arch)
	}

	spinner := ansi.StartNewSpinner(
		ansi.Faint(fmt.Sprintf("downloading Node.js v%s runtime...", runtimeConfig.Version)),
		os.Stdout,
	)

	// Construct download URL
	downloadURL := buildNodeDownloadURL(runtimeConfig.Version, opsys, arch)

	// Download the runtime
	body, err := FetchRemoteResource(downloadURL)
	if err != nil {
		ansi.StopSpinner(spinner, ansi.Faint(fmt.Sprintf("failed to download Node.js runtime: %s", err)), os.Stdout)
		return err
	}

	// Verify checksum
	if err := verifyChecksum(body, checksum); err != nil {
		ansi.StopSpinner(spinner, ansi.Faint(fmt.Sprintf("failed to verify Node.js runtime: %s", err)), os.Stdout)
		return err
	}

	// Extract and install
	runtimePath := GetNodeRuntimePath(cfg, majorVersion)
	if err := extractRuntime(fs, body, runtimePath, opsys); err != nil {
		ansi.StopSpinner(spinner, ansi.Faint(fmt.Sprintf("failed to extract Node.js runtime: %s", err)), os.Stdout)
		return err
	}

	ansi.StopSpinner(spinner, "", os.Stdout)
	return nil
}

// buildNodeDownloadURL constructs the download URL for Node.js binaries
func buildNodeDownloadURL(version, opsys, arch string) string {
	baseURL := "https://nodejs.org/dist"

	// Map Go arch names to Node.js arch names
	nodeArch := arch
	if arch == "amd64" {
		nodeArch = "x64"
	}

	var filename string
	switch opsys {
	case "darwin":
		filename = fmt.Sprintf("node-v%s-darwin-%s.tar.gz", version, nodeArch)
	case "linux":
		filename = fmt.Sprintf("node-v%s-linux-%s.tar.gz", version, nodeArch)
	case "windows":
		filename = fmt.Sprintf("node-v%s-win-%s.zip", version, nodeArch)
	}

	return fmt.Sprintf("%s/v%s/%s", baseURL, version, filename)
}

// verifyChecksum verifies the SHA256 checksum of downloaded data
func verifyChecksum(data []byte, expectedChecksum string) error {
	hash := sha256.New()
	hash.Write(data)
	actualChecksum := hex.EncodeToString(hash.Sum(nil))

	if actualChecksum != expectedChecksum {
		return fmt.Errorf("checksum mismatch: expected %s, got %s", expectedChecksum, actualChecksum)
	}

	return nil
}

// extractRuntime extracts the downloaded runtime archive
func extractRuntime(fs afero.Fs, data []byte, destPath string, opsys string) error {
	// Create destination directory
	if err := fs.MkdirAll(destPath, 0755); err != nil {
		return fmt.Errorf("failed to create runtime directory: %w", err)
	}

	switch opsys {
	case "darwin", "linux":
		return extractTarGz(fs, data, destPath)
	case "windows":
		return fmt.Errorf("Windows runtime extraction not yet implemented")
	default:
		return fmt.Errorf("unsupported operating system: %s", opsys)
	}
}

// extractTarGz extracts a .tar.gz archive
func extractTarGz(fs afero.Fs, data []byte, destPath string) error {
	gzr, err := gzip.NewReader(strings.NewReader(string(data)))
	if err != nil {
		return fmt.Errorf("failed to create gzip reader: %w", err)
	}
	defer gzr.Close()

	tr := tar.NewReader(gzr)

	for {
		header, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("failed to read tar: %w", err)
		}

		// Skip the top-level directory in the archive (e.g., "node-v20.18.1-darwin-arm64/")
		parts := strings.SplitN(header.Name, "/", 2)
		if len(parts) < 2 {
			continue
		}
		relativePath := parts[1]

		targetPath := filepath.Join(destPath, relativePath)

		switch header.Typeflag {
		case tar.TypeDir:
			if err := fs.MkdirAll(targetPath, os.FileMode(header.Mode)); err != nil {
				return fmt.Errorf("failed to create directory: %w", err)
			}
		case tar.TypeReg:
			if err := fs.MkdirAll(filepath.Dir(targetPath), 0755); err != nil {
				return fmt.Errorf("failed to create parent directory: %w", err)
			}

			outFile, err := fs.OpenFile(targetPath, os.O_CREATE|os.O_RDWR|os.O_TRUNC, os.FileMode(header.Mode))
			if err != nil {
				return fmt.Errorf("failed to create file: %w", err)
			}

			if _, err := io.Copy(outFile, tr); err != nil {
				outFile.Close()
				return fmt.Errorf("failed to write file: %w", err)
			}
			outFile.Close()
		case tar.TypeSymlink:
			if err := fs.MkdirAll(filepath.Dir(targetPath), 0755); err != nil {
				return fmt.Errorf("failed to create parent directory for symlink: %w", err)
			}
			// For afero compatibility, we'll skip symlinks for now
			// In production, you might want to handle these properly
		}
	}

	return nil
}

// GetRuntimeRequirement extracts the required Node.js version from a release
func GetRuntimeRequirement(release Release) (string, bool) {
	if release.Runtime == nil {
		return "", false
	}

	nodeVersion, exists := release.Runtime["node"]
	return nodeVersion, exists
}
