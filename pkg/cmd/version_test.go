package cmd

import (
	"bytes"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/stripe/stripe-cli/pkg/version"
)

// withVersion sets version.Version for the duration of the test and restores
// the original value afterward.
func withVersion(t *testing.T, v string) {
	t.Helper()
	original := version.Version
	version.Version = v
	t.Cleanup(func() { version.Version = original })
}

// withReleaseNotesFn stubs version.GetReleaseNotesFn for the duration of the
// test and restores the original afterward.
func withReleaseNotesFn(t *testing.T, fn func(string) (string, error)) {
	t.Helper()
	original := version.GetReleaseNotesFn
	version.GetReleaseNotesFn = fn
	t.Cleanup(func() { version.GetReleaseNotesFn = original })
}

func runVersionNotes(t *testing.T) (stdout, stderr string) {
	t.Helper()
	vc := newVersionCmd()
	vc.notes = true

	outBuf, errBuf := new(bytes.Buffer), new(bytes.Buffer)
	vc.cmd.SetOut(outBuf)
	vc.cmd.SetErr(errBuf)

	vc.cmd.Run(vc.cmd, []string{})
	return outBuf.String(), errBuf.String()
}

func TestVersionNotes_DevBuild(t *testing.T) {
	withVersion(t, "master")

	out, stderr := runVersionNotes(t)
	require.Contains(t, out, "Release notes aren't available for development builds.")
	require.Empty(t, stderr)
}

func TestVersionNotes_Success(t *testing.T) {
	withVersion(t, "1.24.0")
	withReleaseNotesFn(t, func(ver string) (string, error) {
		require.Equal(t, "1.24.0", ver)
		return "## Changelog\n* did a thing", nil
	})

	out, stderr := runVersionNotes(t)
	require.Contains(t, out, "## Changelog\n* did a thing")
	require.Empty(t, stderr)
}

func TestVersionNotes_EmptyNotes(t *testing.T) {
	withVersion(t, "1.24.0")
	withReleaseNotesFn(t, func(ver string) (string, error) {
		return "", nil
	})

	out, stderr := runVersionNotes(t)
	require.Contains(t, out, "No release notes found for this version.")
	require.Empty(t, stderr)
}

func TestVersionNotes_FetchError(t *testing.T) {
	withVersion(t, "1.24.0")
	withReleaseNotesFn(t, func(ver string) (string, error) {
		return "", errors.New("boom")
	})

	out, stderr := runVersionNotes(t)
	require.Contains(t, stderr, "Could not fetch release notes: boom")
	require.NotContains(t, out, "boom")
}
