package palette

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"
)

// TestViewGoldens locks in palette.View() output for representative
// states. Goldens store plaintext (ANSI escapes stripped) so diffs
// stay readable and style tweaks don't break layout assertions.
//
// Regenerate with: go test ./palette/... -run TestViewGoldens -update
func TestViewGoldens(t *testing.T) {
	const width, height = 60, 20

	demoCommands := []Item{
		Command{Name: "Open file", Desc: "open a file in the editor"},
		Command{Name: "Save", Desc: "save the current buffer"},
		Command{Name: "Open in new tab", Desc: "open in a new tab"},
		Command{Name: "Quit", Desc: "exit the program"},
	}

	demoResults := []Item{
		testItem{name: "result one"},
		testItem{name: "result two"},
		testItem{name: "result three"},
	}

	manyItems := make([]Item, 10)
	for i := range manyItems {
		manyItems[i] = Command{Name: string(rune('a' + i))}
	}

	cases := []struct {
		name  string
		setup func() Model
	}{
		{
			name: "empty",
			setup: func() Model {
				m := New()
				m, _ = m.Update(tea.WindowSizeMsg{Width: width, Height: height})
				return m
			},
		},
		{
			name: "command_all",
			setup: func() Model {
				m := New(withSeeded(demoCommands), WithPageSize(4))
				m.input.SetValue(">")
				m, _ = m.Update(tea.WindowSizeMsg{Width: width, Height: height})
				return m
			},
		},
		{
			name: "command_filtered",
			setup: func() Model {
				m := New(withSeeded(demoCommands), WithPageSize(4))
				m.input.SetValue(">open")
				m, _ = m.Update(tea.WindowSizeMsg{Width: width, Height: height})
				return m
			},
		},
		{
			name: "search_loading",
			setup: func() Model {
				m := New(WithModes(searchMode()), WithPageSize(3))
				m.input.SetValue("foo")
				m, _ = m.Update(tea.WindowSizeMsg{Width: width, Height: height})
				m.loading = true
				return m
			},
		},
		{
			name: "search_with_results",
			setup: func() Model {
				m := New(WithModes(searchMode()), WithPageSize(3))
				m.input.SetValue("foo")
				m.results["search"] = demoResults
				m, _ = m.Update(tea.WindowSizeMsg{Width: width, Height: height})
				return m
			},
		},
		{
			name: "search_empty_results",
			setup: func() Model {
				m := New(WithModes(searchMode()), WithPageSize(3))
				m.input.SetValue("foo")
				m.results["search"] = []Item{} // explicitly empty
				m, _ = m.Update(tea.WindowSizeMsg{Width: width, Height: height})
				return m
			},
		},
		{
			name: "search_empty_with_message",
			setup: func() Model {
				m := New(WithModes(searchMode()), WithPageSize(3), WithEmptyMessage("No matches"))
				m.input.SetValue("foo")
				m.results["search"] = []Item{}
				m, _ = m.Update(tea.WindowSizeMsg{Width: width, Height: height})
				return m
			},
		},
		{
			name: "pagination_page_two_of_three",
			setup: func() Model {
				m := New(withSeeded(manyItems), WithPageSize(4))
				m.input.SetValue(">")
				m, _ = m.Update(tea.WindowSizeMsg{Width: width, Height: height})
				m.cursor = 4 // first item of page 2 (pages: [0-3],[4-7],[8-9])
				return m
			},
		},
		{
			name: "help_disabled",
			setup: func() Model {
				m := New(withSeeded(demoCommands), WithPageSize(4), WithHelp(false))
				m.input.SetValue(">")
				m, _ = m.Update(tea.WindowSizeMsg{Width: width, Height: height})
				return m
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			m := tc.setup()
			got := ansi.Strip(m.View())

			path := filepath.Join("testdata", "view_"+tc.name+".golden")
			if *updateGolden {
				if err := os.MkdirAll("testdata", 0o755); err != nil {
					t.Fatalf("mkdir testdata: %v", err)
				}
				if err := os.WriteFile(path, []byte(got), 0o644); err != nil {
					t.Fatalf("write golden: %v", err)
				}
				return
			}

			want, err := os.ReadFile(path)
			if err != nil {
				t.Fatalf("read golden (run with -update to create): %v", err)
			}
			if !bytes.Equal([]byte(got), want) {
				t.Errorf("View mismatch for %s\n--- got ---\n%s\n--- want ---\n%s", tc.name, got, want)
			}
		})
	}
}
