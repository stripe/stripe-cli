package tui

import (
	"image/color"

	"charm.land/bubbles/v2/help"
	"charm.land/lipgloss/v2"
)

func newThemedHelp(t Theme) help.Model {
	h := help.New()
	h.Styles.ShortKey = lipgloss.NewStyle().Foreground(t.HuePurple400).Bold(true)
	h.Styles.ShortDesc = lipgloss.NewStyle().Foreground(t.HueGray300)
	h.Styles.ShortSeparator = lipgloss.NewStyle().Foreground(t.HueGray500)
	h.Styles.Ellipsis = lipgloss.NewStyle().Foreground(t.HueGray400)
	h.Styles.FullKey = h.Styles.ShortKey
	h.Styles.FullDesc = h.Styles.ShortDesc
	h.Styles.FullSeparator = h.Styles.ShortSeparator
	return h
}

// Sail Design System hue palette (from tokensColor.ts)
var (
	HuePurple500 = lipgloss.Color("#625afa")
	HuePurple400 = lipgloss.Color("#8d7ffa")
	HuePurple700 = lipgloss.Color("#3f32a1")

	HueGreen300 = lipgloss.Color("#48c404")
	HueGreen400 = lipgloss.Color("#3fa40d")

	HueBlue400 = lipgloss.Color("#00a3ff")
	HueBlue700 = lipgloss.Color("#0b3a5b")

	HueOrange400 = lipgloss.Color("#ed6704")

	HueGray300 = lipgloss.Color("#a3acba")
	HueGray400 = lipgloss.Color("#87909f")
	HueGray500 = lipgloss.Color("#687385")
	HueGray700 = lipgloss.Color("#414552")
)

type Theme struct {
	IsDark bool

	HuePurple500 color.Color
	HuePurple400 color.Color
	HuePurple700 color.Color
	HueGreen300  color.Color
	HueGreen400  color.Color
	HueBlue400   color.Color
	HueBlue700   color.Color
	HueOrange400 color.Color
	HueGray300   color.Color
	HueGray400   color.Color
	HueGray500   color.Color
	HueGray700   color.Color
	HueError     color.Color
	HueText      color.Color
	HueOnBrand   color.Color
	HueBorder    color.Color
	HuePanel     color.Color
	HueSelection color.Color

	BrandStyle              lipgloss.Style
	SuccessStyle            lipgloss.Style
	AttentionStyle          lipgloss.Style
	ReviewStyle             lipgloss.Style
	MutedStyle              lipgloss.Style
	DimmedStyle             lipgloss.Style
	ErrorStyle              lipgloss.Style
	HeaderStyle             lipgloss.Style
	StepTitleStyle          lipgloss.Style
	StepRuleStyle           lipgloss.Style
	DetailBoxStyle          lipgloss.Style
	ReviewCardStyle         lipgloss.Style
	ConfirmationHeaderStyle lipgloss.Style
	FooterStyle             lipgloss.Style
	FileAnnotationStyle     lipgloss.Style
}

