package autoupdate

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
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
	semver "github.com/hashicorp/go-version"
	log "github.com/sirupsen/logrus"

	"github.com/stripe/stripe-cli/pkg/version"
)

const httpTimeout = 10 * time.Second

const checkInterval = 24 * time.Hour

// UpdateMarker represents a staged update ready to be applied.
type UpdateMarker struct {
	Version      string `json:"version"`
	DownloadURL  string `json:"download_url"`
	Checksum     string `json:"checksum"`
	ReleaseNotes string `json:"release_notes"`

	// StagedAt is when the check wrote this marker, as a Unix timestamp. The
	// gap between it and the moment the update is applied is how long a user
	// sat on a version we already knew was old, which is the number that says
	// whether auto-update is actually keeping the fleet current.
	//
	// Omitted when empty so that a marker written by hand -- as the
	// auto-upgrade canary does -- stays valid, and reported only when set.
	StagedAt int64 `json:"staged_at,omitempty"`
}

// CheckForUpdate checks for a newer CLI version and writes a marker file
// if an update is available. Skips major version changes. This is called
// synchronously after command execution, rate-limited to once per day.
func CheckForUpdate() {
	defer recoverAndReport("check")

	if !shouldCheck() {
		return
	}

	latest, url, checksum, releaseNotes, checkErr := fetchLatestRelease()
	if latest == "" {
		// Reported rather than only debug-logged: the GitHub API this calls is
		// unauthenticated and rate-limited per source IP, so a NAT'd or CI
		// population can stop checking entirely with nothing else to show it.
		// Not rate-limited, because a failed check deliberately writes no
		// timestamp so it can retry immediately -- so this fires once per
		// invocation while the failure lasts, which is the volume worth seeing.
		sendEvent(eventCheckFailed, map[string]string{"reason": checkErr})
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
		sendEvent(eventSkipped, map[string]string{
			"reason": reasonMajorVersion,
			"from":   current,
			"to":     latestClean,
		})
		return
	}

	WriteMarker(UpdateMarker{
		Version:      latestClean,
		DownloadURL:  url,
		Checksum:     checksum,
		ReleaseNotes: releaseNotes,
	})
	sendEvent(eventAvailable, map[string]string{"from": current, "to": latestClean})
}

// recoverAndReport swallows a panic out of the update path and reports that it
// happened.
//
// Swallowing is the point: neither half of auto-update is work the user asked
// for, so neither may take down the command they did ask for. That makes a panic
// here invisible by construction, which is why it is also reported -- a crash
// loop in the updater would otherwise show up as nothing at all.
func recoverAndReport(phase string) {
	r := recover()
	if r == nil {
		return
	}

	log.Debugf("autoupdate %s panicked: %v", phase, r)
	sendEvent(eventPanicked, map[string]string{
		"phase": phase,
		"panic": fmt.Sprintf("%v", r),
	})
}

func isMajorVersionChange(current, latest string) bool {
	cur, err := semver.NewVersion(current)
	if err != nil {
		return false
	}
	lat, err := semver.NewVersion(latest)
	if err != nil {
		return false
	}
	return cur.Segments()[0] != lat.Segments()[0]
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

// fetchLatestRelease resolves the release to update to. On failure every value
// is empty except failureReason, which names which step gave up so the caller
// can report it.
func fetchLatestRelease() (ver string, downloadURL string, checksum string, releaseNotes string, failureReason string) {
	ctx, cancel := context.WithTimeout(context.Background(), httpTimeout)
	defer cancel()

	client := github.NewClient(nil)
	release, _, err := client.Repositories.GetLatestRelease(ctx, "stripe", "stripe-cli")
	if err != nil {
		log.Debug("autoupdate: failed to fetch latest release: ", err)
		return "", "", "", "", reasonReleaseFetch
	}

	ver = release.GetTagName()
	releaseNotes = release.GetBody()
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
		return "", "", "", "", reasonAssetMissing
	}

	if checksumURL != "" {
		checksum = fetchChecksumForAsset(checksumURL, assetName)
	}

	// No checksum, no update. Both ways of ending up without one -- the release
	// publishing no checksums file, or the fetch for it failing -- used to leave
	// checksum empty and stage the update anyway, and VerifyChecksum treats an
	// empty expectation as a pass. A single failed HTTP request was therefore
	// enough to turn integrity checking off for that update, silently. Refusing
	// to stage costs a day: the next check tries again.
	if checksum == "" {
		log.Debug("autoupdate: no checksum published for ", assetName, "; refusing to stage an unverifiable update")
		return "", "", "", "", reasonChecksumMissing
	}

	return ver, binaryURL, checksum, releaseNotes, ""
}

func fetchChecksumForAsset(checksumURL, assetName string) string {
	ctx, cancel := context.WithTimeout(context.Background(), httpTimeout)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, checksumURL, nil)
	if err != nil {
		log.Debug("autoupdate: failed to create checksum request: ", err)
		return ""
	}

	resp, err := http.DefaultClient.Do(req)
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
	return binaryAssetNameFor(ver, runtime.GOOS, runtime.GOARCH)
}

// binaryAssetNameFor is the release archive published for a platform.
//
// The names come from the archive templates in .goreleaser/, which do not use
// the Go names for either half: darwin is published as "mac-os" and amd64 as
// "x86_64". Passing runtime.GOOS straight through asks for an asset that does
// not exist, and a missing asset stops the update silently.
func binaryAssetNameFor(ver, goos, goarch string) string {
	osLabel := goos
	ext := "tar.gz"

	switch goos {
	case "darwin":
		osLabel = "mac-os"
	case "windows":
		ext = "zip"
	}

	return fmt.Sprintf("stripe_%s_%s_%s.%s", ver, osLabel, archAssetLabel(goos, goarch), ext)
}

func archAssetLabel(goos, goarch string) string {
	switch goarch {
	case "amd64":
		return "x86_64"
	case "386":
		return "i386"
	case "arm64":
		// .goreleaser/windows.yml builds amd64 and 386 only. Windows on ARM runs
		// the x64 binary under emulation, so that is the archive to fetch.
		if goos == "windows" {
			return "x86_64"
		}

		return "arm64"
	default:
		return goarch
	}
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

	// Stamped here rather than by the caller, so that "when was this staged" is
	// true of any marker this function writes. A caller that already set it --
	// a test, or a rewrite of an existing marker -- keeps its value.
	if m.StagedAt == 0 {
		m.StagedAt = time.Now().Unix()
	}

	content, err := json.Marshal(m)
	if err != nil {
		return
	}

	markerPath := filepath.Join(stateDir, "update-available")
	_ = os.WriteFile(markerPath, content, 0644)

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

	var m UpdateMarker
	if err := json.Unmarshal(data, &m); err != nil {
		return nil
	}
	return &m
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
