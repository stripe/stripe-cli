package tui

import "github.com/stripe/stripe-cli/pkg/coop"

type navigationKind int

const (
	navigationStep navigationKind = iota
	navigationChapter
)

type navigationItem struct {
	kind         navigationKind
	stepIndex    int
	chapterIndex int
}

func (m Model) navigationItems() []navigationItem {
	if m.session == nil || m.session.IsComplete() {
		return nil
	}

	var items []navigationItem
	stepIndex := 0
	for chapterIndex, chapter := range m.session.Chapters {
		items = append(items, navigationItem{kind: navigationChapter, chapterIndex: chapterIndex})
		if m.chapterCollapsed(chapterIndex) {
			stepIndex += len(chapter.Nodes)
			continue
		}
		for range chapter.Nodes {
			items = append(items, navigationItem{kind: navigationStep, stepIndex: stepIndex, chapterIndex: chapterIndex})
			stepIndex++
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
	if m.selected.kind == navigationStep {
		if chapterIndex, ok := m.chapterIndexForStep(m.cursor); ok {
			for i, item := range items {
				if item.kind == navigationChapter && item.chapterIndex == chapterIndex {
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
	case navigationChapter:
		return m.selected.kind == navigationChapter && m.selected.chapterIndex == item.chapterIndex
	case navigationStep:
		return m.selected.kind == navigationStep && m.cursor == item.stepIndex
	default:
		return false
	}
}

func (m *Model) selectNavigationItem(item navigationItem) {
	switch item.kind {
	case navigationChapter:
		m.selectChapter(item.chapterIndex)
	case navigationStep:
		m.selectStep(item.stepIndex)
	}
}

func (m *Model) selectStep(stepIndex int) {
	m.selected = navigationItem{kind: navigationStep}
	m.cursor = stepIndex
	if chapterIndex, ok := m.chapterIndexForStep(stepIndex); ok {
		m.expandChapter(chapterIndex)
	}
}

func (m *Model) selectChapter(chapterIndex int) {
	m.selected = navigationItem{kind: navigationChapter, chapterIndex: chapterIndex}
	if stepIndex := firstStepIndexInChapter(m.session, chapterIndex); stepIndex >= 0 {
		m.cursor = stepIndex
	}
}

func (m Model) selectedStepIndex() (int, bool) {
	if m.selected.kind != navigationStep {
		return 0, false
	}
	return m.cursor, true
}

func (m Model) selectedChapterIndex() (int, bool) {
	switch m.selected.kind {
	case navigationChapter:
		return m.selected.chapterIndex, true
	case navigationStep:
		return m.chapterIndexForStep(m.cursor)
	default:
		return 0, false
	}
}

func (m Model) chapterCollapsed(chapterIndex int) bool {
	return m.collapsedChapters != nil && m.collapsedChapters[chapterIndex]
}

func (m *Model) collapseChapter(chapterIndex int) {
	if m.collapsedChapters == nil {
		m.collapsedChapters = map[int]bool{}
	}
	m.collapsedChapters[chapterIndex] = true
	if selectedChapter, ok := m.selectedChapterIndex(); ok && selectedChapter == chapterIndex {
		m.selectChapter(chapterIndex)
	}
}

func (m *Model) expandChapter(chapterIndex int) {
	if m.collapsedChapters == nil {
		return
	}
	delete(m.collapsedChapters, chapterIndex)
}

func (m *Model) collapseSelectedChapter() bool {
	chapterIndex, ok := m.selectedChapterIndex()
	if !ok {
		return false
	}
	if m.selected.kind == navigationStep {
		m.selectChapter(chapterIndex)
		return true
	}
	if !m.chapterCollapsed(chapterIndex) {
		m.collapseChapter(chapterIndex)
		return true
	}
	return false
}

func (m *Model) expandSelectedChapter() bool {
	if m.selected.kind != navigationChapter {
		return false
	}
	if m.chapterCollapsed(m.selected.chapterIndex) {
		m.expandChapter(m.selected.chapterIndex)
		return true
	}
	return false
}

func (m Model) chapterIndexForStep(stepIndex int) (int, bool) {
	if m.session == nil || stepIndex < 0 {
		return 0, false
	}
	step := 0
	for chapterIndex := range m.session.Chapters {
		for range m.session.Chapters[chapterIndex].Nodes {
			if step == stepIndex {
				return chapterIndex, true
			}
			step++
		}
	}
	return 0, false
}

func firstStepIndexInChapter(session *coop.Session, chapterIndex int) int {
	if session == nil || chapterIndex < 0 || chapterIndex >= len(session.Chapters) {
		return -1
	}
	stepIndex := 0
	for i := range session.Chapters {
		for range session.Chapters[i].Nodes {
			if i == chapterIndex {
				return stepIndex
			}
			stepIndex++
		}
	}
	return -1
}
