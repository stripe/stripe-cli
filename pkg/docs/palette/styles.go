package palette

import "charm.land/lipgloss/v2"

// Styles holds the lipgloss styles the palette uses to render itself.
type Styles struct {
	// Container wraps the whole palette. The default is a rounded
	// border with no padding — padding is applied manually as a
	// per-line indent so selection backgrounds can fill the full row.
	Container lipgloss.Style
	// Title styles the optional section header at the top of the
	// palette (see WithTitle).
	Title lipgloss.Style
	// Indent is the per-line left margin inside the container, as a
	// literal string. Two spaces by default.
	Indent string
	// Prompt styles the leading mode-prompt glyph rendered in front of
	// the input. Not applied while the spinner is in flight (the
	// spinner owns its own styling).
	Prompt lipgloss.Style
	// Placeholder styles the textinput's placeholder text shown while
	// the input is empty. Propagated into the underlying textinput's
	// Focused / Blurred placeholder styles.
	Placeholder lipgloss.Style
	// EmptyMessage styles the no-results message rendered in place of
	// the item list when the active mode returns no candidates for the
	// current query. See Mode.EmptyMessage and WithEmptyMessage.
	EmptyMessage lipgloss.Style
	// SpinnerLabel styles the text next to the spinner glyph while a
	// search is in flight.
	SpinnerLabel lipgloss.Style
	// FacetHeader styles the facet-completion hint shown in the footer
	// row while the palette is completing a facet token.
	FacetHeader lipgloss.Style
	// Footer wraps the paginator row at the bottom.
	Footer lipgloss.Style
}

// DefaultStyles returns sensible defaults. Override fields individually
// or pass a whole struct via WithStyles.
func DefaultStyles() Styles {
	return Styles{
		Container:    lipgloss.NewStyle().Border(lipgloss.RoundedBorder()),
		Title:        lipgloss.NewStyle().Bold(true),
		Indent:       "  ",
		Prompt:       lipgloss.NewStyle(),
		Placeholder:  lipgloss.NewStyle().Faint(true),
		EmptyMessage: lipgloss.NewStyle().Faint(true),
		SpinnerLabel: lipgloss.NewStyle().Faint(true),
		FacetHeader:  lipgloss.NewStyle().Faint(true),
		Footer:       lipgloss.NewStyle().Faint(true),
	}
}
