package tui

import (
	"fmt"
	"image/color"
	"os"
	"strings"
	"sync"

	"charm.land/glamour/v2"
	glamouransi "charm.land/glamour/v2/ansi"
)

func (m Model) renderMarkdown(content string, width int) string {
	if content == "" {
		return ""
	}
	renderer, err := markdownRenderer(width, m.isDark)
	if err != nil {
		return content
	}
	rendered, err := renderer.Render(content)
	if err != nil {
		return content
	}
	return strings.TrimSpace(rendered)
}

type markdownRendererKey struct {
	width int
	dark  bool
	style string
}

var markdownRenderers = struct {
	sync.Mutex
	byKey map[markdownRendererKey]*glamour.TermRenderer
}{byKey: map[markdownRendererKey]*glamour.TermRenderer{}}

func markdownRenderer(width int, isDark bool) (*glamour.TermRenderer, error) {
	if width < 1 {
		width = 1
	}
	style := os.Getenv("GLAMOUR_STYLE")
	key := markdownRendererKey{width: width, dark: isDark, style: style}

	markdownRenderers.Lock()
	defer markdownRenderers.Unlock()
	if renderer := markdownRenderers.byKey[key]; renderer != nil {
		return renderer, nil
	}

	var styleOpt glamour.TermRendererOption
	if style != "" {
		styleOpt = glamour.WithEnvironmentConfig()
	} else {
		styleOpt = glamour.WithStyles(compactMarkdownStyle(isDark))
	}
	renderer, err := glamour.NewTermRenderer(
		styleOpt,
		glamour.WithWordWrap(width),
		glamour.WithEmoji(),
		glamour.WithPreservedNewLines(),
		glamour.WithTableWrap(false),
	)
	if err != nil {
		return nil, err
	}
	markdownRenderers.byKey[key] = renderer
	return renderer, nil
}

func compactMarkdownStyle(isDark bool) glamouransi.StyleConfig {
	theme := NewTheme(isDark)
	text := colorHex(theme.Text)
	muted := colorHex(theme.Gray400)
	accent := colorHex(theme.Purple400)
	rule := colorHex(theme.Border)
	codeBG := colorHex(theme.Panel)
	codeTheme := "monokai"
	if !isDark {
		codeTheme = "github"
	}

	return glamouransi.StyleConfig{
		Document: glamouransi.StyleBlock{
			StylePrimitive: glamouransi.StylePrimitive{
				Color: stringPtr(text),
			},
			Margin: uintPtr(0),
		},
		Heading: glamouransi.StyleBlock{
			StylePrimitive: glamouransi.StylePrimitive{
				Color: stringPtr(accent),
				Bold:  boolPtr(true),
			},
		},
		List: glamouransi.StyleList{
			StyleBlock: glamouransi.StyleBlock{
				Margin: uintPtr(0),
			},
			LevelIndent: 2,
		},
		BlockQuote: glamouransi.StyleBlock{
			StylePrimitive: glamouransi.StylePrimitive{
				Color: stringPtr(muted),
			},
			Indent:      uintPtr(1),
			IndentToken: stringPtr("│ "),
			Margin:      uintPtr(0),
		},
		Strong: glamouransi.StylePrimitive{
			Bold: boolPtr(true),
		},
		Emph: glamouransi.StylePrimitive{
			Italic: boolPtr(true),
			Color:  stringPtr(muted),
		},
		HorizontalRule: glamouransi.StylePrimitive{
			Color:  stringPtr(rule),
			Format: "\n--------\n",
		},
		Item: glamouransi.StylePrimitive{
			BlockPrefix: "• ",
		},
		Enumeration: glamouransi.StylePrimitive{
			BlockPrefix: ". ",
		},
		Task: glamouransi.StyleTask{
			Ticked:   "[✓] ",
			Unticked: "[ ] ",
		},
		Link: glamouransi.StylePrimitive{
			Color:     stringPtr(accent),
			Underline: boolPtr(true),
		},
		LinkText: glamouransi.StylePrimitive{
			Color: stringPtr(accent),
			Bold:  boolPtr(true),
		},
		Code: glamouransi.StyleBlock{
			StylePrimitive: glamouransi.StylePrimitive{
				Color:           stringPtr(text),
				BackgroundColor: stringPtr(codeBG),
			},
		},
		CodeBlock: glamouransi.StyleCodeBlock{
			StyleBlock: glamouransi.StyleBlock{
				StylePrimitive: glamouransi.StylePrimitive{
					Color: stringPtr(text),
				},
				Margin: uintPtr(0),
			},
			Theme: codeTheme,
		},
		Table: glamouransi.StyleTable{
			StyleBlock: glamouransi.StyleBlock{
				Margin: uintPtr(0),
			},
			CenterSeparator: stringPtr("|"),
			ColumnSeparator: stringPtr("|"),
			RowSeparator:    stringPtr("-"),
		},
	}
}

func stringPtr(v string) *string {
	return &v
}

func boolPtr(v bool) *bool {
	return &v
}

func uintPtr(v uint) *uint {
	return &v
}

func colorHex(c color.Color) string {
	r, g, b, _ := c.RGBA()
	return fmt.Sprintf("#%02x%02x%02x", uint8(r>>8), uint8(g>>8), uint8(b>>8))
}
