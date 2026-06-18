package palette

import (
	"context"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
)

// labelFacet is a minimal sync Facet used across detection / Update
// tests.
var labelFacet = Facet{
	Name: "label",
	Items: func(partial string) []Item {
		values := []string{"bug", "build", "docs", "enhancement"}
		out := make([]Item, 0, len(values))
		for _, v := range values {
			if partial == "" || strings.HasPrefix(v, partial) {
				out = append(out, testItem{name: v})
			}
		}
		return out
	},
}

// facetMode wraps SearchMode-style behavior with a facet attached.
func facetMode(facets ...Facet) Mode {
	return Mode{
		Name:   "search",
		Match:  nil,
		Query:  nil,
		Items:  func(m Model, _ string) []Item { return m.Results("search") },
		Facets: facets,
	}
}

func TestDetectFacetToken(t *testing.T) {
	facets := []Facet{labelFacet}
	tests := []struct {
		name    string
		input   string
		cursor  int
		want    bool
		wantPar string
	}{
		{"empty input", "", 0, false, ""},
		{"plain text", "foo", 3, false, ""},
		{"colon but unknown facet", "kind:bug", 8, false, ""},
		{"in token at end", "label:bu", 8, true, "bu"},
		{"in token mid", "label:bug viewport", 8, true, "bug"},
		{"after space", "label:bug ", 10, false, ""},
		{"cursor at colon", "label:bug", 6, true, "bug"},
		{"empty partial", "label:", 6, true, ""},
		{"cursor inside name", "label:bug", 3, true, "bug"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			_, _, _, partial, ok := detectFacetToken(tc.input, tc.cursor, facets)
			if ok != tc.want {
				t.Fatalf("detectFacetToken ok = %v, want %v", ok, tc.want)
			}
			if ok && partial != tc.wantPar {
				t.Errorf("partial = %q, want %q", partial, tc.wantPar)
			}
		})
	}
}

func TestParseFacets(t *testing.T) {
	facets := []Facet{labelFacet, {Name: "author"}}
	text, parsed := ParseFacets("memory leak label:bug author:alice author:bob label: trailing", facets)
	if text != "memory leak trailing" {
		t.Errorf("text = %q, want %q", text, "memory leak trailing")
	}
	if got := parsed["label"]; len(got) != 1 || got[0] != "bug" {
		t.Errorf("label = %v, want [bug]", got)
	}
	if got := parsed["author"]; len(got) != 2 || got[0] != "alice" || got[1] != "bob" {
		t.Errorf("author = %v, want [alice bob]", got)
	}
}

func TestApplyFacetValue(t *testing.T) {
	out, cursor := applyFacetValue("memory leak label:bu trailing", 12, 20, "label", "bug")
	want := "memory leak label:bug trailing"
	if out != want {
		t.Errorf("input = %q, want %q", out, want)
	}
	if cursor != 21 { // 12 + len("label:bug")
		t.Errorf("cursor = %d, want 21", cursor)
	}
}

func TestFacetEntersCompletionOnInput(t *testing.T) {
	m := New(WithModes(facetMode(labelFacet)))
	m.Focus()

	// Drive a keystroke that lands the cursor inside "label:b".
	m.input.SetValue("label:b")
	m.input.SetCursor(7)
	cmd := m.reconcileInputState()
	if m.facet == nil {
		t.Fatal("expected facet completion to activate")
	}
	if m.facet.facet.Name != "label" {
		t.Errorf("facet name = %q, want label", m.facet.facet.Name)
	}
	if m.facet.partial != "b" {
		t.Errorf("partial = %q, want b", m.facet.partial)
	}
	if cmd != nil {
		t.Errorf("expected nil cmd for sync facet, got %T", cmd())
	}

	items := m.Items()
	if len(items) != 2 || items[0].FilterValue() != "bug" || items[1].FilterValue() != "build" {
		got := make([]string, len(items))
		for i, it := range items {
			got[i] = it.FilterValue()
		}
		t.Errorf("Items = %v, want [bug build]", got)
	}
}

