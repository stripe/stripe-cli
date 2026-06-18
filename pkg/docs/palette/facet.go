package palette

import (
	"context"
	"strings"
	"time"
	"unicode"

	tea "charm.land/bubbletea/v2"
)

// Facet defines a token-based filter inside a Mode. When the cursor
// sits inside a "<Name>:<partial>" token in the input, the palette
// enters a value-completion sub-state listing this facet's matching
// values. Exactly one of Items or Resolve must be set; if both are,
// Items wins.
type Facet struct {
	// Name is the prefix users type before ":" to trigger completion.
	// Should not contain whitespace or ":".
	Name string

	// Desc is an optional context hint shown above the value list
	// while completing this facet.
	Desc string

	// Items returns sync values for the given partial. The host owns
	// the filtering; the palette doesn't post-filter the returned
	// slice.
	Items func(partial string) []Item

	// Resolve dispatches an async value lookup. The returned tea.Cmd
	// must eventually emit a FacetResultMsg whose Facet and Partial
	// match the inputs. The ctx is canceled when the partial changes,
	// the cursor leaves the token, or the palette is Reset.
	Resolve func(ctx context.Context, partial string) tea.Cmd

	// Debounce delays Resolve dispatch after the partial last changed.
	// Ignored when Resolve is nil.
	Debounce time.Duration
}

// FacetResultMsg is what Facet.Resolve eventually emits. The palette
// drops it as stale when Facet/Partial no longer match the cursor's
// current token.
type FacetResultMsg struct {
	Facet   string
	Partial string
	Results []Item
	Err     error
}

// facetCompletion is the palette's transient sub-state while the
// cursor sits inside a "<facet>:<partial>" token.
type facetCompletion struct {
	facet      Facet
	tokenStart int    // rune offset where the token begins
	tokenEnd   int    // rune offset where it ends (exclusive)
	partial    string // value typed so far, after the ":"
	cursor     int    // selected value within the completion list
}

// facetDebounceMsg is an internal tick that fires after a Facet's
// Debounce window. The palette only dispatches Resolve when the
// generation still matches (no newer partial superseded it).
type facetDebounceMsg struct {
	facet   string
	partial string
	gen     int
}

// detectFacetToken inspects the input around the rune-indexed cursor
// position and returns the matching Facet, token bounds, and partial
// when the cursor sits inside a "<name>:<partial>" token whose <name>
// is registered in facets. Token boundaries are whitespace.
func detectFacetToken(input string, cursorRune int, facets []Facet) (Facet, int, int, string, bool) {
	if len(facets) == 0 {
		return Facet{}, 0, 0, "", false
	}
	runes := []rune(input)
	if cursorRune < 0 {
		cursorRune = 0
	}
	if cursorRune > len(runes) {
		cursorRune = len(runes)
	}

	start := cursorRune
	for start > 0 && !unicode.IsSpace(runes[start-1]) {
		start--
	}
	end := cursorRune
	for end < len(runes) && !unicode.IsSpace(runes[end]) {
		end++
	}
	if start == end {
		return Facet{}, 0, 0, "", false
	}

	token := runes[start:end]
	colon := -1
	for i, r := range token {
		if r == ':' {
			colon = i
			break
		}
	}
	if colon < 0 {
		return Facet{}, 0, 0, "", false
	}

	name := string(token[:colon])
	partial := string(token[colon+1:])
	for _, f := range facets {
		if f.Name == name {
			return f, start, end, partial, true
		}
	}
	return Facet{}, 0, 0, "", false
}

// ParseFacets tokenizes input on whitespace and returns the free-text
// portion (tokens that aren't registered facets, joined with single
// spaces) plus a map of facet name → values. Use this from a Mode's
// Items/Search closures to apply facet filters without re-implementing
// the parser.
func ParseFacets(input string, facets []Facet) (text string, parsed map[string][]string) {
	parsed = map[string][]string{}
	if input == "" {
		return "", parsed
	}
	names := make(map[string]bool, len(facets))
	for _, f := range facets {
		names[f.Name] = true
	}
	var textTokens []string
	for _, tok := range strings.Fields(input) {
		colon := strings.Index(tok, ":")
		if colon <= 0 {
			textTokens = append(textTokens, tok)
			continue
		}
		name := tok[:colon]
		if !names[name] {
			textTokens = append(textTokens, tok)
			continue
		}
		value := tok[colon+1:]
		if value == "" {
			// Partial token (e.g. "label:") — skip; it's the in-progress
			// completion, not a finished filter.
			continue
		}
		parsed[name] = append(parsed[name], value)
	}
	return strings.Join(textTokens, " "), parsed
}

