package palette

import (
	"context"
	"strings"
	"testing"

	"charm.land/bubbles/v2/key"
	"charm.land/bubbles/v2/paginator"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
)

// testItem is a minimal Item used to populate the model in tests.
type testItem struct{ name string }

func (t testItem) FilterValue() string { return t.name }

// commandMode returns a ">"-prefix sync mode that fuzzy-filters cmds.
// Mirrors the behavior the deleted built-in CommandMode provided —
// most tests rely on the ">"-prefix mode existing alongside a
// catch-all results bucket.
func commandMode(cmds []Item) Mode {
	return Mode{
		Name:  "command",
		Match: func(s string) bool { return strings.HasPrefix(s, ">") },
		Query: func(s string) string { return strings.TrimSpace(strings.TrimPrefix(s, ">")) },
		Items: func(_ Model, q string) []Item { return FilterFuzzy(cmds, q) },
	}
}

// searchMode returns a catch-all sync mode that reads from
// m.results["search"]. Pair with commandMode to recreate the old
// default-mode setup for tests that depend on it.
func searchMode() Mode {
	return Mode{
		Name:  "search",
		Items: func(m Model, _ string) []Item { return m.results["search"] },
	}
}

// withSeeded wires commandMode + searchMode in the palette so tests
// that used to call withSeeded(cmds) keep working as a drop-in.
func withSeeded(cmds []Item) Option {
	return WithModes(commandMode(cmds), searchMode())
}

func TestModeAndQuery(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		wantMode string // mode Name
		wantQry  string
	}{
		{"empty", "", "search", ""},
		{"plain text", "foo", "search", "foo"},
		{"just bracket", ">", "command", ""},
		{"bracket space", "> ", "command", ""},
		{"command", "> open", "command", "open"},
		{"command no space", ">open", "command", "open"},
		{"bracket inside", "foo > bar", "search", "foo > bar"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			m := New(WithModes(commandMode(nil), searchMode()))
			m.input.SetValue(tc.input)

			if got := m.Mode().Name; got != tc.wantMode {
				t.Errorf("Mode() = %s, want %s", got, tc.wantMode)
			}
			if got := m.Query(); got != tc.wantQry {
				t.Errorf("Query() = %q, want %q", got, tc.wantQry)
			}
			if got := m.Value(); got != tc.input {
				t.Errorf("Value() = %q, want %q", got, tc.input)
			}
		})
	}
}

func TestCustomMode(t *testing.T) {
	// Caller registers a custom "@"-prefixed file-picker mode that
	// sits between CommandMode and SearchMode in priority.
	files := []Item{testItem{name: "main.go"}, testItem{name: "README.md"}, testItem{name: "go.mod"}}

	filesMode := Mode{
		Name: "files",
		Match: func(input string) bool {
			return strings.HasPrefix(input, "@")
		},
		Query: func(input string) string {
			return strings.TrimSpace(strings.TrimPrefix(input, "@"))
		},
		Items: func(_ Model, q string) []Item {
			return FilterFuzzy(files, q)
		},
	}

	m := New(WithModes(commandMode(nil), filesMode, searchMode()))

	cases := []struct {
		input        string
		wantMode     string
		wantItemHead string // FilterValue of the first item, or "" if no items
	}{
		{">foo", "command", ""},    // CommandMode wins on ">"
		{"@go", "files", "go.mod"}, // FilesMode matches "@"; fuzzy ranks go.mod first
		{"@readme", "files", "README.md"},
		{"unprefixed", "search", ""}, // falls through to SearchMode (no results seeded)
	}

	for _, tc := range cases {
		t.Run(tc.input, func(t *testing.T) {
			m.input.SetValue(tc.input)

			if got := m.Mode().Name; got != tc.wantMode {
				t.Errorf("Mode = %s, want %s", got, tc.wantMode)
			}
			items := m.Items()
			if tc.wantItemHead == "" {
				if len(items) != 0 {
					t.Errorf("Items() = %v, want empty", itemNames(items))
				}
			} else {
				if len(items) == 0 || items[0].FilterValue() != tc.wantItemHead {
					t.Errorf("Items()[0] = %q, want %q", itemHead(items), tc.wantItemHead)
				}
			}
		})
	}
}

func itemHead(items []Item) string {
	if len(items) == 0 {
		return ""
	}
	return items[0].FilterValue()
}

func TestItemsReflectsMode(t *testing.T) {
	cmds := []Item{Command{Name: "open"}, Command{Name: "close"}}
	results := []Item{testItem{name: "hit"}}

	m := New(withSeeded(cmds))
	m.results["search"] = results

	m.input.SetValue("foo")
	if got := m.Items(); len(got) != 1 || got[0].FilterValue() != "hit" {
		t.Errorf("SearchMode Items() = %v, want results", got)
	}

	// Empty query in CommandMode returns the full command list, in
	// declared order. Filtering is covered separately.
	m.input.SetValue(">")
	if got := m.Items(); len(got) != 2 || got[0].FilterValue() != "open" {
		t.Errorf("CommandMode Items() = %v, want commands", got)
	}
}