func TestFacetExitsOnSpace(t *testing.T) {
	m := New(WithModes(facetMode(labelFacet)))
	m.Focus()
	m.input.SetValue("label:bug")
	m.input.SetCursor(9)
	m.reconcileInputState()
	if m.facet == nil {
		t.Fatal("setup: facet should be active")
	}

	// User types a space after — cursor lands outside the token.
	m.input.SetValue("label:bug ")
	m.input.SetCursor(10)
	m.reconcileInputState()
	if m.facet != nil {
		t.Errorf("expected facet to clear when cursor leaves token; still active for %q", m.facet.facet.Name)
	}
}

func TestFacetEnterInsertsValue(t *testing.T) {
	m := New(WithModes(facetMode(labelFacet)))
	m.Focus()
	m.input.SetValue("memory leak label:bu")
	m.input.SetCursor(20)
	m.reconcileInputState()
	if m.facet == nil {
		t.Fatal("setup: facet should be active")
	}
	// Facet items in this partial: ["bug", "build"]. Cursor starts at 0
	// (bug). Apply.
	m, _ = m.applyFacetCompletion()
	if m.facet != nil {
		t.Error("facet should clear after apply")
	}
	// applyFacetValue appends a trailing space when no whitespace
	// follows, so completion exits cleanly and the user can keep typing.
	if got := m.input.Value(); got != "memory leak label:bug " {
		t.Errorf("input = %q, want %q", got, "memory leak label:bug ")
	}
	if got := m.input.Position(); got != 22 {
		t.Errorf("cursor = %d, want 22", got)
	}
}

func TestFacetEscExitsWithoutInserting(t *testing.T) {
	m := New(WithModes(facetMode(labelFacet)))
	m.Focus()
	m.input.SetValue("label:bu")
	m.input.SetCursor(8)
	m.reconcileInputState()
	if m.facet == nil {
		t.Fatal("setup: facet should be active")
	}

	escMsg := tea.KeyPressMsg{Code: tea.KeyEscape}
	m, _ = m.Update(escMsg)
	if m.facet != nil {
		t.Errorf("facet should clear on Esc; still active")
	}
	if got := m.input.Value(); got != "label:bu" {
		t.Errorf("input mutated by Esc: %q, want %q (unchanged)", got, "label:bu")
	}
}

func TestFacetAsyncResolveLifecycle(t *testing.T) {
	asyncFacet := Facet{
		Name:     "author",
		Debounce: 0,
		Resolve: func(_ context.Context, partial string) tea.Cmd {
			return func() tea.Msg {
				return FacetResultMsg{
					Facet:   "author",
					Partial: partial,
					Results: []Item{testItem{name: partial + "-bot"}},
				}
			}
		},
	}
	m := New(WithModes(facetMode(asyncFacet)))
	m.Focus()
	m.input.SetValue("author:al")
	m.input.SetCursor(9)
	cmd := m.reconcileInputState()
	if m.facet == nil {
		t.Fatal("facet should activate")
	}
	if !m.pending {
		t.Error("expected pending=true while debounce window is open")
	}
	if cmd == nil {
		t.Fatal("expected debounce tick cmd from reconcileInputState")
	}

	dbMsg := findFacetDebounceMsg(t, cmd)

	mm, dispatchCmd := m.Update(dbMsg)
	m = mm
	if m.pending {
		t.Error("expected pending=false after dispatch")
	}
	if !m.loading {
		t.Error("expected loading=true after dispatch")
	}
	if dispatchCmd == nil {
		t.Fatal("expected Resolve cmd")
	}
	resultMsg, ok := dispatchCmd().(FacetResultMsg)
	if !ok {
		t.Fatalf("dispatchCmd produced %T, want FacetResultMsg", dispatchCmd())
	}
	if resultMsg.Facet != "author" || resultMsg.Partial != "al" {
		t.Errorf("result = %+v, want Facet=author Partial=al", resultMsg)
	}

	m, _ = m.Update(resultMsg)
	if m.loading {
		t.Error("expected loading=false after result")
	}
	items := m.Items()
	if len(items) != 1 || items[0].FilterValue() != "al-bot" {
		t.Errorf("Items = %v, want [al-bot]", items)
	}
}

