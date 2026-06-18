// Package palette provides a command-palette bubble for Bubble Tea
// programs. Hosts compose one or more Modes: each Mode owns its own
// Match (which inputs it claims), Items (the candidate list), and
// optionally an async Search dispatcher and typeable Facets. With no
// WithModes, the palette uses a single empty catch-all mode; the
// palette renders cleanly but shows no items until hosts wire one up.
package palette

import (
	"context"
	"strings"
	"time"

	"charm.land/bubbles/v2/help"
	"charm.land/bubbles/v2/key"
	"charm.land/bubbles/v2/paginator"
	"charm.land/bubbles/v2/spinner"
	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/sahilm/fuzzy"
)

// Mode describes how the palette interprets the current input. The
// active mode is the first one in the configured list whose Match
// returns true; a nil Match matches anything, so it's typically used
// as the fallback (last entry).
type Mode struct {
	// Name identifies the mode for logging and host status displays.
	// Also used as the cache key for Search results — see
	// Model.Results.
	Name string

	// Prompt is the glyph rendered before the input field. Should be
	// the same display width as the configured spinner so the input
	// text doesn't shift when the spinner swaps in during search. An
	// empty Prompt falls back to defaultPrompt ("⣿ ").
	Prompt string

	// Placeholder is the hint text shown in the input while it's empty
	// and this mode is active. Empty falls back to the palette-level
	// placeholder set via WithPlaceholder.
	Placeholder string

	// EmptyMessage is shown in place of the item list when this mode
	// is active, the user has typed a query, and Items returns no
	// candidates. Empty falls back to the palette-level message set
	// via WithEmptyMessage; when both are empty no message is rendered.
	EmptyMessage string

	// Debounce is how long the palette waits after the input stops
	// changing before invoking Search. Zero means dispatch on the
	// next tick. Ignored when Search is nil.
	Debounce time.Duration

	// Match reports whether this mode applies to the given raw input.
	// A nil Match matches anything.
	Match func(input string) bool

	// Query extracts the meaningful query string from the raw input —
	// typically by stripping a leading prefix. A nil Query returns
	// the input unchanged.
	Query func(input string) string

	// Items returns the candidate items for this mode given the
	// palette state and the extracted query. For sync modes it does
	// the filtering inline; for async modes it typically reads from
	// the palette's Results cache that Search populates. A nil Items
	// returns nil.
	Items func(m Model, query string) []Item

	// Search is the async dispatcher. When the input changes inside
	// this mode and the debounce window elapses, the palette calls
	// Search and the returned tea.Cmd must eventually yield a
	// SearchResultMsg with the matching Mode name. The ctx is
	// cancelled when a newer search supersedes this one or when the
	// active mode changes — implementations should pass it through
	// to their HTTP/DB call. Nil means the mode is purely synchronous.
	Search func(ctx context.Context, query string) tea.Cmd

	// Facets registers typeable "<Name>:<value>" filters for this
	// mode. When the cursor sits inside such a token, the palette
	// swaps the item list for a value picker scoped to the matching
	// Facet. See palette.ParseFacets to apply parsed filters from a
	// Mode's Items/Search closure.
	Facets []Facet
}

// defaultPrompt is the fallback prompt glyph for modes that don't set
// their own. A static full braille block — same character family as
// the Dot spinner so idle and loading read as one shape morphing
// rather than two unrelated glyphs. Two cells wide to match the
// spinner.
const defaultPrompt = "⣿ "

// emptyMode is the implicit default when no Modes are configured. It
// claims any input but produces no items — the palette renders, but
// is otherwise inert until hosts pass their own Mode list via
// WithModes.
var emptyMode = Mode{
	Name:   "default",
	Prompt: defaultPrompt,
}

// debounceMsg is an internal tick that fires after a mode's Debounce
// window. The palette dispatches the mode's Search closure only when
// the generation hasn't moved on (no newer keystroke).
type debounceMsg struct {
	mode string
	gen  int
}

