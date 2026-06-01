package tui

import (
	"image/color"

	"charm.land/lipgloss/v2"
)

// Styles defines the shared visual styles for the TUI.
type Styles struct {
	StatusBar     lipgloss.Style
	StatusTitle   lipgloss.Style
	StatusMessage lipgloss.Style
	StatusHelp    lipgloss.Style

	// Landing screen
	LandingTitle     lipgloss.Style
	LandingSubtitle  lipgloss.Style
	LandingHint      lipgloss.Style
	LandingDotBright color.Color
	LandingDotMid    color.Color
	LandingDotDim    color.Color
}

// DefaultStyles returns the default set of styles.
func DefaultStyles() Styles {
	return Styles{
		StatusBar: lipgloss.NewStyle().
			Background(lipgloss.Color("#1A2C44")).
			Foreground(lipgloss.Color("#ECF1F6")),
		StatusTitle: lipgloss.NewStyle().
			Bold(true).
			Background(lipgloss.Color("#533afd")).
			Foreground(lipgloss.Color("#FFFFFF")).
			Padding(0, 1),
		StatusMessage: lipgloss.NewStyle().
			Background(lipgloss.Color("#2b8700")).
			Foreground(lipgloss.Color("#FFFFFF")),
		StatusHelp: lipgloss.NewStyle().
			Background(lipgloss.Color("#273951")).
			Foreground(lipgloss.Color("#ECF1F6")).
			Padding(0, 1),

		LandingTitle: lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#675dff")),
		LandingSubtitle: lipgloss.NewStyle().
			Foreground(lipgloss.Color("#667691")),
		LandingHint: lipgloss.NewStyle().
			Foreground(lipgloss.Color("#50617A")),
		LandingDotBright: lipgloss.Color("#3c147c"),
		LandingDotMid:    lipgloss.Color("#5f4cfe"),
		LandingDotDim:    lipgloss.Color("#b1a7fd"),
	}
}
