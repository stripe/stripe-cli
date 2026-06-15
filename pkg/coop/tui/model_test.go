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

	assert.Equal(t, navigationStep, updated.selected.kind)
	assert.Equal(t, 1, updated.selected.stepIndex)
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
	m.cursor = m.session.TotalNodes() - 1

	result, _ := m.Update(tea.KeyPressMsg{Code: 'j', Text: "j"})
	updated := result.(Model)

	assert.Equal(t, m.session.TotalNodes()-1, updated.cursor)
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

func TestUpdateKeyTabCyclesDetailStep(t *testing.T) {
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
	m.session.Steps[0].Nodes[0].State = coop.NodeReview
	m.session.Steps[0].Nodes[1].State = coop.NodeDone
	m.cursor = 0
	store.Write(m.session)

	result, _ := m.Update(tea.KeyPressMsg{Code: 'c', Text: "c"})
	updated := result.(Model)

	node, _ := updated.session.NodeByNumber(1)
	assert.Equal(t, coop.NodeDone, node.State)
}

func TestUpdateKeyConfirmIgnoresRepeat(t *testing.T) {
	dir := t.TempDir()
	store, _ := coop.NewStoreAt(dir)

	m := readyModel()
	m.store = store
	m.session.Steps[0].Nodes[0].State = coop.NodeReview
	m.session.Steps[0].Nodes[1].State = coop.NodeDone
	m.cursor = 0
	store.Write(m.session)

	result, _ := m.Update(tea.KeyPressMsg{Code: 'c', Text: "c", IsRepeat: true})
	updated := result.(Model)

	node, _ := updated.session.NodeByNumber(1)
	assert.Equal(t, coop.NodeReview, node.State)
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
	claimURL := "https://dashboard.stripe.com/sandbox/claim_abc"
	m.session.UsedSandbox = true
	m.sandboxClaimURL = claimURL
	var opened string
	oldOpen := openBrowserFn
	openBrowserFn = func(url string) error {
		opened = url
		return nil
	}
	t.Cleanup(func() { openBrowserFn = oldOpen })

	result, cmd := m.Update(tea.MouseClickMsg(tea.Mouse{Y: 1, Button: tea.MouseLeft}))
	_ = result.(Model)
	assert.Nil(t, cmd)
	assert.Empty(t, opened)

	view := m.View()
	mouseCmd := view.OnMouse(tea.MouseClickMsg(tea.Mouse{Y: 1, Button: tea.MouseLeft}))
	require.NotNil(t, mouseCmd)
	action, ok := mouseCmd().(mouseActionMsg)
	require.True(t, ok)

	result, cmd = m.Update(action)
	_ = result.(Model)
	require.NotNil(t, cmd)
	msg := cmd()
	assert.Nil(t, msg)

	assert.Equal(t, claimURL, opened)
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
	m.sdkSnippetNode = 0
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

	assert.Equal(t, 2, updated.cursor)
	assert.True(t, updated.userMoved)
}

func TestNavigationMovesBetweenStepAndStepRows(t *testing.T) {
	m := readyModel()
	m.session.Steps[0].Nodes[0].State = coop.NodeReview
	m.session.Steps[0].Nodes[1].State = coop.NodeReview
	m.selectStep(0)

	m.syncViewport()

	m.moveCursorUp()
	assert.Equal(t, navigationStep, m.selected.kind)
	assert.Equal(t, 0, m.selected.stepIndex)

	m.moveCursorDown()
	assert.Equal(t, navigationNode, m.selected.kind)
	assert.Equal(t, 0, m.cursor)
}

func TestMouseActionSelectsStep(t *testing.T) {
	m := readyModel()
	m.session.Steps[0].Nodes[0].State = coop.NodeReview
	m.session.Steps[0].Nodes[1].State = coop.NodeReview

	result, _ := m.Update(mouseActionMsg{action: mouseActionSelectStep, index: 0})
	updated := result.(Model)

	assert.Equal(t, navigationStep, updated.selected.kind)
	assert.Equal(t, 0, updated.selected.stepIndex)
	assert.True(t, updated.userMoved)
}