// Model is the palette bubble.
type Model struct {
	input     textinput.Model
	spinner   spinner.Model
	paginator paginator.Model
	help      help.Model

	modes     []Mode
	results   map[string][]Item // per-mode cache: keyed by Mode.Name
	delegate  ItemDelegate
	onExecute func(Item) tea.Cmd // optional synchronous Enter hook; see WithOnExecute

	// search machinery
	searchGen    int                // increments on each input change for stale-tick rejection
	searchCancel context.CancelFunc // cancels the in-flight Search context

	// facet sub-state
	facet        *facetCompletion
	facetResults map[string][]Item  // per-facet cache for async Resolve
	facetGen     int                // increments on each partial change
	facetCancel  context.CancelFunc // cancels the in-flight Resolve context

	title        string
	placeholder  string
	emptyMessage string
	cursor       int
	pageSize     int
	pending      bool // debounce scheduled but Search not yet dispatched
	loading      bool // Search dispatched, awaiting SearchResultMsg
	width        int
	height       int
	showHelp     bool

	// render-context fields populated by View before each call to
	// ItemDelegate.Render. renderRow is the visible-row index of the
	// selected item in the current page (or < 0 when no selection
	// hits this page / we're outside Render). renderWidth is the
	// available width for the row content. Both are unexported; the
	// delegate reads them via IsSelected() and Width().
	renderRow   int
	renderWidth int

	KeyMap KeyMap
	Styles Styles
}

// New constructs a palette Model with sensible defaults. Apply Options
// to override.
func New(opts ...Option) Model {
	ti := textinput.New()
	ti.Prompt = ""

	sp := spinner.New()
	sp.Spinner = spinner.Dot
	// Tighter than the upstream 100 ms default so the spinner reads
	// as smooth motion rather than a single-frame flicker on short
	// searches.
	sp.Spinner.FPS = 60 * time.Millisecond

	pg := paginator.New()
	pg.Type = paginator.Dots
	pg.ActiveDot = "● "
	pg.InactiveDot = "○ "

	m := Model{
		input:        ti,
		spinner:      sp,
		paginator:    pg,
		help:         help.New(),
		modes:        []Mode{emptyMode},
		delegate:     NewDefaultDelegate(),
		results:      map[string][]Item{},
		facetResults: map[string][]Item{},
		renderRow:    -1, // sentinel: IsSelected returns false outside Render
		showHelp:     true,
		KeyMap:       DefaultKeyMap(),
		Styles:       DefaultStyles(),
	}
	for _, o := range opts {
		o(&m)
	}
	return m
}

// Init is part of the tea.Model contract. The palette emits no startup
// command — callers compose it into their own program's Init.
func (m Model) Init() tea.Cmd { return nil }

// Update handles cursor navigation, the async search lifecycle
// (debounce → dispatch → result), spinner ticks, and forwards
// remaining messages to the textinput.
func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height

	case debounceMsg:
		return m.handleDebounce(msg)

	case SearchResultMsg:
		return m.handleSearchResult(msg)

	case facetDebounceMsg:
		return m.handleFacetDebounce(msg)

	case FacetResultMsg:
		return m.handleFacetResult(msg)

	case spinner.TickMsg:
		if !m.loading && !m.pending {
			return m, nil
		}
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd

	case tea.MouseWheelMsg:
		// Wheel scrolls the item cursor one row at a time. Routes
		// through moveCursor so facet-completion navigation is handled
		// the same way as Up/Down. The host has to enable mouse capture
		// (e.g. tea.View.MouseMode = tea.MouseModeCellMotion) for these
		// to arrive.
		switch msg.Button {
		case tea.MouseWheelUp:
			m.moveCursor(-1)
		case tea.MouseWheelDown:
			m.moveCursor(1)
		}
		return m, nil

	case tea.KeyPressMsg:
		// Navigation and Execute keys are consumed by the palette and
		// NOT forwarded to the textinput.
		switch {
		case key.Matches(msg, m.KeyMap.Cancel) && m.facet != nil:
			m.clearFacet()
			m.pending = false
			m.loading = false
			return m, nil
		case key.Matches(msg, m.KeyMap.Down):
			m.moveCursor(1)
			return m, nil
		case key.Matches(msg, m.KeyMap.Up):
			m.moveCursor(-1)
			return m, nil
		case key.Matches(msg, m.KeyMap.NextPage):
			m.pageBy(1)
			return m, nil
		case key.Matches(msg, m.KeyMap.PrevPage):
			m.pageBy(-1)
			return m, nil
		case key.Matches(msg, m.KeyMap.Execute):
			if m.facet != nil {
				return m.applyFacetCompletion()
			}
			return m, m.execute()
		}
	}

	prev := m.input.Value()
	var cmd tea.Cmd
	m.input, cmd = m.input.Update(msg)
	if m.input.Value() != prev {
		m.cursor = 0
		if reconcileCmd := m.reconcileInputState(); reconcileCmd != nil {
			return m, tea.Batch(cmd, reconcileCmd)
		}
	}
	return m, cmd
}