func TestCommandModeFuzzyFilter(t *testing.T) {
	cmds := []Item{
		Command{Name: "Open file"},
		Command{Name: "Save"},
		Command{Name: "Open in new tab"},
		Command{Name: "Quit"},
	}

	tests := []struct {
		name  string
		query string
		want  []string // expected FilterValues in order
	}{
		{
			name:  "empty query returns all in declared order",
			query: "",
			want:  []string{"Open file", "Save", "Open in new tab", "Quit"},
		},
		{
			name:  "exact substring matches",
			query: "open",
			want:  []string{"Open file", "Open in new tab"},
		},
		{
			name:  "case insensitive",
			query: "QuI",
			want:  []string{"Quit"},
		},
		{
			name:  "fuzzy non-contiguous chars",
			query: "qt", // Q...uiT
			want:  []string{"Quit"},
		},
		{
			name:  "no matches returns empty",
			query: "zzzz",
			want:  []string{},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			m := New(withSeeded(cmds))
			m.input.SetValue(">" + tc.query)

			got := m.Items()
			if len(got) != len(tc.want) {
				t.Fatalf("Items() len = %d, want %d\ngot:  %v\nwant: %v", len(got), len(tc.want), itemNames(got), tc.want)
			}
			for i, w := range tc.want {
				if got[i].FilterValue() != w {
					t.Errorf("Items()[%d] = %q, want %q (full got: %v)", i, got[i].FilterValue(), w, itemNames(got))
				}
			}
		})
	}
}

func TestCursorResetsOnInputChange(t *testing.T) {
	m := New(withSeeded([]Item{
		Command{Name: "Open file"},
		Command{Name: "Save"},
		Command{Name: "Quit"},
	}))
	m.Focus()
	m.input.SetValue(">")
	m.cursor = 2

	// Simulate a keypress that mutates the input value.
	m, _ = m.Update(typeKey(t, "a"))

	if m.cursor != 0 {
		t.Errorf("cursor = %d after input change, want 0", m.cursor)
	}
}

func TestCursorUnchangedWhenInputUnchanged(t *testing.T) {
	m := New(withSeeded([]Item{Command{Name: "a"}, Command{Name: "b"}, Command{Name: "c"}}))
	m.Focus()
	m.input.SetValue(">")
	m.cursor = 1

	// Send a message that the textinput doesn't consume as text (e.g.,
	// a window-size change). Cursor must not reset.
	m, _ = m.Update(tea.WindowSizeMsg{Width: 80, Height: 24})

	if m.cursor != 1 {
		t.Errorf("cursor = %d after non-input msg, want 1 (unchanged)", m.cursor)
	}
}

func TestCursorNavigation(t *testing.T) {
	newPaletteIn := func(t *testing.T) Model {
		t.Helper()
		m := New(withSeeded([]Item{
			Command{Name: "Open file"},
			Command{Name: "Save"},
			Command{Name: "Quit"},
		}))
		m.Focus()
		m.input.SetValue(">")
		return m
	}

	tests := []struct {
		name string
		seed int
		key  tea.KeyPressMsg
		want int
	}{
		{"down from top", 0, arrowKey(tea.KeyDown), 1},
		{"down again", 1, arrowKey(tea.KeyDown), 2},
		{"down wraps from last", 2, arrowKey(tea.KeyDown), 0},
		{"up from middle", 1, arrowKey(tea.KeyUp), 0},
		{"up wraps from first", 0, arrowKey(tea.KeyUp), 2},
		{"ctrl+n is down", 0, ctrlKey('n'), 1},
		{"ctrl+p is up", 1, ctrlKey('p'), 0},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			m := newPaletteIn(t)
			m.cursor = tc.seed
			m, _ = m.Update(tc.key)
			if m.cursor != tc.want {
				t.Errorf("cursor = %d, want %d", m.cursor, tc.want)
			}
		})
	}
}

func TestNavigationDoesNotAlterInput(t *testing.T) {
	m := New(withSeeded([]Item{Command{Name: "a"}, Command{Name: "b"}}))
	m.Focus()
	m.input.SetValue(">")

	m, _ = m.Update(arrowKey(tea.KeyDown))

	if v := m.input.Value(); v != ">" {
		t.Errorf("input mutated by navigation key: got %q, want %q", v, ">")
	}
	if m.cursor != 1 {
		t.Errorf("cursor = %d, want 1", m.cursor)
	}
}

func TestPaginationCursorJumpsByPage(t *testing.T) {
	// 10 items, pageSize 4 → pages of [0-3], [4-7], [8-9].
	cmds := make([]Item, 10)
	for i := range cmds {
		cmds[i] = Command{Name: string(rune('a' + i))}
	}
	m := New(withSeeded(cmds), WithPageSize(4))
	m.Focus()
	m.input.SetValue(">")

	// PageDown from page 0 → cursor at start of page 1.
	m, _ = m.Update(arrowKey(tea.KeyRight))
	if m.cursor != 4 {
		t.Errorf("after PageDown: cursor = %d, want 4", m.cursor)
	}

	// PageDown again → start of page 2 (the last page).
	m, _ = m.Update(arrowKey(tea.KeyRight))
	if m.cursor != 8 {
		t.Errorf("after second PageDown: cursor = %d, want 8", m.cursor)
	}

	// PageDown at last page → wraps to first page.
	m, _ = m.Update(arrowKey(tea.KeyRight))
	if m.cursor != 0 {
		t.Errorf("PageDown at last page: cursor = %d, want 0 (wrapped)", m.cursor)
	}

	// PageUp from first page → wraps to last page.
	m, _ = m.Update(arrowKey(tea.KeyLeft))
	if m.cursor != 8 {
		t.Errorf("PageUp at first page: cursor = %d, want 8 (wrapped)", m.cursor)
	}

	// PageUp → back to page 1 start.
	m, _ = m.Update(arrowKey(tea.KeyLeft))
	if m.cursor != 4 {
		t.Errorf("after PageUp: cursor = %d, want 4", m.cursor)
	}
}

