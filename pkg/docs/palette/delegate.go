package palette

import (
	"fmt"
	"io"
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"
)

const ellipsis = "…"

// Item is anything that can appear in the palette list. Both predefined
// commands and async search results implement it.
type Item interface {
	FilterValue() string
}

// DefaultItem is the convention DefaultDelegate knows how to render.
// Implement this if you want title/description rendering out of the box.
type DefaultItem interface {
	Item
	Title() string
	Description() string
}

// ItemDelegate controls how an Item is rendered and how key events
// reach the currently selected item. Mirrors bubbles/list.ItemDelegate.
type ItemDelegate interface {
	// Height is the number of terminal rows one item occupies.
	Height() int
	// Spacing is the number of blank rows between adjacent items.
	Spacing() int
	// Update receives messages while the delegate is active. Implement
	// item-level keybindings here, or return nil to opt out.
	Update(msg tea.Msg, m *Model) tea.Cmd
	// Render draws one item at the given visible index to w.
	Render(w io.Writer, m Model, index int, item Item)
}

// DelegateStyles holds the styles used by DefaultDelegate.
type DelegateStyles struct {
	Title            lipgloss.Style
	Description      lipgloss.Style
	SelectedTitle    lipgloss.Style
	SelectedDesc     lipgloss.Style
	SelectionMarker  string
	UnselectedMarker string
}

// DefaultDelegate renders DefaultItems with a title/description layout
// and a selection marker. When ShowDescription is false, only the
// title line is drawn.
type DefaultDelegate struct {
	Styles          DelegateStyles
	ShowDescription bool
}

// NewDefaultDelegate returns a DefaultDelegate with sensible defaults.
// Selected rows are rendered with reverse-video so the highlight reads
// against any terminal palette without picking a brand colour.
func NewDefaultDelegate() DefaultDelegate {
	return DefaultDelegate{
		ShowDescription: true,
		Styles: DelegateStyles{
			Title:            lipgloss.NewStyle(),
			Description:      lipgloss.NewStyle().Faint(true),
			SelectedTitle:    lipgloss.NewStyle().Reverse(true),
			SelectedDesc:     lipgloss.NewStyle().Reverse(true).Faint(true),
			SelectionMarker:  "  ",
			UnselectedMarker: "  ",
		},
	}
}

// Height reports two rows when ShowDescription is on, one otherwise.
func (d DefaultDelegate) Height() int {
	if d.ShowDescription {
		return 2
	}
	return 1
}

// Spacing reports zero blank rows between items by default.
func (d DefaultDelegate) Spacing() int { return 0 }

// Update is a no-op by default. Override by wrapping or replacing.
func (d DefaultDelegate) Update(_ tea.Msg, _ *Model) tea.Cmd { return nil }

// Render draws one item: a selection marker followed by the title, and
// (when ShowDescription is on and the item exposes one) a faint
// description line indented under the title. Selected rows fill the
// palette's width so the highlight background reaches the right edge.
// Text is truncated to the palette's current width when known.
func (d DefaultDelegate) Render(w io.Writer, m Model, index int, item Item) {
	s := &d.Styles

	var title, desc string
	if di, ok := item.(DefaultItem); ok {
		title = di.Title()
		desc = di.Description()
	} else {
		title = item.FilterValue()
	}

	isSelected := m.IsSelected(index)
	marker := s.UnselectedMarker
	titleStyle := s.Title
	descStyle := s.Description
	if isSelected {
		marker = s.SelectionMarker
		titleStyle = s.SelectedTitle
		descStyle = s.SelectedDesc
	}

	width := m.Width()
	if width > 0 {
		avail := width - lipgloss.Width(marker)
		if avail < 1 {
			avail = 1
		}
		title = ansi.Truncate(title, avail, ellipsis)
		if desc != "" {
			desc = ansi.Truncate(desc, avail, ellipsis)
		}
	}

	titleLine := marker + title
	descLine := ""
	if d.ShowDescription && desc != "" {
		descLine = strings.Repeat(" ", lipgloss.Width(marker)) + desc
	}

	if isSelected && width > 0 {
		titleLine = titleStyle.Width(width).Render(titleLine)
		if descLine != "" {
			descLine = descStyle.Width(width).Render(descLine)
		}
	} else {
		titleLine = titleStyle.Render(titleLine)
		if descLine != "" {
			descLine = descStyle.Render(descLine)
		}
	}

	_, _ = fmt.Fprint(w, titleLine)
	if descLine != "" {
		_, _ = fmt.Fprintf(w, "\n%s", descLine)
	}
}