// reconcileInputState re-evaluates facet completion against the cursor
// token, then schedules a mode Search (or cancels it when completion
// is now active). Call after any input change.
func (m *Model) reconcileInputState() tea.Cmd {
	var cmds []tea.Cmd
	if c := m.evaluateFacet(); c != nil {
		cmds = append(cmds, c)
	}
	if m.facet == nil {
		if c := m.scheduleSearch(); c != nil {
			cmds = append(cmds, c)
		}
	} else if m.searchCancel != nil {
		// Facet completion took over — drop any in-flight mode search.
		m.searchCancel()
		m.searchCancel = nil
	}
	switch len(cmds) {
	case 0:
		return nil
	case 1:
		return cmds[0]
	default:
		return tea.Batch(cmds...)
	}
}

// scheduleSearch is called whenever the input value changes. It
// cancels any in-flight Search and, if the now-active mode has a
// Search closure, schedules a debounce tick that will dispatch it.
// The spinner runs from the moment the debounce is scheduled (pending)
// through Search dispatch (loading) so it reflects "input not yet
// reconciled with results" rather than just "Search in flight."
func (m *Model) scheduleSearch() tea.Cmd {
	// Cancel any in-flight search — we're either coalescing keystrokes
	// or switching away from the mode that started it.
	if m.searchCancel != nil {
		m.searchCancel()
		m.searchCancel = nil
	}
	spinnerActive := m.loading || m.pending
	m.loading = false

	mode := m.Mode()
	if mode.Search == nil {
		m.pending = false
		return nil
	}
	m.searchGen++
	gen := m.searchGen
	name := mode.Name
	d := mode.Debounce
	tickCmd := tea.Tick(d, func(_ time.Time) tea.Msg {
		return debounceMsg{mode: name, gen: gen}
	})
	m.pending = true
	if spinnerActive {
		// Spinner tick chain already running; don't reseed it or the
		// glyph advances twice per frame.
		return tickCmd
	}
	return tea.Batch(tickCmd, m.spinner.Tick)
}

// handleDebounce dispatches the active mode's Search closure when the
// debounce tick is still current (no newer keystroke superseded it
// and the user hasn't switched mode).
func (m Model) handleDebounce(msg debounceMsg) (Model, tea.Cmd) {
	if msg.gen != m.searchGen {
		return m, nil // stale: a newer keystroke is pending
	}
	mode := m.Mode()
	if mode.Name != msg.mode || mode.Search == nil {
		return m, nil // mode switched out from under this tick
	}
	ctx, cancel := context.WithCancel(context.Background())
	m.searchCancel = cancel
	m.pending = false
	m.loading = true
	// Spinner is already ticking from scheduleSearch — no need to reseed.
	return m, mode.Search(ctx, m.Query())
}

// handleSearchResult stores result items in the per-mode cache and
// clears loading. Stale results (whose Mode or Query no longer
// matches the current state) are dropped.
func (m Model) handleSearchResult(msg SearchResultMsg) (Model, tea.Cmd) {
	mode := m.Mode()
	if msg.Mode != mode.Name || msg.Query != m.Query() {
		return m, nil // stale
	}
	if m.results == nil {
		m.results = map[string][]Item{}
	}
	m.results[msg.Mode] = msg.Results
	m.loading = false
	return m, nil
}

// ShortHelp returns the compact key list rendered by the help bubble
// at the bottom of the palette. Combines Up/Down into a single
// synthetic "↑↓ navigate" entry for legibility; the actual KeyMap
// bindings remain split since they're separate actions.
func (m Model) ShortHelp() []key.Binding {
	return []key.Binding{m.KeyMap.Navigate, m.KeyMap.Execute, m.KeyMap.Cancel}
}

// FullHelp returns the expanded key groups for help bubbles displaying
// the full layout (not used by the palette itself by default, but
// available for hosts that wire up "?"-toggled help).
func (m Model) FullHelp() [][]key.Binding {
	return [][]key.Binding{
		{m.KeyMap.Up, m.KeyMap.Down},
		{m.KeyMap.PrevPage, m.KeyMap.NextPage},
		{m.KeyMap.Execute, m.KeyMap.Cancel},
	}
}

