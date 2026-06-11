package markdown_test

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/stripe/stripe-cli/internal/markdown"
)

var (
	update  = flag.Bool("update", false, "update golden files")
	preview = flag.Bool("preview", false, "render showcase to stdout for visual inspection")
)

// TestRendererPreview renders the showcase document for visual inspection.
// To run: go test ./markdown/ -run TestRendererPreview -preview -v
func TestRendererPreview(t *testing.T) {
	if !*preview {
		t.Skip("run with -preview to render showcase to stdout")
	}

	src, err := os.ReadFile("testdata/showcase.md")
	require.NoError(t, err)

	doc, err := markdown.Parse(src)
	require.NoError(t, err)

	for _, style := range []string{"dark", "light", "notty"} {
		r, err := markdown.NewRenderer(markdown.WithStyle(style))
		require.NoError(t, err)

		out, err := r.Render(doc)
		require.NoError(t, err)

		_, _ = fmt.Fprintf(os.Stdout, "\n\033[1m=== style: %s ===\033[0m\n%s", style, out)
	}
}

func TestNewRenderer(t *testing.T) {
	tests := []struct {
		name string
		opts []markdown.RendererOption
	}{
		{name: "no options"},
		{name: "with style", opts: []markdown.RendererOption{markdown.WithStyle("dark")}},
		{name: "with word wrap", opts: []markdown.RendererOption{markdown.WithWordWrap(120)}},
		{name: "with style and word wrap", opts: []markdown.RendererOption{markdown.WithStyle("light"), markdown.WithWordWrap(60)}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r, err := markdown.NewRenderer(tt.opts...)
			require.NoError(t, err)
			assert.NotNil(t, r)
		})
	}
}

func TestRendererRenderShowcase(t *testing.T) {
	src, err := os.ReadFile("testdata/showcase.md")
	require.NoError(t, err)

	doc, err := markdown.Parse(src)
	require.NoError(t, err)

	for _, style := range []string{"dark", "light", "notty"} {
		t.Run(style, func(t *testing.T) {
			r, err := markdown.NewRenderer(markdown.WithStyle(style))
			require.NoError(t, err)

			out, err := r.Render(doc)
			require.NoError(t, err)

			golden := filepath.Join("testdata", "showcase."+style+".golden")
			if *update {
				require.NoError(t, os.WriteFile(golden, []byte(out), 0600))
			}
			want, err := os.ReadFile(golden)
			require.NoError(t, err)
			assert.Equal(t, string(want), out)
		})
	}
}

func TestRendererRender(t *testing.T) {
	tests := []struct {
		name        string
		src         string
		wantContain []string
	}{
		{
			name:        "heading and paragraph",
			src:         "# Hello\n\nThis is a paragraph.",
			wantContain: []string{"Hello", "This is a paragraph."},
		},
		{
			name:        "code block",
			src:         "```go\nfmt.Println(\"hello\")\n```",
			wantContain: []string{"fmt.Println"},
		},
		{
			name:        "bold and italic",
			src:         "**bold** and *italic*",
			wantContain: []string{"bold", "italic"},
		},
		{
			name:        "unordered list",
			src:         "- alpha\n- beta\n- gamma",
			wantContain: []string{"alpha", "beta", "gamma"},
		},
		{
			name:        "empty",
			src:         "",
			wantContain: nil,
		},
	}

	r, err := markdown.NewRenderer(markdown.WithStyle("notty"))
	require.NoError(t, err)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			doc, err := markdown.Parse([]byte(tt.src))
			require.NoError(t, err)

			out, err := r.Render(doc)
			require.NoError(t, err)
			for _, want := range tt.wantContain {
				assert.Contains(t, out, want)
			}
		})
	}
}
