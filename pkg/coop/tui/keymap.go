package tui

import (
	"charm.land/bubbles/v2/help"
	"charm.land/bubbles/v2/key"
)

type keyMap struct {
	Quit      key.Binding
	Up        key.Binding
	Down      key.Binding
	Left      key.Binding
	Right     key.Binding
	PageUp    key.Binding
	PageDown  key.Binding
	Top       key.Binding
	Bottom    key.Binding
	Expand    key.Binding
	Enter     key.Binding
	Tab       key.Binding
	Escape    key.Binding
	Follow    key.Binding
	Confirm   key.Binding
	Reject    key.Binding
	Copy      key.Binding
	OpenClaim key.Binding
}

func newKeyMap() keyMap {
	return keyMap{
		Quit: key.NewBinding(
			key.WithKeys("q", "ctrl+c"),
			key.WithHelp("q", "quit"),
		),
		Up: key.NewBinding(
			key.WithKeys("up", "k"),
			key.WithHelp("↑/k", "up"),
		),
		Down: key.NewBinding(
			key.WithKeys("down", "j"),
			key.WithHelp("↓/j", "down"),
		),
		Left: key.NewBinding(
			key.WithKeys("left"),
			key.WithHelp("←", "collapse"),
		),
		Right: key.NewBinding(
			key.WithKeys("right"),
			key.WithHelp("→", "expand"),
		),
		PageUp: key.NewBinding(
			key.WithKeys("pgup", "b"),
			key.WithHelp("pgup/b", "page up"),
		),
		PageDown: key.NewBinding(
			key.WithKeys("pgdown", "space"),
			key.WithHelp("pgdn/space", "page down"),
		),
		Top: key.NewBinding(
			key.WithKeys("home", "g"),
			key.WithHelp("g", "top"),
		),
		Bottom: key.NewBinding(
			key.WithKeys("end", "G"),
			key.WithHelp("G", "bottom"),
		),
		Expand: key.NewBinding(
			key.WithKeys("e", "?"),
			key.WithHelp("e/?", "details"),
		),
		Enter: key.NewBinding(
			key.WithKeys("enter"),
			key.WithHelp("enter", "expand"),
		),
		Tab: key.NewBinding(
			key.WithKeys("tab"),
			key.WithHelp("tab", "next tab"),
		),
		Escape: key.NewBinding(
			key.WithKeys("esc"),
			key.WithHelp("esc", "close"),
		),
		Follow: key.NewBinding(
			key.WithKeys("f"),
			key.WithHelp("f", "follow"),
		),
		Confirm: key.NewBinding(
			key.WithKeys("c"),
			key.WithHelp("c", "confirm"),
		),
		Reject: key.NewBinding(
			key.WithKeys("r"),
			key.WithHelp("r", "request changes"),
		),
		Copy: key.NewBinding(
			key.WithKeys("y"),
			key.WithHelp("y", "copy"),
		),
		OpenClaim: key.NewBinding(
			key.WithKeys("o"),
			key.WithHelp("o", "claim"),
		),
	}
}

func (m Model) ShortHelp() []key.Binding {
	if m.rejecting {
		return []key.Binding{
			key.NewBinding(key.WithKeys("enter"), key.WithHelp("enter", "send")),
			key.NewBinding(key.WithKeys("esc"), key.WithHelp("esc", "cancel")),
		}
	}

	var bindings []key.Binding

	if m.userMoved {
		bindings = append(bindings, m.keys.Follow)
	}

	if target, ok := m.selectedReviewTarget(); ok {
		if m.selectedReviewCommand() != "" {
			bindings = append(bindings, m.keys.Copy)
		}
		confirm := m.keys.Confirm
		reject := m.keys.Reject
		if m.width > 0 && m.width < 56 {
			confirm.SetHelp("c", "confirm")
			reject.SetHelp("r", "changes")
		} else if target.kind == "chapter" {
			confirm.SetHelp("c", "confirm all")
			reject.SetHelp("r", "changes")
		}
		bindings = append(bindings, confirm, reject)
	}

	bindings = append(bindings, m.keys.Enter, m.keys.Quit)

	if m.selected.kind == navigationChapter {
		if m.chapterCollapsed(m.selected.chapterIndex) {
			bindings = append(bindings, m.keys.Right)
		} else {
			bindings = append(bindings, m.keys.Left)
		}
	}

	if m.expanded {
		bindings = append(bindings, m.keys.Tab, m.keys.Escape)
	}

	if m.session != nil && m.session.ClaimURL != "" {
		bindings = append(bindings, m.keys.OpenClaim)
	}

	return bindings
}

func (m Model) FullHelp() [][]key.Binding {
	return [][]key.Binding{m.ShortHelp()}
}

var _ help.KeyMap = Model{}
