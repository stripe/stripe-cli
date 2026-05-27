package markdown_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/yuin/goldmark/ast"

	"github.com/stripe/stripe-cli-docs-plugin/markdown"
)

func TestParse(t *testing.T) {
	tests := []struct {
		name        string
		src         string
		wantNodeKind ast.NodeKind
	}{
		{
			name:        "valid markdown",
			src:         "# Hello\n\nSome paragraph text.",
			wantNodeKind: ast.KindDocument,
		},
		{
			name:        "empty input",
			src:         "",
			wantNodeKind: ast.KindDocument,
		},
		{
			name:        "whitespace only",
			src:         "   \n\n\t",
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
