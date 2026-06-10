package tui

import (
	"image/color"
	"strings"
	"testing"
	"time"

	"charm.land/bubbles/v2/viewport"
	tea "charm.land/bubbletea/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/stripe/stripe-cli/pkg/coop"
)

func readyModel() Model {
	m := testModel()
	m.ready = true
	m.viewport = viewport.New(viewport.WithWidth(80), viewport.WithHeight(20))
	return m
}

func TestUpdateKeyDown(t *testing.T) {
	m := readyModel()
	m.cursor = 0

	result, _ := m.Update(tea.KeyPressMsg{Code: 'j', Text: "j"})
	updated := result.(Model)

	assert.Equal(t, 1, updated.cursor)
	assert.True(t, updated.userMoved)
}

func TestUpdateKeyUp(t *testing.T) {
	m := readyModel()
	m.cursor = 2

	result, _ := m.Update(tea.KeyPressMsg{Code: 'k', Text: "k"})
	updated := result.(Model)

	assert.Equal(t, navigationChapter, updated.selected.kind)
	assert.Equal(t, 1, updated.selected.chapterIndex)
}

func TestUpdateKeyUpAtTop(t *testing.T) {
	m := readyModel()
	m.cursor = 0

	result, _ := m.Update(tea.KeyPressMsg{Code: 'k', Text: "k"})
	updated := result.(Model)

	assert.Equal(t, 0, updated.cursor)
}

func TestUpdateKeyDownAtBottom(t *testing.T) {
	m := readyModel()
	m.cursor = m.session.TotalSteps() - 1

	result, _ := m.Update(tea.KeyPressMsg{Code: 'j', Text: "j"})
	updated := result.(Model)

	assert.Equal(t, m.session.TotalSteps()-1, updated.cursor)
}

func TestUpdatePageKeysMoveViewport(t *testing.T) {
	m := readyModel()
	m.viewport.SetContent(strings.Join([]string{
		"one", "two", "three", "four", "five", "six", "seven", "eight",
	}, "\n"))
	m.viewport.SetHeight(3)

	result, _ := m.Update(tea.KeyPressMsg{Code: ' '})
	updated := result.(Model)

	assert.True(t, updated.viewport.YOffset() > 0)
	assert.True(t, updated.userMoved)
}

func TestUpdateKeyExpand(t *testing.T) {
	m := readyModel()
	m.expanded = false

	result, _ := m.Update(tea.KeyPressMsg{Code: 'e', Text: "e"})
	updated := result.(Model)

	assert.True(t, updated.expanded)
}

func TestUpdateKeyExpandToggle(t *testing.T) {
	m := readyModel()
	m.expanded = true

	result, _ := m.Update(tea.KeyPressMsg{Code: 'e', Text: "e"})
	updated := result.(Model)

	assert.False(t, updated.expanded)
}

func TestUpdateKeyTabCyclesDetailSection(t *testing.T) {
	m := readyModel()
	m.expanded = true
	m.detailTab = 0

	result, _ := m.Update(tea.KeyPressMsg{Code: tea.KeyTab})
	updated := result.(Model)

	assert.Equal(t, 1, updated.detailTab)
}

func TestUpdateKeyEscClosesDetails(t *testing.T) {
	m := readyModel()
	m.expanded = true

	result, _ := m.Update(tea.KeyPressMsg{Code: tea.KeyEsc})
	updated := result.(Model)

	assert.False(t, updated.expanded)
}

func TestUpdateKeyConfirm(t *testing.T) {
	dir := t.TempDir()
	store, _ := coop.NewStoreAt(dir)

	m := readyModel()
	m.store = store
	m.session.Chapters[0].Nodes[0].State = coop.StepReview
	m.cursor = 0
	store.Write(m.session)

	result, _ := m.Update(tea.KeyPressMsg{Code: 'c', Text: "c"})
	updated := result.(Model)

	node, _ := updated.session.NodeByNumber(1)
	assert.Equal(t, coop.StepDone, node.State)
}

func TestUpdateKeyConfirmIgnoresRepeat(t *testing.T) {
	dir := t.TempDir()
	store, _ := coop.NewStoreAt(dir)

	m := readyModel()
	m.store = store
	m.session.Chapters[0].Nodes[0].State = coop.StepReview
	m.cursor = 0
	store.Write(m.session)

	result, _ := m.Update(tea.KeyPressMsg{Code: 'c', Text: "c", IsRepeat: true})
	updated := result.(Model)

	node, _ := updated.session.NodeByNumber(1)
	assert.Equal(t, coop.StepReview, node.State)
}

