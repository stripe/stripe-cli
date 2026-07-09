package autoupdate

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/google/go-github/v72/github"
	log "github.com/sirupsen/logrus"

	"github.com/stripe/stripe-cli/pkg/version"
)

const checkInterval = 24 * time.Hour

// UpdateMarker represents a staged update ready to be applied.
type UpdateMarker struct {
	Version     string
	DownloadURL string
	Checksum    string
}

// CheckForUpdate checks for a newer CLI version and writes a marker file
// if an update is available. Skips major version changes. This is called
// synchronously after command execution, rate-limited to once per day.
func CheckForUpdate() {
	defer func() {
		if r := recover(); r != nil {
			log.Debugf("autoupdate check panicked: %v", r)
		}
	}()

	if !shouldCheck() {
		return
	}

	latest, url, checksum := fetchLatestRelease()
	if latest == "" {
		return
	}

	current := strings.TrimPrefix(version.Version, "v")
	latestClean := strings.TrimPrefix(latest, "v")

	if current == latestClean {
		recordLastCheck()
		return
	}

	if isMajorVersionChange(current, latestClean) {
		log.Debugf("autoupdate: skipping major version change %s → %s", current, latestClean)
		recordLastCheck()
		return
	}

	WriteMarker(UpdateMarker{
		Version:     latestClean,
		DownloadURL: url,
		Checksum:    checksum,
	})
}

func isMajorVersionChange(current, latest string) bool {
	currentMajor := majorVersion(current)
	latestMajor := majorVersion(latest)
	return currentMajor != "" && latestMajor != "" && currentMajor != latestMajor
}

func majorVersion(v string) string {
	parts := strings.SplitN(v, ".", 2)
	if len(parts) == 0 {
		return ""
	}
	return parts[0]
}

func shouldCheck() bool {
	if version.Version == "master" {
		return false
	}
	if IsOptedOut() {
		return false
	}
	if !IsCurlInstall() {
		return false
	}

	stateDir := GetStateDir()
	if stateDir == "" {
		return false
	}

	lastCheckFile := filepath.Join(stateDir, "last_update_check")
	data, err := os.ReadFile(lastCheckFile)
	if err != nil {
		return true
	}

	ts, err := strconv.ParseInt(strings.TrimSpace(string(data)), 10, 64)
	if err != nil {
		return true
	}

	return time.Since(time.Unix(ts, 0)) >= checkInterval
}

func fetchLatestRelease() (ver string, downloadURL string, checksum string) {
	client := github.NewClient(nil)
	release, _, err := client.Repositories.GetLatestRelease(context.Background(), "stripe", "stripe-cli")
	if err != nil {
		log.Debug("autoupdate: failed to fetch latest release: ", err)
		return "", "", ""
	}

	ver = release.GetTagName()
	assetName := binaryAssetName(strings.TrimPrefix(ver, "v"))
	checksumAsset := checksumAssetName()

	var binaryURL, checksumURL string
	for _, asset := range release.Assets {
		name := asset.GetName()
		if name == assetName {
			binaryURL = asset.GetBrowserDownloadURL()
		}
		if name == checksumAsset {
			checksumURL = asset.GetBrowserDownloadURL()
		}
	}

	if binaryURL == "" {
		log.Debug("autoupdate: binary asset not found: ", assetName)
		return "", "", ""
	}

	if checksumURL != "" {
		checksum = fetchChecksumForAsset(checksumURL, assetName)
	}

	return ver, binaryURL, checksum
}

func fetchChecksumForAsset(checksumURL, assetName string) string {
	resp, err := http.Get(checksumURL) //nolint:gosec
	if err != nil {
		log.Debug("autoupdate: failed to fetch checksums: ", err)
		return ""
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return ""
	}

	for _, line := range strings.Split(string(body), "\n") {
		parts := strings.Fields(line)
		if len(parts) == 2 && parts[1] == assetName {
			return parts[0]
		}
	}
	return ""
}

func binaryAssetName(ver string) string {
	goos := runtime.GOOS
	goarch := runtime.GOARCH

	var archName string
	switch goarch {
	case "amd64":
		archName = "x86_64"
	case "arm64":
		archName = "arm64"
	default:
		archName = goarch
	}

	ext := "tar.gz"
	if goos == "windows" {
		ext = "zip"
	}

	return fmt.Sprintf("stripe_%s_%s_%s.%s", ver, goos, archName, ext)
}

func checksumAssetName() string {
	switch runtime.GOOS {
	case "darwin":
		return "stripe-mac-checksums.txt"
	case "linux":
		return "stripe-linux-checksums.txt"
	case "windows":
		return "stripe-windows-checksums.txt"
	default:
		return ""
	}
}

// WriteMarker writes an update marker to the state directory.
func WriteMarker(m UpdateMarker) {
	stateDir := GetStateDir()
	if stateDir == "" {
		return
	}

	if err := os.MkdirAll(stateDir, 0755); err != nil {
		return
	}

	content := fmt.Sprintf("%s\n%s\n%s\n", m.Version, m.DownloadURL, m.Checksum)
	markerPath := filepath.Join(stateDir, "update-available")
	_ = os.WriteFile(markerPath, []byte(content), 0644)

	recordLastCheck()
}

func recordLastCheck() {
	stateDir := GetStateDir()
	if stateDir == "" {
		return
	}
	if err := os.MkdirAll(stateDir, 0755); err != nil {
		return
	}
	now := strconv.FormatInt(time.Now().Unix(), 10)
	_ = os.WriteFile(filepath.Join(stateDir, "last_update_check"), []byte(now), 0644)
}

// ReadMarker reads a pending update marker, or returns nil if none exists.
func ReadMarker() *UpdateMarker {
	stateDir := GetStateDir()
	if stateDir == "" {
		return nil
	}

	data, err := os.ReadFile(filepath.Join(stateDir, "update-available"))
	if err != nil {
		return nil
	}

	lines := strings.Split(strings.TrimSpace(string(data)), "\n")
	if len(lines) < 2 {
		return nil
	}

	m := &UpdateMarker{
		Version:     lines[0],
		DownloadURL: lines[1],
	}
	if len(lines) >= 3 {
		m.Checksum = lines[2]
	}
	return m
}

// ClearMarker removes the pending update marker.
func ClearMarker() {
	stateDir := GetStateDir()
	if stateDir == "" {
		return
	}
	_ = os.Remove(filepath.Join(stateDir, "update-available"))
}

// VerifyChecksum verifies the SHA256 checksum of a file.
func VerifyChecksum(filePath, expected string) bool {
	if expected == "" {
		return true
	}

	f, err := os.Open(filePath)
	if err != nil {
		return false
	}
	defer f.Close()

	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return false
	}

	actual := hex.EncodeToString(h.Sum(nil))
	return strings.EqualFold(actual, expected)
}
