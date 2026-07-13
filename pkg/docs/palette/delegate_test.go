package palette

import (
	"bytes"
	"flag"
	"io"
	"os"
	"path/filepath"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"
)

var updateGolden = flag.Bool("update", false, "regenerate golden files in testdata/")

type plainItem struct{ name string }

func (p plainItem) FilterValue() string { return p.name }

type fullItem struct {
	title string
	desc  string
}

func (f fullItem) FilterValue() string { return f.title }
func (f fullItem) Title() string       { return f.title }
func (f fullItem) Description() string { return f.desc }

func TestDefaultDelegateRender(t *testing.T) {
	cases := []struct {
		name   string
		item   Item
		index  int
		cursor int
		width  int
		setup  func(*DefaultDelegate)
	}{
		{
			name:   "selected_with_desc",
			item:   fullItem{title: "Open file", desc: "Open a file in the editor"},
			index:  0,
			cursor: 0,
			width:  40,
		},
		{
			name:   "unselected_with_desc",
			item:   fullItem{title: "Open file", desc: "Open a file in the editor"},
			index:  1,
			cursor: 0,
			width:  40,
		},
		{
			name:   "selected_without_desc",
			item:   fullItem{title: "Save", desc: ""},
			index:  0,
			cursor: 0,
			width:  40,
		},
		{
			name:   "narrow_truncated",
			item:   fullItem{title: "Open a very long command name", desc: "with a description that also overflows"},
			index:  0,
			cursor: 0,
			width:  16,
		},
		{
			name:   "wide_no_truncation",
			item:   fullItem{title: "Open file", desc: "Short desc"},
			index:  0,
			cursor: 0,
			width:  120,
		},
		{
			name:   "plain_item_filter_value",
			item:   plainItem{name: "raw-result"},
			index:  0,
			cursor: 0,
			width:  40,
		},
		{
			name:   "show_description_off",
			item:   fullItem{title: "Open file", desc: "hidden in this mode"},
			index:  0,
			cursor: 0,
			width:  40,
			setup: func(d *DefaultDelegate) {
				d.ShowDescription = false
			},
		},
		{
			name:   "unknown_width_no_truncation",
			item:   fullItem{title: "Open a very long command name", desc: "with a long description"},
			index:  0,
			cursor: 0,
			width:  0, // width unset: render full text, no ellipsis
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			d := NewDefaultDelegate()
			if tc.setup != nil {
				tc.setup(&d)
			}
			m := New()
			// Simulate the render context the palette's View loop sets
			// up before each delegate call: renderRow is the visible-row
			// index of the selected item (page-local), renderWidth is
			// the available column width for the row.
			m.cursor = tc.cursor
			m.renderRow = tc.cursor
			m.renderWidth = tc.width

			var buf bytes.Buffer
			d.Render(&buf, m, tc.index, tc.item)

			// Golden files store plaintext (ANSI escape codes stripped) so
			// they remain readable in diffs and stable across lipgloss
			// styling tweaks. Layout is what we're locking in here.
			got := ansi.Strip(buf.String())

			path := filepath.Join("testdata", tc.name+".golden")
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
			if got != string(want) {
				t.Errorf("render mismatch\n--- got ---\n%s\n--- want ---\n%s", got, want)
			}
		})
	}
}

// captureDelegate is an external-style ItemDelegate that records each
// Render call's view of the Model — the page-local selection check,
// the row width, and the absolute Selected item / cursor. Used to
// verify the palette's render-context contract for third-party
// delegates.
type captureDelegate struct {
	width       int
	selectedAt  []int // visible indices where IsSelected returned true
	cursor      int
	selectedVal string
}

func (d *captureDelegate) Height() int                        { return 1 }
func (d *captureDelegate) Spacing() int                       { return 0 }
func (d *captureDelegate) Update(_ tea.Msg, _ *Model) tea.Cmd { return nil }
func (d *captureDelegate) Render(_ io.Writer, m Model, i int, _ Item) {
	d.width = m.Width()
	d.cursor = m.Cursor()
	if sel := m.Selected(); sel != nil {
		d.selectedVal = sel.FilterValue()
	}
	if m.IsSelected(i) {
		d.selectedAt = append(d.selectedAt, i)
	}
}

func TestRenderContextExposesSelectionAndWidth(t *testing.T) {
	items := []Item{
		Command{Name: "alpha"},
		Command{Name: "bravo"},
		Command{Name: "charlie"},
		Command{Name: "delta"},
		Command{Name: "echo"},
	}
	mode := Mode{
		Name:  "letters",
		Items: func(_ Model, _ string) []Item { return items },
	}

	d := &captureDelegate{}
	m := New(
		WithModes(mode),
		WithDelegate(d),
		WithPageSize(2),
		WithHelp(false),
	)
	m, _ = m.Update(tea.WindowSizeMsg{Width: 60, Height: 20})

	// Cursor on page 2 (third page; pageSize=2, items 0/1, 2/3, 4).
	m.cursor = 4
	_ = m.View()

	if d.cursor != 4 {
		t.Errorf("Cursor() = %d, want 4 (absolute)", d.cursor)
	}
	if d.selectedVal != "echo" {
		t.Errorf("Selected().FilterValue() = %q, want %q", d.selectedVal, "echo")
	}
	if d.width <= 0 {
		t.Errorf("Width() = %d, want >0", d.width)
	}
	// On page 2 we render a single row (item "echo") at visible index 0.
	if len(d.selectedAt) != 1 || d.selectedAt[0] != 0 {
		t.Errorf("IsSelected fired at %v, want [0]", d.selectedAt)
	}

	// Now flip to page 0 — the cursor is no longer in this page, so
	// IsSelected should never fire.
	d.selectedAt = nil
	m.cursor = 0
	_ = m.View()
	if len(d.selectedAt) != 1 || d.selectedAt[0] != 0 {
		t.Errorf("page 0 IsSelected fired at %v, want [0]", d.selectedAt)
	}

	// IsSelected returns false outside any Render call.
	if m.IsSelected(0) {
		t.Error("IsSelected(0) returned true outside Render context")
	}
}
