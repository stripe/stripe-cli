package tui

import "net/url"

// Page holds the raw content and metadata needed by the TUI to display a
// documentation page. Callers construct a Page from a fetched docs response
// and pass it via WithPage.
type Page struct {
	Content []byte
	URL     *url.URL
}
