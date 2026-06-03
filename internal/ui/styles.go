// Package ui provides shared visual styles and components for the TUI.
package ui

import (
	"image/color"

	"charm.land/lipgloss/v2"
)

// Semantic color palette.
var (
	text    = lipgloss.Color("#eceef1")
	subdued = lipgloss.Color("#a9b2c2")
	primary = lipgloss.Color("#9289fe")
	accent  = lipgloss.Color("#533afd") // deeper primary, for solid fills
	surface = lipgloss.Color("#1A2C44") // nav bar background
	overlay = lipgloss.Color("#273951") // elevated surface
	success = lipgloss.Color("#2b8700")
	danger  = lipgloss.Color("#FF0000")

	dotBright = lipgloss.Color("#3c147c")
	dotMid    = lipgloss.Color("#5f4cfe")
	dotDim    = lipgloss.Color("#b1a7fd")
)

// Styles defines the shared visual styles for the TUI.
type Styles struct {
	// Text hierarchy
	Title       lipgloss.Style
	Description lipgloss.Style
	Muted       lipgloss.Style
	Error       lipgloss.Style
	SuccessText lipgloss.Style

	// Status bar
	Bar     lipgloss.Style
	Brand   lipgloss.Style
	Success lipgloss.Style
	Help    lipgloss.Style

	// Logo animation dot colors
	DotBright color.Color
	DotMid    color.Color
	DotDim    color.Color
}

// DefaultStyles returns the default set of styles.
func DefaultStyles() Styles {
	return Styles{
		Title:       lipgloss.NewStyle().Bold(true).Foreground(primary),
		Description: lipgloss.NewStyle().Foreground(subdued),
		Muted:       lipgloss.NewStyle().Foreground(subdued),
		Error:       lipgloss.NewStyle().Foreground(danger),
		SuccessText: lipgloss.NewStyle().Foreground(success),

		Bar:     lipgloss.NewStyle().Background(surface).Foreground(text),
		Brand:   lipgloss.NewStyle().Bold(true).Background(accent).Foreground(text).Padding(0, 1),
		Success: lipgloss.NewStyle().Background(success).Foreground(text),
		Help:    lipgloss.NewStyle().Background(overlay).Foreground(text).Padding(0, 1),

		DotBright: dotBright,
		DotMid:    dotMid,
		DotDim:    dotDim,
	}
}
