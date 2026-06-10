package tui

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
		if m.chapterReviewReady(chapterIndex) {
			items = append(items, navigationItem{kind: navigationChapter, chapterIndex: chapterIndex})
			stepIndex += len(chapter.Nodes)
			continue
		}
		for range chapter.Nodes {
			items = append(items, navigationItem{kind: navigationStep, stepIndex: stepIndex})
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

	if !m.chapterSelected {
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
		return m.chapterSelected && m.chapterCursor == item.chapterIndex
	case navigationStep:
		return !m.chapterSelected && m.cursor == item.stepIndex
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
	m.cursor = stepIndex
	m.chapterSelected = false
}

func (m *Model) selectChapter(chapterIndex int) {
	m.chapterSelected = true
	m.chapterCursor = chapterIndex
	if stepIndex := firstReviewStepIndex(m.session, chapterIndex); stepIndex >= 0 {
		m.cursor = stepIndex
	}
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