func TestLeftRightCollapseAndExpandSelectedStep(t *testing.T) {
	m := readyModel()
	m.selectStep(0)

	result, _ := m.Update(tea.KeyPressMsg{Code: tea.KeyLeft})
	updated := result.(Model)

	assert.True(t, updated.stepCollapsed(0))
	assert.Equal(t, navigationStep, updated.selected.kind)

	result, _ = updated.Update(tea.KeyPressMsg{Code: tea.KeyRight})
	updated = result.(Model)

	assert.False(t, updated.stepCollapsed(0))
	assert.Equal(t, navigationStep, updated.selected.kind)
}

func TestLeftFromStepMovesToParentStep(t *testing.T) {
	m := readyModel()
	m.selectStep(1)

	result, _ := m.Update(tea.KeyPressMsg{Code: tea.KeyLeft})
	updated := result.(Model)

	assert.Equal(t, navigationStep, updated.selected.kind)
	assert.Equal(t, 1, updated.selected.stepIndex)
	assert.True(t, updated.stepCollapsed(1))
}

func TestUpdateKeyConfirmNotOnReviewStep(t *testing.T) {
	m := readyModel()
	m.cursor = 0 // step is Done, not Review

	result, _ := m.Update(tea.KeyPressMsg{Code: 'c', Text: "c"})
	updated := result.(Model)

	// Should not change
	node, _ := updated.session.NodeByNumber(1)
	assert.Equal(t, coop.NodeDone, node.State)
}

func TestUpdateKeyReject(t *testing.T) {
	dir := t.TempDir()
	store, _ := coop.NewStoreAt(dir)

	m := readyModel()
	m.store = store
	m.session.Steps[0].Nodes[0].State = coop.NodeReview
	m.session.Steps[0].Nodes[1].State = coop.NodeDone
	m.session.Steps[0].Nodes[0].Implementation = &coop.Implementation{File: "a.js"}
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
	assert.Equal(t, coop.NodeActive, node.State)
	assert.Nil(t, node.Implementation)
	assert.Equal(t, "Needs tests", node.RejectionNote)
	assert.False(t, updated.rejecting)
}

func TestUpdateKeyRejectRequiresNote(t *testing.T) {
	dir := t.TempDir()
	store, _ := coop.NewStoreAt(dir)

	m := readyModel()
	m.store = store
	m.session.Steps[0].Nodes[0].State = coop.NodeReview
	m.session.Steps[0].Nodes[1].State = coop.NodeDone
	m.cursor = 0
	store.Write(m.session)

	result, _ := m.Update(tea.KeyPressMsg{Code: 'r', Text: "r"})
	updated := result.(Model)
	result, _ = updated.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	updated = result.(Model)

	node, _ := updated.session.NodeByNumber(1)
	assert.Equal(t, coop.NodeReview, node.State)
	assert.True(t, updated.rejecting)
	assert.Contains(t, updated.rejectionError, "short note")
}

