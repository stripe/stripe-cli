# markdown

This package provides markdown parsing and terminal rendering for the `stripe docs` CLI.

## Usage

```go
// Parse markdown into a Document (gives you a goldmark AST + source bytes).
doc, err := markdown.Parse(src)

// Walk the AST to extract structure (headings, links, code blocks, etc.).
ast.Walk(doc.Node, func(n ast.Node, entering bool) (ast.WalkStatus, error) {
    if entering {
        if h, ok := n.(*ast.Heading); ok {
            fmt.Println("heading level", h.Level)
        }
    }
    return ast.WalkContinue, nil
})

// Render to a terminal-styled string. Auto-detects dark/light using our own
// DarkStyleConfig / LightStyleConfig. Override with WithStyle or WithStyleConfig.
r, err := markdown.NewRenderer()
out, err := r.Render(doc)
fmt.Print(out)

// Use a built-in glamour style by name.
r, err := markdown.NewRenderer(markdown.WithStyle("dracula"))

// Supply a fully custom style, using our configs as a starting point.
cfg := markdown.DarkStyleConfig
cfg.H1.StylePrimitive.Color = strPtr("#FF6600")
r, err := markdown.NewRenderer(markdown.WithStyleConfig(cfg))
```

## Conventions

- `RendererOption` follows the functional options pattern; add new options as `func(...) RendererOption`
- Style customisation starts from `DarkStyleConfig` or `LightStyleConfig` in `styles.go` — edit those rather than touching glamour's built-ins
- The `TermRenderer` is built once in `NewRenderer`, not on every `Render` call
- To preview all styles in the terminal: `go test ./markdown/ -run TestRendererPreview -preview -v`
- To regenerate golden files after a style change: `go test ./markdown/ -run TestRendererRenderShowcase -update`
