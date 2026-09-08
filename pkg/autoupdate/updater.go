package autoupdate

import (
	"archive/tar"
	"archive/zip"
	"bytes"
	"compress/gzip"
	"fmt"
	"io"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"runtime"
	"strings"

	log "github.com/sirupsen/logrus"

	"github.com/stripe/stripe-cli/pkg/version"
)

const (
	maxReleaseNoteLines = 15
	maxReleaseNoteRunes = 1000
)

// ApplyIfPending checks for a pending update marker and applies it.
// If an update is applied, it re-execs the current process with the new binary.
// This function only returns if no update was applied — or, on Windows, if the
// new binary could not be started, in which case this invocation carries on
// running the image it already has.
func ApplyIfPending() {
	if version.Version == "master" {
		return
	}
	if !IsCurlInstall() {
		return
	}

	exe, err := resolvedExecutable()
	if err != nil {
		log.Debug("autoupdate: cannot determine executable path: ", err)
		return
	}

	// Ahead of the opt-out check, so that turning auto-update off does not leave
	// the outgoing binary from an earlier update parked next to the new one
	// forever. On Windows the update that installed it could not delete it: a
	// process was still running from that file. This one is not, so it can.
	removeSupersededBinary(exe)

	if IsOptedOut() {
		return
	}

	marker := ReadMarker()
	if marker == nil {
		return
	}

	current := strings.TrimPrefix(version.Version, "v")
	target := strings.TrimPrefix(marker.Version, "v")
	if current == target {
		ClearMarker()
		return
	}

	fmt.Fprintf(os.Stderr, "Automatically updating Stripe CLI from %s to %s.\n", current, target)
	fmt.Fprintf(os.Stderr, "To disable auto-update, set STRIPE_NO_AUTO_UPDATE=1 or add auto_update = false to ~/.config/stripe/config.toml\n")

	if err := downloadAndReplace(marker, exe); err != nil {
		fmt.Fprintf(os.Stderr, "Auto-update failed: %v. Continuing with current version.\n", err)
		sendTelemetryEvent("Auto-Update Failed", fmt.Sprintf("from=%s to=%s error=%s", current, target, err.Error()))
		ClearMarker()
		return
	}

	ClearMarker()
	fmt.Fprintf(os.Stderr, "Updated successfully ✓\n")
	if releaseNotes := formatReleaseNotes(marker.ReleaseNotes, marker.Version); releaseNotes != "" {
		fmt.Fprint(os.Stderr, releaseNotes)
		if !strings.HasSuffix(releaseNotes, "\n") {
			fmt.Fprintln(os.Stderr)
		}
	}
	sendTelemetryEvent("Auto-Update Succeeded", fmt.Sprintf("from=%s to=%s", current, target))

	reexec(exe)
}

func formatReleaseNotes(notes, markerVersion string) string {
	if notes == "" {
		return ""
	}

	runes := []rune(notes)
	cutoff := len(runes)
	truncated := false

	if cutoff > maxReleaseNoteRunes {
		cutoff = maxReleaseNoteRunes
		truncated = true
	}

	lineCount := 0
	for i, r := range runes {
		if r != '\n' {
			continue
		}
		lineCount++
		if lineCount == maxReleaseNoteLines && i+1 < len(runes) {
			if i+1 < cutoff {
				cutoff = i + 1
			}
			truncated = true
			break
		}
	}

	formatted := string(runes[:cutoff])
	if !truncated {
		return formatted
	}

	if !strings.HasSuffix(formatted, "\n") {
		formatted += "\n"
	}
	version := strings.TrimLeft(markerVersion, "v")
	return fmt.Sprintf("%sFull release notes: https://github.com/stripe/stripe-cli/releases/tag/v%s", formatted, version)
}

func downloadAndReplace(marker *UpdateMarker, exePath string) error {
	resp, err := http.Get(marker.DownloadURL) //nolint:gosec
	if err != nil {
		return fmt.Errorf("download failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("download returned status %d", resp.StatusCode)
	}

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

	if marker.Checksum != "" && !VerifyChecksum(tmpArchivePath, marker.Checksum) {
		return fmt.Errorf("checksum verification failed")
	}

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

	if err := replaceBinary(exePath, tmpBinaryPath); err != nil {
		os.Remove(tmpBinaryPath)
		return err
	}

	return nil
}

// resolvedExecutable is the path of the running binary with every symlink
// resolved. A user may have symlinked stripe onto their PATH, or a component of
// the install path may itself be a link, and the update has to land on the real
// file.
func resolvedExecutable() (string, error) {
	exe, err := os.Executable()
	if err != nil {
		return "", err
	}

	return filepath.EvalSymlinks(exe)
}

// binaryName is the name of the executable inside the release archive, which is
// also the name it is installed under. goreleaser appends .exe to Windows builds.
func binaryName() string {
	if runtime.GOOS == "windows" {
		return "stripe.exe"
	}

	return "stripe"
}

// extractBinary pulls the stripe executable out of a release archive.
//
// Which format that is is read off the file rather than its name: the archive was
// downloaded to a temporary path whose name says nothing about its contents. Mac
// and Linux releases ship a .tar.gz, Windows a .zip.
func extractBinary(archivePath, destPath string) error {
	zipped, err := isZipArchive(archivePath)
	if err != nil {
		return err
	}

	if zipped {
		return extractFromZip(archivePath, destPath)
	}

	return extractFromTarGz(archivePath, destPath)
}

func isZipArchive(archivePath string) (bool, error) {
	f, err := os.Open(archivePath)
	if err != nil {
		return false, err
	}
	defer f.Close()

	magic := make([]byte, 4)
	if _, err := io.ReadFull(f, magic); err != nil {
		// Too short to be either format. Let the tar reader say so, so that the
		// error names what the file was expected to be.
		return false, nil
	}

	return bytes.Equal(magic, []byte("PK\x03\x04")), nil
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

		if path.Base(hdr.Name) == binaryName() && hdr.Typeflag == tar.TypeReg {
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
	r, err := zip.OpenReader(archivePath)
	if err != nil {
		return err
	}
	defer r.Close()

	for _, entry := range r.File {
		// Only this one entry is extracted, to a path this function chose, so a
		// crafted archive cannot write anywhere else.
		if entry.FileInfo().IsDir() || path.Base(entry.Name) != binaryName() {
			continue
		}

		in, err := entry.Open()
		if err != nil {
			return err
		}
		defer in.Close()

		out, err := os.Create(destPath)
		if err != nil {
			return err
		}
		if _, err := io.Copy(out, in); err != nil {
			out.Close()
			return err
		}
		return out.Close()
	}

	return fmt.Errorf("stripe binary not found in archive")
}
