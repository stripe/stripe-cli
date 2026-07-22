package palette

import (
	"charm.land/bubbles/v2/paginator"
	tea "charm.land/bubbletea/v2"
)

// Option configures a palette Model. Apply with New(...Option).
type Option func(*Model)

// WithModes replaces the default empty mode with the supplied modes,
// in priority order — the first whose Match returns true wins. Make
// the last entry a fallback (Match: nil) so some mode always applies.
func WithModes(modes ...Mode) Option {
	return func(m *Model) { m.modes = modes }
}

// WithTitle sets the optional section header rendered above the input.
// Pass an empty string (the default) for no title row.
func WithTitle(s string) Option {
	return func(m *Model) { m.title = s }
}

// WithPlaceholder sets the default hint text shown in the input while
// it's empty. Modes can override this for their own context via
// Mode.Placeholder.
func WithPlaceholder(s string) Option {
	return func(m *Model) { m.placeholder = s }
}

// WithEmptyMessage sets the default text shown in place of the item
// list when the user has typed a query but the active mode returns
// no candidates. Modes can override this for their own context via
// Mode.EmptyMessage. The message is suppressed while a search is in
// flight (the spinner is the loading indicator) and while a facet
// completion is active.
func WithEmptyMessage(s string) Option {
	return func(m *Model) { m.emptyMessage = s }
}

// WithHelp toggles the short-help row at the bottom of the palette.
// On by default.
func WithHelp(show bool) Option {
	return func(m *Model) { m.showHelp = show }
}

// WithDelegate overrides the ItemDelegate used to render items.
func WithDelegate(d ItemDelegate) Option {
	return func(m *Model) { m.delegate = d }
}

// WithOnExecute registers a callback fired synchronously inside the
// palette's Update when the user presses Execute (Enter) on a
// highlighted item. The returned tea.Cmd is batched alongside the
// SelectedMsg dispatch and any Command.Run() cmd, so hosts can:
//
//   - read the selected Item without pattern-matching on SelectedMsg;
//   - return a "close overlay" / "dispatch action" cmd in the same
//     tick the keypress was received;
//   - keep Command.Run chaining cleanly (the hook is additive).
//
// Return nil to opt out of adding a cmd for a particular item.
func WithOnExecute(fn func(Item) tea.Cmd) Option {
	return func(m *Model) { m.onExecute = fn }
}

// WithKeyMap overrides the default keybindings.
func WithKeyMap(km KeyMap) Option {
	return func(m *Model) { m.KeyMap = km }
}

// WithStyles overrides the default visual styles.
func WithStyles(s Styles) Option {
	return func(m *Model) { m.Styles = s }
}

// WithPageSize sets a fixed number of items per page. Pass 0 to
// auto-fit to the available terminal height (the default).
func WithPageSize(n int) Option {
	return func(m *Model) {
		m.pageSize = n
		if n > 0 {
			m.paginator.PerPage = n
		}
	}
}

// WithPaginatorType selects between dot indicators and "1/N" numeric.
func WithPaginatorType(t paginator.Type) Option {
	return func(m *Model) { m.paginator.Type = t }
}