func TestPaginationDisabledWhenPageSizeZero(t *testing.T) {
	// pageSize=0 means no pagination — PageNext key is a no-op,
	// cursor doesn't snap.
	cmds := make([]Item, 10)
	for i := range cmds {
		cmds[i] = Command{Name: string(rune('a' + i))}
	}
	m := New(withSeeded(cmds))
	m.Focus()
	m.input.SetValue(">")
	m.cursor = 3

	m, _ = m.Update(arrowKey(tea.KeyRight))
	if m.cursor != 3 {
		t.Errorf("PageNext with pageSize=0: cursor = %d, want 3 (unchanged)", m.cursor)
	}
}

func TestPaginationSelectedHonoursAbsoluteCursor(t *testing.T) {
	// After paging forward, Selected() must return the item the cursor
	// points to — not the first item of the visible page slice.
	cmds := make([]Item, 6)
	for i := range cmds {
		cmds[i] = Command{Name: string(rune('a' + i))}
	}
	m := New(withSeeded(cmds), WithPageSize(3))
	m.Focus()
	m.input.SetValue(">")

	m, _ = m.Update(arrowKey(tea.KeyRight)) // cursor=3, page 1
	if got := m.Selected().FilterValue(); got != "d" {
		t.Errorf("Selected on page 1 = %q, want d", got)
	}
}

func TestPaginationFooterRendersWhenMultiplePages(t *testing.T) {
	cmds := []Item{
		Command{Name: "one"}, Command{Name: "two"},
		Command{Name: "three"}, Command{Name: "four"},
	}
	m := New(withSeeded(cmds), WithPageSize(2), WithHelp(false))
	m.input.SetValue(">")
	m, _ = m.Update(tea.WindowSizeMsg{Width: 40, Height: 20})

	out := m.View()
	// The dots paginator emits at least two bullet glyphs (• or ○)
	// for a 2-page list. Check for the active-dot character used by
	// the bubbles paginator default config.
	if !strings.Contains(out, "•") && !strings.Contains(out, "○") {
		t.Errorf("View() missing paginator footer dots, got:\n%s", out)
	}
}

func TestPaginationNoFooterForSinglePage(t *testing.T) {
	cmds := []Item{Command{Name: "only"}}
	m := New(withSeeded(cmds), WithPageSize(5), WithHelp(false))
	m.input.SetValue(">")
	m, _ = m.Update(tea.WindowSizeMsg{Width: 40, Height: 20})

	out := m.View()
	// Single page: paginator footer should NOT appear. Lines below the
	// items section should be padding only — no dots in the output's
	// footer area. We look for either of the paginator's default glyphs.
	// Simpler check: with only one item, there should be no "•" char
	// outside of any help line (help is disabled here).
	if strings.Contains(out, "•") {
		t.Errorf("View() rendered paginator footer for single page:\n%s", out)
	}
}

// asyncMode builds a Mode whose Search emits a known SearchResultMsg
// synchronously (test cmds are invoked inline) so we can drive the
// async lifecycle deterministically without a real clock.
func asyncMode(name string, results []Item) Mode {
	return Mode{
		Name:     name,
		Debounce: 0,
		Match: func(s string) bool {
			return strings.HasPrefix(s, "@"+name) || (name == "search" && !strings.HasPrefix(s, "@"))
		},
		Query: func(s string) string { return strings.TrimPrefix(s, "@"+name) },
		Items: func(m Model, _ string) []Item { return m.Results(name) },
		Search: func(_ context.Context, q string) tea.Cmd {
			return func() tea.Msg {
				return SearchResultMsg{Mode: name, Query: q, Results: results}
			}
		},
	}
}

func TestSearchDebounceDispatchesAfterTick(t *testing.T) {
	mode := asyncMode("files", []Item{testItem{name: "hit"}})
	m := New(WithModes(mode, searchMode()))
	m.Focus()

	// Simulate a keystroke that lands "@files foo" in the input.
	m.input.SetValue("@filesfoo")
	cmd := m.scheduleSearch()
	if cmd == nil {
		t.Fatal("scheduleSearch returned nil; expected a debounce tick cmd")
	}
	if !m.pending {
		t.Error("expected pending=true after scheduling search; spinner needs it during debounce window")
	}

	// scheduleSearch returns a batch of (debounce tick, spinner.Tick) on
	// the first scheduling so the spinner starts ticking before Search
	// dispatches.
	dbMsg := findDebounceMsg(t, cmd)

	// Feed the debounceMsg back; this should dispatch the Search.
	m, dispatchCmd := m.Update(dbMsg)
	if m.pending {
		t.Error("expected pending=false after debounce dispatch")
	}
	if !m.loading {
		t.Error("expected loading=true after debounce dispatch")
	}
	if dispatchCmd == nil {
		t.Fatal("expected a search cmd after debounce")
	}

	// dispatchCmd is the Search closure (spinner already ticks from the
	// scheduleSearch batch, so handleDebounce doesn't re-seed it).
	resultMsg, ok := dispatchCmd().(SearchResultMsg)
	if !ok {
		t.Fatalf("dispatch cmd produced %T, want SearchResultMsg", dispatchCmd())
	}
	if resultMsg.Mode != "files" || resultMsg.Query != "foo" {
		t.Errorf("SearchResultMsg = %+v, want Mode=files Query=foo", resultMsg)
	}

	// Apply the result.
	m, _ = m.Update(resultMsg)
	if m.loading {
		t.Error("expected loading=false after result")
	}
	if got := m.Results("files"); len(got) != 1 || got[0].FilterValue() != "hit" {
		t.Errorf("Results(files) = %v, want [hit]", got)
	}
}

