package markdown

import (
	"bytes"
	"net/url"

	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/text"
)

// ParseOption configures Parse behaviour.
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
func (d *Document) References() []Reference {
	var refs []Reference
	_ = ast.Walk(d.Node, func(n ast.Node, entering bool) (ast.WalkStatus, error) {
		if !entering {
			return ast.WalkContinue, nil
		}
		if link, ok := n.(*ast.Link); ok {
			if u, err := url.Parse(string(link.Destination)); err == nil {
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
