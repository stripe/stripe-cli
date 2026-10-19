// Package releasenotes tidies the release notes GitHub generates for a release.
package releasenotes

import (
	"regexp"
	"strings"
)

var (
	// prSuffix matches the " by @author in <pull request>" GitHub appends to every
	// entry. GitHub uses the full url for pull requests in this repository and the
	// short "#1974" form for pull requests from forks.
	prSuffix = regexp.MustCompile(`\s+by\s+@\S+\s+in\s+(?:https://github\.com/[^\s/]+/[^\s/]+/pull/|#)(\d+)\s*$`)
	// categoryHeading matches a per-category heading such as "### Bug fixes".
	categoryHeading = regexp.MustCompile(`^#{3,6}\s`)
	// newContributorsHeading matches the heading of the section listing first-time
	// contributors.
	newContributorsHeading = regexp.MustCompile(`^##\s+New Contributors\s*$`)
	// fullChangelog matches the compare link GitHub appends at the end.
	fullChangelog = regexp.MustCompile(`^\*\*Full Changelog\*\*:`)

	heading = regexp.MustCompile(`^#{1,6}\s`)
	bullet  = regexp.MustCompile(`^\s*[*-]\s`)
)

// Tidy strips the boilerplate out of the release notes GitHub generates from
// .github/release.yml: the per-category headings, the " by @author in <pull
// request>" suffix on every entry, the "New Contributors" section, and the
// "Full Changelog" compare link. Every entry keeps a short "(#1974)" reference
// to its pull request.
//
// The categories in .github/release.yml still decide which pull requests appear
// in the notes and in what order.
func Tidy(body string) string {
	var kept []string
	inNewContributors := false

	for _, line := range strings.Split(body, "\n") {
		line = strings.TrimRight(line, " \t\r")

		if newContributorsHeading.MatchString(line) {
			inNewContributors = true
			continue
		}
		if inNewContributors {
			// The section is a run of bullets, so anything else has ended it.
			if line == "" || bullet.MatchString(line) {
				continue
			}
			inNewContributors = false
		}
		if categoryHeading.MatchString(line) || fullChangelog.MatchString(line) {
			continue
		}

		kept = append(kept, prSuffix.ReplaceAllString(line, " (#$1)"))
	}

	return strings.Join(collapseBlankLines(kept), "\n")
}

// collapseBlankLines squeezes the blank lines the removals leave behind, keeps a
// single blank line around every remaining heading, and drops trailing blanks.
func collapseBlankLines(lines []string) []string {
	var out []string

	for _, line := range lines {
		if line == "" {
			if len(out) > 0 && out[len(out)-1] != "" {
				out = append(out, "")
			}
			continue
		}
		if heading.MatchString(line) && len(out) > 0 && out[len(out)-1] != "" {
			out = append(out, "")
		}
		out = append(out, line)
		if heading.MatchString(line) {
			out = append(out, "")
		}
	}

	for len(out) > 0 && out[len(out)-1] == "" {
		out = out[:len(out)-1]
	}

	return out
}