// findDebounceMsg extracts the debounceMsg from cmd's output, whether
// cmd is the raw tick or a batch wrapping (tick, spinner.Tick).
func findDebounceMsg(t *testing.T, cmd tea.Cmd) debounceMsg {
	t.Helper()
	msg := cmd()
	if db, ok := msg.(debounceMsg); ok {
		return db
	}
	batch, ok := msg.(tea.BatchMsg)
	if !ok {
		t.Fatalf("cmd produced %T, want debounceMsg or tea.BatchMsg", msg)
	}
	for _, c := range batch {
		if db, ok := c().(debounceMsg); ok {
			return db
		}
	}
	t.Fatal("no debounceMsg in batch")
	return debounceMsg{}
}

func TestSearchStaleDebounceIgnored(t *testing.T) {
	mode := asyncMode("files", []Item{testItem{name: "hit"}})
	m := New(WithModes(mode, searchMode()))

	m.input.SetValue("@filesa")
	_ = m.scheduleSearch() // gen=1
	m.input.SetValue("@filesab")
	cmd := m.scheduleSearch() // gen=2
	if cmd == nil {
		t.Fatal("nil cmd from scheduleSearch")
	}

	// First (stale) debounce: gen=1. Should be ignored.
	m, dispatch := m.Update(debounceMsg{mode: "files", gen: 1})
	if m.loading {
		t.Error("loading=true after stale debounce; expected ignored")
	}
	if !m.pending {
		t.Error("pending=false after stale debounce; expected unchanged (newer debounce still in flight)")
	}
	if dispatch != nil {
		t.Error("dispatch should be nil for stale debounce")
	}
}

func TestSearchStaleResultIgnored(t *testing.T) {
	mode := asyncMode("files", []Item{testItem{name: "hit"}})
	m := New(WithModes(mode, searchMode()))

	m.input.SetValue("@filesfoo")
	// Stale result targeting a different query.
	m, _ = m.Update(SearchResultMsg{
		Mode: "files", Query: "old", Results: []Item{testItem{name: "stale"}},
	})
	if got := m.Results("files"); got != nil {
		t.Errorf("stale result accepted: %v", got)
	}
}

func TestSearchModeSwitchCancelsInFlight(t *testing.T) {
	// Capture the ctx the search receives so we can assert it gets
	// canceled when the mode switches.
	var capturedCtx context.Context
	mode := Mode{
		Name:     "files",
		Debounce: 0,
		Match:    func(s string) bool { return strings.HasPrefix(s, "@") },
		Query:    func(s string) string { return strings.TrimPrefix(s, "@") },
		Items:    func(m Model, _ string) []Item { return m.Results("files") },
		Search: func(ctx context.Context, _ string) tea.Cmd {
			capturedCtx = ctx
			// Return a cmd that never produces — simulates a long
			// in-flight request.
			return func() tea.Msg { return nil }
		},
	}
	m := New(WithModes(mode, searchMode()))
	m.input.SetValue("@foo")
	cmd := m.scheduleSearch()
	dbMsg := findDebounceMsg(t, cmd)
	m, _ = m.Update(dbMsg)

	if capturedCtx == nil {
		t.Fatal("Search wasn't dispatched")
	}
	if capturedCtx.Err() != nil {
		t.Error("ctx already canceled before mode switch")
	}

	// Switch modes by changing input to something CommandMode claims.
	m.input.SetValue(">cmd")
	m.scheduleSearch() // cancel-only path; CommandMode has no Search

	if capturedCtx.Err() == nil {
		t.Error("expected ctx to be canceled after mode switch, got nil err")
	}
}

// runMarker is what a test Command's Run() emits — lets us assert
// that Run actually fired when Enter dispatches.
type runMarker struct{ id string }

func TestEnterDispatchesCommandRun(t *testing.T) {
	cmd := Command{
		Name: "Open file",
		Run:  func() tea.Cmd { return func() tea.Msg { return runMarker{id: "open"} } },
	}

	m := New(withSeeded([]Item{cmd}))
	m.Focus()
	m.input.SetValue(">")

	m, dispatched := m.Update(arrowKey(tea.KeyEnter))
	if dispatched == nil {
		t.Fatal("Enter returned nil cmd, expected dispatch")
	}

	// The batched cmd produces a BatchMsg containing the SelectedMsg
	// emitter and the Command's Run cmd.
	batchMsg, ok := dispatched().(tea.BatchMsg)
	if !ok {
		t.Fatalf("dispatched msg = %T, want tea.BatchMsg", dispatched())
	}
	if len(batchMsg) != 2 {
		t.Fatalf("BatchMsg holds %d cmds, want 2", len(batchMsg))
	}

	var sawSelected, sawRun bool
	for _, c := range batchMsg {
		switch v := c().(type) {
		case SelectedMsg:
			if v.Item.FilterValue() != "Open file" {
				t.Errorf("SelectedMsg.Item = %q, want Open file", v.Item.FilterValue())
			}
			sawSelected = true
		case runMarker:
			if v.id != "open" {
				t.Errorf("runMarker id = %q, want open", v.id)
			}
			sawRun = true
		}
	}
	if !sawSelected {
		t.Error("batch missing SelectedMsg")
	}
	if !sawRun {
		t.Error("batch missing Command.Run output")
	}
}

