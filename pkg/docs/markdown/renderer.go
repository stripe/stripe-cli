package markdown

import (
	"fmt"
	"os"
	"strings"

	"charm.land/glamour/v2"
	"charm.land/glamour/v2/ansi"
	"charm.land/lipgloss/v2"
	"github.com/acarl005/stripansi"
)

const (
	defaultWordWrap = 140
)

// Renderer renders a Document to a terminal-friendly string.
type Renderer interface {
	Render(doc *Document) (string, error)
}

// RenderedDocument contains rendered content and the line offset of each heading.
type RenderedDocument struct {
	Content        string
	HeadingOffsets map[string]int
}

// RenderWithHeadingOffsets renders doc and locates each heading in the output.
func RenderWithHeadingOffsets(r Renderer, doc *Document) (RenderedDocument, error) {
	content, err := r.Render(doc)
	if err != nil {
		return RenderedDocument{}, err
	}

	result := RenderedDocument{
		Content:        content,
		HeadingOffsets: make(map[string]int),
	}
	lines := strings.Split(content, "\n")
	searchFrom := 0
	for _, heading := range doc.Headings() {
		for i := searchFrom; i < len(lines); i++ {
			if renderedLineMatchesHeading(lines[i], heading) {
				result.HeadingOffsets[heading.Fragment] = i
				searchFrom = i + 1
				break
			}
		}
	}
	return result, nil
}

func renderedLineMatchesHeading(line string, heading Heading) bool {
	line = strings.TrimSpace(stripansi.Strip(line))
	if heading.Level > 1 && !strings.HasPrefix(line, strings.Repeat("#", heading.Level)+" ") {
		return false
	}
	line = strings.TrimSpace(strings.TrimLeft(line, "#"))
	lineFragment := NormalizeFragment(line)
	headingFragment := NormalizeFragment(heading.Text)
	return lineFragment != "" && (lineFragment == headingFragment || strings.HasPrefix(headingFragment, lineFragment+"-"))
}

// WithStyle sets a built-in glamour style by name (e.g. "dark", "light", "dracula", "notty").
// The default auto-detects dark/light from the terminal and applies our own style configs.
func WithStyle(style string) RendererOption {
	return func(r *rendererConfig) { r.style = style }
}

// WithStyleConfig sets a fully custom ansi.StyleConfig, overriding both the named style
// and the auto-detected default. Use DarkStyleConfig or LightStyleConfig as a starting point.
func WithStyleConfig(cfg ansi.StyleConfig) RendererOption {
	return func(r *rendererConfig) { r.styleConfig = &cfg }
}

// WithWordWrap sets the column width at which output is wrapped.
func WithWordWrap(width int) RendererOption {
	return func(r *rendererConfig) { r.wordWrap = width }
}

type glamourRenderer struct {
	tr *glamour.TermRenderer
}

// NewRenderer constructs a Renderer backed by glamour with sensible defaults
// (auto-detected dark/light using our own style configs, 140-column word wrap).
// Options override individual defaults.
func NewRenderer(opts ...RendererOption) (Renderer, error) {
	cfg := rendererConfig{wordWrap: defaultWordWrap}
	for _, opt := range opts {
		opt(&cfg)
	}

	glamourOpts := []glamour.TermRendererOption{
		glamour.WithWordWrap(cfg.wordWrap),
	}
	switch {
	case cfg.styleConfig != nil:
		glamourOpts = append(glamourOpts, glamour.WithStyles(*cfg.styleConfig))
	case cfg.style != "":
		glamourOpts = append(glamourOpts, glamour.WithStylePath(cfg.style))
	default:
		if lipgloss.HasDarkBackground(os.Stdin, os.Stdout) {
			glamourOpts = append(glamourOpts, glamour.WithStyles(DarkStyleConfig))
		} else {
			glamourOpts = append(glamourOpts, glamour.WithStyles(LightStyleConfig))
		}
	}

	tr, err := glamour.NewTermRenderer(glamourOpts...)
	if err != nil {
		return nil, fmt.Errorf("creating term renderer: %w", err)
	}
	return &glamourRenderer{tr: tr}, nil
}

// Render renders doc to a terminal-styled string using glamour.
func (r *glamourRenderer) Render(doc *Document) (string, error) {
	out, err := r.tr.Render(string(doc.Source))
	if err != nil {
		return "", fmt.Errorf("rendering markdown: %w", err)
	}
	return out, nil
}

// rendererConfig holds options accumulated before the TermRenderer is built.
type rendererConfig struct {
	style       string
	styleConfig *ansi.StyleConfig
	wordWrap    int
}

// RendererOption configures a Renderer created by NewRenderer.
type RendererOption func(*rendererConfig)
