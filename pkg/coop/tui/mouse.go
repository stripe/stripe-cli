package tui

import (
	"strings"

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

	if m.session.ClaimURL != "" {
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
			case navigationChapter:
				targets = append(targets, mouseTarget{y: y, action: mouseActionSelectChapter, index: item.chapterIndex})
			case navigationStep:
				targets = append(targets, mouseTarget{y: y, action: mouseActionSelectStep, index: item.stepIndex})
			}
		}
	}
	return targets
}

func (m Model) navigationContentLines() map[int]navigationItem {
	result := map[int]navigationItem{}
	if m.session == nil {
		return result
	}

	line := 0
	stepIdx := 0

	for chIdx, ch := range m.session.Chapters {
		chapterItem := navigationItem{kind: navigationChapter, chapterIndex: chIdx}
		chapterSelected := m.navigationItemSelected(chapterItem)
		line++ // blank line before chapter
		result[line] = chapterItem
		line++ // chapter line
		line++ // rule line
		if m.expanded && chapterSelected {
			if detail := m.renderDetail(); detail != "" {
				line += lipgloss.Height(detail)
			}
		}

		if m.chapterCollapsed(chIdx) {
			stepIdx += len(ch.Nodes)
			continue
		}
		chapterReviewReady := m.chapterReviewReady(chIdx)
		for _, node := range ch.Nodes {
			stepItem := navigationItem{kind: navigationStep, stepIndex: stepIdx, chapterIndex: chIdx}
			stepSelected := m.navigationItemSelected(stepItem)
			result[line] = stepItem
			line += lipgloss.Height(m.renderStepLine(node, stepIdx, chapterReviewReady, stepSelected))
			if m.expanded && stepSelected {
				if detail := m.renderDetail(); detail != "" {
					line += lipgloss.Height(detail)
				}
			}
			stepIdx++
		}
	}
	return result
}

func (m Model) completionSuggestionLines() map[int]int {
	result := map[int]int{}
	suggestions := m.getCompletionSuggestions()
	if len(suggestions) == 0 {
		return result
	}
	lines := strings.Split(m.renderCompletionBody(), "\n")
	for suggestionIdx, suggestion := range suggestions {
		for lineIdx, line := range lines {
			if _, exists := result[lineIdx]; exists {
				continue
			}
			if strings.Contains(line, suggestion.title) {
				result[lineIdx] = suggestionIdx
				break
			}
		}
	}
	return result
}

func (m Model) handleMouseAction(msg mouseActionMsg) (tea.Model, tea.Cmd) {
	switch msg.action {
	case mouseActionSelectStep:
		if m.session == nil || msg.index < 0 || msg.index >= m.session.TotalSteps() {
			return m, nil
		}
		m.selectStep(msg.index)
		m.userMoved = true
		m.resizeViewport()
		m.syncViewport()
		return m, nil
	case mouseActionSelectChapter:
		if m.session == nil || msg.index < 0 || msg.index >= len(m.session.Chapters) {
			return m, nil
		}
		m.selectChapter(msg.index)
		m.userMoved = true
		m.resizeViewport()
		m.syncViewport()
		return m, nil
	case mouseActionSelectCompletion:
		suggestions := m.getCompletionSuggestions()
		if msg.index < 0 || msg.index >= len(suggestions) {
			return m, nil
		}
		m.cursor = msg.index
		m.syncViewport()
		return m.handleEnter()
	case mouseActionOpenClaim:
		if m.session != nil && m.session.ClaimURL != "" {
			openBrowser(m.session.ClaimURL)
		}
		return m, nil
	default:
		return m, nil
	}
}