func TestEnterEmitsSelectedMsgWhenRunNil(t *testing.T) {
	m := New(withSeeded([]Item{Command{Name: "Save", Run: nil}}))
	m.Focus()
	m.input.SetValue(">")

	_, dispatched := m.Update(arrowKey(tea.KeyEnter))
	if dispatched == nil {
		t.Fatal("Enter returned nil cmd")
	}
	msg := dispatched()
	sel, ok := msg.(SelectedMsg)
	if !ok {
		t.Fatalf("dispatched = %T, want SelectedMsg", msg)
	}
	if sel.Item.FilterValue() != "Save" {
		t.Errorf("SelectedMsg.Item = %q, want Save", sel.Item.FilterValue())
	}
}

func TestEnterOnNonCommandItem(t *testing.T) {
	// A non-Command result item: Enter should still emit SelectedMsg
	// so the host knows what was picked.
	m := New(WithModes(searchMode()))
	m.Focus()
	m.input.SetValue("query")
	m.results["search"] = []Item{testItem{name: "hit"}}

	_, dispatched := m.Update(arrowKey(tea.KeyEnter))
	if dispatched == nil {
		t.Fatal("Enter returned nil cmd")
	}
	sel, ok := dispatched().(SelectedMsg)
	if !ok {
		t.Fatalf("dispatched = %T, want SelectedMsg", dispatched())
	}
	if sel.Item.FilterValue() != "hit" {
		t.Errorf("SelectedMsg.Item = %q, want hit", sel.Item.FilterValue())
	}
}

func TestEnterWithNoSelection(t *testing.T) {
	// Empty items list → no cmd, no panic.
	m := New()
	m.Focus()
	m.input.SetValue("nothing")

	_, dispatched := m.Update(arrowKey(tea.KeyEnter))
	if dispatched != nil {
		t.Errorf("Enter with no selection should return nil cmd, got %v", dispatched())
	}
}

func TestEnterNotForwardedToInput(t *testing.T) {
	// Enter must be consumed; the textinput must not see it (which
	// in some configurations would otherwise submit/blur the input).
	m := New(withSeeded([]Item{Command{Name: "a"}}))
	m.Focus()
	m.input.SetValue(">a")

	m, _ = m.Update(arrowKey(tea.KeyEnter))
	if v := m.input.Value(); v != ">a" {
		t.Errorf("input mutated by Enter: got %q, want %q", v, ">a")
	}
}

func TestNavigationWithNoItems(t *testing.T) {
	// SearchMode + no results → no items. Navigation must be a no-op,
	// not panic or set cursor out of bounds.
	m := New()
	m.Focus()
	m.input.SetValue("query")
	m.cursor = 0

	m, _ = m.Update(arrowKey(tea.KeyDown))
	if m.cursor != 0 {
		t.Errorf("cursor moved with no items: got %d, want 0", m.cursor)
	}
}

// arrowKey returns a tea.KeyPressMsg for a special key like tea.KeyDown.
func arrowKey(code rune) tea.KeyPressMsg {
	return tea.KeyPressMsg{Code: code}
}

// ctrlKey returns a tea.KeyPressMsg for ctrl+<rune>.
func ctrlKey(r rune) tea.KeyPressMsg {
	return tea.KeyPressMsg{Code: r, Mod: tea.ModCtrl}
}

// itemNames extracts FilterValues for readable test failures.
func itemNames(items []Item) []string {
	out := make([]string, len(items))
	for i, it := range items {
		out[i] = it.FilterValue()
	}
	return out
}

// typeKey returns a tea.KeyPressMsg corresponding to a single typed
// character, suitable for feeding through Model.Update.
func typeKey(t *testing.T, ch string) tea.KeyPressMsg {
	t.Helper()
	r := []rune(ch)
	if len(r) != 1 {
		t.Fatalf("typeKey expects a single rune, got %q", ch)
	}
	return tea.KeyPressMsg{Code: r[0], Text: ch}
}

func TestSelectedBounds(t *testing.T) {
	m := New(withSeeded([]Item{Command{Name: "a"}, Command{Name: "b"}}))
	m.input.SetValue(">")

	m.cursor = 0
	if got := m.Selected(); got == nil || got.FilterValue() != "a" {
		t.Errorf("cursor=0 Selected() = %v, want a", got)
	}

	m.cursor = -1
	if got := m.Selected(); got != nil {
		t.Errorf("cursor=-1 Selected() = %v, want nil", got)
	}

	m.cursor = 99
	if got := m.Selected(); got != nil {
		t.Errorf("cursor=99 Selected() = %v, want nil", got)
	}
}

func TestReset(t *testing.T) {
	m := New(withSeeded([]Item{Command{Name: "a"}}))
	m.input.SetValue("hello")
	m.results["search"] = []Item{testItem{name: "x"}}
	m.cursor = 3
	m.paginator.Page = 2
	m.loading = true
	m.pending = true

	m.Reset()

	if m.Value() != "" {
		t.Errorf("Value() = %q after Reset, want empty", m.Value())
	}
	if len(m.results) != 0 {
		t.Errorf("results = %v after Reset, want empty", m.results)
	}
	if m.cursor != 0 {
		t.Errorf("cursor = %d after Reset, want 0", m.cursor)
	}
	if m.paginator.Page != 0 {
		t.Errorf("paginator.Page = %d after Reset, want 0", m.paginator.Page)
	}
	if m.loading {
		t.Error("loading = true after Reset, want false")
	}
	if m.pending {
		t.Error("pending = true after Reset, want false")
	}
}

