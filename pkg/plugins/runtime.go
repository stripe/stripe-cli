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
//
//  1. Download checksums:
//     curl -fsO "https://nodejs.org/dist/vX.Y.Z/SHASUMS256.txt"
//
//  2. (Optional) Verify GPG signature:
//     curl -fsO "https://nodejs.org/dist/vX.Y.Z/SHASUMS256.txt.asc"
//     curl -fsLo "nodejs-keyring.kbx" "https://github.com/nodejs/release-keys/raw/HEAD/gpg/pubring.kbx"
//     gpgv --keyring="nodejs-keyring.kbx" --output SHASUMS256.txt < SHASUMS256.txt.asc
//
//  3. Extract checksums for each platform:
//     grep "darwin-x64.tar.gz" SHASUMS256.txt      # macOS Intel
//     grep "darwin-arm64.tar.gz" SHASUMS256.txt    # macOS Apple Silicon
//     grep "linux-x64.tar.gz" SHASUMS256.txt       # Linux Intel
//     grep "linux-arm64.tar.gz" SHASUMS256.txt     # Linux ARM
//     grep "win-x64.zip" SHASUMS256.txt            # Windows
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
	"22": {
		Version: "22.13.0",
		// Checksums verified from https://nodejs.org/dist/v22.13.0/SHASUMS256.txt
		// Verified on 2026-03-04
		Checksums: map[string]map[string]string{
			"darwin": {
				"amd64": "cfaaf5edde585a15547f858f5b3b62a292cf5929a23707b6f1e36c29a32487be", // node-v22.13.0-darwin-x64.tar.gz
				"arm64": "bc1e374e7393e2f4b20e5bbc157d02e9b1fb2c634b2f992136b38fb8ca2023b7", // node-v22.13.0-darwin-arm64.tar.gz
			},
			"linux": {
				"amd64": "9a33e89093a0d946c54781dcb3ccab4ccf7538a7135286528ca41ca055e9b38f", // node-v22.13.0-linux-x64.tar.gz
				"arm64": "e0cc088cb4fb2e945d3d5c416c601e1101a15f73e0f024c9529b964d9f6dce5b", // node-v22.13.0-linux-arm64.tar.gz
			},
			"windows": {
				"amd64": "b0feb09ebf41328628e7383f7a092fb7342ce1e05c867a90cf8f1379205a8429", // node-v22.13.0-win-x64.zip
			},
		},
	},
	"24": {
		Version: "24.0.0",
		// Checksums verified from https://nodejs.org/dist/v24.0.0/SHASUMS256.txt
		// Verified on 2026-03-04
		Checksums: map[string]map[string]string{
			"darwin": {
				"amd64": "f716b3ce14a7e37a6cbf97c9de10d444d7da07ef833cd8da81dd944d111e6a4a", // node-v24.0.0-darwin-x64.tar.gz
				"arm64": "194e2f3dd3ec8c2adcaa713ed40f44c5ca38467880e160974ceac1659be60121", // node-v24.0.0-darwin-arm64.tar.gz
			},
			"linux": {
				"amd64": "b760ed6de40c35a25eb011b3cf5943d35d7a76f0c8c331d5a801e10925826cb3", // node-v24.0.0-linux-x64.tar.gz
				"arm64": "4104136ddd3d2f167d799f1b21bac72ccf500d80c24be849195f831df6371b83", // node-v24.0.0-linux-arm64.tar.gz
			},
			"windows": {
				"amd64": "3d0fff80c87bb9a8d7f49f2f27832aa34a1477d137af46f5b14df5498be81304", // node-v24.0.0-win-x64.zip
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

		// Prevent Zip Slip: ensure the resolved path stays within destPath
		if !strings.HasPrefix(targetPath+string(os.PathSeparator), filepath.Clean(destPath)+string(os.PathSeparator)) {
			return fmt.Errorf("illegal file path in archive: %s", header.Name)
		}

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
