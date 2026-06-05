package tui

import (
	"github.com/charmbracelet/huh"
	"github.com/charmbracelet/lipgloss"
)

// HuhTheme returns a Sail-styled huh theme for interactive prompts.
func HuhTheme() *huh.Theme {
	t := huh.ThemeBase()

	t.Focused.Base = t.Focused.Base.BorderForeground(HueGray700)
	t.Focused.Title = t.Focused.Title.Foreground(HuePurple500).Bold(true)
	t.Focused.Description = t.Focused.Description.Foreground(HueGray400)
	t.Focused.SelectSelector = t.Focused.SelectSelector.Foreground(HuePurple500)
	t.Focused.NextIndicator = t.Focused.NextIndicator.Foreground(HuePurple400)
	t.Focused.PrevIndicator = t.Focused.PrevIndicator.Foreground(HuePurple400)
	t.Focused.Option = t.Focused.Option.Foreground(HueGray300)
	t.Focused.SelectedOption = t.Focused.SelectedOption.Foreground(HueGreen300)
	t.Focused.SelectedPrefix = lipgloss.NewStyle().Foreground(HueGreen400).SetString("▶ ")
	t.Focused.UnselectedPrefix = lipgloss.NewStyle().Foreground(HueGray500).SetString("  ")
	t.Focused.UnselectedOption = t.Focused.UnselectedOption.Foreground(HueGray400)
	t.Focused.FocusedButton = t.Focused.FocusedButton.Foreground(lipgloss.Color("#ffffff")).Background(HuePurple500)
	t.Focused.BlurredButton = t.Focused.BlurredButton.Foreground(HueGray300).Background(HueGray700)
	t.Blurred = t.Focused

	return t
}

// Sail Design System hue palette (from tokensColor.ts)
var (
	HuePurple500 = lipgloss.Color("#625afa")
	HuePurple400 = lipgloss.Color("#8d7ffa")
	HuePurple700 = lipgloss.Color("#3f32a1")

	HueGreen300 = lipgloss.Color("#48c404")
	HueGreen400 = lipgloss.Color("#3fa40d")

	HueOrange400 = lipgloss.Color("#ed6704")

	HueGray300 = lipgloss.Color("#a3acba")
	HueGray400 = lipgloss.Color("#87909f")
	HueGray500 = lipgloss.Color("#687385")
	HueGray700 = lipgloss.Color("#414552")
)

// Semantic styles
var (
	BrandStyle     = lipgloss.NewStyle().Foreground(HuePurple500)
	SuccessStyle   = lipgloss.NewStyle().Foreground(HueGreen400)
	AttentionStyle = lipgloss.NewStyle().Foreground(HueOrange400)
	MutedStyle     = lipgloss.NewStyle().Foreground(HueGray400)
	DimmedStyle    = lipgloss.NewStyle().Foreground(HueGray500).Italic(true)
	ErrorStyle     = lipgloss.NewStyle().Foreground(lipgloss.Color("#df1b41"))

	HeaderStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#ffffff")).
			Background(HuePurple700).
			Padding(0, 1)

	ChapterTitleStyle = lipgloss.NewStyle().
				Bold(true).
				Foreground(HueGray300)

	ChapterRuleStyle = lipgloss.NewStyle().
				Foreground(HueGray700)

	DetailBoxStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(HueGray700).
			Padding(0, 1).
			MarginTop(1)

	FooterStyle = lipgloss.NewStyle().
			Foreground(HueGray500)

	FileAnnotationStyle = lipgloss.NewStyle().
				Foreground(HueGray500).
				Italic(true)
)