func TestViewProgressBarReflectsSessionProgress(t *testing.T) {
	m := readyModel()

	view := m.View()

	require.NotNil(t, view.ProgressBar)
	assert.Equal(t, tea.ProgressBarDefault, view.ProgressBar.State)
	assert.Equal(t, 33, view.ProgressBar.Value)
}

func TestUpdateMouseClickOpensClaimURL(t *testing.T) {
	m := readyModel()
	m.session.ClaimURL = "https://dashboard.stripe.com/sandbox/claim_abc"
	var opened string
	oldOpen := openBrowserFn
	openBrowserFn = func(url string) { opened = url }
	t.Cleanup(func() { openBrowserFn = oldOpen })

	result, _ := m.Update(tea.MouseClickMsg(tea.Mouse{Y: 1, Button: tea.MouseLeft}))
	_ = result.(Model)

	assert.Equal(t, m.session.ClaimURL, opened)
}

func TestUpdateMouseWheelScrollsViewport(t *testing.T) {
	m := readyModel()
	m.viewport.SetHeight(3)
	m.viewport.SetContent(strings.Join([]string{
		"one", "two", "three", "four", "five", "six", "seven",
	}, "\n"))

	result, _ := m.Update(tea.MouseWheelMsg(tea.Mouse{Button: tea.MouseWheelDown}))
	updated := result.(Model)

	assert.True(t, updated.userMoved)
	assert.Greater(t, updated.viewport.YOffset(), 0)
}

func TestSyncViewportPreservesManualScroll(t *testing.T) {
	m := readyModel()
	m.ready = true
	m.width = 80
	m.height = 12
	m.resizeViewport()
	m.cursor = 0
	m.expanded = true
	m.sdkSnippet = strings.Repeat("const product = await stripe.products.create({});\n", 20)
	m.sdkSnippetStep = 0
	m.syncViewport()
	m.viewport.SetYOffset(6)
	m.userMoved = true

	m.syncViewport()

	assert.Equal(t, 6, m.viewport.YOffset())
}

func TestViewInstallsMouseHandler(t *testing.T) {
	m := readyModel()

	view := m.View()

	assert.NotNil(t, view.OnMouse)
}

func TestMouseActionSelectsVisibleStep(t *testing.T) {
	m := readyModel()
	m.ready = true
	m.width = 80
	m.height = 24
	m.resizeViewport()
	m.syncViewport()

	result, _ := m.Update(mouseActionMsg{action: mouseActionSelectStep, index: 1})
	updated := result.(Model)

	assert.Equal(t, 1, updated.cursor)
	assert.True(t, updated.userMoved)
}

func TestNavigationMovesBetweenChapterAndStepRows(t *testing.T) {
	m := readyModel()
	m.session.Chapters[0].ReviewGranularity = coop.ReviewGranularityChapter
	m.session.Chapters[0].Nodes[0].State = coop.StepReview
	m.session.Chapters[0].Nodes[1].State = coop.StepReview
	m.selectStep(0)

	m.syncViewport()

	m.moveCursorUp()
	assert.Equal(t, navigationChapter, m.selected.kind)
	assert.Equal(t, 0, m.selected.chapterIndex)

	m.moveCursorDown()
	assert.Equal(t, navigationStep, m.selected.kind)
	assert.Equal(t, 0, m.selected.stepIndex)
}

func TestMouseActionSelectsChapter(t *testing.T) {
	m := readyModel()
	m.session.Chapters[0].ReviewGranularity = coop.ReviewGranularityChapter
	m.session.Chapters[0].Nodes[0].State = coop.StepReview
	m.session.Chapters[0].Nodes[1].State = coop.StepReview

	result, _ := m.Update(mouseActionMsg{action: mouseActionSelectChapter, index: 0})
	updated := result.(Model)

	assert.Equal(t, navigationChapter, updated.selected.kind)
	assert.Equal(t, 0, updated.selected.chapterIndex)
	assert.True(t, updated.userMoved)
}

func TestLeftRightCollapseAndExpandSelectedChapter(t *testing.T) {
	m := readyModel()
	m.selectChapter(0)

	result, _ := m.Update(tea.KeyPressMsg{Code: tea.KeyLeft})
	updated := result.(Model)

	assert.True(t, updated.chapterCollapsed(0))
	assert.Equal(t, navigationChapter, updated.selected.kind)

	result, _ = updated.Update(tea.KeyPressMsg{Code: tea.KeyRight})
	updated = result.(Model)

	assert.False(t, updated.chapterCollapsed(0))
	assert.Equal(t, navigationChapter, updated.selected.kind)
}

