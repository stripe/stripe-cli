package tui

import (
	"charm.land/bubbles/v2/help"
	"charm.land/lipgloss/v2"

	"github.com/stripe/stripe-cli/pkg/coop/colors"
)

func newThemedHelp(t Theme) help.Model {
	h := help.New()
	h.Styles.ShortKey = lipgloss.NewStyle().Foreground(t.Purple400).Bold(true)
	h.Styles.ShortDesc = lipgloss.NewStyle().Foreground(t.Gray300)
	h.Styles.ShortSeparator = lipgloss.NewStyle().Foreground(t.Gray500)
	h.Styles.Ellipsis = lipgloss.NewStyle().Foreground(t.Gray400)
	h.Styles.FullKey = h.Styles.ShortKey
	h.Styles.FullDesc = h.Styles.ShortDesc
	h.Styles.FullSeparator = h.Styles.ShortSeparator
	return h
}

type Theme struct {
	IsDark bool
	colors.Palette

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
		IsDark:  isDark,
		Palette: palette,
	}
	t.BrandStyle = lipgloss.NewStyle().Foreground(t.Purple500)
	t.SuccessStyle = lipgloss.NewStyle().Foreground(t.Green400)
	t.AttentionStyle = lipgloss.NewStyle().Foreground(t.Orange400)
	t.ReviewStyle = lipgloss.NewStyle().Foreground(t.Purple400)
	t.MutedStyle = lipgloss.NewStyle().Foreground(t.Gray400)
	t.DimmedStyle = lipgloss.NewStyle().Foreground(t.Gray500).Italic(true)
	t.ErrorStyle = lipgloss.NewStyle().Foreground(t.Error)
	t.HeaderStyle = lipgloss.NewStyle().
		Bold(true).
		Foreground(t.OnBrand).
		Background(t.Purple700).
		Padding(0, 1)
	t.StepTitleStyle = lipgloss.NewStyle().
		Bold(true).
		Foreground(t.Gray300)
	t.StepRuleStyle = lipgloss.NewStyle().
		Foreground(t.Gray700)
	t.DetailBoxStyle = lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(t.Border).
		Padding(0, 1)
	t.ReviewCardStyle = lipgloss.NewStyle().
		Border(lipgloss.NormalBorder()).
		BorderForeground(t.Selection).
		Padding(0, 1)
	t.ConfirmationHeaderStyle = lipgloss.NewStyle().
		Foreground(t.Text).
		Background(t.Selection).
		Bold(true).
		Padding(0, 1)
	t.FooterStyle = lipgloss.NewStyle().
		Foreground(t.Gray300)
	t.FileAnnotationStyle = lipgloss.NewStyle().
		Foreground(t.Gray500).
		Italic(true)
	return t
}
