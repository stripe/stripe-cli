package tui

import (
	"strings"

	"github.com/charmbracelet/x/ansi"

	"charm.land/lipgloss/v2"
)

// viewportIndicatorRows is what the overflow indicator costs: a blank line and
// its own row.
const viewportIndicatorRows = 2

// viewportFooterGap is the blank space between the scrolling region and the
// pinned footer.
const viewportFooterGap = 2

// viewportRegionHeight is how many rows the scrolling region actually gets.
//
// This is the single definition. It used to be computed twice — once here when
// drawing and once in resizeViewport when sizing — with different arithmetic,
// so the viewport believed it was two rows taller than the region it was drawn
// into. Anything reasoning about what is on screen, EnsureVisible above all,
// was working from the wrong number, and the difference showed up as dead rows
// above the footer.
func (m Model) viewportRegionHeight(header, footer string) int {
	available := m.height - (lipgloss.Height(header) + 1) - lipgloss.Height(footer) - viewportFooterGap
	if floor := m.minViewportRows(); available < floor {
		return floor
	}
	return available
}

func (m Model) renderViewportRegionWithHeight(height int) string {
	if m.width <= 0 || height <= 0 {
		return m.viewport.View()
	}
	hasMoreBelow := m.viewport.YOffset()+height < m.viewport.TotalLineCount()
	// The indicator costs a blank line plus its own row. At a three-row region
	// that left a single body row, which landed on the outline's leading blank
	// and hid the step entirely — pointing at content the user could no longer
	// see. Below four rows the content wins.
	if hasMoreBelow && height >= 4 {
		vp := m.viewport
		vp.SetHeight(height - 2)
		body := lipgloss.NewStyle().
			Width(m.width).
			Height(height - 2).
			MaxHeight(height - 2).
			Render(vp.View())
		body = closeOpenBoxAtViewportBoundary(body)
		indicator := m.renderMoreBelowIndicator()
		return strings.Join([]string{body, "", indicator}, "\n")
	}
	// Same tidy on this path. It only ran when the overflow indicator was being
	// drawn, so the shortest viewports — the ones most likely to cut a card in
	// half — were the ones left showing the fragment.
	view := closeOpenBoxAtViewportBoundary(m.viewport.View())
	return lipgloss.NewStyle().
		Width(m.width).
		Height(height).
		MaxHeight(height).
		Render(view)
}

func (m Model) renderMoreBelowIndicator() string {
	// Matches the outline's "▼ N more below" so the two affordances teach the
	// same pattern rather than unlearning each other two lines apart.
	label := m.theme.MutedStyle.Render("▼ scroll for more")
	width := m.outlineRuleWidth()
	if width < lipgloss.Width(label) {
		width = lipgloss.Width(label)
	}
	centered := lipgloss.PlaceHorizontal(width, lipgloss.Center, label)
	return lipgloss.NewStyle().
		Width(m.width).
		MaxWidth(m.width).
		Render(strings.Repeat(" ", rowCursorWidth) + centered)
}

// closeOpenBoxAtViewportBoundary tidies a card the viewport cut in half.
//
// Two ways it used to leave a mess. It gave up the moment any box had closed
// earlier in the frame, so in the split workspace — where the nav column draws
// a card under every step — it never ran at all. And when the cut landed on a
// card's top border it returned the frame untouched, leaving a lone "╭────╮"
// hanging over blank space with no body and no bottom: the most broken-looking
// thing the layout could produce.
//
// So: find the box that is still open at the end, and either close it or, when
// there is not enough room left to show anything of it, drop it.
// closeDanglingBoxBody closes a card whose top border is scrolled out of view.
func closeDanglingBoxBody(lines []string) string {
	last := len(lines) - 1
	for last >= 0 && strings.TrimSpace(ansi.Strip(lines[last])) == "" {
		last--
	}
	if last < 0 {
		return strings.Join(lines, "\n")
	}
	left, right, ok := boxSideColumns(ansi.Strip(lines[last]))
	if !ok {
		return strings.Join(lines, "\n")
	}
	// Keep whatever sits left of the card — the timeline rail — with its own
	// styling, rather than replacing the whole row with spaces and dropping the
	// rail for one line.
	prefix := ansi.Truncate(lines[last], left, "")
	lines[last] = prefix + "╰" + strings.Repeat("─", right-left-1) + "╯"
	return strings.Join(lines, "\n")
}

// boxSideColumns finds a card row's own left and right borders. The timeline
// rail is drawn with the same glyph, so the leftmost one is not necessarily the
// card's — the card's is the one at its indent.
func boxSideColumns(plain string) (int, int, bool) {
	var columns []int
	for i, r := range []rune(plain) {
		if string(r) == boxSide {
			columns = append(columns, i)
		}
	}
	if len(columns) < 2 {
		return 0, 0, false
	}
	left, right := columns[len(columns)-2], columns[len(columns)-1]
	if right-left < 2 {
		return 0, 0, false
	}
	return left, right, true
}

func closeOpenBoxAtViewportBoundary(s string) string {
	lines := strings.Split(s, "\n")
	topLine := -1
	for i, line := range lines {
		switch {
		case strings.Contains(line, "╭") && strings.Contains(line, "╮"):
			topLine = i
		case strings.Contains(line, "╰") && strings.Contains(line, "╯"):
			topLine = -1
		}
	}
	if topLine == -1 {
		// The window can start below a card's top border, in which case there
		// is no "╭" to find and the card runs to the bottom of the region as a
		// pair of side borders with nothing closing them. Close it from the
		// geometry of the last body row.
		return closeDanglingBoxBody(lines)
	}

	// A card needs its top, at least one row of content, and its bottom. With
	// less than that there is nothing worth showing, so drop the fragment and
	// leave the rows blank rather than draw a box that says nothing.
	const minVisibleBoxRows = 3
	if len(lines)-topLine < minVisibleBoxRows {
		for i := topLine; i < len(lines); i++ {
			lines[i] = ""
		}
		return strings.Join(lines, "\n")
	}

	lines[len(lines)-1] = strings.NewReplacer("╭", "╰", "╮", "╯").Replace(lines[topLine])
	return strings.Join(lines, "\n")
}

func (m Model) renderPinnedViewport(header, footer string) string {
	footerGap := viewportFooterGap
	viewHeight := m.viewport.Height()
	if m.height > 0 {
		available := m.viewportRegionHeight(header, footer)
		if viewHeight <= 0 || viewHeight > available {
			viewHeight = available
		}
	}
	view := m.renderViewportRegionWithHeight(viewHeight)
	rendered := header + "\n" + view + strings.Repeat("\n", footerGap) + footer
	if m.height <= 0 {
		return rendered
	}
	if pad := m.height - lipgloss.Height(rendered); pad > 0 {
		rendered = header + "\n" + view + strings.Repeat("\n", footerGap+pad) + footer
	}
	return rendered
}

// minViewportRows is the floor the outline keeps so the user does not lose
// their place in the blueprint. While they are typing feedback they are not
// navigating, so the editor outranks the floor and it drops back to one row.
func (m Model) minViewportRows() int {
	if m.rejecting {
		return 1
	}
	return minViewportHeight
}

func (m Model) footerHeightBudget() int {
	if m.height <= 0 {
		return 0
	}
	headerHeight := lipgloss.Height(m.renderHeader())
	budget := m.height - headerHeight - m.minViewportRows() - 2 - terminalScrollGuard
	if budget < 1 {
		return 1
	}
	return budget
}
