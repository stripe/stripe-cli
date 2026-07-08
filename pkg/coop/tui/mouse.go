package tui

import (
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
)

type mouseTarget struct {
	y      int
	action mouseAction
	index  int
}

func (m Model) mouseActionFor(mouse tea.Mouse) (mouseActionMsg, bool) {
	if mouse.Button != tea.MouseLeft {
		return mouseActionMsg{}, false
	}
	for _, target := range m.mouseTargets() {
		if mouse.Y == target.y {
			return mouseActionMsg{action: target.action, index: target.index}, true
		}
	}
	return mouseActionMsg{}, false
}

func (m Model) mouseTargets() []mouseTarget {
	var targets []mouseTarget
	if m.session == nil {
		return targets
	}

	if m.sandboxClaimLink() != "" {
		targets = append(targets, mouseTarget{y: lipgloss.Height(m.renderHeader()) - 1, action: mouseActionOpenClaim})
	}

	headerHeight := lipgloss.Height(m.renderHeader())
	viewportTop := headerHeight
	viewportBottom := viewportTop + m.viewport.Height()
	offset := m.viewport.YOffset()

	if m.session.IsComplete() {
		for line, suggestion := range m.completionSuggestionLines() {
			y := viewportTop + line - offset
			if y >= viewportTop && y < viewportBottom {
				targets = append(targets, mouseTarget{y: y, action: mouseActionSelectCompletion, index: suggestion})
			}
		}
		return targets
	}

	for line, item := range m.navigationContentLines() {
		y := viewportTop + line - offset
		if y >= viewportTop && y < viewportBottom {
			switch item.kind {
			case navigationStep:
				targets = append(targets, mouseTarget{y: y, action: mouseActionSelectStep, index: item.stepIndex})
			case navigationNode:
				targets = append(targets, mouseTarget{y: y, action: mouseActionSelectNode, index: item.nodeIndex})
			}
		}
	}
	return targets
}

func (m Model) navigationContentLines() map[int]navigationItem {
	return m.renderStepOutline().navigationLine
}

func (m Model) completionSuggestionLines() map[int]int {
	return m.renderCompletionBodyWithLines().suggestionLines
}

func (m Model) handleMouseAction(msg mouseActionMsg) (tea.Model, tea.Cmd) {
	switch msg.action {
	case mouseActionSelectNode:
		if m.rejecting {
			return m, nil
		}
		if m.session == nil || msg.index < 0 || msg.index >= m.session.TotalNodes() {
			return m, nil
		}
		m.selectNode(msg.index)
		m.userMoved = true
		m.resizeViewport()
		m.syncViewport()
		return m, nil
	case mouseActionSelectStep:
		if m.rejecting {
			return m, nil
		}
		if m.session == nil || msg.index < 0 || msg.index >= len(m.session.Steps) {
			return m, nil
		}
		m.selectStep(msg.index)
		m.userMoved = true
		m.resizeViewport()
		m.syncViewport()
		return m, nil
	case mouseActionSelectCompletion:
		suggestions := m.getCompletionSuggestions()
		if msg.index < 0 || msg.index >= len(suggestions) {
			return m, nil
		}
		m.selectionCursor = msg.index
		m.syncViewport()
		return m.handleEnter()
	case mouseActionOpenClaim:
		if claimURL := m.sandboxClaimLink(); claimURL != "" {
			return m, openBrowserCmd(claimURL)
		}
		return m, nil
	default:
		return m, nil
	}
}
