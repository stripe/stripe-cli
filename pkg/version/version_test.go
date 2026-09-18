package version

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/google/go-github/v72/github"
	"github.com/stretchr/testify/require"
)

func TestNeedsToUpgrade(t *testing.T) {
	require.False(t, needsToUpgrade("4.2.4.2", "v4.2.4.2"))
	require.False(t, needsToUpgrade("4.2.4.2", "4.2.4.2"))
	require.True(t, needsToUpgrade("4.2.4.2", "4.2.4.3"))
	require.True(t, needsToUpgrade("4.2.4.2", "v4.2.4.3"))
	require.True(t, needsToUpgrade("v4.2.4.2", "v4.2.4.3"))
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
