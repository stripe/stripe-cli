package markdown

import (
	"bytes"
	"net/url"
	"strconv"
	"strings"
	"unicode"

	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/text"
)

// ParseOption configures Parse behavior.
type ParseOption func(*parseConfig)

type parseConfig struct {
	relativeOrigins []string
}

// WithRelativeURLs strips the given origin (e.g. "https://docs.stripe.com")
// from absolute URLs in the document source before parsing, converting them to
// root-relative paths. It may be passed multiple times to strip multiple origins.
func WithRelativeURLs(origin string) ParseOption {
	return func(c *parseConfig) { c.relativeOrigins = append(c.relativeOrigins, origin) }
}

// Reference is a hyperlink extracted from a markdown document.
type Reference struct {
	Title    string
	URL      *url.URL
	External bool // true when the URL points outside docs.stripe.com
}

// Document holds a parsed markdown document, including the goldmark AST root and
// the original source bytes. Callers can walk Node to extract headings, links,
// code blocks, and other elements.
type Document struct {
	Node   ast.Node
	Source []byte
}

// Heading describes a document heading and its URL fragment.
type Heading struct {
	Level    int
	Text     string
	Fragment string
}

// Headings returns the document headings in source order. Repeated fragments
// receive the same numeric suffixes used by documentation URLs.
func (d *Document) Headings() []Heading {
	var headings []Heading
	used := make(map[string]bool)
	_ = ast.Walk(d.Node, func(n ast.Node, entering bool) (ast.WalkStatus, error) {
		if !entering {
			return ast.WalkContinue, nil
		}
		h, ok := n.(*ast.Heading)
		if !ok {
			return ast.WalkContinue, nil
		}
		text := string(nodeText(h, d.Source))
		baseFragment := NormalizeFragment(text)
		fragment := baseFragment
		for suffix := 1; used[fragment]; suffix++ {
			fragment = baseFragment + "-" + strconv.Itoa(suffix)
		}
		used[fragment] = true
		headings = append(headings, Heading{Level: h.Level, Text: text, Fragment: fragment})
		return ast.WalkContinue, nil
	})
	return headings
}

// HeadingForFragment returns the heading matching fragment.
func (d *Document) HeadingForFragment(fragment string) (Heading, bool) {
	fragment = strings.TrimPrefix(fragment, "#")
	for _, heading := range d.Headings() {
		if heading.Fragment == fragment {
			return heading, true
		}
	}
	return Heading{}, false
}

// NormalizeFragment converts heading text to the fragment format used by docs URLs.
func NormalizeFragment(s string) string {
	var b strings.Builder
	separator := false
	for _, r := range strings.ToLower(strings.TrimSpace(s)) {
		switch {
		case unicode.IsLetter(r), unicode.IsNumber(r), r == '_':
			if separator && b.Len() > 0 {
				b.WriteByte('-')
			}
			separator = false
			b.WriteRune(r)
		case unicode.IsSpace(r), r == '-':
			separator = true
		}
	}
	return strings.TrimSuffix(b.String(), "-")
}

// Title returns the text of the first h1 heading in the document, or an empty
// string if none is found.
func (d *Document) Title() string {
	var title string
	_ = ast.Walk(d.Node, func(n ast.Node, entering bool) (ast.WalkStatus, error) {
		if !entering {
			return ast.WalkContinue, nil
		}
		if h, ok := n.(*ast.Heading); ok && h.Level == 1 {
			title = string(nodeText(h, d.Source))
			return ast.WalkStop, nil
		}
		return ast.WalkContinue, nil
	})
	return title
}

// References returns all hyperlinks found in the document, in document order.
// currentURL is the URL of the page being viewed; links that resolve to the
// same page (with or without a fragment) are excluded so that on-page anchors
// do not clutter the reference palette.
func (d *Document) References(currentURL *url.URL) []Reference {
	var refs []Reference
	_ = ast.Walk(d.Node, func(n ast.Node, entering bool) (ast.WalkStatus, error) {
		if !entering {
			return ast.WalkContinue, nil
		}
		if link, ok := n.(*ast.Link); ok {
			if u, err := url.Parse(string(link.Destination)); err == nil {
				if isSamePageAnchor(u, currentURL) {
					return ast.WalkContinue, nil
				}
				refs = append(refs, Reference{
					Title:    string(nodeText(link, d.Source)),
					URL:      u,
					External: u.Host != "" && u.Host != "docs.stripe.com",
				})
			}
		}
		return ast.WalkContinue, nil
	})
	return refs
}

// isSamePageAnchor reports whether u is an anchor link that resolves to the
// same page as currentURL. Pure fragment-only links (#section) always match.
func isSamePageAnchor(u, currentURL *url.URL) bool {
	if u.Host == "" && u.Path == "" {
		return true // pure #fragment
	}
	if u.Fragment == "" || currentURL == nil {
		return false
	}
	if u.Host != "" && u.Host != currentURL.Host {
		return false // different site
	}
	return u.Path == currentURL.Path && u.Query().Encode() == currentURL.Query().Encode()
}

func nodeText(n ast.Node, source []byte) []byte {
	var buf []byte
	for c := n.FirstChild(); c != nil; c = c.NextSibling() {
		if t, ok := c.(*ast.Text); ok {
			buf = append(buf, t.Value(source)...)
		} else {
			buf = append(buf, nodeText(c, source)...)
		}
	}
	return buf
}

// Parse parses src as markdown and returns a Document containing the goldmark AST
// root and the original source bytes. The source is required for extracting text
// from AST nodes via Node.Text(source).
func Parse(src []byte, opts ...ParseOption) (*Document, error) {
	cfg := parseConfig{}
	for _, opt := range opts {
		opt(&cfg)
	}
	for _, origin := range cfg.relativeOrigins {
		src = bytes.ReplaceAll(src, []byte(origin), []byte(""))
	}
	reader := text.NewReader(src)
	parser := goldmark.DefaultParser()
	node := parser.Parse(reader)
	return &Document{Node: node, Source: src}, nil
}