func TestOptionsApply(t *testing.T) {
	cmds := []Item{Command{Name: "open"}}
	custom := KeyMap{}
	styles := Styles{}

	m := New(
		withSeeded(cmds),
		WithKeyMap(custom),
		WithStyles(styles),
		WithPageSize(7),
		WithPaginatorType(paginator.Arabic),
	)

	if len(m.modes) != 2 {
		t.Errorf("modes not applied: got %d, want 2 (commandMode + searchMode)", len(m.modes))
	}
	if m.paginator.PerPage != 7 {
		t.Errorf("paginator.PerPage = %d, want 7", m.paginator.PerPage)
	}
	if m.paginator.Type != paginator.Arabic {
		t.Errorf("paginator.Type = %v, want Arabic", m.paginator.Type)
	}
}

func TestPageSizeZeroSkipsPaginatorPerPage(t *testing.T) {
	// pageSize=0 means auto-fit, which lands in a later milestone.
	// Until then, it must NOT clobber the paginator's default PerPage.
	m := New(WithPageSize(0))

	if m.pageSize != 0 {
		t.Errorf("pageSize = %d, want 0", m.pageSize)
	}
	if m.paginator.PerPage == 0 {
		t.Error("paginator.PerPage should retain its default when pageSize=0")
	}
}

func TestCommandImplementsDefaultItem(t *testing.T) {
	var _ DefaultItem = Command{Name: "x", Desc: "y"}
}

func TestShortHelpBindings(t *testing.T) {
	m := New()
	bindings := m.ShortHelp()
	if len(bindings) != 3 {
		t.Fatalf("ShortHelp returned %d bindings, want 3", len(bindings))
	}

	// First entry is the synthetic "navigate" combo (no matched keys,
	// just a help label).
	if got := bindings[0].Help(); got.Key != "↑↓" || got.Desc != "navigate" {
		t.Errorf("ShortHelp[0] help = %+v, want ↑↓/navigate", got)
	}
	if got := bindings[1].Help(); got.Desc != "execute" {
		t.Errorf("ShortHelp[1] desc = %q, want execute", got.Desc)
	}
	if got := bindings[2].Help(); got.Desc != "cancel" {
		t.Errorf("ShortHelp[2] desc = %q, want cancel", got.Desc)
	}
}

func TestFullHelpGroups(t *testing.T) {
	groups := New().FullHelp()
	if len(groups) != 3 {
		t.Fatalf("FullHelp groups = %d, want 3", len(groups))
	}
	// Each group should hold ≥1 binding with non-empty help text.
	for i, g := range groups {
		if len(g) == 0 {
			t.Errorf("FullHelp group %d is empty", i)
		}
		for j, b := range g {
			if b.Help().Desc == "" {
				t.Errorf("FullHelp[%d][%d] missing help desc", i, j)
			}
		}
	}
}

func TestMouseWheelMovesCursor(t *testing.T) {
	cmds := []Item{Command{Name: "a"}, Command{Name: "b"}, Command{Name: "c"}}
	m := New(withSeeded(cmds))
	m.Focus()
	m.input.SetValue(">")
	m.cursor = 1

	m, _ = m.Update(tea.MouseWheelMsg{Button: tea.MouseWheelDown})
	if m.cursor != 2 {
		t.Errorf("wheel down: cursor = %d, want 2", m.cursor)
	}
	m, _ = m.Update(tea.MouseWheelMsg{Button: tea.MouseWheelUp})
	m, _ = m.Update(tea.MouseWheelMsg{Button: tea.MouseWheelUp})
	if m.cursor != 0 {
		t.Errorf("wheel up x2: cursor = %d, want 0", m.cursor)
	}
}

func TestWithOnExecuteFiresInline(t *testing.T) {
	cmds := []Item{Command{Name: "Open"}, Command{Name: "Save"}}

	var seen Item
	m := New(
		withSeeded(cmds),
		WithOnExecute(func(item Item) tea.Cmd {
			seen = item
			return func() tea.Msg { return runMarker{id: "hook"} }
		}),
	)
	m.Focus()
	m.input.SetValue(">")
	m.cursor = 1 // "Save"

	_, dispatch := m.Update(arrowKey(tea.KeyEnter))
	if dispatch == nil {
		t.Fatal("Enter returned nil cmd")
	}
	if seen == nil || seen.FilterValue() != "Save" {
		t.Errorf("OnExecute callback saw item %v, want Save", seen)
	}

	// The hook's cmd is batched alongside SelectedMsg; drain the batch
	// and assert both ride out of the same Update tick.
	var sawSelected, sawHook bool
	for _, msg := range drainBatch(dispatch) {
		switch v := msg.(type) {
		case SelectedMsg:
			sawSelected = v.Item.FilterValue() == "Save"
		case runMarker:
			sawHook = v.id == "hook"
		}
	}
	if !sawSelected {
		t.Error("expected SelectedMsg in dispatch batch")
	}
	if !sawHook {
		t.Error("expected OnExecute's runMarker cmd in dispatch batch")
	}
}

// drainBatch invokes cmd and returns every message it produced. Used
// to inspect the contents of a tea.BatchMsg in tests without coupling
// to its internal layout. Tolerates a single-cmd (non-batched) return
// by treating it as a singleton.
func drainBatch(cmd tea.Cmd) []tea.Msg {
	if cmd == nil {
		return nil
	}
	msg := cmd()
	batch, ok := msg.(tea.BatchMsg)
	if !ok {
		return []tea.Msg{msg}
	}
	var msgs []tea.Msg
	for _, c := range batch {
		if c == nil {
			continue
		}
		msgs = append(msgs, c())
	}
	return msgs
}