func TestLeftFromStepMovesToParentChapter(t *testing.T) {
	m := readyModel()
	m.selectStep(1)

	result, _ := m.Update(tea.KeyPressMsg{Code: tea.KeyLeft})
	updated := result.(Model)

	assert.Equal(t, navigationChapter, updated.selected.kind)
	assert.Equal(t, 0, updated.selected.chapterIndex)
	assert.False(t, updated.chapterCollapsed(0))
}

func TestUpdateKeyConfirmNotOnReviewStep(t *testing.T) {
	m := readyModel()
	m.cursor = 0 // step is Done, not Review

	result, _ := m.Update(tea.KeyPressMsg{Code: 'c', Text: "c"})
	updated := result.(Model)

	// Should not change
	node, _ := updated.session.NodeByNumber(1)
	assert.Equal(t, coop.StepDone, node.State)
}

func TestUpdateKeyReject(t *testing.T) {
	dir := t.TempDir()
	store, _ := coop.NewStoreAt(dir)

	m := readyModel()
	m.store = store
	m.session.Chapters[0].Nodes[0].State = coop.StepReview
	m.session.Chapters[0].Nodes[0].Implementation = &coop.Implementation{File: "a.js"}
	m.cursor = 0
	store.Write(m.session)

	result, _ := m.Update(tea.KeyPressMsg{Code: 'r', Text: "r"})
	updated := result.(Model)
	assert.True(t, updated.rejecting)

	result, _ = updated.Update(tea.KeyPressMsg{Code: 'N', Text: "Needs tests"})
	updated = result.(Model)
	result, _ = updated.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	updated = result.(Model)

	node, _ := updated.session.NodeByNumber(1)
	assert.Equal(t, coop.StepActive, node.State)
	assert.Nil(t, node.Implementation)
	assert.Equal(t, "Needs tests", node.RejectionNote)
	assert.False(t, updated.rejecting)
}

func TestUpdateKeyRejectRequiresNote(t *testing.T) {
	dir := t.TempDir()
	store, _ := coop.NewStoreAt(dir)

	m := readyModel()
	m.store = store
	m.session.Chapters[0].Nodes[0].State = coop.StepReview
	m.cursor = 0
	store.Write(m.session)

	result, _ := m.Update(tea.KeyPressMsg{Code: 'r', Text: "r"})
	updated := result.(Model)
	result, _ = updated.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	updated = result.(Model)

	node, _ := updated.session.NodeByNumber(1)
	assert.Equal(t, coop.StepReview, node.State)
	assert.True(t, updated.rejecting)
	assert.Contains(t, updated.rejectionError, "short note")
}

func TestRejectingViewSetsRealCursor(t *testing.T) {
	m := readyModel()
	m.ready = true
	m.width = 69
	m.height = 20
	m.session.Chapters[0].Nodes[0].State = coop.StepReview
	m.cursor = 0
	m.startReject()

	view := m.View()

	require.NotNil(t, view.Cursor)
	assert.GreaterOrEqual(t, view.Cursor.X, 0)
	assert.GreaterOrEqual(t, view.Cursor.Y, 0)
	assert.Equal(t, tea.CursorBar, view.Cursor.Shape)
}

func TestBackgroundColorUpdatesTheme(t *testing.T) {
	m := readyModel()
	require.True(t, m.theme.IsDark)

	result, _ := m.Update(tea.BackgroundColorMsg{Color: color.White})
	updated := result.(Model)

	assert.False(t, updated.theme.IsDark)
	assert.False(t, updated.isDark)
	assert.NotEqual(t, m.theme.HueGray300, updated.theme.HueGray300)
}

func TestUpdateKeyConfirmChapterReview(t *testing.T) {
	dir := t.TempDir()
	store, _ := coop.NewStoreAt(dir)

	m := readyModel()
	m.store = store
	m.session.Chapters[0].ReviewGranularity = coop.ReviewGranularityChapter
	m.session.Chapters[0].Nodes[0].State = coop.StepReview
	m.session.Chapters[0].Nodes[1].State = coop.StepReview
	m.selectChapter(0)
	m.userMoved = true
	store.Write(m.session)

	result, _ := m.Update(tea.KeyPressMsg{Code: 'c', Text: "c"})
	updated := result.(Model)

	node1, _ := updated.session.NodeByNumber(1)
	node2, _ := updated.session.NodeByNumber(2)
	assert.Equal(t, coop.StepDone, node1.State)
	assert.Equal(t, coop.StepDone, node2.State)
	assert.False(t, updated.userMoved)
}

