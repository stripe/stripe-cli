package markdown_test

import (
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
	tests := []struct {
		name string
		src  string
		want []struct {
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
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			doc, err := markdown.Parse([]byte(tt.src))
			require.NoError(t, err)

			refs := doc.References()
			require.Len(t, refs, len(tt.want))
			for i, w := range tt.want {
				assert.Equal(t, w.title, refs[i].Title)
				assert.Equal(t, w.url, refs[i].URL.String())
				assert.Equal(t, w.external, refs[i].External)
			}
		})
	}
}

func TestParseWithRelativeURLs(t *testing.T) {
	tests := []struct {
		name     string
		src      string
		wantURLs []string
	}{
		{
			name:     "strips docs.stripe.com origin",
			src:      "[Payments](https://docs.stripe.com/payments)",
			wantURLs: []string{"/payments"},
		},
		{
			name:     "preserves external links unchanged",
			src:      "[Stripe](https://stripe.com/blog)",
			wantURLs: []string{"https://stripe.com/blog"},
		},
		{
			name:     "preserves already-relative links",
			src:      "[Get started](/get-started)",
			wantURLs: []string{"/get-started"},
		},
		{
			name: "handles mixed links",
			src:  "[A](https://docs.stripe.com/a)\n\n[B](https://stripe.com/b)\n\n[C](/c)",
			wantURLs: []string{"/a", "https://stripe.com/b", "/c"},
		},
	}

	t.Run("strips multiple origins", func(t *testing.T) {
		src := "[A](https://docs.stripe.com/a)\n\n[B](https://stripe.com/b)"
		doc, err := markdown.Parse([]byte(src),
			markdown.WithRelativeURLs("https://docs.stripe.com"),
			markdown.WithRelativeURLs("https://stripe.com"),
		)
		require.NoError(t, err)
		refs := doc.References()
		require.Len(t, refs, 2)
		assert.Equal(t, "/a", refs[0].URL.String())
		assert.Equal(t, "/b", refs[1].URL.String())
	})

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			doc, err := markdown.Parse([]byte(tt.src), markdown.WithRelativeURLs("https://docs.stripe.com"))

			require.NoError(t, err)
			refs := doc.References()
			require.Len(t, refs, len(tt.wantURLs))
			for i, want := range tt.wantURLs {
				assert.Equal(t, want, refs[i].URL.String())
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