func TestStylesPromptWrapsLeadingGlyph(t *testing.T) {
	mode := Mode{
		Name:   "tagged",
		Prompt: "» ",
		Items:  func(_ Model, _ string) []Item { return nil },
	}
	m := New(WithModes(mode))
	// A distinctive style we can pattern-match in the rendered output.
	m.Styles.Prompt = lipgloss.NewStyle().Bold(true)
	m, _ = m.Update(tea.WindowSizeMsg{Width: 40, Height: 10})

	out := m.View()
	// lipgloss bold opens with ESC[1m. The prompt glyph "» " must be
	// wrapped by it; without the new Styles.Prompt application this
	// assertion would fail.
	if !strings.Contains(out, "\x1b[1m» ") {
		t.Errorf("View() prompt glyph not styled with Styles.Prompt:\n%q", out)
	}
}

func TestStylesPlaceholderReachesInput(t *testing.T) {
	m := New(WithPlaceholder("type to search"))
	m.Styles.Placeholder = lipgloss.NewStyle().Italic(true)
	m.Focus()
	m, _ = m.Update(tea.WindowSizeMsg{Width: 60, Height: 10})

	out := m.View()
	// Italic opens with ESC[3m. The placeholder text is rendered by
	// the underlying textinput; Styles.Placeholder propagates through
	// SetStyles each View call.
	if !strings.Contains(out, "\x1b[3m") {
		t.Errorf("View() placeholder not styled (no italic escape):\n%q", out)
	}
}

func TestKeyMapNavigateLabelDrivesShortHelp(t *testing.T) {
	m := New()
	m.KeyMap.Navigate = key.NewBinding(
		key.WithKeys("up", "down"),
		key.WithHelp("↑↓", "select"),
	)
	bindings := m.ShortHelp()
	if len(bindings) == 0 || bindings[0].Help().Desc != "select" {
		t.Errorf("ShortHelp()[0].Help().Desc = %q, want %q",
			bindings[0].Help().Desc, "select")
	}
}

// TestNewNoOptionsRendersSafely guards the empty-default mode: with no
// WithModes (and no other options), New() must produce a Model whose
// common methods don't panic. The palette won't be useful — it has no
// items — but it must still render cleanly so hosts can wire it up
// incrementally without crashing.
func TestNewNoOptionsRendersSafely(t *testing.T) {
	m := New()
	m.Focus()
	m, _ = m.Update(tea.WindowSizeMsg{Width: 60, Height: 20})
	m, _ = m.Update(typeKey(t, "a"))

	if got := m.Items(); got != nil {
		t.Errorf("Items() = %v, want nil for empty default mode", got)
	}
	if got := m.Selected(); got != nil {
		t.Errorf("Selected() = %v, want nil when there are no items", got)
	}
	if got := m.Mode().Name; got != "default" {
		t.Errorf("Mode().Name = %q, want %q", got, "default")
	}
	if out := m.View(); out == "" {
		t.Error("View() returned empty string")
	}
}

func TestViewRendersModePrompt(t *testing.T) {
	m := New()
	m, _ = m.Update(tea.WindowSizeMsg{Width: 40, Height: 20})

	out := m.View()
	if !strings.Contains(out, "⣿") {
		t.Errorf("View() missing default prompt glyph ⣿, got:\n%s", out)
	}
}

func TestViewSwapsSpinnerWhenLoading(t *testing.T) {
	m := New()
	m, _ = m.Update(tea.WindowSizeMsg{Width: 40, Height: 20})
	m.loading = true

	out := m.View()
	if strings.Contains(out, "⣿") {
		t.Errorf("View() rendered idle prompt glyph while loading, got:\n%s", out)
	}
	// Dot frames are dense braille blocks — confirm one appears.
	if !strings.ContainsAny(out, "⣾⣽⣻⢿⡿⣟⣯⣷") {
		t.Errorf("View() missing spinner glyph while loading, got:\n%s", out)
	}
}

// The spinner should also run while a debounce is queued so users see
// "I heard you typing" feedback, not just "Search is running."
func TestViewSwapsSpinnerWhilePending(t *testing.T) {
	m := New()
	m, _ = m.Update(tea.WindowSizeMsg{Width: 40, Height: 20})
	m.pending = true

	out := m.View()
	if strings.Contains(out, "⣿") {
		t.Errorf("View() rendered idle prompt glyph while pending, got:\n%s", out)
	}
	if !strings.ContainsAny(out, "⣾⣽⣻⢿⡿⣟⣯⣷") {
		t.Errorf("View() missing spinner glyph while pending, got:\n%s", out)
	}
}

func TestCustomModePrompt(t *testing.T) {
	custom := Mode{
		Name:   "files",
		Prompt: "@ ",
		Match:  func(s string) bool { return strings.HasPrefix(s, "@") },
		Query:  func(s string) string { return strings.TrimPrefix(s, "@") },
		Items:  func(_ Model, _ string) []Item { return nil },
	}
	m := New(WithModes(custom, searchMode()))
	m.input.SetValue("@foo")
	m, _ = m.Update(tea.WindowSizeMsg{Width: 40, Height: 20})

	out := m.View()
	if !strings.Contains(out, "@ ") {
		t.Errorf("View() missing custom mode prompt '@ ', got:\n%s", out)
	}
}

