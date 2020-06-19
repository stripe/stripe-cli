package version

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/google/go-github/v28/github"
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

func getLatestVersion() string {
	client := github.NewClient(nil)
	rep, _, err := client.Repositories.GetLatestRelease(context.Background(), "stripe", "stripe-cli")

	l := log.StandardLogger()

	if err != nil {
		// We don't want to fail any functionality or display errors for this
		// so fail silently and output to debug log
		l.Debug(err)
		return ""
	}

	return *rep.TagName
}
