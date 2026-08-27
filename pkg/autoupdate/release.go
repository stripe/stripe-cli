package autoupdate

import (
	"archive/tar"
	"archive/zip"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	goversion "github.com/hashicorp/go-version"

	"github.com/google/go-github/v72/github"

	"github.com/stripe/stripe-cli/pkg/errorcategory"
	"github.com/stripe/stripe-cli/pkg/version"
)

const (
	githubOwner = "stripe"
	githubRepo  = "stripe-cli"

	metadataTimeout = 10 * time.Second
	downloadTimeout = 3 * time.Minute

	// maxChecksumsSize and maxArchiveSize bound what a hostile or broken response
	// can make the CLI buffer or write.
	maxChecksumsSize = 1 << 20 // 1 MiB
	maxArchiveSize   = 1 << 28 // 256 MiB
)

// releaseBaseURL is where release archives are downloaded from. Tests point it at
// a local server.
var releaseBaseURL = fmt.Sprintf("https://github.com/%s/%s/releases/download", githubOwner, githubRepo)

var httpClient = &http.Client{Timeout: downloadTimeout}

// latestVersion returns the newest published release, without a leading "v".
func latestVersion(ctx context.Context) (string, error) {
	ctx, cancel := context.WithTimeout(ctx, metadataTimeout)
	defer cancel()

	client := github.NewClient(nil)

	// Unauthenticated GitHub API calls are capped at 60 per hour per IP, which
	// shared egress and CI runners exhaust; the install scripts take a token for
	// the same reason.
	if token := os.Getenv("GITHUB_TOKEN"); token != "" {
		client = client.WithAuthToken(token)
	}

	release, _, err := client.Repositories.GetLatestRelease(ctx, githubOwner, githubRepo)
	if err != nil {
		return "", err
	}

	if release.TagName == nil {
		return "", errorcategory.Errorf(errorcategory.API, "latest release has no tag name")
	}

	return strings.TrimPrefix(*release.TagName, "v"), nil
}

func isNewer(current, latest string) bool {
	from, err := goversion.NewVersion(strings.TrimPrefix(current, "v"))
	if err != nil {
		return false
	}

	to, err := goversion.NewVersion(strings.TrimPrefix(latest, "v"))
	if err != nil {
		return false
	}

	return to.GreaterThan(from)
}

// assetNames returns the archive and checksums file the release publishes for a
// platform. These have to match what the .goreleaser configs emit and what
// scripts/install.sh and scripts/install.ps1 download.
func assetNames(release, goos, goarch string) (archive, checksums string, err error) {
	var osLabel, extension string

	switch goos {
	case "darwin":
		osLabel, extension, checksums = "mac-os", "tar.gz", "stripe-mac-checksums.txt"
	case "linux":
		osLabel, extension, checksums = "linux", "tar.gz", "stripe-linux-checksums.txt"
	case "windows":
		osLabel, extension, checksums = "windows", "zip", "stripe-windows-checksums.txt"
	default:
		return "", "", errorcategory.Errorf(errorcategory.Internal, "unsupported operating system: %s", goos)
	}

	archLabel, err := archLabel(goos, goarch)
	if err != nil {
		return "", "", err
	}

	return fmt.Sprintf("stripe_%s_%s_%s.%s", release, osLabel, archLabel, extension), checksums, nil
}

func archLabel(goos, goarch string) (string, error) {
	switch {
	case goarch == "amd64":
		return "x86_64", nil
	case goarch == "arm64" && goos == "windows":
		// .goreleaser/windows.yml builds amd64 and 386 only. Windows on ARM runs
		// the x64 binary under emulation, which is also what install.ps1 fetches.
		return "x86_64", nil
	case goarch == "arm64":
		return "arm64", nil
	case goarch == "386" && goos == "windows":
		return "i386", nil
	default:
		return "", errorcategory.Errorf(errorcategory.Internal, "unsupported architecture: %s/%s", goos, goarch)
	}
}

// binaryName is the name of the executable inside the release archive, which is
// also the name it is installed under.
func binaryName() string {
	if runtime.GOOS == "windows" {
		return "stripe.exe"
	}

	return "stripe"
}

// download fetches the release archive into destDir, verifies its checksum, and
// returns the path of the extracted binary.
func download(ctx context.Context, release, destDir string) (string, error) {
	archive, checksums, err := assetNames(release, runtime.GOOS, runtime.GOARCH)
	if err != nil {
		return "", errorcategory.Errorf(errorcategory.Internal, "%v", err)
	}

	base := fmt.Sprintf("%s/v%s", releaseBaseURL, release)

	archivePath := filepath.Join(destDir, archive)
	if err := httpDownload(ctx, base+"/"+archive, archivePath); err != nil {
		return "", errorcategory.Errorf(errorcategory.Network, "could not download %s: %v", archive, err)
	}

	raw, err := httpGet(ctx, base+"/"+checksums)
	if err != nil {
		return "", errorcategory.Errorf(errorcategory.Network, "could not download %s: %v", checksums, err)
	}

	expected, err := lookupChecksum(string(raw), archive)
	if err != nil {
		return "", errorcategory.Errorf(errorcategory.API, "%v", err)
	}

	if err := verifySHA256(archivePath, expected); err != nil {
		return "", errorcategory.Errorf(errorcategory.API, "%v", err)
	}

	return extractBinary(archivePath, destDir)
}

