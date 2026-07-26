package tui

import (
	"strings"

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
	view := m.viewport.View()
	rendered := lipgloss.NewStyle().
		Width(m.width).
		Height(height).
		MaxHeight(height).
		Render(view)
	return rendered
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

func closeOpenBoxAtViewportBoundary(s string) string {
	if !strings.Contains(s, "╭") || strings.Contains(s, "╰") {
		return s
	}
	lines := strings.Split(s, "\n")
	if len(lines) == 0 {
		return s
	}
	topLine := -1
	bottomLine := -1
	for i, line := range lines {
		if strings.Contains(line, "╭") && strings.Contains(line, "╮") {
			topLine = i
		}
		if strings.Contains(line, "╰") && strings.Contains(line, "╯") {
			bottomLine = i
		}
	}
	if topLine == -1 || bottomLine > topLine || topLine >= len(lines)-1 {
		return s
	}
	top := lines[topLine]
	bottom := strings.NewReplacer("╭", "╰", "╮", "╯").Replace(top)
	lines[len(lines)-1] = bottom
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
