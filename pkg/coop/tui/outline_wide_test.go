package tui

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/stretchr/testify/require"
)

// TestSplitWorkspaceOutlineFitsLeftColumn guards against the split-workspace
// regression where the outline's divider rules and titles were sized to the
// full terminal width and wrapped into a broken multi-line mess inside the
// narrow left column.
func TestSplitWorkspaceOutlineFitsLeftColumn(t *testing.T) {
	m := testModel()
	updated, _ := m.Update(tea.WindowSizeMsg{Width: 140, Height: 50})
	m = updated.(Model)

	require.True(t, m.useSplitWorkspace(), "expected split workspace at width 140")

	leftW := m.width / 3
	if leftW > 48 {
		leftW = 48
	}

	navModel := m
	navModel.outlineWidthOverride = leftW
	out := navModel.renderStepOutline().content

	for i, ln := range strings.Split(out, "\n") {
		require.LessOrEqualf(t, lipgloss.Width(ln), leftW,
			"outline line %d overflows left column (%d): %q", i, leftW, ln)
	}

	// Sanity: the full split render still produces both columns and does not panic.
	require.NotEmpty(t, m.renderSplitWorkspace())
}