func TestUpdateKeyConfirmChapterReviewRequiresChapterSelection(t *testing.T) {
	dir := t.TempDir()
	store, _ := coop.NewStoreAt(dir)

	m := readyModel()
	m.store = store
	m.session.Chapters[0].ReviewGranularity = coop.ReviewGranularityChapter
	m.session.Chapters[0].Nodes[0].State = coop.StepReview
	m.session.Chapters[0].Nodes[1].State = coop.StepReview
	m.selectStep(0)
	store.Write(m.session)

	result, _ := m.Update(tea.KeyPressMsg{Code: 'c', Text: "c"})
	updated := result.(Model)

	node1, _ := updated.session.NodeByNumber(1)
	node2, _ := updated.session.NodeByNumber(2)
	assert.Equal(t, coop.StepReview, node1.State)
	assert.Equal(t, coop.StepReview, node2.State)
}

func TestSelectedReviewTargetChapterRequiresReadyChapter(t *testing.T) {
	m := readyModel()
	m.session.Chapters[0].ReviewGranularity = coop.ReviewGranularityChapter
	m.session.Chapters[0].Nodes[0].State = coop.StepReview
	m.session.Chapters[0].Nodes[1].State = coop.StepPending
	m.cursor = 0

	_, ok := m.selectedReviewTarget()
	assert.False(t, ok)
	assert.False(t, m.reviewIsActionable(1))

	m.session.Chapters[0].Nodes[1].State = coop.StepReview
	_, ok = m.selectedReviewTarget()
	assert.False(t, ok)

	m.selectChapter(0)
	target, ok := m.selectedReviewTarget()

	assert.True(t, ok)
	assert.Equal(t, "chapter", target.kind)
	assert.Equal(t, "Set up product", target.title)
	assert.Equal(t, []int{1, 2}, target.steps)
	assert.True(t, m.reviewIsActionable(1))
}

func TestUpdateKeyRejectChapterReview(t *testing.T) {
	dir := t.TempDir()
	store, _ := coop.NewStoreAt(dir)

	m := readyModel()
	m.store = store
	m.session.Chapters[0].ReviewGranularity = coop.ReviewGranularityChapter
	m.session.Chapters[0].Nodes[0].State = coop.StepReview
	m.session.Chapters[0].Nodes[0].Implementation = &coop.Implementation{File: "product.js"}
	m.session.Chapters[0].Nodes[0].Verifications = []coop.Verification{{Check: "product test", Passed: true}}
	m.session.Chapters[0].Nodes[1].State = coop.StepReview
	m.session.Chapters[0].Nodes[1].Implementation = &coop.Implementation{File: "checkout.js"}
	m.session.Chapters[0].Nodes[1].Verifications = []coop.Verification{{Check: "checkout test", Passed: true}}
	m.selectChapter(0)
	m.userMoved = true
	store.Write(m.session)

	result, _ := m.Update(tea.KeyPressMsg{Code: 'r', Text: "r"})
	updated := result.(Model)
	result, _ = updated.Update(tea.KeyPressMsg{Code: 'R', Text: "Rework both steps"})
	updated = result.(Model)
	result, _ = updated.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	updated = result.(Model)

	node1, _ := updated.session.NodeByNumber(1)
	node2, _ := updated.session.NodeByNumber(2)
	assert.Equal(t, coop.StepActive, node1.State)
	assert.Equal(t, coop.StepActive, node2.State)
	assert.Equal(t, "Rework both steps", node1.RejectionNote)
	assert.Equal(t, "Rework both steps", node2.RejectionNote)
	assert.Nil(t, node1.Implementation)
	assert.Nil(t, node2.Implementation)
	assert.Nil(t, node1.Verifications)
	assert.Nil(t, node2.Verifications)
	assert.False(t, updated.rejecting)
	assert.False(t, updated.userMoved)
}

func TestUpdateKeyRejectCancel(t *testing.T) {
	m := readyModel()
	m.session.Chapters[0].Nodes[0].State = coop.StepReview
	m.cursor = 0

	result, _ := m.Update(tea.KeyPressMsg{Code: 'r', Text: "r"})
	updated := result.(Model)
	assert.True(t, updated.rejecting)

	result, _ = updated.Update(tea.KeyPressMsg{Code: tea.KeyEsc})
	updated = result.(Model)

	node, _ := updated.session.NodeByNumber(1)
	assert.Equal(t, coop.StepReview, node.State)
	assert.False(t, updated.rejecting)
	assert.Contains(t, updated.statusMessage, "canceled")
}