// execute builds the tea.Cmd dispatched when the user presses Enter
// on a highlighted item. Always emits a SelectedMsg so the host can
// react (close the palette, log, etc.); when the item is a Command
// with a non-nil Run, batches Run()'s cmd alongside. Returns nil when
// no item is selected.
func (m Model) execute() tea.Cmd {
	sel := m.Selected()
	if sel == nil {
		return nil
	}
	cmds := []tea.Cmd{
		func() tea.Msg { return SelectedMsg{Item: sel} },
	}
	if c, ok := sel.(Command); ok && c.Run != nil {
		if runCmd := c.Run(); runCmd != nil {
			cmds = append(cmds, runCmd)
		}
	}
	// OnExecute fires synchronously here so hosts can react to the
	// selection inside the same Update tick — Command.Run still chains
	// in the batch alongside it.
	if m.onExecute != nil {
		if hookCmd := m.onExecute(sel); hookCmd != nil {
			cmds = append(cmds, hookCmd)
		}
	}
	if len(cmds) == 1 {
		return cmds[0]
	}
	return tea.Batch(cmds...)
}

// moveCursor shifts the selection by delta, wrapping at both ends. A
// no-op when there are no items. Operates on the facet-completion
// cursor when that sub-state is active.
func (m *Model) moveCursor(delta int) {
	n := len(m.Items())
	if n == 0 {
		return
	}
	if m.facet != nil {
		m.facet.cursor = ((m.facet.cursor+delta)%n + n) % n
		return
	}
	m.cursor = ((m.cursor+delta)%n + n) % n
}

// pageBy snaps the cursor to the start of the page delta away from
// the current one, wrapping at the first and last page. No-op when
// pagination is disabled (pageSize == 0) or there are no items.
func (m *Model) pageBy(delta int) {
	if m.pageSize <= 0 {
		return
	}
	n := len(m.Items())
	if n == 0 {
		return
	}
	totalPages := (n + m.pageSize - 1) / m.pageSize
	cursor := m.cursor
	if m.facet != nil {
		cursor = m.facet.cursor
	}
	currentPage := cursor / m.pageSize
	targetPage := ((currentPage+delta)%totalPages + totalPages) % totalPages
	newCursor := targetPage * m.pageSize
	if m.facet != nil {
		m.facet.cursor = newCursor
		return
	}
	m.cursor = newCursor
}

// InnerWidth is the usable width inside the Container border/padding.
// Returns 0 when the outer width has not yet been set.
func (m Model) InnerWidth() int {
	if m.width <= 0 {
		return 0
	}
	inner := m.width - m.Styles.Container.GetHorizontalFrameSize()
	if inner < 0 {
		return 0
	}
	return inner
}

// Cursor returns the absolute index of the highlighted item in the
// currently visible items list. Stable across page changes and safe
// to call from inside ItemDelegate.Render.
func (m Model) Cursor() int { return m.cursor }

// Width returns the column width available for rendering a row.
// During ItemDelegate.Render this is the inner row width (set by the
// palette for that render pass); elsewhere it falls back to
// InnerWidth so callers can ask "how wide is my content area?" with
// one method regardless of context.
func (m Model) Width() int {
	if m.renderWidth > 0 {
		return m.renderWidth
	}
	return m.InnerWidth()
}

// IsSelected reports whether the given visible-row index in the
// current page is the highlighted row. ItemDelegate.Render
// implementations should use this instead of comparing against a raw
// cursor field — it's correct on every page and zero outside a
// Render call (so it's safe to call defensively).
func (m Model) IsSelected(visibleIndex int) bool {
	return m.renderRow >= 0 && visibleIndex == m.renderRow
}

