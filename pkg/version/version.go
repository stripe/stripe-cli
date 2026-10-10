// Package version manages CLI version checking and display.
package version

import (
	"context"
	"fmt"
	"os"
	"runtime"
	"strings"
	"time"

	"github.com/google/go-github/v72/github"
	goversion "github.com/hashicorp/go-version"
	log "github.com/sirupsen/logrus"

	"github.com/stripe/stripe-cli/pkg/ansi"
	"github.com/stripe/stripe-cli/pkg/installmethod"
)

// Version of the CLI.
// This is set to the actual version by GoReleaser, identify by the
// git tag assigned to the release. Versions built from source will
// show master or the commit SHA.
var Version = "master"

// Template for the version string.
var Template = fmt.Sprintf("stripe version %s\n", Version)

// releaseGracePeriod is how long after a release is published on GitHub before
// the upgrade notice mentions it. Package managers (Homebrew, npm, Scoop, ...)
// pick up a release hours after GitHub does, so announcing it right away tells
// users to run an upgrade command that cannot find the new version yet.
const releaseGracePeriod = 24 * time.Hour

// CheckLatestVersion makes a request to the GitHub API to pull the latest
// release of the CLI
func CheckLatestVersion() {
	if _, err := goversion.NewSemver(Version); err == nil {
		s := ansi.StartNewSpinner("Checking for new versions...", os.Stdout)
		latest, publishedAt := getLatestVersion()

		ansi.StopSpinner(s, "", os.Stdout)

		if needsToUpgrade(Version, latest) && releaseIsSettled(publishedAt, time.Now()) {
			method := installmethod.Detect(installmethod.OSEnv())
			if notice := upgradeNotice(latest, installmethod.UpgradeAdvice(method, runtime.GOOS)); notice != "" {
				fmt.Println(notice)
			}
		}
	}
}

// upgradeNotice builds the out-of-date message, naming the command that upgrades
// the CLI when the install method is one we can name a command for. Returns ""
// when the notice should not be printed at all.
func upgradeNotice(latest string, advice installmethod.Advice) string {
	if advice.Suppress {
		return ""
	}

	notice := fmt.Sprintf("%s %s",
		ansi.Italic("A newer version of the Stripe CLI is available, please update to:"),
		ansi.Italic(latest),
	)

	if advice.Command != "" {
		notice += fmt.Sprintf("\n%s %s %s", ansi.Italic("Run"), ansi.Bold(advice.Command), ansi.Italic("to upgrade."))
	}

	return notice
}

func needsToUpgrade(version, latest string) bool {
	return latest != "" && (strings.TrimPrefix(latest, "v") != strings.TrimPrefix(version, "v"))
}

// releaseIsSettled reports whether a release published at publishedAt has been
// out long enough for package managers to carry it. A missing publish time is
// treated as settled, so the notice degrades to its old behavior rather than
// going silent.
func releaseIsSettled(publishedAt, now time.Time) bool {
	return publishedAt.IsZero() || now.Sub(publishedAt) >= releaseGracePeriod
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

func getLatestVersion() (string, time.Time) {
	rep, _, err := newGithubClient().Repositories.GetLatestRelease(context.Background(), "stripe", "stripe-cli")

	l := log.StandardLogger()

	if err != nil {
		// We don't want to fail any functionality or display errors for this
		// so fail silently and output to debug log
		l.Debug(err)
		return "", time.Time{}
	}

	return rep.GetTagName(), rep.GetPublishedAt().Time
}
