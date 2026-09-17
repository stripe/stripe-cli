// Package version manages CLI version checking and display.
package version

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/google/go-github/v72/github"
	log "github.com/sirupsen/logrus"

	"github.com/stripe/stripe-cli/pkg/ansi"
)

// Version of the CLI.
// This is set to the actual version by GoReleaser, identify by the
// git tag assigned to the release. Versions built from source will
// always show master.
var Version = "master"

// Template for the version string.
var Template = fmt.Sprintf("stripe version %s\n", Version)

// CheckLatestVersion makes a request to the GitHub API to pull the latest
// release of the CLI
func CheckLatestVersion() {
	// master is the dev version, we don't want to check against that every time
	if Version != "master" {
		s := ansi.StartNewSpinner("Checking for new versions...", os.Stdout)
		latest := getLatestVersion()

		ansi.StopSpinner(s, "", os.Stdout)

		if needsToUpgrade(Version, latest) {
			fmt.Println(ansi.Italic("A newer version of the Stripe CLI is available, please update to:"), ansi.Italic(latest))
		}
	}
}

func needsToUpgrade(version, latest string) bool {
	return latest != "" && (strings.TrimPrefix(latest, "v") != strings.TrimPrefix(version, "v"))
}

// newGithubClient is overridden in tests to point at a fake server instead of
// the real GitHub API.
var newGithubClient = func() *github.Client {
	return github.NewClient(nil)
}

// GetReleaseNotesFn fetches the GitHub release notes for the given version
// (with or without a leading "v"). Exposed as a var so callers outside this
// package can stub it out in tests.
var GetReleaseNotesFn = func(ver string) (string, error) {
	release, _, err := newGithubClient().Repositories.GetReleaseByTag(context.Background(), "stripe", "stripe-cli", releaseTag(ver))
	if err != nil {
		return "", err
	}

	return release.GetBody(), nil
}

// releaseTag normalizes a version string into the "vX.Y.Z" tag GitHub releases
// are published under.
func releaseTag(ver string) string {
	if strings.HasPrefix(ver, "v") {
		return ver
	}

	return "v" + ver
}

func getLatestVersion() string {
	rep, _, err := newGithubClient().Repositories.GetLatestRelease(context.Background(), "stripe", "stripe-cli")

	l := log.StandardLogger()

	if err != nil {
		// We don't want to fail any functionality or display errors for this
		// so fail silently and output to debug log
		l.Debug(err)
		return ""
	}

	return *rep.TagName
}