// View composes the palette layout: an optional title, the text
// input, and the visible items rendered through the configured
// ItemDelegate, wrapped in the Container style. Items are passed the
// inner width so the delegate's selection background can fill the row.
// The spinner row and paginator footer land in later milestones.
func (m Model) View() string {
	indent := m.Styles.Indent

	// Render against the facet-completion cursor when that sub-state
	// is active. Done here so the rest of View doesn't branch.
	if m.facet != nil {
		m.cursor = m.facet.cursor
	}

	var sections []string
	if m.title != "" {
		sections = append(sections, indent+m.Styles.Title.Render(m.title), "")
	}

	// Pick the leading glyph: the spinner while a search is pending,
	// otherwise the active mode's prompt (or the global default).
	// Styles.Prompt wraps the static glyph; the spinner owns its own
	// rendering so we don't restyle it.
	glyph := m.Mode().Prompt
	if glyph == "" {
		glyph = defaultPrompt
	}
	if m.loading || m.pending {
		glyph = m.spinner.View()
	} else {
		glyph = m.Styles.Prompt.Render(glyph)
	}

	// Per-mode placeholder takes precedence over the palette-level one.
	if ph := m.Mode().Placeholder; ph != "" {
		m.input.Placeholder = ph
	} else {
		m.input.Placeholder = m.placeholder
	}
	// Sync the placeholder style into the underlying textinput each
	// render so callers don't have to reach into m.input themselves.
	tiStyles := m.input.Styles()
	tiStyles.Focused.Placeholder = m.Styles.Placeholder
	tiStyles.Blurred.Placeholder = m.Styles.Placeholder
	m.input.SetStyles(tiStyles)

	// Size the textinput to the available row width so it doesn't
	// overflow the container.
	inner := m.InnerWidth()
	if inner > 0 {
		w := inner - lipgloss.Width(indent) - lipgloss.Width(glyph)
		if w < 1 {
			w = 1
		}
		m.input.SetWidth(w)
	}
	sections = append(sections, indent+glyph+m.input.View())

	items := m.Items()
	desiredRows := 0
	if m.pageSize > 0 {
		desiredRows = m.pageSize * m.delegate.Height()
	}

	// Slice items down to the current page when pagination is on.
	pageItems := items
	pageStart := 0
	totalPages := 1
	if m.pageSize > 0 && len(items) > 0 {
		totalPages = (len(items) + m.pageSize - 1) / m.pageSize
		currentPage := m.cursor / m.pageSize
		pageStart = currentPage * m.pageSize
		pageEnd := pageStart + m.pageSize
		if pageEnd > len(items) {
			pageEnd = len(items)
		}
		pageItems = items[pageStart:pageEnd]
	}

	// Resolve the no-results message for the current mode when the
	// item list is empty, the user has typed a query, and no other
	// surface owns the items section (loading spinner, facet picker).
	emptyMsg := ""
	if len(items) == 0 && m.input.Value() != "" && !m.loading && !m.pending && m.facet == nil {
		emptyMsg = m.Mode().EmptyMessage
		if emptyMsg == "" {
			emptyMsg = m.emptyMessage
		}
	}

	if len(pageItems) > 0 || desiredRows > 0 || emptyMsg != "" {
		// Populate the render-context fields the delegate reads via
		// IsSelected() / Width(). cursor and width on rowModel stay
		// unchanged so Model.Selected() / Cursor() / Width() return
		// the same values inside Render that hosts see outside.
		rowModel := m
		rowModel.renderWidth = inner
		rowModel.renderRow = m.cursor - pageStart

		var lines []string
		if len(pageItems) == 0 && emptyMsg != "" {
			lines = append(lines, indent+m.Styles.EmptyMessage.Render(emptyMsg))
		} else {
			for i, item := range pageItems {
				var buf strings.Builder
				m.delegate.Render(&buf, rowModel, i, item)
				lines = append(lines, strings.Split(buf.String(), "\n")...)
			}
		}

		// Pad or truncate to a stable height so the palette doesn't
		// jump between modes with different item counts.
		if desiredRows > 0 {
			for len(lines) < desiredRows {
				lines = append(lines, "")
			}
			if len(lines) > desiredRows {
				lines = lines[:desiredRows]
			}
		}

		sections = append(sections, "", strings.Join(lines, "\n"))
	}

	// Footer: paginator dots on the left, facet-completion hint on the
	// right of the same line. Reserve the slot whenever pagination is on
	// OR any configured mode declares facets, so the palette height
	// doesn't jump when facets activate or modes switch.
	hasFacetSlot := false
	for _, md := range m.modes {
		if len(md.Facets) > 0 {
			hasFacetSlot = true
			break
		}
	}
	if m.pageSize > 0 || hasFacetSlot {
		if m.pageSize > 0 {
			m.paginator.TotalPages = totalPages
			m.paginator.Page = m.cursor / m.pageSize
		}
		var parts []string
		if m.pageSize > 0 && totalPages > 1 {
			parts = append(parts, m.paginator.View())
		}
		if m.facet != nil {
			hint := m.facet.facet.Desc
			if hint == "" {
				hint = m.facet.facet.Name + ":"
			}
			parts = append(parts, m.Styles.FacetHeader.Render(hint))
		}
		footer := ""
		if len(parts) > 0 {
			footer = indent + strings.Join(parts, " • ")
		}
		sections = append(sections, "", footer)
	}

	if m.showHelp {
		helpWidth := inner - lipgloss.Width(indent)
		if helpWidth > 0 {
			m.help.SetWidth(helpWidth)
		}
		helpLine := m.help.View(m)
		if helpLine != "" {
			sections = append(sections, "", indent+helpLine)
		}
	}

	body := strings.Join(sections, "\n")

	// Pin the container's block width (lipgloss treats Width as the
	// outer block — border included — not the content width). Pass
	// m.width so the rendered output is exactly the terminal width
	// and lipgloss pads short body rows to the inner width without
	// truncating the delegate's already-styled padding.
	container := m.Styles.Container
	if m.width > 0 {
		container = container.Width(m.width)
	}
	return container.Render(body)
}

