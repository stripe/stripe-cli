package tui

import "github.com/stripe/stripe-cli/pkg/coop"

type navigationKind int

const (
	navigationNode navigationKind = iota
	navigationStep
)

type navigationItem struct {
	kind      navigationKind
	nodeIndex int
	stepIndex int
}

func (m Model) navigationItems() []navigationItem {
	if m.session == nil || m.session.IsComplete() {
		return nil
	}

	var items []navigationItem
	nodeIndex := 0
	for stepIndex, step := range m.session.Steps {
		items = append(items, navigationItem{kind: navigationStep, stepIndex: stepIndex})
		if m.stepCollapsed(stepIndex) {
			nodeIndex += len(step.Nodes)
			continue
		}
		for range step.Nodes {
			items = append(items, navigationItem{kind: navigationNode, nodeIndex: nodeIndex, stepIndex: stepIndex})
			nodeIndex++
		}
	}
	return items
}

func (m Model) selectedNavigationIndex() int {
	items := m.navigationItems()
	if len(items) == 0 {
		return -1
	}
	for i, item := range items {
		if m.navigationItemSelected(item) {
			return i
		}
	}
	if m.selected.kind == navigationNode {
		if stepIndex, ok := m.stepIndexForNode(m.selectionCursor); ok {
			for i, item := range items {
				if item.kind == navigationStep && item.stepIndex == stepIndex {
					return i
				}
			}
		}
	}
	return 0
}

func (m *Model) ensureValidNavigationSelection() {
	items := m.navigationItems()
	if len(items) == 0 {
		return
	}
	for _, item := range items {
		if m.navigationItemSelected(item) {
			return
		}
	}
	idx := m.selectedNavigationIndex()
	if idx < 0 || idx >= len(items) {
		idx = 0
	}
	m.selectNavigationItem(items[idx])
}

func (m Model) navigationItemSelected(item navigationItem) bool {
	switch item.kind {
	case navigationStep:
		return m.selected.kind == navigationStep && m.selected.stepIndex == item.stepIndex
	case navigationNode:
		return m.selected.kind == navigationNode && m.selectionCursor == item.nodeIndex
	default:
		return false
	}
}

func (m *Model) selectNavigationItem(item navigationItem) {
	switch item.kind {
	case navigationStep:
		m.selectStep(item.stepIndex)
	case navigationNode:
		m.selectNode(item.nodeIndex)
	}
}

func (m *Model) selectNode(nodeIndex int) {
	m.selected = navigationItem{kind: navigationNode}
	m.selectionCursor = nodeIndex
	if stepIndex, ok := m.stepIndexForNode(nodeIndex); ok {
		m.expandStep(stepIndex)
	}
}

func (m *Model) selectStep(stepIndex int) {
	m.selected = navigationItem{kind: navigationStep, stepIndex: stepIndex}
	if nodeIndex := firstNodeIndexInStep(m.session, stepIndex); nodeIndex >= 0 {
		m.selectionCursor = nodeIndex
	}
}

func (m Model) selectedNodeIndex() (int, bool) {
	if m.selected.kind != navigationNode {
		return 0, false
	}
	return m.selectionCursor, true
}

func (m Model) selectedStepIndex() (int, bool) {
	switch m.selected.kind {
	case navigationStep:
		return m.selected.stepIndex, true
	case navigationNode:
		return m.stepIndexForNode(m.selectionCursor)
	default:
		return 0, false
	}
}

func (m Model) stepCollapsed(stepIndex int) bool {
	return m.collapsedSteps != nil && m.collapsedSteps[stepIndex]
}

func (m *Model) collapseStep(stepIndex int) {
	if m.collapsedSteps == nil {
		m.collapsedSteps = map[int]bool{}
	}
	m.collapsedSteps[stepIndex] = true
	if selectedStep, ok := m.selectedStepIndex(); ok && selectedStep == stepIndex {
		m.selectStep(stepIndex)
	}
}

func (m *Model) expandStep(stepIndex int) {
	if m.collapsedSteps == nil {
		return
	}
	delete(m.collapsedSteps, stepIndex)
}

func (m *Model) collapseSelectedStep() bool {
	stepIndex, ok := m.selectedStepIndex()
	if !ok {
		return false
	}
	if m.selected.kind == navigationNode {
		m.selectStep(stepIndex)
		return true
	}
	if !m.stepCollapsed(stepIndex) {
		m.collapseStep(stepIndex)
		return true
	}
	return false
}

func (m *Model) expandSelectedStep() bool {
	if m.selected.kind != navigationStep {
		return false
	}
	if m.stepCollapsed(m.selected.stepIndex) {
		m.expandStep(m.selected.stepIndex)
		return true
	}
	return false
}

func (m Model) stepIndexForNode(nodeIndex int) (int, bool) {
	if m.session == nil || nodeIndex < 0 {
		return 0, false
	}
	step := 0
	for stepIndex := range m.session.Steps {
		for range m.session.Steps[stepIndex].Nodes {
			if step == nodeIndex {
				return stepIndex, true
			}
			step++
		}
	}
	return 0, false
}

func firstNodeIndexInStep(session *coop.Session, stepIndex int) int {
	if session == nil || stepIndex < 0 || stepIndex >= len(session.Steps) {
		return -1
	}
	nodeIndex := 0
	for i := range session.Steps {
		for range session.Steps[i].Nodes {
			if i == stepIndex {
				return nodeIndex
			}
			nodeIndex++
		}
	}
	return -1
}