func TestUpdateKeyQuit(t *testing.T) {
	m := readyModel()

	_, cmd := m.Update(tea.KeyPressMsg{Code: 'q', Text: "q"})
	// tea.Quit returns a special command
	assert.NotNil(t, cmd)
}

func TestUpdateWindowSize(t *testing.T) {
	m := readyModel()

	result, _ := m.Update(tea.WindowSizeMsg{Width: 120, Height: 40})
	updated := result.(Model)

	assert.Equal(t, 120, updated.width)
	assert.Equal(t, 40, updated.height)
	assert.Equal(t, 120, updated.viewport.Width())
}

func TestUpdateSessionUpdated(t *testing.T) {
	m := readyModel()
	m.lastVersion = 1

	newSession := &coop.Session{
		ID:      "test_123",
		Version: 2,
		Status:  coop.SessionActive,
		Chapters: []coop.SessionChapter{
			{Key: "ch1", Title: "Ch", Nodes: []coop.SessionNode{
				{Key: "n1", Title: "Step", State: coop.StepActive},
			}},
		},
	}

	result, _ := m.Update(sessionUpdatedMsg{session: newSession})
	updated := result.(Model)

	assert.Equal(t, 2, updated.lastVersion)
	assert.Equal(t, "test_123", updated.session.ID)
}

func TestUpdateSessionDiscovered(t *testing.T) {
	dir := t.TempDir()
	store, _ := coop.NewStoreAt(dir)
	store.Write(&coop.Session{ID: "new_session", Status: coop.SessionActive})

	m := readyModel()
	m.store = store
	m.waiting = true

	result, cmd := m.Update(sessionDiscoveredMsg{sessionID: "new_session"})
	updated := result.(Model)

	assert.False(t, updated.waiting)
	assert.Equal(t, "new_session", updated.sessionID)
	assert.NotNil(t, cmd)
}

func TestAutoScrollToReview(t *testing.T) {
	m := readyModel()
	m.session.Chapters[0].Nodes[0].State = coop.StepDone
	m.session.Chapters[0].Nodes[1].State = coop.StepReview
	m.cursor = 0

	m.autoScroll()

	assert.Equal(t, navigationChapter, m.selected.kind)
	assert.Equal(t, 0, m.selected.chapterIndex)
}

func TestAutoScrollToActive(t *testing.T) {
	m := readyModel()
	m.session.Chapters[0].Nodes[0].State = coop.StepDone
	m.session.Chapters[0].Nodes[1].State = coop.StepActive
	m.cursor = 0

	m.autoScroll()

	assert.Equal(t, 1, m.cursor)
}

func TestAutoScrollReviewPriority(t *testing.T) {
	m := readyModel()
	m.session.Chapters[0].Nodes[0].State = coop.StepReview
	m.session.Chapters[0].Nodes[1].State = coop.StepActive
	m.cursor = 2

	m.autoScroll()

	// Should go to review (index 0), not active (index 1)
	assert.Equal(t, 0, m.cursor)
}

func TestFollowKeyResumesAutoFollow(t *testing.T) {
	m := readyModel()
	m.session.Chapters[0].Nodes[0].State = coop.StepDone
	m.session.Chapters[0].Nodes[1].State = coop.StepReview
	m.cursor = 0
	m.userMoved = true

	result, _ := m.Update(tea.KeyPressMsg{Code: 'f', Text: "f"})
	updated := result.(Model)

	assert.False(t, updated.userMoved)
	assert.Equal(t, navigationChapter, updated.selected.kind)
	assert.Equal(t, 0, updated.selected.chapterIndex)
}

func TestActionableReviewCountCollapsesChapterReview(t *testing.T) {
	m := readyModel()
	m.session.Chapters[0].ReviewGranularity = coop.ReviewGranularityChapter
	m.session.Chapters[0].Nodes[0].State = coop.StepReview
	m.session.Chapters[0].Nodes[1].State = coop.StepReview
	m.session.Chapters[1].Nodes[0].State = coop.StepReview

	assert.Equal(t, 2, m.actionableReviewCount())
}