// Focus directs keyboard input to the palette.
func (m *Model) Focus() tea.Cmd { return m.input.Focus() }

// Blur removes keyboard focus from the palette.
func (m *Model) Blur() { m.input.Blur() }

// Mode returns the currently active Mode — the first in the
// configured list whose Match returns true (or whose Match is nil).
// Returns a zero Mode when no modes are configured.
func (m Model) Mode() Mode {
	input := m.input.Value()
	for _, mode := range m.modes {
		if mode.Match == nil || mode.Match(input) {
			return mode
		}
	}
	return Mode{}
}

// Query returns the active mode's interpretation of the input value
// (typically with a leading prefix stripped). Falls back to the raw
// input when the active mode has no Query function.
func (m Model) Query() string {
	mode := m.Mode()
	if mode.Query == nil {
		return m.input.Value()
	}
	return mode.Query(m.input.Value())
}

// Value returns the raw input value.
func (m Model) Value() string { return m.input.Value() }

// Results returns the cached items for the named mode (typically
// populated by that mode's Search closure via SearchResultMsg). A
// mode's Items closure usually reads from here.
func (m Model) Results(modeName string) []Item {
	return m.results[modeName]
}

// Loading reports whether a Search is currently in flight.
func (m Model) Loading() bool { return m.loading }

// SetWidth overrides the palette's outer width. Useful when the
// palette is laid out manually (e.g., as a fixed-width modal overlay)
// rather than filling the host's WindowSizeMsg.
func (m *Model) SetWidth(w int) { m.width = w }

// SetHeight overrides the palette's outer height. Currently advisory —
// pagination is driven by WithPageSize.
func (m *Model) SetHeight(h int) { m.height = h }

// Items returns the candidate items currently visible in the palette.
// During facet completion this is the active Facet's value list (sync
// via Facet.Items or the cached async results); otherwise it's the
// active Mode's items.
func (m Model) Items() []Item {
	if m.facet != nil {
		if m.facet.facet.Items != nil {
			return m.facet.facet.Items(m.facet.partial)
		}
		return m.facetResults[m.facet.facet.Name]
	}
	mode := m.Mode()
	if mode.Items == nil {
		return nil
	}
	return mode.Items(m, m.Query())
}

// FilterFuzzy returns items whose FilterValue is a fuzzy-subsequence
// match for query, ordered by relevance (best first). An empty query
// returns the input unchanged. Exported so a Mode's Items closure can
// fuzzy-filter its own item slice without re-implementing the matcher.
func FilterFuzzy(items []Item, query string) []Item {
	if query == "" {
		return items
	}
	targets := make([]string, len(items))
	for i, c := range items {
		targets[i] = c.FilterValue()
	}
	matches := fuzzy.Find(query, targets)
	out := make([]Item, len(matches))
	for i, mt := range matches {
		out[i] = items[mt.Index]
	}
	return out
}

// Selected returns the highlighted item, or nil if none.
func (m Model) Selected() Item {
	items := m.Items()
	cursor := m.cursor
	if m.facet != nil {
		cursor = m.facet.cursor
	}
	if cursor < 0 || cursor >= len(items) {
		return nil
	}
	return items[cursor]
}

// Page returns the current page (0-indexed).
func (m Model) Page() int { return m.paginator.Page }

// TotalPages returns the number of pages.
func (m Model) TotalPages() int { return m.paginator.TotalPages }

// Reset clears the input and result state, and cancels any in-flight
// Search or facet Resolve.
func (m *Model) Reset() {
	if m.searchCancel != nil {
		m.searchCancel()
		m.searchCancel = nil
	}
	if m.facetCancel != nil {
		m.facetCancel()
		m.facetCancel = nil
	}
	m.input.SetValue("")
	m.results = map[string][]Item{}
	m.facetResults = map[string][]Item{}
	m.facet = nil
	m.cursor = 0
	m.paginator.Page = 0
	m.loading = false
	m.pending = false
}