func TestViewIncludesHelp(t *testing.T) {
	m := New(withSeeded([]Item{Command{Name: "open"}}))
	m.input.SetValue(">")
	m, _ = m.Update(tea.WindowSizeMsg{Width: 50, Height: 20})

	out := m.View()
	if !strings.Contains(out, "navigate") {
		t.Errorf("View() missing help text 'navigate', got:\n%s", out)
	}
	if !strings.Contains(out, "execute") {
		t.Errorf("View() missing help text 'execute', got:\n%s", out)
	}
}

func TestWithHelpFalseHidesHelp(t *testing.T) {
	m := New(withSeeded([]Item{Command{Name: "open"}}), WithHelp(false))
	m.input.SetValue(">")
	m, _ = m.Update(tea.WindowSizeMsg{Width: 50, Height: 20})

	out := m.View()
	if strings.Contains(out, "navigate") {
		t.Errorf("View() rendered help despite WithHelp(false), got:\n%s", out)
	}
}

func TestViewRendersInputAndItems(t *testing.T) {
	m := New(withSeeded([]Item{
		Command{Name: "open", Desc: "open it"},
		Command{Name: "save"},
	}))
	m.input.SetValue(">")
	m.width = 40

	out := m.View()

	// Both command titles should appear in the rendered output.
	for _, want := range []string{"open", "save"} {
		if !strings.Contains(out, want) {
			t.Errorf("View() missing %q\n--- output ---\n%s", want, out)
		}
	}
}

func TestViewWithNoItems(t *testing.T) {
	// SearchMode + empty results should render the input inside the
	// container but no items list under it.
	m := New(withSeeded([]Item{Command{Name: "open"}, Command{Name: "save"}}))
	m.input.SetValue("hello")

	out := m.View()
	if !strings.Contains(out, "hello") {
		t.Errorf("View() missing input value, got %q", out)
	}
	for _, name := range []string{"open", "save"} {
		if strings.Contains(out, name) {
			t.Errorf("View() should not render commands in SearchMode, but found %q in:\n%s", name, out)
		}
	}
}

func TestEmptyMessagePaletteDefault(t *testing.T) {
	m := New(WithModes(searchMode()), WithEmptyMessage("Nothing here"))
	m.input.SetValue("foo")
	m, _ = m.Update(tea.WindowSizeMsg{Width: 40, Height: 20})

	out := m.View()
	if !strings.Contains(out, "Nothing here") {
		t.Errorf("View() missing palette-level empty message\n--- output ---\n%s", out)
	}
}

func TestEmptyMessageModeOverridesPalette(t *testing.T) {
	mode := searchMode()
	mode.EmptyMessage = "Mode says nothing"
	m := New(WithModes(mode), WithEmptyMessage("Palette default"))
	m.input.SetValue("foo")
	m, _ = m.Update(tea.WindowSizeMsg{Width: 40, Height: 20})

	out := m.View()
	if !strings.Contains(out, "Mode says nothing") {
		t.Errorf("View() missing per-mode empty message\n--- output ---\n%s", out)
	}
	if strings.Contains(out, "Palette default") {
		t.Errorf("View() should not render palette default when mode overrides\n--- output ---\n%s", out)
	}
}

func TestEmptyMessageSuppressedOnEmptyInput(t *testing.T) {
	m := New(WithModes(searchMode()), WithEmptyMessage("Nothing here"))
	m, _ = m.Update(tea.WindowSizeMsg{Width: 40, Height: 20})

	out := m.View()
	if strings.Contains(out, "Nothing here") {
		t.Errorf("View() should suppress empty message when input is empty\n--- output ---\n%s", out)
	}
}

func TestEmptyMessageSuppressedWhileLoading(t *testing.T) {
	m := New(WithModes(searchMode()), WithEmptyMessage("Nothing here"))
	m.input.SetValue("foo")
	m.loading = true
	m, _ = m.Update(tea.WindowSizeMsg{Width: 40, Height: 20})

	out := m.View()
	if strings.Contains(out, "Nothing here") {
		t.Errorf("View() should suppress empty message while a search is in flight\n--- output ---\n%s", out)
	}
}

func TestEmptyMessageSuppressedDuringFacetCompletion(t *testing.T) {
	mode := Mode{
		Name:         "search",
		EmptyMessage: "Nothing here",
		Items:        func(_ Model, _ string) []Item { return nil },
		Facets:       []Facet{labelFacet},
	}
	m := New(WithModes(mode))
	m, _ = m.Update(tea.WindowSizeMsg{Width: 40, Height: 20})
	m.input.SetValue("label:b")
	if cmd := m.reconcileInputState(); cmd != nil {
		cmd()
	}
	if m.facet == nil {
		t.Fatal("setup: facet completion should be active")
	}

	out := m.View()
	if strings.Contains(out, "Nothing here") {
		t.Errorf("View() should suppress empty message during facet completion\n--- output ---\n%s", out)
	}
}

func TestDefaultKeyMapHasBindings(t *testing.T) {
	km := DefaultKeyMap()
	bindings := []struct {
		name string
		keys []string
	}{
		{"Execute", km.Execute.Keys()},
		{"Cancel", km.Cancel.Keys()},
		{"Down", km.Down.Keys()},
		{"Up", km.Up.Keys()},
		{"NextPage", km.NextPage.Keys()},
		{"PrevPage", km.PrevPage.Keys()},
	}
	for _, b := range bindings {
		if len(b.keys) == 0 {
			t.Errorf("DefaultKeyMap.%s has no keys", b.name)
		}
	}
}