func NewTheme(isDark bool) Theme {
	lightDark := lipgloss.LightDark(isDark)
	t := Theme{
		IsDark:       isDark,
		HuePurple500: lightDark(lipgloss.Color("#4f46d8"), HuePurple500),
		HuePurple400: lightDark(lipgloss.Color("#5f52e8"), HuePurple400),
		HuePurple700: lightDark(lipgloss.Color("#3f32a1"), HuePurple700),
		HueGreen300:  lightDark(lipgloss.Color("#237500"), HueGreen300),
		HueGreen400:  lightDark(lipgloss.Color("#2f8506"), HueGreen400),
		HueBlue400:   lightDark(lipgloss.Color("#006bb6"), HueBlue400),
		HueBlue700:   lightDark(lipgloss.Color("#dff2ff"), HueBlue700),
		HueOrange400: lightDark(lipgloss.Color("#b34800"), HueOrange400),
		HueGray300:   lightDark(lipgloss.Color("#2f3640"), HueGray300),
		HueGray400:   lightDark(lipgloss.Color("#4f5967"), HueGray400),
		HueGray500:   lightDark(lipgloss.Color("#697384"), HueGray500),
		HueGray700:   lightDark(lipgloss.Color("#d9dee7"), HueGray700),
		HueError:     lightDark(lipgloss.Color("#b00020"), lipgloss.Color("#df1b41")),
		HueText:      lightDark(lipgloss.Color("#1f2430"), lipgloss.Color("#ffffff")),
		HueOnBrand:   lipgloss.Color("#ffffff"),
	}
	t.HueBorder = lipgloss.Blend1D(3, t.HueGray500, t.HuePurple400)[1]
	t.HuePanel = lightDark(lipgloss.Darken(t.HueGray700, 0.04), lipgloss.Lighten(t.HueGray700, 0.08))
	t.HueSelection = lipgloss.Blend1D(4, t.HuePurple700, t.HuePurple500)[1]
	t.BrandStyle = lipgloss.NewStyle().Foreground(t.HuePurple500)
	t.SuccessStyle = lipgloss.NewStyle().Foreground(t.HueGreen400)
	t.AttentionStyle = lipgloss.NewStyle().Foreground(t.HueOrange400)
	t.ReviewStyle = lipgloss.NewStyle().Foreground(t.HuePurple400)
	t.MutedStyle = lipgloss.NewStyle().Foreground(t.HueGray400)
	t.DimmedStyle = lipgloss.NewStyle().Foreground(t.HueGray500).Italic(true)
	t.ErrorStyle = lipgloss.NewStyle().Foreground(t.HueError)
	t.HeaderStyle = lipgloss.NewStyle().
		Bold(true).
		Foreground(t.HueOnBrand).
		Background(t.HuePurple700).
		Padding(0, 1)
	t.StepTitleStyle = lipgloss.NewStyle().
		Bold(true).
		Foreground(t.HueGray300)
	t.StepRuleStyle = lipgloss.NewStyle().
		Foreground(t.HueGray700)
	t.DetailBoxStyle = lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(t.HueBorder).
		Padding(0, 1)
	t.ReviewCardStyle = lipgloss.NewStyle().
		Border(lipgloss.NormalBorder()).
		BorderForeground(t.HueSelection).
		Padding(0, 1)
	t.ConfirmationHeaderStyle = lipgloss.NewStyle().
		Foreground(t.HueText).
		Background(t.HueSelection).
		Bold(true).
		Padding(0, 1)
	t.FooterStyle = lipgloss.NewStyle().
		Foreground(t.HueGray300)
	t.FileAnnotationStyle = lipgloss.NewStyle().
		Foreground(t.HueGray500).
		Italic(true)
	return t
}

// Semantic styles
var (
	BrandStyle     = lipgloss.NewStyle().Foreground(HuePurple500)
	SuccessStyle   = lipgloss.NewStyle().Foreground(HueGreen400)
	AttentionStyle = lipgloss.NewStyle().Foreground(HueOrange400)
	ReviewStyle    = lipgloss.NewStyle().Foreground(HuePurple400)
	MutedStyle     = lipgloss.NewStyle().Foreground(HueGray400)
	DimmedStyle    = lipgloss.NewStyle().Foreground(HueGray500).Italic(true)
	ErrorStyle     = lipgloss.NewStyle().Foreground(lipgloss.Color("#df1b41"))

	HeaderStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#ffffff")).
			Background(HuePurple700).
			Padding(0, 1)

	StepTitleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(HueGray300)

	StepRuleStyle = lipgloss.NewStyle().
			Foreground(HueGray700)

	DetailBoxStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(HueGray700).
			Padding(0, 1)

	ReviewCardStyle = lipgloss.NewStyle().
			Border(lipgloss.NormalBorder()).
			BorderForeground(HuePurple400).
			Padding(0, 1)

	ConfirmationHeaderStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#ffffff")).
				Background(HuePurple700).
				Bold(true).
				Padding(0, 1)

	FooterStyle = lipgloss.NewStyle().
			Foreground(HueGray300)

	FileAnnotationStyle = lipgloss.NewStyle().
				Foreground(HueGray500).
				Italic(true)
)