func TestActionableReviewCountIgnoresUnreadyChapterReview(t *testing.T) {
	m := readyModel()
	m.session.Chapters[0].ReviewGranularity = coop.ReviewGranularityChapter
	m.session.Chapters[0].Nodes[0].State = coop.StepReview
	m.session.Chapters[0].Nodes[1].State = coop.StepPending

	assert.Equal(t, 0, m.actionableReviewCount())
}

func TestCompletionViewKeyDown(t *testing.T) {
	m := readyModel()
	// Make session complete
	for i := range m.session.Chapters {
		for j := range m.session.Chapters[i].Nodes {
			m.session.Chapters[i].Nodes[j].State = coop.StepDone
		}
	}
	m.cursor = 0

	result, _ := m.Update(tea.KeyPressMsg{Code: 'j', Text: "j"})
	updated := result.(Model)

	assert.Equal(t, 1, updated.cursor)
}

func TestCompletionViewportKeepsReceiptAtTop(t *testing.T) {
	m := readyModel()
	m.height = 10
	m.viewport.SetHeight(3)
	for i := range m.session.Chapters {
		for j := range m.session.Chapters[i].Nodes {
			m.session.Chapters[i].Nodes[j].State = coop.StepDone
		}
	}
	m.session.NextSteps = &coop.NextStepsState{
		Suggestions: []coop.NextStepSuggestion{
			{ID: "a", Title: "First", Description: "Long first option"},
			{ID: "b", Title: "Second", Description: "Long second option"},
			{ID: "c", Title: "Third", Description: "Long third option"},
			{ID: "d", Title: "Fourth", Description: "Long fourth option"},
			{ID: "done", Title: "Finish", Description: "Close this session"},
		},
	}
	m.cursor = 4

	m.syncViewport()

	assert.Equal(t, 0, m.viewport.YOffset())
	assert.Contains(t, m.viewport.View(), "Integration complete")
}

func TestCompletionViewKeyDownWraps(t *testing.T) {
	m := readyModel()
	for i := range m.session.Chapters {
		for j := range m.session.Chapters[i].Nodes {
			m.session.Chapters[i].Nodes[j].State = coop.StepDone
		}
	}
	suggestions := m.getCompletionSuggestions()
	m.cursor = len(suggestions) - 1

	result, _ := m.Update(tea.KeyPressMsg{Code: 'j', Text: "j"})
	updated := result.(Model)

	assert.Equal(t, 0, updated.cursor)
}

func TestCompletionEnterSelectsDone(t *testing.T) {
	dir := t.TempDir()
	store, _ := coop.NewStoreAt(dir)

	m := readyModel()
	m.store = store
	for i := range m.session.Chapters {
		for j := range m.session.Chapters[i].Nodes {
			m.session.Chapters[i].Nodes[j].State = coop.StepDone
		}
	}
	m.session.ID = "done_selection"
	store.Write(m.session)
	// "Finish" is the last suggestion
	suggestions := m.getCompletionSuggestions()
	m.cursor = len(suggestions) - 1

	_, cmd := m.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	// Should return tea.Quit
	assert.NotNil(t, cmd)

	session, err := store.Read("done_selection")
	require.NoError(t, err)
	assert.Equal(t, "done", session.NextSteps.Selected)
}

func TestCompletionEnterWritesSelection(t *testing.T) {
	dir := t.TempDir()
	store, _ := coop.NewStoreAt(dir)

	m := readyModel()
	m.store = store
	for i := range m.session.Chapters {
		for j := range m.session.Chapters[i].Nodes {
			m.session.Chapters[i].Nodes[j].State = coop.StepDone
		}
	}
	m.session.ID = "completion_test"
	store.Write(m.session)
	m.cursor = 0 // "Write a STRIPE.md summary"

	m.Update(tea.KeyPressMsg{Code: tea.KeyEnter})

	// Read back from store
	session, err := store.Read("completion_test")
	require.NoError(t, err)
	assert.Equal(t, "summarize", session.NextSteps.Selected)
}

func TestSyncViewportSetsContent(t *testing.T) {
	m := readyModel()
	m.syncViewport()

	// Viewport should have content
	view := m.viewport.View()
	assert.NotEmpty(t, view)
	assert.Contains(t, view, "Set up product")
}

func TestSpinnerTickDoesNotPanic(t *testing.T) {
	m := readyModel()
	now := time.Now()
	// Simulate spinner tick
	assert.NotPanics(t, func() {
		m.Update(m.spinner.Tick())
		_ = now
	})
}

