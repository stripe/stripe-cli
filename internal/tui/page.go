package tui

import (
	"context"
	"fmt"
	"net/url"

	"github.com/atotto/clipboard"

	"github.com/stripe/stripe-cli/pkg/open"
)

var docsAllowedHosts = map[string]bool{"docs.stripe.com": true}

// Page holds the raw content and metadata needed by the TUI to display a
// documentation page. Callers construct a Page from a fetched docs response
// and pass it via WithPage.
type Page struct {
	Content []byte
	URL     *url.URL
}

// Copy writes the raw markdown content to the system clipboard.
func (p Page) Copy() error {
	if err := clipboard.WriteAll(string(p.Content)); err != nil {
		return fmt.Errorf("copying to clipboard: %w", err)
	}
	return nil
}

// Open opens the page URL in the user's default browser.
func (p Page) Open(ctx context.Context) error {
	if p.URL == nil {
		return nil
	}
	if err := open.OpenURL(ctx, p.URL, docsAllowedHosts); err != nil {
		return fmt.Errorf("opening browser: %w", err)
	}
	return nil
}
