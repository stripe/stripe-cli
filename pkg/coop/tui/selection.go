package tui

import (
	"strings"

	"github.com/stripe/stripe-cli/pkg/coop"
)

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

	// The step is the unit of work and the unit of review: confirming and
	// requesting changes always act on a whole step, never on one task. So the
	// step is also the only cursor target. Tasks stay visible beneath it as
	// status rows and are reachable through the step's detail view.
	var items []navigationItem
	for stepIndex := range m.session.Steps {
		items = append(items, navigationItem{kind: navigationStep, stepIndex: stepIndex})
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

// selectNode points the cursor at a task. Tasks are status rows rather than
// cursor targets, so the selection itself lands on the step that owns the task;
// the cursor index is kept so per-task rendering still tracks the right row.
func (m *Model) selectNode(nodeIndex int) {
	m.selectionCursor = nodeIndex
	stepIndex, ok := m.stepIndexForNode(nodeIndex)
	if !ok {
		m.selected = navigationItem{kind: navigationNode}
		return
	}
	m.expandStep(stepIndex)
	m.selected = navigationItem{kind: navigationStep, stepIndex: stepIndex}
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

// stepCollapsed decides whether a step shows its tasks.
//
// The default is derived from state rather than fixed: only the step in play —
// the one being worked on or awaiting review — is worth spending rows on, and
// an eight-step blueprint otherwise fills the screen with history and things
// that have not started. A manual toggle overrides that, but only until the
// step's own state changes, so a step opened out of curiosity while pending
// cannot silently suppress its own review box later.
func (m Model) stepCollapsed(stepIndex int) bool {
	if override, ok := m.collapseOverride(stepIndex); ok {
		return override
	}
	return !m.stepInPlay(stepIndex)
}

// stepInPlay reports whether a step is the one the session is currently about.
func (m Model) stepInPlay(stepIndex int) bool {
	if m.session == nil || stepIndex < 0 || stepIndex >= len(m.session.Steps) {
		return false
	}
	if m.stepReviewReady(stepIndex) {
		return true
	}
	for _, node := range m.session.Steps[stepIndex].Nodes {
		if node.State == coop.NodeActive {
			return true
		}
	}
	return false
}

// collapseOverride returns a manual toggle if one still applies to the step's
// current state.
func (m Model) collapseOverride(stepIndex int) (bool, bool) {
	if m.collapsedSteps == nil {
		return false, false
	}
	override, ok := m.collapsedSteps[stepIndex]
	if !ok || m.stepStateSignatures[stepIndex] != m.stepStateSignature(stepIndex) {
		return false, false
	}
	return override, true
}

// stepStateSignature summarizes a step's task states, so a manual collapse can
// be dropped the moment the step becomes something else.
func (m Model) stepStateSignature(stepIndex int) string {
	if m.session == nil || stepIndex < 0 || stepIndex >= len(m.session.Steps) {
		return ""
	}
	var b strings.Builder
	for _, node := range m.session.Steps[stepIndex].Nodes {
		b.WriteString(string(node.State))
		b.WriteByte('|')
	}
	return b.String()
}

func (m *Model) recordCollapseOverride(stepIndex int) {
	if m.stepStateSignatures == nil {
		m.stepStateSignatures = map[int]string{}
	}
	m.stepStateSignatures[stepIndex] = m.stepStateSignature(stepIndex)
}

func (m *Model) collapseStep(stepIndex int) {
	if m.collapsedSteps == nil {
		m.collapsedSteps = map[int]bool{}
	}
	m.collapsedSteps[stepIndex] = true
	m.recordCollapseOverride(stepIndex)
	if selectedStep, ok := m.selectedStepIndex(); ok && selectedStep == stepIndex {
		m.selectStep(stepIndex)
	}
}

func (m *Model) expandStep(stepIndex int) {
	if m.collapsedSteps == nil {
		m.collapsedSteps = map[int]bool{}
	}
	m.collapsedSteps[stepIndex] = false
	m.recordCollapseOverride(stepIndex)
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

// selectedStepIndexOrZero is the selected step, or the first one when the
// selection has not resolved to a step yet.
func (m Model) selectedStepIndexOrZero() int {
	if idx, ok := m.selectedStepIndex(); ok {
		return idx
	}
	return 0
}
