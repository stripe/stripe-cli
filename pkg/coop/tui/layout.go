package tui

import (
	"strings"

	"charm.land/lipgloss/v2"
)

func (m Model) renderViewportRegionWithHeight(height int) string {
	if m.width <= 0 || height <= 0 {
		return m.viewport.View()
	}
	hasMoreBelow := m.viewport.YOffset()+height < m.viewport.TotalLineCount()
	if hasMoreBelow && height >= 3 {
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
	footerGap := 2
	viewHeight := m.viewport.Height()
	if m.height > 0 {
		headerH := lipgloss.Height(header) + 1
		footerH := lipgloss.Height(footer)
		available := m.height - headerH - footerH - footerGap
		if floor := m.minViewportRows(); available < floor {
			available = floor
		}
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
