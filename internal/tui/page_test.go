package tui

import (
	"context"
	"fmt"
	"net/url"
	"os/exec"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/stripe/stripe-cli/internal/markdown"
	"github.com/stripe/stripe-cli/pkg/open"
)

func stubBrowser(t *testing.T) *[]*exec.Cmd {
	t.Helper()
	var calls []*exec.Cmd
	original := open.StartCommand
	open.StartCommand = func(cmd *exec.Cmd) error {
		calls = append(calls, cmd)
		return nil
	}
	t.Cleanup(func() { open.StartCommand = original })
	return &calls
}

// docWithReferences builds a minimal markdown.Document containing a link to each of the given URLs.
func docWithReferences(urls ...*url.URL) *markdown.Document {
	src := "# Title\n\n"
	for i, u := range urls {
		src += fmt.Sprintf("[Link %d](%s)\n\n", i, u.String())
	}
	doc, err := markdown.Parse([]byte(src))
	if err != nil {
		panic(err)
	}
	return doc
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
