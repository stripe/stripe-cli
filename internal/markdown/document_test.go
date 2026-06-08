package markdown_test

import (
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stripe/stripe-cli-docs-plugin/internal/markdown"
	"github.com/yuin/goldmark/ast"
)

func TestParse(t *testing.T) {
	tests := []struct {
		name         string
		src          string
		wantNodeKind ast.NodeKind
	}{
		{
			name:         "valid markdown",
			src:          "# Hello\n\nSome paragraph text.",
			wantNodeKind: ast.KindDocument,
		},
		{
			name:         "empty input",
			src:          "",
			wantNodeKind: ast.KindDocument,
		},
		{
			name:         "whitespace only",
			src:          "   \n\n\t",
			wantNodeKind: ast.KindDocument,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			doc, err := markdown.Parse([]byte(tt.src))
			require.NoError(t, err)
			require.NotNil(t, doc)
			assert.Equal(t, tt.wantNodeKind, doc.Node.Kind())
			assert.Equal(t, []byte(tt.src), doc.Source)
		})
	}
}

func TestTitle(t *testing.T) {
	tests := []struct {
		name string
		src  string
		want string
	}{
		{
			name: "extracts first h1",
			src:  "# Hello World\n\nSome content.",
			want: "Hello World",
		},
		{
			name: "ignores h2",
			src:  "## Not a title\n\n# Real Title",
			want: "Real Title",
		},
		{
			name: "returns empty when no h1",
			src:  "## Heading Two\n\nParagraph.",
			want: "",
		},
		{
			name: "empty document",
			src:  "",
			want: "",
		},
		{
			name: "inline formatting in heading",
			src:  "# Hello **World**\n\nContent.",
			want: "Hello World",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			doc, err := markdown.Parse([]byte(tt.src))
			require.NoError(t, err)
			assert.Equal(t, tt.want, doc.Title())
		})
	}
}

func TestReferences(t *testing.T) {
	mustURL := func(raw string) *url.URL {
		u, err := url.Parse(raw)
		if err != nil {
			panic(err)
		}
		return u
	}

	tests := []struct {
		name       string
		src        string
		currentURL *url.URL
		want       []struct {
			title    string
			url      string
			external bool
		}
	}{
		{
			name: "internal docs.stripe.com link",
			src:  "[Accept a payment](https://docs.stripe.com/payments/accept-a-payment)",
			want: []struct {
				title    string
				url      string
				external bool
			}{
				{title: "Accept a payment", url: "https://docs.stripe.com/payments/accept-a-payment", external: false},
			},
		},
		{
			name: "relative link",
			src:  "[Get started](/get-started)",
			want: []struct {
				title    string
				url      string
				external bool
			}{
				{title: "Get started", url: "/get-started", external: false},
			},
		},
		{
			name: "external link",
			src:  "[Stripe](https://stripe.com/blog)",
			want: []struct {
				title    string
				url      string
				external bool
			}{
				{title: "Stripe", url: "https://stripe.com/blog", external: true},
			},
		},
		{
			name: "multiple links",
			src:  "[A](/a)\n\n[B](https://stripe.com/b)",
			want: []struct {
				title    string
				url      string
				external bool
			}{
				{title: "A", url: "/a", external: false},
				{title: "B", url: "https://stripe.com/b", external: true},
			},
		},
		{
			name: "no links",
			src:  "# Just a heading\n\nSome text.",
			want: nil,
		},
		{
			name: "pure anchor filtered",
			src:  "[Jump](#section-id)",
			want: nil,
		},
		{
			name:       "relative same-page anchor filtered",
			src:        "[Jump](/payments/accept-a-payment#section)",
			currentURL: mustURL("https://docs.stripe.com/payments/accept-a-payment"),
			want:       nil,
		},
		{
			name:       "absolute same-page anchor filtered",
			src:        "[Jump](https://docs.stripe.com/payments/accept-a-payment#section)",
			currentURL: mustURL("https://docs.stripe.com/payments/accept-a-payment"),
			want:       nil,
		},
		{
			name:       "different page anchor kept",
			src:        "[Other](/other-page#section)",
			currentURL: mustURL("https://docs.stripe.com/payments/accept-a-payment"),
			want: []struct {
				title    string
				url      string
				external bool
			}{
				{title: "Other", url: "/other-page#section", external: false},
			},
		},
		{
			name:       "external site same-path anchor kept",
			src:        "[Ext](https://stripe.com/payments/accept-a-payment#section)",
			currentURL: mustURL("https://docs.stripe.com/payments/accept-a-payment"),
			want: []struct {
				title    string
				url      string
				external bool
			}{
				{title: "Ext", url: "https://stripe.com/payments/accept-a-payment#section", external: true},
			},
		},
		{
			name:       "same path same query anchor filtered",
			src:        "[Jump](/payments/accept-a-payment?client=ios#section)",
			currentURL: mustURL("https://docs.stripe.com/payments/accept-a-payment?client=ios"),
			want:       nil,
		},
		{
			name:       "same path different query anchor kept",
			src:        "[Jump](/payments/accept-a-payment?client=ios#section)",
			currentURL: mustURL("https://docs.stripe.com/payments/accept-a-payment?client=web"),
			want: []struct {
				title    string
				url      string
				external bool
			}{
				{title: "Jump", url: "/payments/accept-a-payment?client=ios#section", external: false},
			},
		},
		{
			name:       "same path same query different order anchor filtered",
			src:        "[Jump](/payments/accept-a-payment?lang=en&client=ios#section)",
			currentURL: mustURL("https://docs.stripe.com/payments/accept-a-payment?client=ios&lang=en"),
			want:       nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			doc, err := markdown.Parse([]byte(tt.src))
			require.NoError(t, err)

			refs := doc.References(tt.currentURL)
			require.Len(t, refs, len(tt.want))
			for i, w := range tt.want {
				assert.Equal(t, w.title, refs[i].Title)
				assert.Equal(t, w.url, refs[i].URL.String())
				assert.Equal(t, w.external, refs[i].External)
			}
		})
	}
}

func TestParseAST(t *testing.T) {
	src := "# Hello\n\nParagraph."
	doc, err := markdown.Parse([]byte(src))
	require.NoError(t, err)

	var kinds []ast.NodeKind
	_ = ast.Walk(doc.Node, func(n ast.Node, entering bool) (ast.WalkStatus, error) {
		if entering {
			kinds = append(kinds, n.Kind())
		}
		return ast.WalkContinue, nil
	})

	assert.Contains(t, kinds, ast.KindHeading)
	assert.Contains(t, kinds, ast.KindParagraph)
}
