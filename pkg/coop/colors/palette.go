// Package colors provides shared Sail color tokens for co-op prompts and TUI themes.
package colors

import (
	"image/color"

	"charm.land/lipgloss/v2"
)

var (
	// Sail Design System hue palette (from tokensColor.ts).
	Purple500 = lipgloss.Color("#625afa")
	Purple400 = lipgloss.Color("#8d7ffa")
	Purple700 = lipgloss.Color("#3f32a1")

	Green300 = lipgloss.Color("#48c404")
	Green400 = lipgloss.Color("#3fa40d")

	Blue400 = lipgloss.Color("#00a3ff")
	Blue700 = lipgloss.Color("#0b3a5b")

	Orange400 = lipgloss.Color("#ed6704")

	Gray300 = lipgloss.Color("#a3acba")
	Gray400 = lipgloss.Color("#87909f")
	Gray500 = lipgloss.Color("#687385")
	Gray700 = lipgloss.Color("#414552")
)

type Palette struct {
	Purple500 color.Color
	Purple400 color.Color
	Purple700 color.Color
	Green300  color.Color
	Green400  color.Color
	Blue400   color.Color
	Blue700   color.Color
	Orange400 color.Color
	Gray300   color.Color
	Gray400   color.Color
	Gray500   color.Color
	Gray700   color.Color
	Error     color.Color
	Text      color.Color
	OnBrand   color.Color
	Border    color.Color
	Panel     color.Color
	Selection color.Color
}

func NewPalette(isDark bool) Palette {
	lightDark := lipgloss.LightDark(isDark)
	p := Palette{
		Purple500: lightDark(lipgloss.Color("#4f46d8"), Purple500),
		Purple400: lightDark(lipgloss.Color("#5f52e8"), Purple400),
		Purple700: lightDark(lipgloss.Color("#3f32a1"), Purple700),
		Green300:  lightDark(lipgloss.Color("#237500"), Green300),
		Green400:  lightDark(lipgloss.Color("#2f8506"), Green400),
		Blue400:   lightDark(lipgloss.Color("#006bb6"), Blue400),
		Blue700:   lightDark(lipgloss.Color("#dff2ff"), Blue700),
		Orange400: lightDark(lipgloss.Color("#b34800"), Orange400),
		Gray300:   lightDark(lipgloss.Color("#2f3640"), Gray300),
		Gray400:   lightDark(lipgloss.Color("#4f5967"), Gray400),
		Gray500:   lightDark(lipgloss.Color("#697384"), Gray500),
		Gray700:   lightDark(lipgloss.Color("#d9dee7"), Gray700),
		Error:     lightDark(lipgloss.Color("#b00020"), lipgloss.Color("#df1b41")),
		Text:      lightDark(lipgloss.Color("#1f2430"), lipgloss.Color("#ffffff")),
		OnBrand:   lipgloss.Color("#ffffff"),
	}
	p.Border = lipgloss.Blend1D(3, p.Gray500, p.Purple400)[1]
	p.Panel = lightDark(lipgloss.Darken(p.Gray700, 0.04), lipgloss.Lighten(p.Gray700, 0.08))
	p.Selection = lipgloss.Blend1D(4, p.Purple700, p.Purple500)[1]
	return p
}
