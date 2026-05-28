package tui

import (
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/joelzwarrington/foam/palette"
)

const paletteWidth = 60

type closePaletteMsg struct{}

// Palette wraps the foam palette model with visibility state and
// overlay rendering.
type Palette struct {
	palette.Model
	visible bool
}

func newPalette(page Page) Palette {
	commands := []palette.Item{
		palette.Command{
			ID:   "copy-markdown",
			Name: "Copy page as Markdown",
			Desc: "Copy the raw markdown source to clipboard",
			Run: func() tea.Cmd {
				return func() tea.Msg {
					_ = page.Copy()
					return nil
				}
			},
		},
	}

	p := palette.New(
		palette.WithModes(
			palette.Mode{
				Name:  "commands",
				Match: func(s string) bool { return strings.HasPrefix(s, ">") },
				Query: func(s string) string { return strings.TrimSpace(strings.TrimPrefix(s, ">")) },
				Items: func(_ palette.Model, q string) []palette.Item {
					return palette.FilterFuzzy(commands, q)
				},
			}),
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
