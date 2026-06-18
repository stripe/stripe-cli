package markdown

import (
	"fmt"
	"os"

	"charm.land/glamour/v2"
	"charm.land/glamour/v2/ansi"
	"charm.land/lipgloss/v2"
)

const (
	defaultWordWrap = 140
)

// Renderer renders a Document to a terminal-friendly string.
type Renderer interface {
	Render(doc *Document) (string, error)
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