func TestRejectingViewSetsRealCursor(t *testing.T) {
	m := readyModel()
	m.ready = true
	m.width = 69
	m.height = 20
	m.session.Steps[0].Nodes[0].State = coop.NodeReview
	m.session.Steps[0].Nodes[1].State = coop.NodeDone
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

func TestUpdateKeyConfirmStepReview(t *testing.T) {
	dir := t.TempDir()
	store, _ := coop.NewStoreAt(dir)

	m := readyModel()
	m.store = store
	m.session.Steps[0].Nodes[0].State = coop.NodeReview
	m.session.Steps[0].Nodes[1].State = coop.NodeReview
	m.selectStep(0)
	m.userMoved = true
	store.Write(m.session)

	result, _ := m.Update(tea.KeyPressMsg{Code: 'c', Text: "c"})
	updated := result.(Model)

	node1, _ := updated.session.NodeByNumber(1)
	node2, _ := updated.session.NodeByNumber(2)
	assert.Equal(t, coop.NodeDone, node1.State)
	assert.Equal(t, coop.NodeDone, node2.State)
	assert.False(t, updated.userMoved)
}

func TestUpdateKeyConfirmStepReviewFromNodeSelection(t *testing.T) {
	dir := t.TempDir()
	store, _ := coop.NewStoreAt(dir)

	m := readyModel()
	m.store = store
	m.session.Steps[0].Nodes[0].State = coop.NodeReview
	m.session.Steps[0].Nodes[1].State = coop.NodeReview
	m.selectNode(0)
	store.Write(m.session)

	result, _ := m.Update(tea.KeyPressMsg{Code: 'c', Text: "c"})
	updated := result.(Model)

	node1, _ := updated.session.NodeByNumber(1)
	node2, _ := updated.session.NodeByNumber(2)
	assert.Equal(t, coop.NodeDone, node1.State)
	assert.Equal(t, coop.NodeDone, node2.State)
}

func TestSelectedReviewTargetStepRequiresReadyStep(t *testing.T) {
	m := readyModel()
	m.session.Steps[0].Nodes[0].State = coop.NodeReview
	m.session.Steps[0].Nodes[1].State = coop.NodePending
	m.cursor = 0

	_, ok := m.selectedReviewTarget()
	assert.False(t, ok)
	assert.False(t, m.reviewIsActionable(1))

	m.session.Steps[0].Nodes[1].State = coop.NodeReview
	_, ok = m.selectedReviewTarget()
	assert.True(t, ok)

	m.selectStep(0)
	target, ok := m.selectedReviewTarget()

	assert.True(t, ok)
	assert.Equal(t, "step", target.kind)
	assert.Equal(t, "Set up product", target.title)
	assert.Equal(t, []int{1, 2}, target.nodeNumbers)
	assert.True(t, m.reviewIsActionable(1))
}

func TestUpdateKeyRejectStepReview(t *testing.T) {
	dir := t.TempDir()
	store, _ := coop.NewStoreAt(dir)

	m := readyModel()
	m.store = store
	m.session.Steps[0].Nodes[0].State = coop.NodeReview
	m.session.Steps[0].Nodes[0].Implementation = &coop.Implementation{File: "product.js"}
	m.session.Steps[0].Nodes[0].Verifications = []coop.Verification{{Check: "product test", Passed: true}}
	m.session.Steps[0].Nodes[1].State = coop.NodeReview
	m.session.Steps[0].Nodes[1].Implementation = &coop.Implementation{File: "checkout.js"}
	m.session.Steps[0].Nodes[1].Verifications = []coop.Verification{{Check: "checkout test", Passed: true}}
	m.selectStep(0)
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
	assert.Equal(t, coop.NodeActive, node1.State)
	assert.Equal(t, coop.NodeActive, node2.State)
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
	m.session.Steps[0].Nodes[0].State = coop.NodeReview
	m.session.Steps[0].Nodes[1].State = coop.NodeDone
	m.cursor = 0

	result, _ := m.Update(tea.KeyPressMsg{Code: 'r', Text: "r"})
	updated := result.(Model)
	assert.True(t, updated.rejecting)

	result, _ = updated.Update(tea.KeyPressMsg{Code: tea.KeyEsc})
	updated = result.(Model)

	node, _ := updated.session.NodeByNumber(1)
	assert.Equal(t, coop.NodeReview, node.State)
	assert.False(t, updated.rejecting)
	assert.Contains(t, updated.statusMessage, "canceled")
}

func TestRejectSubmissionCancelsWhenTargetChanges(t *testing.T) {
	dir := t.TempDir()
	store, _ := coop.NewStoreAt(dir)

	m := readyModel()
	m.store = store
	m.session.Steps[0].Nodes[0].State = coop.NodeReview
	m.session.Steps[0].Nodes[1].State = coop.NodeDone
	m.cursor = 0
	store.Write(m.session)

	result, _ := m.Update(tea.KeyPressMsg{Code: 'r', Text: "r"})
	updated := result.(Model)
	assert.True(t, updated.rejecting)

	updated.session.Steps[0].Nodes[0].State = coop.NodeDone
	result, _ = updated.Update(tea.KeyPressMsg{Code: 'N', Text: "Needs tests"})
	updated = result.(Model)
	result, _ = updated.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	updated = result.(Model)

	node, _ := updated.session.NodeByNumber(1)
	assert.Equal(t, coop.NodeDone, node.State)
	assert.Empty(t, node.RejectionNote)
	assert.False(t, updated.rejecting)
	assert.Contains(t, updated.statusMessage, "Review target changed")
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
		Steps: []coop.SessionStep{
			{
				StepDefinition: coop.StepDefinition{Key: "ch1", Title: "Ch"},
				Nodes: []coop.SessionNode{
					{
						NodeDefinition: coop.NodeDefinition{Key: "n1", Title: "Step"},
						State:          coop.NodeActive,
					},
				},
			},
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

func TestCheckForUpdatesNoChangeReturnsNoUpdate(t *testing.T) {
	dir := t.TempDir()
	store, _ := coop.NewStoreAt(dir)

	m := readyModel()
	m.store = store
	m.sessionID = m.session.ID
	require.NoError(t, store.Write(m.session))
	stored, err := store.Read(m.session.ID)
	require.NoError(t, err)
	m.lastVersion = stored.Version

	cmd := m.checkForUpdates()
	require.NotNil(t, cmd)
	msg := cmd()

	_, ok := msg.(noUpdateMsg)
	require.True(t, ok)
}

func TestDiscoverNewSessionNoChangeReturnsNoUpdate(t *testing.T) {
	dir := t.TempDir()
	store, _ := coop.NewStoreAt(dir)
	require.NoError(t, store.Write(&coop.Session{ID: "old_session", Status: coop.SessionActive}))

	m := NewWaitingModel(store, map[string]bool{"old_session": true})
	cmd := m.discoverNewSession()
	require.NotNil(t, cmd)
	msg := cmd()

	_, ok := msg.(noUpdateMsg)
	assert.True(t, ok)
}

func TestAutoScrollToReview(t *testing.T) {
	m := readyModel()
	m.session.Steps[0].Nodes[0].State = coop.NodeDone
	m.session.Steps[0].Nodes[1].State = coop.NodeReview
	m.cursor = 0

	m.autoScroll()

	assert.Equal(t, navigationStep, m.selected.kind)
	assert.Equal(t, 0, m.selected.stepIndex)
}

func TestAutoScrollToActive(t *testing.T) {
	m := readyModel()
	m.session.Steps[0].Nodes[0].State = coop.NodeDone
	m.session.Steps[0].Nodes[1].State = coop.NodeActive
	m.cursor = 0

	m.autoScroll()

	assert.Equal(t, 1, m.cursor)
}

func TestAutoScrollReviewPriority(t *testing.T) {
	m := readyModel()
	m.session.Steps[0].Nodes[0].State = coop.NodeReview
	m.session.Steps[0].Nodes[1].State = coop.NodeDone
	m.cursor = 2

	m.autoScroll()

	// Should go to review (index 0), not active (index 1)
	assert.Equal(t, 0, m.cursor)
}

func TestFollowKeyResumesAutoFollow(t *testing.T) {
	m := readyModel()
	m.session.Steps[0].Nodes[0].State = coop.NodeDone
	m.session.Steps[0].Nodes[1].State = coop.NodeReview
	m.cursor = 0
	m.userMoved = true

	result, _ := m.Update(tea.KeyPressMsg{Code: 'f', Text: "f"})
	updated := result.(Model)

	assert.False(t, updated.userMoved)
	assert.Equal(t, navigationStep, updated.selected.kind)
	assert.Equal(t, 0, updated.selected.stepIndex)
}

func TestActionableReviewCountCollapsesStepReview(t *testing.T) {
	m := readyModel()
	m.session.Steps[0].Nodes[0].State = coop.NodeReview
	m.session.Steps[0].Nodes[1].State = coop.NodeReview
	m.session.Steps[1].Nodes[0].State = coop.NodeReview

	assert.Equal(t, 2, m.actionableReviewCount())
}

func TestActionableReviewCountIgnoresUnreadyStepReview(t *testing.T) {
	m := readyModel()
	m.session.Steps[0].Nodes[0].State = coop.NodeReview
	m.session.Steps[0].Nodes[1].State = coop.NodePending

	assert.Equal(t, 0, m.actionableReviewCount())
}

func TestCompletionViewKeyDown(t *testing.T) {
	m := withCompletionSuggestions(readyModel())
	// Make session complete
	for i := range m.session.Steps {
		for j := range m.session.Steps[i].Nodes {
			m.session.Steps[i].Nodes[j].State = coop.NodeDone
		}
	}
	m.cursor = 0

	result, _ := m.Update(tea.KeyPressMsg{Code: 'j', Text: "j"})
	updated := result.(Model)

	assert.Equal(t, 1, updated.cursor)
}

func TestCompletionViewWaitsForAgentSuggestions(t *testing.T) {
	m := readyModel()
	for i := range m.session.Steps {
		for j := range m.session.Steps[i].Nodes {
			m.session.Steps[i].Nodes[j].State = coop.NodeDone
		}
	}
	m.cursor = 0

	result, cmd := m.Update(tea.KeyPressMsg{Code: 'j', Text: "j"})
	updated := result.(Model)

	assert.Nil(t, cmd)
	assert.Equal(t, 0, updated.cursor)
	assert.Contains(t, updated.renderCompletionView(), "Waiting for agent to publish next steps")
}

func TestCompletionViewportKeepsReceiptAtTop(t *testing.T) {
	m := readyModel()
	m.height = 10
	m.viewport.SetHeight(3)
	for i := range m.session.Steps {
		for j := range m.session.Steps[i].Nodes {
			m.session.Steps[i].Nodes[j].State = coop.NodeDone
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

	line, ok := m.completionLineForCursor()
	require.True(t, ok)
	assert.GreaterOrEqual(t, line, m.viewport.YOffset())
	assert.Less(t, line, m.viewport.YOffset()+m.viewport.Height())
	assert.Contains(t, m.viewport.View(), "Finish")
}

func TestCompletionViewKeyDownWraps(t *testing.T) {
	m := withCompletionSuggestions(readyModel())
	for i := range m.session.Steps {
		for j := range m.session.Steps[i].Nodes {
			m.session.Steps[i].Nodes[j].State = coop.NodeDone
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

	m := withCompletionSuggestions(readyModel())
	m.store = store
	for i := range m.session.Steps {
		for j := range m.session.Steps[i].Nodes {
			m.session.Steps[i].Nodes[j].State = coop.NodeDone
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

	m := withCompletionSuggestions(readyModel())
	m.store = store
	for i := range m.session.Steps {
		for j := range m.session.Steps[i].Nodes {
			m.session.Steps[i].Nodes[j].State = coop.NodeDone
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

func TestCompletionEnterAddIntegrationWaitsForAgentSession(t *testing.T) {
	dir := t.TempDir()
	store, _ := coop.NewStoreAt(dir)

	m := withCompletionSuggestions(readyModel())
	m.store = store
	for i := range m.session.Steps {
		for j := range m.session.Steps[i].Nodes {
			m.session.Steps[i].Nodes[j].State = coop.NodeDone
		}
	}
	m.session.ID = "parent_session"
	store.Write(m.session)

	suggestions := m.getCompletionSuggestions()
	for i, s := range suggestions {
		if s.id == "add-integration" {
			m.cursor = i
			break
		}
	}

	result, cmd := m.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	updated := result.(Model)

	session, err := store.Read("parent_session")
	require.NoError(t, err)
	assert.Equal(t, suggestions[m.cursor].id, session.NextSteps.Selected)
	assert.True(t, updated.waiting)
	require.NotNil(t, cmd)
	msg := cmd()
	baseline, ok := msg.(waitingBaselineMsg)
	require.True(t, ok)
	assert.NotNil(t, baseline.existingIDs)

	assert.Equal(t, "add-integration", session.NextSteps.Selected)
}

func TestSelectCompletionOptionSummarize(t *testing.T) {
	dir := t.TempDir()
	store, _ := coop.NewStoreAt(dir)

	m := withCompletionSuggestions(readyModel())
	m.store = store
	for i := range m.session.Steps {
		for j := range m.session.Steps[i].Nodes {
			m.session.Steps[i].Nodes[j].State = coop.NodeDone
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
	assert.Equal(t, -1, m.sdkSnippetNode)
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
	openBrowserFn = func(url string) error {
		opened = url
		return nil
	}
	defer func() { openBrowserFn = orig }()

	m := readyModel()
	m.session.UsedSandbox = true
	m.sandboxClaimURL = "https://example.com"
	result, cmd := m.Update(tea.KeyPressMsg{Code: 'o', Text: "o"})
	updated := result.(Model)
	assert.NotNil(t, updated)
	require.NotNil(t, cmd)
	msg := cmd()
	assert.Nil(t, msg)
	assert.Equal(t, "https://example.com", opened)
}

func TestHandleKeyCopyReviewCommand(t *testing.T) {
	m := readyModel()
	m.session.Steps[1].Nodes[0].State = coop.NodeReview
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
	m.session.Steps[0].Nodes[0].State = coop.NodeReview
	m.session.Steps[0].Nodes[1].State = coop.NodeDone
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
		Steps: []coop.SessionStep{
			{
				StepDefinition: coop.StepDefinition{Key: "ch1", Title: "Ch"},
				Nodes: []coop.SessionNode{
					{
						NodeDefinition: coop.NodeDefinition{Key: "n1", Title: "Step 1"},
						State:          coop.NodeDone,
					},
					{
						NodeDefinition: coop.NodeDefinition{Key: "n2", Title: "Step 2"},
						State:          coop.NodeDone,
					},
					{
						NodeDefinition: coop.NodeDefinition{Key: "n3", Title: "Step 3"},
						State:          coop.NodeDone,
					},
				},
			},
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
	m := withCompletionSuggestions(readyModel())
	for i := range m.session.Steps {
		for j := range m.session.Steps[i].Nodes {
			m.session.Steps[i].Nodes[j].State = coop.NodeDone
		}
	}

	suggestions := m.getCompletionSuggestions()

	for i, s := range suggestions {
		if s.id == "add-integration" {
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
	m.sdkSnippetNode = 0 // already cached for this step
	cmd := m.fetchSnippetIfNeeded()
	assert.Nil(t, cmd) // should not re-fetch
}

func TestAgentIdleNoSession(t *testing.T) {
	m := readyModel()
	m.session = nil
	m.updateAgentIdle(10*time.Second, true, time.Now())
	assert.False(t, m.agentIdle())
}

func TestAgentIdleSessionComplete(t *testing.T) {
	m := readyModel()
	for i := range m.session.Steps {
		for j := range m.session.Steps[i].Nodes {
			m.session.Steps[i].Nodes[j].State = coop.NodeDone
		}
	}
	m.lastUpdateTime = time.Now().Add(-5 * time.Minute)
	m.updateAgentIdle(10*time.Second, true, time.Now())
	assert.False(t, m.agentIdle())
}

func TestAgentIdleNoUpdateTime(t *testing.T) {
	m := readyModel()
	m.updateAgentIdle(10*time.Second, true, time.Now())
	assert.False(t, m.agentIdle())
}

func TestAgentIdleRecentUpdate(t *testing.T) {
	m := readyModel()
	m.lastUpdateTime = time.Now()
	m.updateAgentIdle(10*time.Second, true, time.Now())
	assert.False(t, m.agentIdle())
}

func TestAgentIdleStaleNoHeartbeat(t *testing.T) {
	m := readyModel()
	m.lastUpdateTime = time.Now().Add(-3 * time.Minute)
	m.updateAgentIdle(0, false, time.Now())
	assert.False(t, m.agentIdle())
}

func TestAgentIdleFreshHeartbeat(t *testing.T) {
	m := readyModel()
	m.lastUpdateTime = time.Now().Add(-3 * time.Minute)
	m.updateAgentIdle(time.Second, true, time.Now())
	assert.False(t, m.agentIdle())
}

func TestAgentIdleStaleHeartbeat(t *testing.T) {
	m := readyModel()
	m.lastUpdateTime = time.Now().Add(-3 * time.Minute)
	m.updateAgentIdle(10*time.Second, true, time.Now())
	assert.True(t, m.agentIdle())
}

func TestNoUpdateMsgRefreshesCachedAgentIdle(t *testing.T) {
	m := readyModel()
	m.lastUpdateTime = time.Now().Add(-3 * time.Minute)

	result, _ := m.Update(noUpdateMsg{heartbeatAge: 10 * time.Second, heartbeatOK: true})
	updated := result.(Model)

	assert.True(t, updated.agentIdle())
}

func TestResizeViewportOnSessionUpdate(t *testing.T) {
	m := readyModel()
	m.width = 80
	m.height = 25
	m.resizeViewport()

	initialHeight := m.viewport.Height()

	// Simulate a session with a review step (footer grows by 1 line)
	m.session.Steps[0].Nodes[0].State = coop.NodeReview
	m.resizeViewport()

	// Footer now has the review notice line — viewport should shrink
	assert.True(t, m.viewport.Height() <= initialHeight)
}
