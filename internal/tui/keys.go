package tui

import (
	"charm.land/bubbles/v2/key"
)

// KeyMap defines the keybindings for the TUI.
type KeyMap struct {
	Quit          key.Binding
	Up            key.Binding
	Down          key.Binding
	PageUp        key.Binding
	PageDown      key.Binding
	Help          key.Binding
	Palette       key.Binding
	Search        key.Binding
	OpenInBrowser key.Binding
	Enter         key.Binding
}

// ShortHelp returns bindings for the short help view.
func (k KeyMap) ShortHelp() []key.Binding {
	return []key.Binding{k.Help}
}

// FullHelp returns bindings grouped by column for the full help view.
func (k KeyMap) FullHelp() [][]key.Binding {
	return [][]key.Binding{
		{k.Up, k.Down},
		{k.PageUp, k.PageDown},
		{k.OpenInBrowser},
		{k.Search, k.Palette},
		{k.Quit},
	}
}

// DefaultKeyMap returns the default set of keybindings.
func DefaultKeyMap() KeyMap {
	return KeyMap{
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
		PageUp: key.NewBinding(
			key.WithKeys("pgup", "b"),
			key.WithHelp("b/pgup", "page up"),
		),
		PageDown: key.NewBinding(
			key.WithKeys("pgdown", "space", "f"),
			key.WithHelp("f/pgdn", "page down"),
		),
		Help: key.NewBinding(
			key.WithKeys("?"),
			key.WithHelp("?", "help"),
		),
		Palette: key.NewBinding(
			key.WithKeys(">"),
			key.WithHelp(">", "commands"),
		),
		Search: key.NewBinding(
			key.WithKeys("/"),
			key.WithHelp("/", "search"),
		),
		OpenInBrowser: key.NewBinding(
			key.WithKeys("o"),
			key.WithHelp("o", "open in browser"),
		),
		Enter: key.NewBinding(
			key.WithKeys("enter"),
			key.WithHelp("↵", "browse"),
		),
	}
}
