package tui

import (
	"context"
	"net/url"
	"os/exec"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/stripe/stripe-cli/internal/browser"
)

func stubBrowser(t *testing.T) *[]*exec.Cmd {
	t.Helper()
	var calls []*exec.Cmd
	original := browser.StartCommand
	browser.StartCommand = func(cmd *exec.Cmd) error {
		calls = append(calls, cmd)
		return nil
	}
	t.Cleanup(func() { browser.StartCommand = original })
	return &calls
}

func TestPage_Open_NilURL(t *testing.T) {
	stubBrowser(t)
	p := Page{}
	err := p.Open(context.Background())
	assert.NoError(t, err)
}

func TestPage_Open_OpensURL(t *testing.T) {
	calls := stubBrowser(t)
	p := Page{
		URL: &url.URL{Scheme: "https", Host: "docs.stripe.com", Path: "/payments"},
	}
	err := p.Open(context.Background())
	require.NoError(t, err)
	require.Len(t, *calls, 1)
	assert.Contains(t, (*calls)[0].Args, "https://docs.stripe.com/payments")
}

func TestPage_Open_PreservesQueryParams(t *testing.T) {
	calls := stubBrowser(t)
	p := Page{
		URL: &url.URL{
			Scheme:   "https",
			Host:     "docs.stripe.com",
			Path:     "/api/charges",
			RawQuery: "lang=go",
		},
	}
	err := p.Open(context.Background())
	require.NoError(t, err)
	require.Len(t, *calls, 1)
	assert.Contains(t, (*calls)[0].Args, "https://docs.stripe.com/api/charges?lang=go")
}
