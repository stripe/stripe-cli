package version

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"

	"github.com/google/go-github/v72/github"
	"github.com/stretchr/testify/require"

	"github.com/stripe/stripe-cli/pkg/ansi"
	"github.com/stripe/stripe-cli/pkg/installmethod"
)

func TestUpgradeNotice(t *testing.T) {
	// ansi styling is on by default even when stdout is not a terminal, so turn it
	// off to assert on the text itself.
	ansi.DisableColors = true
	t.Cleanup(func() { ansi.DisableColors = false })

	t.Run("names the command for a known install method", func(t *testing.T) {
		notice := upgradeNotice("v1.2.3", installmethod.UpgradeAdvice(installmethod.Homebrew, "darwin"))

		require.Contains(t, notice, "A newer version of the Stripe CLI is available, please update to: v1.2.3")
		require.Contains(t, notice, "Run brew upgrade stripe to upgrade.")
	})

	t.Run("reports the version alone when no command can be named", func(t *testing.T) {
		notice := upgradeNotice("v1.2.3", installmethod.UpgradeAdvice(installmethod.Unknown, "linux"))

		require.Contains(t, notice, "please update to: v1.2.3")
		require.NotContains(t, notice, "Run")
	})

	t.Run("prints nothing for a self-updating install method", func(t *testing.T) {
		require.Empty(t, upgradeNotice("v1.2.3", installmethod.UpgradeAdvice(installmethod.NPX, "darwin")))
	})
}

func TestNeedsToUpgrade(t *testing.T) {
	require.False(t, needsToUpgrade("4.2.4.2", "v4.2.4.2"))
	require.False(t, needsToUpgrade("4.2.4.2", "4.2.4.2"))
	require.True(t, needsToUpgrade("4.2.4.2", "4.2.4.3"))
	require.True(t, needsToUpgrade("4.2.4.2", "v4.2.4.3"))
	require.True(t, needsToUpgrade("v4.2.4.2", "v4.2.4.3"))
}

func TestReleaseIsSettled(t *testing.T) {
	now := time.Date(2026, 9, 28, 12, 0, 0, 0, time.UTC)

	require.False(t, releaseIsSettled(now.Add(-time.Hour), now))
	require.False(t, releaseIsSettled(now.Add(-releaseGracePeriod+time.Minute), now))
	require.True(t, releaseIsSettled(now.Add(-releaseGracePeriod), now))
	require.True(t, releaseIsSettled(now.Add(-72*time.Hour), now))
	require.True(t, releaseIsSettled(time.Time{}, now))
}

func TestReleaseTag(t *testing.T) {
	require.Equal(t, "v1.24.0", releaseTag("1.24.0"))
	require.Equal(t, "v1.24.0", releaseTag("v1.24.0"))
}

// withFakeGithub points the package's GitHub client at a test server for the
// duration of the test, and restores the real client afterward.
func withFakeGithub(t *testing.T, handler http.HandlerFunc) {
	t.Helper()

	server := httptest.NewServer(handler)
	t.Cleanup(server.Close)

	baseURL, err := url.Parse(server.URL + "/")
	require.NoError(t, err)

	client := github.NewClient(server.Client())
	client.BaseURL = baseURL

	original := newGithubClient
	newGithubClient = func() *github.Client { return client }
	t.Cleanup(func() { newGithubClient = original })
}

func TestGetReleaseNotes_Success(t *testing.T) {
	withFakeGithub(t, func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, "/repos/stripe/stripe-cli/releases/tags/v1.24.0", r.URL.Path)
		fmt.Fprint(w, `{"body": "## Changelog\n* did a thing"}`)
	})

	notes, err := GetReleaseNotesFn("1.24.0")
	require.NoError(t, err)
	require.Equal(t, "## Changelog\n* did a thing", notes)
}

func TestGetReleaseNotes_NormalizesVersionWithoutLeadingV(t *testing.T) {
	var gotPath string
	withFakeGithub(t, func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		fmt.Fprint(w, `{"body": ""}`)
	})

	_, err := GetReleaseNotesFn("1.24.0")
	require.NoError(t, err)
	require.Equal(t, "/repos/stripe/stripe-cli/releases/tags/v1.24.0", gotPath)
}

func TestGetReleaseNotes_EmptyBody(t *testing.T) {
	withFakeGithub(t, func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `{"body": ""}`)
	})

	notes, err := GetReleaseNotesFn("1.24.0")
	require.NoError(t, err)
	require.Empty(t, notes)
}

func TestGetReleaseNotes_NotFound(t *testing.T) {
	withFakeGithub(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		fmt.Fprint(w, `{"message": "Not Found"}`)
	})

	_, err := GetReleaseNotesFn("999.0.0")
	require.Error(t, err)
}