// evaluateFacet inspects the cursor against the active Mode's Facets
// and synchronizes m.facet accordingly. Returns commands to dispatch
// (debounce tick + spinner.Tick) when entering or updating async
// facet completion; nil otherwise.
func (m *Model) evaluateFacet() tea.Cmd {
	mode := m.Mode()
	detected, start, end, partial, ok := detectFacetToken(m.input.Value(), m.input.Position(), mode.Facets)
	if !ok {
		m.clearFacet()
		return nil
	}

	if m.facet != nil && m.facet.facet.Name == detected.Name && m.facet.partial == partial {
		m.facet.tokenStart, m.facet.tokenEnd = start, end
		return nil
	}

	if m.facet == nil {
		m.facet = &facetCompletion{}
	}
	m.facet.facet = detected
	m.facet.tokenStart = start
	m.facet.tokenEnd = end
	m.facet.partial = partial
	m.facet.cursor = 0

	// Sync facets: items are returned inline by Facet.Items — no
	// scheduling or spinner needed.
	if detected.Items != nil {
		if m.facetCancel != nil {
			m.facetCancel()
			m.facetCancel = nil
		}
		m.loading = false
		m.pending = false
		return nil
	}
	if detected.Resolve == nil {
		m.loading = false
		m.pending = false
		return nil
	}

	// Async: cancel any prior in-flight resolve, schedule a debounce
	// tick, and seed the spinner tick chain if it isn't already running.
	if m.facetCancel != nil {
		m.facetCancel()
		m.facetCancel = nil
	}
	spinnerActive := m.loading || m.pending
	m.loading = false
	m.facetGen++
	gen := m.facetGen
	name := detected.Name
	partialCopy := partial
	d := detected.Debounce
	tickCmd := tea.Tick(d, func(_ time.Time) tea.Msg {
		return facetDebounceMsg{facet: name, partial: partialCopy, gen: gen}
	})
	m.pending = true
	if spinnerActive {
		return tickCmd
	}
	return tea.Batch(tickCmd, m.spinner.Tick)
}

// clearFacet exits facet completion and cancels any in-flight Resolve.
// Leaves m.pending/m.loading untouched — the caller decides what state
// the spinner should be in next.
func (m *Model) clearFacet() {
	if m.facet == nil {
		return
	}
	if m.facetCancel != nil {
		m.facetCancel()
		m.facetCancel = nil
	}
	m.facet = nil
}

// handleFacetDebounce dispatches the active Facet's Resolve closure
// when the debounce tick is still current.
func (m Model) handleFacetDebounce(msg facetDebounceMsg) (Model, tea.Cmd) {
	if msg.gen != m.facetGen {
		return m, nil
	}
	if m.facet == nil || m.facet.facet.Name != msg.facet || m.facet.partial != msg.partial {
		return m, nil
	}
	if m.facet.facet.Resolve == nil {
		return m, nil
	}
	ctx, cancel := context.WithCancel(context.Background())
	m.facetCancel = cancel
	m.pending = false
	m.loading = true
	return m, m.facet.facet.Resolve(ctx, msg.partial)
}

// handleFacetResult stores facet values in the per-facet cache and
// clears loading. Stale results (whose Facet or Partial no longer
// matches the cursor's current token) are dropped.
func (m Model) handleFacetResult(msg FacetResultMsg) (Model, tea.Cmd) {
	if m.facet == nil || msg.Facet != m.facet.facet.Name || msg.Partial != m.facet.partial {
		return m, nil
	}
	if m.facetResults == nil {
		m.facetResults = map[string][]Item{}
	}
	m.facetResults[msg.Facet] = msg.Results
	m.loading = false
	return m, nil
}

// applyFacetCompletion splices the currently selected facet value into
// the input at the active token's position, exits completion, and
// re-reconciles the input state so any mode Search kicks in for the
// resulting query.
func (m Model) applyFacetCompletion() (Model, tea.Cmd) {
	if m.facet == nil {
		return m, nil
	}
	items := m.Items()
	if m.facet.cursor < 0 || m.facet.cursor >= len(items) {
		return m, nil
	}
	sel := items[m.facet.cursor]
	newInput, cursor := applyFacetValue(
		m.input.Value(),
		m.facet.tokenStart, m.facet.tokenEnd,
		m.facet.facet.Name, sel.FilterValue(),
	)
	m.input.SetValue(newInput)
	m.input.SetCursor(cursor)
	m.clearFacet()
	m.pending = false
	m.loading = false
	m.cursor = 0
	return m, m.reconcileInputState()
}

// applyFacetValue splices the given facet value into input, replacing
// the partial token at [tokenStart, tokenEnd). A trailing space is
// added when the character after the token isn't already whitespace so
// the cursor lands outside the token and completion exits cleanly.
// Returns the new input and the rune position for the cursor.
func applyFacetValue(input string, tokenStart, tokenEnd int, name, value string) (string, int) {
	runes := []rune(input)
	if tokenStart < 0 {
		tokenStart = 0
	}
	if tokenEnd > len(runes) {
		tokenEnd = len(runes)
	}
	insertion := []rune(name + ":" + value)
	needsSpace := tokenEnd >= len(runes) || !unicode.IsSpace(runes[tokenEnd])
	if needsSpace {
		insertion = append(insertion, ' ')
	}
	out := make([]rune, 0, len(runes)-(tokenEnd-tokenStart)+len(insertion))
	out = append(out, runes[:tokenStart]...)
	out = append(out, insertion...)
	out = append(out, runes[tokenEnd:]...)
	return string(out), tokenStart + len(insertion)
}
