package autoupdate

import (
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/stripe/stripe-cli/pkg/version"
)

func withVersionPolicyState(t *testing.T) {
	t.Helper()
	originalState, originalVersion := GetStateDirFn, version.Version
	t.Cleanup(func() { GetStateDirFn = originalState; version.Version = originalVersion })
	dir := t.TempDir()
	GetStateDirFn = func() string { return dir }
	version.Version = "1.55.6"
	t.Setenv("STRIPE_INSTALL_METHOD", "script")
	t.Setenv("STRIPE_NO_AUTO_UPDATE", "")
	t.Setenv("STRIPE_CLI_TELEMETRY_OPTOUT", "1")
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
}

func TestCheckForUpdateOnlyStagesNewerVersions(t *testing.T) {
	for _, tc := range []struct {
		target string
		staged bool
	}{
		{"v1.55.10", true},
		{"v1.55.6", false},
		{"v1.51.1", false},
		{"v1.55.6-rc.1", false},
		{"v2.0.0", false},
		{"invalid", false},
	} {
		t.Run(tc.target, func(t *testing.T) {
			withVersionPolicyState(t)
			original := http.DefaultTransport
			t.Cleanup(func() { http.DefaultTransport = original })
			http.DefaultTransport = versionPolicyTransport(func(r *http.Request) (*http.Response, error) {
				var body string
				switch r.URL.Path {
				case "/repos/stripe/stripe-cli/releases/latest":
					body = fmt.Sprintf(`{"tag_name":%q,"assets":[{"name":%q,"browser_download_url":"https://example.test/archive"},{"name":%q,"browser_download_url":"https://example.test/checksums"}]}`, tc.target, binaryAssetName(strings.TrimPrefix(tc.target, "v")), checksumAssetName())
				case "/checksums":
					body = strings.Repeat("a", 64) + "  " + binaryAssetName(strings.TrimPrefix(tc.target, "v"))
				default:
					t.Fatalf("unexpected request: %s", r.URL)
				}
				return &http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(body)), Header: make(http.Header), Request: r}, nil
			})
			CheckForUpdate()
			if tc.staged {
				require.NotNil(t, ReadMarker())
				assert.Equal(t, strings.TrimPrefix(tc.target, "v"), ReadMarker().Version)
			} else {
				assert.Nil(t, ReadMarker())
			}
		})
	}
}

func TestApplyIfPendingRejectsUnsafeVersions(t *testing.T) {
	for _, target := range []string{"1.51.1", "1.55.6", "1.55.6-rc.1", "2.0.0", "invalid"} {
		t.Run(target, func(t *testing.T) {
			withVersionPolicyState(t)
			original := http.DefaultTransport
			t.Cleanup(func() { http.DefaultTransport = original })
			requests := 0
			http.DefaultTransport = versionPolicyTransport(func(r *http.Request) (*http.Response, error) {
				requests++
				return &http.Response{StatusCode: http.StatusInternalServerError, Body: io.NopCloser(strings.NewReader("")), Header: make(http.Header), Request: r}, nil
			})
			WriteMarker(UpdateMarker{Version: target, DownloadURL: "https://example.test/archive", Checksum: strings.Repeat("a", 64)})
			ApplyIfPending()
			assert.Zero(t, requests)
			assert.Nil(t, ReadMarker())
		})
	}
}

type versionPolicyTransport func(*http.Request) (*http.Response, error)

func (f versionPolicyTransport) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }
