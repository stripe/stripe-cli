// Command tidy-release-notes reads the release notes GitHub generated for a
// release on stdin and writes the tidied notes on stdout.
//
// The release workflow pipes the body from GitHub's "generate release notes" api
// into it and hands the result to GoReleaser with --release-notes, so a release
// is published with tidied notes rather than having them corrected afterwards.
package main

import (
	"fmt"
	"io"
	"os"

	"github.com/stripe/stripe-cli/internal/releasenotes"
)

func main() {
	body, err := io.ReadAll(os.Stdin)
	if err != nil {
		fmt.Fprintf(os.Stderr, "tidy-release-notes: cannot read stdin: %v\n", err)
		os.Exit(1)
	}
	if len(body) == 0 {
		fmt.Fprintln(os.Stderr, "tidy-release-notes: stdin is empty, expected generated release notes")
		os.Exit(1)
	}

	fmt.Println(releasenotes.Tidy(string(body)))
}
