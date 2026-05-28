package tui

import (
	"fmt"
	"net/url"

	"github.com/atotto/clipboard"
)

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