func TestFacetStaleResolveIgnored(t *testing.T) {
	asyncFacet := Facet{
		Name:     "author",
		Resolve:  func(_ context.Context, _ string) tea.Cmd { return nil },
		Debounce: 0,
	}
	m := New(WithModes(facetMode(asyncFacet)))
	m.Focus()
	m.input.SetValue("author:al")
	m.input.SetCursor(9)
	m.reconcileInputState()

	// Result for a different partial — should be dropped.
	stale := FacetResultMsg{Facet: "author", Partial: "old", Results: []Item{testItem{name: "x"}}}
	m, _ = m.Update(stale)
	if got := m.facetResults["author"]; got != nil {
		t.Errorf("stale facet result accepted: %v", got)
	}
}

func TestFacetResetClearsState(t *testing.T) {
	asyncFacet := Facet{
		Name:    "author",
		Resolve: func(_ context.Context, _ string) tea.Cmd { return nil },
	}
	m := New(WithModes(facetMode(asyncFacet)))
	m.Focus()
	m.input.SetValue("author:al")
	m.input.SetCursor(9)
	m.reconcileInputState()
	m.facetResults["author"] = []Item{testItem{name: "x"}}
	m.Reset()
	if m.facet != nil {
		t.Error("Reset should clear facet")
	}
	if len(m.facetResults) != 0 {
		t.Errorf("Reset should clear facetResults, got %v", m.facetResults)
	}
}

func TestFacetHintMovesToFooter(t *testing.T) {
	m := New(WithModes(facetMode(labelFacet)), WithPageSize(4))
	m, _ = m.Update(tea.WindowSizeMsg{Width: 60, Height: 20})
	m.Focus()

	idleLines := strings.Split(m.View(), "\n")

	// Activate facet completion by typing into a facet token.
	m.input.SetValue("label:b")
	m.input.SetCursor(7)
	m.reconcileInputState()
	activeLines := strings.Split(m.View(), "\n")

	if len(idleLines) != len(activeLines) {
		t.Errorf("palette height changed when facet activated: idle=%d active=%d",
			len(idleLines), len(activeLines))
	}
	if !strings.Contains(m.View(), "label:") {
		t.Errorf("expected facet hint 'label:' in footer, got:\n%s", m.View())
	}
}

func TestFacetUpDownNavigatesValues(t *testing.T) {
	m := New(WithModes(facetMode(labelFacet)))
	m.Focus()
	m.input.SetValue("label:b")
	m.input.SetCursor(7)
	m.reconcileInputState()
	// Two items: bug, build.
	m.moveCursor(1)
	if m.facet.cursor != 1 {
		t.Errorf("facet cursor = %d, want 1 after Down", m.facet.cursor)
	}
	if m.cursor != 0 {
		t.Errorf("mode cursor mutated by facet nav: %d, want 0", m.cursor)
	}
	m.moveCursor(1) // wraps to 0
	if m.facet.cursor != 0 {
		t.Errorf("facet cursor = %d, want wrap to 0", m.facet.cursor)
	}
}

// findFacetDebounceMsg extracts a facetDebounceMsg from cmd's output
// whether cmd is raw or batched with spinner.Tick.
func findFacetDebounceMsg(t *testing.T, cmd tea.Cmd) facetDebounceMsg {
	t.Helper()
	msg := cmd()
	if db, ok := msg.(facetDebounceMsg); ok {
		return db
	}
	batch, ok := msg.(tea.BatchMsg)
	if !ok {
		t.Fatalf("cmd produced %T, want facetDebounceMsg or tea.BatchMsg", msg)
	}
	for _, c := range batch {
		if db, ok := c().(facetDebounceMsg); ok {
			return db
		}
	}
	t.Fatal("no facetDebounceMsg in batch")
	return facetDebounceMsg{}
}
