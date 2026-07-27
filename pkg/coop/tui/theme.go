package tui

import (
	"image/color"

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

	// Section headings inside the review box and the detail pane.
	ActionHeadingStyle     lipgloss.Style
	TaskHeadingStyle       lipgloss.Style
	EvidenceStyle          lipgloss.Style
	IdentifierStyle        lipgloss.Style
	StatusCardStyle        lipgloss.Style
	SelectedStepTitleStyle lipgloss.Style
	TabActiveStyle         lipgloss.Style
	TabInactiveStyle       lipgloss.Style
	TabGapStyle            lipgloss.Style
	SoftSuccessStyle       lipgloss.Style
	SoftErrorStyle         lipgloss.Style
	SoftAttentionStyle     lipgloss.Style
	PromptStyle            lipgloss.Style
	InstructionStyle       lipgloss.Style
	KeyStyle               lipgloss.Style
	KeyDescriptionStyle    lipgloss.Style
	FooterStyle            lipgloss.Style
	FileAnnotationStyle    lipgloss.Style
}

func NewTheme(isDark bool) Theme {
	palette := colors.NewPalette(isDark)
	t := Theme{
		IsDark:  isDark,
		Palette: palette,
	}
	t.BrandStyle = lipgloss.NewStyle().Foreground(t.Purple500)
	t.SuccessStyle = lipgloss.NewStyle().Foreground(t.Green400)
	// Bold as well as colored: this is the only style that means "you are
	// blocking progress", and it has to stay legible when color is stripped.
	t.AttentionStyle = lipgloss.NewStyle().Foreground(t.Orange400).Bold(true)
	t.ReviewStyle = lipgloss.NewStyle().Foreground(t.Purple400).Bold(true)
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
	// The step the cursor is on, filled so it separates from the steps around
	// it without depending on a two-character marker.
	t.SelectedStepTitleStyle = lipgloss.NewStyle().
		Foreground(t.Text).
		Background(t.Selection).
		Bold(true).
		Padding(0, 1)
	// The small card under a step header. Quieter than the review card: it
	// carries a single line of status, so a heavy border would outweigh it.
	t.StatusCardStyle = lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(t.Gray700).
		Padding(0, 1)
	// Tabs read as tabs when they have a filled body, not when they are words
	// separated by dots. The active one is a solid block; the others sit on a
	// dim ground so the strip still reads as a row of controls.
	t.TabActiveStyle = lipgloss.NewStyle().
		Foreground(t.Text).
		Background(t.Selection).
		Bold(true).
		Padding(0, 1)
	t.TabInactiveStyle = lipgloss.NewStyle().
		Foreground(t.Gray400).
		Background(t.Panel).
		Padding(0, 1)
	t.TabGapStyle = lipgloss.NewStyle().Foreground(t.Gray700)
	t.ConfirmationHeaderStyle = lipgloss.NewStyle().
		Foreground(t.Text).
		Background(t.Selection).
		Bold(true).
		Padding(0, 1)
	// Accents at full strength read as neon against a dark panel, which is both
	// distracting and off-brand. Blending each toward the muted text color keeps
	// the hue identifiable while dropping its saturation, so the box reads as
	// tinted rather than lit up.
	soften := func(c color.Color) color.Color {
		return lipgloss.Blend1D(5, t.Gray400, c)[2]
	}
	// A failure keeps more of its hue: muting it to the same degree as the rest
	// made it read as decoration rather than as something that went wrong.
	softenLess := func(c color.Color) color.Color {
		return lipgloss.Blend1D(5, t.Gray400, c)[3]
	}

	// The instruction is the one thing the reader must act on, so it is the
	// brightest text rather than a fourth hue competing with the others.
	// Headings stay the prominent element. The instruction is lifted by hue
	// instead of weight: full-strength text against the grays everything else
	// uses, so it is the brightest *content* without out-weighting its label.
	// Underline gives the heading a second channel, so it stays distinct from
	// the other bold headings once color is stripped.
	t.ActionHeadingStyle = lipgloss.NewStyle().Foreground(t.Text).Bold(true)
	// Event names and env vars are the subject of a check; they carry the
	// review purple so they can be picked out of the sentence around them.
	t.IdentifierStyle = lipgloss.NewStyle().Foreground(t.Purple400)
	t.InstructionStyle = lipgloss.NewStyle().Foreground(t.Text)
	// Blue for structure: purple already marks the cursor and review state, and
	// orange is reserved for "you are blocking progress".
	t.TaskHeadingStyle = lipgloss.NewStyle().Foreground(soften(t.Blue400)).Bold(true)
	// Agent notes were Gray500 and italic — the dimmest value in the palette,
	// with a face that reduces legibility further. One step brighter, upright.
	t.EvidenceStyle = lipgloss.NewStyle().Foreground(t.Gray400)
	t.SoftSuccessStyle = lipgloss.NewStyle().Foreground(soften(t.Green400)).Bold(true)
	t.SoftErrorStyle = lipgloss.NewStyle().Foreground(softenLess(t.Error)).Bold(true)
	t.SoftAttentionStyle = lipgloss.NewStyle().Foreground(soften(t.Orange400)).Bold(true)
	// "Waiting for you" is a prompt, not a fault. Orange reads as a warning, so
	// the line is plain emphasis and only its keys carry color.
	t.PromptStyle = lipgloss.NewStyle().Foreground(t.Text).Bold(true)
	// "Waiting for you" is a prompt, not a fault. Orange reads as a warning, so
	// the line is plain emphasis and only the keys carry color.
	t.PromptStyle = lipgloss.NewStyle().Foreground(t.Text).Bold(true)

	// Matches the footer's help component, so a key looks like a key anywhere.
	t.KeyStyle = lipgloss.NewStyle().Foreground(t.Purple400).Bold(true)
	t.KeyDescriptionStyle = lipgloss.NewStyle().Foreground(t.Gray300)

	t.FooterStyle = lipgloss.NewStyle().
		Foreground(t.Gray300)
	t.FileAnnotationStyle = lipgloss.NewStyle().
		Foreground(t.Gray500).
		Italic(true)
	return t
}
