package tui

import (
	"image/color"

	"charm.land/lipgloss/v2"
)

// Brand colors used across the TUI.
var (
	colorBlurple    = lipgloss.Color("#635BFF")
	colorBlurpleMid = lipgloss.Color("#463FB0")
	colorBlurpleDim = lipgloss.Color("#2A2560")
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
			Background(lipgloss.Color("#2D2D2D")).
			Foreground(lipgloss.Color("#EEEEEE")),
		StatusTitle: lipgloss.NewStyle().
			Bold(true).
			Background(lipgloss.Color("#7D56F4")).
			Foreground(lipgloss.Color("#FFFFFF")).
			Padding(0, 1),
		StatusMessage: lipgloss.NewStyle().
			Background(lipgloss.Color("#3B9C5E")).
			Foreground(lipgloss.Color("#FFFFFF")),
		StatusHelp: lipgloss.NewStyle().
			Background(lipgloss.Color("#3D3D3D")).
			Foreground(lipgloss.Color("#EEEEEE")).
			Padding(0, 1),

		LandingTitle: lipgloss.NewStyle().
			Bold(true).
			Foreground(colorBlurple),
		LandingSubtitle: lipgloss.NewStyle().
			Foreground(lipgloss.Color("#999999")),
		LandingHint: lipgloss.NewStyle().
			Foreground(lipgloss.Color("#666666")),
		LandingDotBright: colorBlurple,
		LandingDotMid:    colorBlurpleMid,
		LandingDotDim:    colorBlurpleDim,
	}
}
