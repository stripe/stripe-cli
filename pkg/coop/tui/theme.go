package tui

import (
	"image/color"

	"charm.land/bubbles/v2/help"
	"charm.land/lipgloss/v2"

	"github.com/stripe/stripe-cli/pkg/coop/colors"
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
	HuePurple500 = colors.Purple500
	HuePurple400 = colors.Purple400
	HuePurple700 = colors.Purple700

	HueGreen300 = colors.Green300
	HueGreen400 = colors.Green400

	HueBlue400 = colors.Blue400
	HueBlue700 = colors.Blue700

	HueOrange400 = colors.Orange400

	HueGray300 = colors.Gray300
	HueGray400 = colors.Gray400
	HueGray500 = colors.Gray500
	HueGray700 = colors.Gray700
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
	palette := colors.NewPalette(isDark)
	t := Theme{
		IsDark:       isDark,
		HuePurple500: palette.Purple500,
		HuePurple400: palette.Purple400,
		HuePurple700: palette.Purple700,
		HueGreen300:  palette.Green300,
		HueGreen400:  palette.Green400,
		HueBlue400:   palette.Blue400,
		HueBlue700:   palette.Blue700,
		HueOrange400: palette.Orange400,
		HueGray300:   palette.Gray300,
		HueGray400:   palette.Gray400,
		HueGray500:   palette.Gray500,
		HueGray700:   palette.Gray700,
		HueError:     palette.Error,
		HueText:      palette.Text,
		HueOnBrand:   palette.OnBrand,
		HueBorder:    palette.Border,
		HuePanel:     palette.Panel,
		HueSelection: palette.Selection,
	}
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
