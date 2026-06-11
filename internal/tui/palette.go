package tui

import (
	"context"
	"net/url"
	"strings"
	"time"

	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/joelzwarrington/foam/palette"

	"github.com/stripe/stripe-cli/internal/docs"
	"github.com/stripe/stripe-cli/internal/markdown"
)

const (
	paletteWidth       = 60
	searchModeName     = "search"
	searchModeDebounce = 300 * time.Millisecond
	referenceModeName  = "reference"
	docsHost           = "docs.stripe.com"
)

type closePaletteMsg struct{}

// searchHit is a docs search result item displayed in the palette.
type searchHit struct {
	title string
	url   string
}

func (h searchHit) FilterValue() string { return h.title }
func (h searchHit) Title() string       { return h.title }
func (h searchHit) Description() string {
	const docsOrigin = "https://" + docsHost
	if strings.HasPrefix(h.url, docsOrigin) {
		return h.url[len(docsOrigin):]
	}
	return h.url
}

// reference is a link extracted from the current document.
type reference struct {
	title    string
	url      *url.URL
	external bool
}

func (r reference) FilterValue() string { return r.title + " " + r.url.Path }
func (r reference) Title() string       { return r.title }
func (r reference) Description() string {
	if r.url.Host == docsHost {
		return r.url.RequestURI()
	}
	if r.url.Host != "" {
		return r.url.Host + r.url.RequestURI()
	}
	return r.url.String()
}

// Palette wraps the foam palette model with visibility state and
// overlay rendering.
type Palette struct {
	palette.Model
	visible bool
}

func newPalette(page Page, doc *markdown.Document, client *docs.Client) Palette {
	commands := []palette.Item{
		palette.Command{
			ID:   "copy-markdown",
			Name: "Copy page as Markdown",
			Desc: "Copy the raw Markdown source to clipboard",
			Run: func() tea.Cmd {
				return func() tea.Msg {
					_ = page.Copy()
					return statusMsg("Copied!")
				}
			},
		},
		palette.Command{
			ID:   "open-in-browser",
			Name: "Open in browser",
			Desc: "Open this page on docs.stripe.com",
			Run: func() tea.Cmd {
				return func() tea.Msg {
					_ = page.Open(context.Background())
					return statusMsg("Opened!")
				}
			},
		},
	}

	commandsMode := palette.Mode{
		Name:  "commands",
		Match: func(s string) bool { return strings.HasPrefix(s, ">") },
		Query: func(s string) string { return strings.TrimSpace(strings.TrimPrefix(s, ">")) },
		Items: func(_ palette.Model, q string) []palette.Item {
			return palette.FilterFuzzy(commands, q)
		},
	}

	var refs []palette.Item
	if doc != nil {
		for _, r := range doc.References(page.URL) {
			refs = append(refs, reference{title: r.Title, url: r.URL, external: r.External})
		}
	}
	referenceMode := palette.Mode{
		Name:         referenceModeName,
		Placeholder:  "References...",
		EmptyMessage: "No references",
		Match:        func(s string) bool { return strings.HasPrefix(s, "@") },
		Query:        func(s string) string { return strings.TrimSpace(strings.TrimPrefix(s, "@")) },
		Items: func(_ palette.Model, q string) []palette.Item {
			return palette.FilterFuzzy(refs, q)
		},
	}

	searchMode := palette.Mode{
		Name:         searchModeName,
		Placeholder:  "Search...",
		EmptyMessage: "No results",
		Debounce:     searchModeDebounce,
		Match:        nil, // catch-all
		Items: func(m palette.Model, _ string) []palette.Item {
			return m.Results(searchModeName)
		},
		Search: func(ctx context.Context, q string) tea.Cmd {
			if client == nil || q == "" {
				return func() tea.Msg {
					return palette.SearchResultMsg{Mode: searchModeName, Query: q}
				}
			}
			return func() tea.Msg {
				resp, err := client.Search(ctx, q)
				if err != nil {
					if ctx.Err() != nil {
						return nil
					}
					return palette.SearchResultMsg{Mode: searchModeName, Query: q, Err: err}
				}
				items := make([]palette.Item, 0, len(resp.Hits))
				for _, h := range resp.Hits {
					items = append(items, searchHit{title: h.Title, url: h.URL})
				}
				return palette.SearchResultMsg{Mode: searchModeName, Query: q, Results: items}
			}
		},
	}

	p := palette.New(
		palette.WithModes(commandsMode, referenceMode, searchMode),
		palette.WithPageSize(5),
		palette.WithOnExecute(func(_ palette.Item) tea.Cmd {
			return func() tea.Msg { return closePaletteMsg{} }
		}),
	)
	p.SetWidth(paletteWidth)
	return Palette{Model: p}
}

// Visible reports whether the palette overlay is showing.
func (p Palette) Visible() bool { return p.visible }

// Open shows the palette and focuses its input.
func (p *Palette) Open() tea.Cmd {
	p.visible = true
	return p.Focus()
}

// syncKeyMap updates the Execute binding label and enabled state to match
// the active mode and whether an item is currently selected.
func (p *Palette) syncKeyMap() {
	label := "execute"
	if p.Mode().Name == searchModeName || p.Mode().Name == referenceModeName {
		label = "view"
	}
	p.KeyMap.Execute = key.NewBinding(
		key.WithKeys("enter"),
		key.WithHelp("enter", label),
	)
	p.KeyMap.Execute.SetEnabled(p.Selected() != nil)
}

// Dismiss hides the palette and resets its state.
func (p *Palette) Dismiss() {
	p.visible = false
	p.Blur()
	p.Reset()
}

// View composites the palette over the given background using the
// lipgloss Canvas/Layer/Compositor overlay system.
func (p Palette) View(bg string, width, height int) string {
	paletteLayer := lipgloss.NewLayer(p.Model.View()).
		X((width - paletteWidth) / 2).
		Y(1)
	canvas := lipgloss.NewCanvas(width, height)
	canvas.Compose(lipgloss.NewCompositor(
		lipgloss.NewLayer(bg),
		paletteLayer,
	))
	return canvas.Render()
}