func TestCompletionEnterDeployWaitsForAgentSession(t *testing.T) {
	dir := t.TempDir()
	store, _ := coop.NewStoreAt(dir)

	m := readyModel()
	m.store = store
	for i := range m.session.Chapters {
		for j := range m.session.Chapters[i].Nodes {
			m.session.Chapters[i].Nodes[j].State = coop.StepDone
		}
	}
	m.session.ID = "parent_session"
	store.Write(m.session)

	// Find deploy position in default suggestions
	suggestions := m.getCompletionSuggestions()
	for i, s := range suggestions {
		if s.id == "deploy" || s.id == "deploy-update" {
			m.cursor = i
			break
		}
	}

	result, _ := m.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	updated := result.(Model)

	session, err := store.Read("parent_session")
	require.NoError(t, err)
	assert.Equal(t, suggestions[m.cursor].id, session.NextSteps.Selected)
	assert.True(t, updated.waiting)

	_, err = store.Read("coop_deploy")
	assert.Error(t, err)
}

func TestSelectCompletionOptionSummarize(t *testing.T) {
	dir := t.TempDir()
	store, _ := coop.NewStoreAt(dir)

	m := readyModel()
	m.store = store
	for i := range m.session.Chapters {
		for j := range m.session.Chapters[i].Nodes {
			m.session.Chapters[i].Nodes[j].State = coop.StepDone
		}
	}
	m.session.ID = "test_summarize"
	store.Write(m.session)
	m.cursor = 0 // "Write a STRIPE.md summary" is first

	m.selectCompletionOption()

	// Should have written selection to session
	session, _ := store.Read("test_summarize")
	assert.Equal(t, "summarize", session.NextSteps.Selected)
}

func TestNewModel(t *testing.T) {
	dir := t.TempDir()
	store, _ := coop.NewStoreAt(dir)
	m := NewModel(store, "test_id")
	assert.Equal(t, "test_id", m.sessionID)
	assert.False(t, m.waiting)
	assert.Equal(t, -1, m.sdkSnippetStep)
}

func TestNewWaitingModel(t *testing.T) {
	dir := t.TempDir()
	store, _ := coop.NewStoreAt(dir)
	ids := map[string]bool{"old_session": true}
	m := NewWaitingModel(store, ids)
	assert.True(t, m.waiting)
	assert.Equal(t, ids, m.existingIDs)
}

func TestHandleKeyOpenBrowser(t *testing.T) {
	orig := openBrowserFn
	var opened string
	openBrowserFn = func(url string) { opened = url }
	defer func() { openBrowserFn = orig }()

	m := readyModel()
	m.session.ClaimURL = "https://example.com"
	result, _ := m.Update(tea.KeyPressMsg{Code: 'o', Text: "o"})
	updated := result.(Model)
	assert.NotNil(t, updated)
	assert.Equal(t, "https://example.com", opened)
}

func TestHandleKeyCopyReviewCommand(t *testing.T) {
	m := readyModel()
	m.session.Chapters[1].Nodes[0].State = coop.StepReview
	m.cursor = 2

	result, cmd := m.Update(tea.KeyPressMsg{Code: 'y', Text: "y"})
	updated := result.(Model)

	assert.NotNil(t, cmd, "should return a clipboard command")
	assert.Contains(t, updated.statusMessage, "Copied")
}

func TestHandleKeyQuestionMark(t *testing.T) {
	m := readyModel()
	m.expanded = false

	result, _ := m.Update(tea.KeyPressMsg{Code: '?', Text: "?"})
	updated := result.(Model)
	assert.True(t, updated.expanded)
}

func TestAutoScrollFocusesReviewWithoutExpanding(t *testing.T) {
	m := readyModel()
	m.session.Chapters[0].Nodes[0].State = coop.StepReview
	m.expanded = false

	m.autoScroll()

	assert.False(t, m.expanded)
	assert.Equal(t, 0, m.cursor)
}

func TestCompletionTransitionResetsCursor(t *testing.T) {
	m := readyModel()
	m.cursor = 2
	m.viewport.SetYOffset(5)

	// Simulate session becoming complete
	newSession := &coop.Session{
		ID:      "test_123",
		Version: 5,
		Status:  coop.SessionActive,
		Chapters: []coop.SessionChapter{
			{Key: "ch1", Title: "Ch", Nodes: []coop.SessionNode{
				{Key: "n1", Title: "Step 1", State: coop.StepDone},
				{Key: "n2", Title: "Step 2", State: coop.StepDone},
				{Key: "n3", Title: "Step 3", State: coop.StepDone},
			}},
		},
	}

	result, _ := m.Update(sessionUpdatedMsg{session: newSession})
	updated := result.(Model)

	assert.Equal(t, 0, updated.cursor)
	assert.False(t, updated.expanded)
	assert.Equal(t, 0, updated.viewport.YOffset())
}

