package open

import (
	"context"
	"net/url"
	"os/exec"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var docsHosts = map[string]bool{"docs.stripe.com": true}

func stubBrowser(t *testing.T) *[]*exec.Cmd {
	t.Helper()
	var calls []*exec.Cmd
	original := StartCommand
	StartCommand = func(cmd *exec.Cmd) error {
		calls = append(calls, cmd)
		return nil
	}
	t.Cleanup(func() { StartCommand = original })
	return &calls
}

func TestOpenURL_NilURL(t *testing.T) {
	stubBrowser(t)
	err := OpenURL(context.Background(), nil, docsHosts)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "nil URL")
}

func TestOpenURL_DisallowedHost(t *testing.T) {
	stubBrowser(t)
	u := &url.URL{Scheme: "https", Host: "evil.com", Path: "/foo"}
	err := OpenURL(context.Background(), u, docsHosts)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not allowed")
}

func TestOpenURL_HTTPSchemeRejected(t *testing.T) {
	stubBrowser(t)
	u := &url.URL{Scheme: "http", Host: "docs.stripe.com", Path: "/payments"}
	err := OpenURL(context.Background(), u, docsHosts)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not allowed")
}

func TestOpenURL_AllowedURL(t *testing.T) {
	calls := stubBrowser(t)
	u := &url.URL{Scheme: "https", Host: "docs.stripe.com", Path: "/payments"}
	err := OpenURL(context.Background(), u, docsHosts)
	require.NoError(t, err)
	require.Len(t, *calls, 1)
	assert.Contains(t, (*calls)[0].Args, "https://docs.stripe.com/payments")
}

func TestOpenURL_PreservesQueryParams(t *testing.T) {
	calls := stubBrowser(t)
	u := &url.URL{
		Scheme:   "https",
		Host:     "docs.stripe.com",
		Path:     "/api/charges",
		RawQuery: "lang=go&api_version=2024-06-30",
	}
	err := OpenURL(context.Background(), u, docsHosts)
	require.NoError(t, err)
	require.Len(t, *calls, 1)
	assert.Contains(t, (*calls)[0].Args, "https://docs.stripe.com/api/charges?lang=go&api_version=2024-06-30")
}

func TestOpenURL_NilAllowedHosts(t *testing.T) {
	calls := stubBrowser(t)
	u := &url.URL{Scheme: "https", Host: "example.com", Path: "/foo"}
	err := OpenURL(context.Background(), u, nil)
	require.NoError(t, err)
	require.Len(t, *calls, 1)
}

func TestOpenCommand_ReturnsPlatformCommand(t *testing.T) {
	cmd, err := openCommand()
	require.NoError(t, err)
	assert.NotEmpty(t, cmd)
	assert.NotEmpty(t, cmd[0])
}
