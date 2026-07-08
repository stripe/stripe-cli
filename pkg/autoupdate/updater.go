package autoupdate

import (
	"archive/tar"
	"compress/gzip"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"syscall"

	log "github.com/sirupsen/logrus"

	"github.com/stripe/stripe-cli/pkg/version"
)

// ApplyIfPending checks for a pending update marker and applies it.
// If an update is applied, it re-execs the current process with the new binary.
// This function only returns if no update was applied.
func ApplyIfPending() {
	if version.Version == "master" {
		return
	}
	if isOptedOut() {
		return
	}
	if !isCurlInstall() {
		return
	}

	marker := readMarker()
	if marker == nil {
		return
	}

	// Don't apply if we're already on this version
	current := strings.TrimPrefix(version.Version, "v")
	target := strings.TrimPrefix(marker.Version, "v")
	if current == target {
		clearMarker()
		return
	}

	exe, err := os.Executable()
	if err != nil {
		log.Debug("autoupdate: cannot determine executable path: ", err)
		clearMarker()
		return
	}

	exe, err = filepath.EvalSymlinks(exe)
	if err != nil {
		log.Debug("autoupdate: cannot resolve symlinks: ", err)
		clearMarker()
		return
	}

	fmt.Fprintf(os.Stderr, "Automatically updating Stripe CLI from %s to %s.\n", current, target)
	fmt.Fprintf(os.Stderr, "To disable auto-update, set STRIPE_NO_AUTO_UPDATE=1 or add auto_update = false to ~/.config/stripe/config.toml\n")

	if err := downloadAndReplace(marker, exe); err != nil {
		fmt.Fprintf(os.Stderr, "Auto-update failed: %v. Continuing with current version.\n", err)
		clearMarker()
		return
	}

	clearMarker()
	fmt.Fprintf(os.Stderr, "Updated successfully ✓\n")

	// Re-exec with the updated binary
	reexec(exe)
}

func downloadAndReplace(marker *updateMarker, exePath string) error {
	resp, err := http.Get(marker.DownloadURL) //nolint:gosec
	if err != nil {
		return fmt.Errorf("download failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("download returned status %d", resp.StatusCode)
	}

	// Download archive to a temp file
	tmpArchive, err := os.CreateTemp(filepath.Dir(exePath), "stripe-update-archive-*")
	if err != nil {
		return fmt.Errorf("cannot create temp file: %w", err)
	}
	tmpArchivePath := tmpArchive.Name()
	defer os.Remove(tmpArchivePath)

	if _, err := io.Copy(tmpArchive, resp.Body); err != nil {
		tmpArchive.Close()
		return fmt.Errorf("download interrupted: %w", err)
	}
	tmpArchive.Close()

	// Verify checksum of the archive
	if marker.Checksum != "" && !verifyChecksum(tmpArchivePath, marker.Checksum) {
		return fmt.Errorf("checksum verification failed")
	}

	// Extract binary from archive
	tmpBinary, err := os.CreateTemp(filepath.Dir(exePath), "stripe-update-*")
	if err != nil {
		return fmt.Errorf("cannot create temp binary: %w", err)
	}
	tmpBinaryPath := tmpBinary.Name()
	tmpBinary.Close()

	if err := extractBinary(tmpArchivePath, tmpBinaryPath); err != nil {
		os.Remove(tmpBinaryPath)
		return fmt.Errorf("extraction failed: %w", err)
	}

	// Make executable
	if err := os.Chmod(tmpBinaryPath, 0755); err != nil {
		os.Remove(tmpBinaryPath)
		return fmt.Errorf("chmod failed: %w", err)
	}

	// Atomic rename
	if err := os.Rename(tmpBinaryPath, exePath); err != nil {
		os.Remove(tmpBinaryPath)
		return fmt.Errorf("cannot replace binary: %w", err)
	}

	return nil
}

func extractBinary(archivePath, destPath string) error {
	if runtime.GOOS == "windows" {
		return extractFromZip(archivePath, destPath)
	}
	return extractFromTarGz(archivePath, destPath)
}

func extractFromTarGz(archivePath, destPath string) error {
	f, err := os.Open(archivePath)
	if err != nil {
		return err
	}
	defer f.Close()

	gz, err := gzip.NewReader(f)
	if err != nil {
		return err
	}
	defer gz.Close()

	tr := tar.NewReader(gz)
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}

		if filepath.Base(hdr.Name) == "stripe" && hdr.Typeflag == tar.TypeReg {
			out, err := os.Create(destPath)
			if err != nil {
				return err
			}
			if _, err := io.Copy(out, tr); err != nil {
				out.Close()
				return err
			}
			return out.Close()
		}
	}
	return fmt.Errorf("stripe binary not found in archive")
}

func extractFromZip(archivePath, destPath string) error {
	// Windows zip extraction - using archive/zip
	// Import is at the top level but only used on windows path
	return extractFromZipImpl(archivePath, destPath)
}

func reexec(exe string) {
	err := syscall.Exec(exe, os.Args, os.Environ())
	if err != nil {
		log.Debug("autoupdate: re-exec failed: ", err)
	}
}
