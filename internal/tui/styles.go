package tui

import "charm.land/lipgloss/v2"

// Styles defines the shared visual styles for the TUI.
type Styles struct {
	StatusBar   lipgloss.Style
	StatusTitle lipgloss.Style
	StatusHelp  lipgloss.Style
}

// DefaultStyles returns the default set of styles.
func DefaultStyles() Styles {
	return Styles{
		StatusBar: lipgloss.NewStyle().
			Background(lipgloss.Color("#2D2D2D")).
			Foreground(lipgloss.Color("#EEEEEE")),
		StatusTitle: lipgloss.NewStyle().
			Bold(true).
			Background(lipgloss.Color("#7D56F4")).
			Foreground(lipgloss.Color("#FFFFFF")).
			Padding(0, 1),
		StatusHelp: lipgloss.NewStyle().
			Background(lipgloss.Color("#3D3D3D")).
			Foreground(lipgloss.Color("#EEEEEE")).
			Padding(0, 1),
	}
}