func TestStatusExpiresOnTick(t *testing.T) {
	m := readyModel()
	m.setStatus("Temporary status", time.Second)

	assert.Equal(t, "Temporary status", m.statusMessage)

	m.clearExpiredStatus(time.Now().Add(2 * time.Second))

	assert.Equal(t, "", m.statusMessage)
	assert.True(t, m.statusExpiresAt.IsZero())
}

func TestStatusWithoutTTLDoesNotExpire(t *testing.T) {
	m := readyModel()
	m.setStatus("Persistent status", 0)

	m.clearExpiredStatus(time.Now().Add(2 * time.Second))

	assert.Equal(t, "Persistent status", m.statusMessage)
	assert.True(t, m.statusExpiresAt.IsZero())
}

func TestShouldTransitionToNewSession(t *testing.T) {
	m := readyModel()
	for i := range m.session.Chapters {
		for j := range m.session.Chapters[i].Nodes {
			m.session.Chapters[i].Nodes[j].State = coop.StepDone
		}
	}

	suggestions := m.getCompletionSuggestions()

	// Find deploy
	for i, s := range suggestions {
		if s.id == "deploy" || s.id == "deploy-update" {
			m.cursor = i
			assert.True(t, m.shouldTransitionToNewSession())
			break
		}
	}

	// Find "Finish" — should NOT transition
	for i, s := range suggestions {
		if s.id == "done" {
			m.cursor = i
			assert.False(t, m.shouldTransitionToNewSession())
			break
		}
	}
}

func TestFetchSnippetNotAPIRequest(t *testing.T) {
	m := readyModel()
	m.cursor = 2 // asyncHandler node
	cmd := m.fetchSnippetIfNeeded()
	assert.Nil(t, cmd) // should not fetch for non-apiRequest
}

func TestFetchSnippetCached(t *testing.T) {
	m := readyModel()
	m.cursor = 0
	m.sdkSnippetStep = 0 // already cached for this step
	cmd := m.fetchSnippetIfNeeded()
	assert.Nil(t, cmd) // should not re-fetch
}

func TestAgentIdleNoSession(t *testing.T) {
	m := readyModel()
	m.session = nil
	assert.False(t, m.agentIdle())
}

func TestAgentIdleSessionComplete(t *testing.T) {
	m := readyModel()
	for i := range m.session.Chapters {
		for j := range m.session.Chapters[i].Nodes {
			m.session.Chapters[i].Nodes[j].State = coop.StepDone
		}
	}
	m.lastUpdateTime = time.Now().Add(-5 * time.Minute)
	assert.False(t, m.agentIdle())
}

func TestAgentIdleNoUpdateTime(t *testing.T) {
	m := readyModel()
	// lastUpdateTime is zero value — should not show warning
	assert.False(t, m.agentIdle())
}

func TestAgentIdleRecentUpdate(t *testing.T) {
	m := readyModel()
	m.lastUpdateTime = time.Now()
	assert.False(t, m.agentIdle())
}

func TestAgentIdleStaleNoHeartbeat(t *testing.T) {
	dir := t.TempDir()
	store, _ := coop.NewStoreAt(dir)

	m := readyModel()
	m.store = store
	m.sessionID = "test_123"
	m.lastUpdateTime = time.Now().Add(-3 * time.Minute)
	// No heartbeat file, stale update → idle
	assert.True(t, m.agentIdle())
}

func TestAgentIdleFreshHeartbeat(t *testing.T) {
	dir := t.TempDir()
	store, _ := coop.NewStoreAt(dir)
	store.WriteHeartbeat("test_123")

	m := readyModel()
	m.store = store
	m.sessionID = "test_123"
	m.lastUpdateTime = time.Now().Add(-3 * time.Minute)
	// Heartbeat is fresh — agent is polling via await
	assert.False(t, m.agentIdle())
}

func TestResizeViewportOnSessionUpdate(t *testing.T) {
	m := readyModel()
	m.width = 80
	m.height = 25
	m.resizeViewport()

	initialHeight := m.viewport.Height()

	// Simulate a session with a review step (footer grows by 1 line)
	m.session.Chapters[0].Nodes[0].State = coop.StepReview
	m.resizeViewport()

	// Footer now has the review notice line — viewport should shrink
	assert.True(t, m.viewport.Height() <= initialHeight)
}