// lookupChecksum finds the digest for exactly this asset name.
//
// The comparison is on the whole field, not a substring or a regex: asset names
// differ only in an architecture suffix, and "." is a regex metacharacter, so a
// loose match can silently verify a different archive than the one downloaded.
func lookupChecksum(contents, asset string) (string, error) {
	for _, line := range strings.Split(contents, "\n") {
		fields := strings.Fields(line)
		if len(fields) != 2 {
			continue
		}

		// Some checksum writers mark binary mode with a leading "*".
		if strings.TrimPrefix(fields[1], "*") == asset {
			return fields[0], nil
		}
	}

	return "", errorcategory.Errorf(errorcategory.API, "no checksum published for %s", asset)
}

func verifySHA256(path, expected string) error {
	file, err := os.Open(path)
	if err != nil {
		return err
	}

	defer file.Close()

	digest := sha256.New()
	if _, err := io.Copy(digest, file); err != nil {
		return err
	}

	actual := hex.EncodeToString(digest.Sum(nil))
	if !strings.EqualFold(actual, expected) {
		return errorcategory.Errorf(errorcategory.API, "checksum mismatch for %s: expected %s, got %s", filepath.Base(path), expected, actual)
	}

	return nil
}

// extractBinary pulls the stripe executable out of a release archive. Only that
// one entry is extracted, and it is written to a path this function chooses, so a
// crafted archive cannot place a file anywhere else.
func extractBinary(archivePath, destDir string) (string, error) {
	name := binaryName()
	dest := filepath.Join(destDir, name)

	var err error
	if strings.HasSuffix(archivePath, ".zip") {
		err = extractFromZip(archivePath, name, dest)
	} else {
		err = extractFromTarGz(archivePath, name, dest)
	}

	if err != nil {
		return "", errorcategory.Errorf(errorcategory.API, "could not extract %s from %s: %v", name, filepath.Base(archivePath), err)
	}

	return dest, nil
}

func extractFromZip(archivePath, name, dest string) error {
	reader, err := zip.OpenReader(archivePath)
	if err != nil {
		return err
	}

	defer reader.Close()

	for _, entry := range reader.File {
		if path.Base(entry.Name) != name {
			continue
		}

		contents, err := entry.Open()
		if err != nil {
			return err
		}

		defer contents.Close()

		return writeFile(dest, contents)
	}

	return errorcategory.Errorf(errorcategory.API, "archive does not contain %s", name)
}

func extractFromTarGz(archivePath, name, dest string) error {
	file, err := os.Open(archivePath)
	if err != nil {
		return err
	}

	defer file.Close()

	gzipReader, err := gzip.NewReader(file)
	if err != nil {
		return err
	}

	defer gzipReader.Close()

	tarReader := tar.NewReader(gzipReader)

	for {
		header, err := tarReader.Next()
		if err == io.EOF {
			return errorcategory.Errorf(errorcategory.API, "archive does not contain %s", name)
		}

		if err != nil {
			return err
		}

		if header.Typeflag != tar.TypeReg || path.Base(header.Name) != name {
			continue
		}

		return writeFile(dest, tarReader)
	}
}

func writeFile(dest string, contents io.Reader) error {
	file, err := os.OpenFile(dest, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0755)
	if err != nil {
		return err
	}

	// Read one byte past the cap so that hitting it is an error rather than a
	// silently truncated file, which for a binary means a broken install.
	written, err := io.Copy(file, io.LimitReader(contents, maxArchiveSize+1))
	if err == nil && written > maxArchiveSize {
		err = errorcategory.Errorf(errorcategory.API, "%s is larger than the %d byte limit", filepath.Base(dest), maxArchiveSize)
	}

	if err != nil {
		_ = file.Close()

		return err
	}

	return file.Close()
}

func httpGet(ctx context.Context, url string) ([]byte, error) {
	body, err := httpBody(ctx, url)
	if err != nil {
		return nil, err
	}

	defer body.Close()

	return io.ReadAll(io.LimitReader(body, maxChecksumsSize))
}

func httpDownload(ctx context.Context, url, dest string) error {
	body, err := httpBody(ctx, url)
	if err != nil {
		return err
	}

	defer body.Close()

	return writeFile(dest, body)
}

// httpBody performs the GET behind both helpers above.
//
// No Authorization header, even when GITHUB_TOKEN is set: release asset URLs
// redirect to a pre-signed storage URL that rejects a request carrying a second
// set of credentials. Only the API metadata call authenticates.
func httpBody(ctx context.Context, url string) (io.ReadCloser, error) {
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}

	request.Header.Set("User-Agent", "stripe-cli/"+version.Version)

	response, err := httpClient.Do(request)
	if err != nil {
		return nil, err
	}

	if response.StatusCode != http.StatusOK {
		_ = response.Body.Close()

		return nil, errorcategory.Errorf(errorcategory.API, "GET %s: %s", url, response.Status)
	}

	return response.Body, nil
}
