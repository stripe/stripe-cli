package tui

import (
	"strings"
	"testing"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/stripe/stripe-cli/pkg/coop"
)

func TestReviewInteractionJourney(t *testing.T) {
	m := reviewStepLongPromptLayoutModel()
	m = attachTestStore(t, m)
	m = prepareInteractiveModel(m, 56, 18)

	assertInteractionLayout(t, m, "initial review")
	assert.False(t, m.expanded)
	assert.Contains(t, m.renderFooter(), "request changes")

	m = updateWithKey(t, m, tea.KeyEnter)
	assert.True(t, m.expanded)
	assert.Empty(t, m.statusMessage)
	assertInteractionLayout(t, m, "details opened")

	m = updateWithKey(t, m, tea.KeyTab)
	assert.Equal(t, 1, m.detailTab)
	assertInteractionLayout(t, m, "details tabbed")

	m = updateWithKey(t, m, tea.KeyEsc)
	assert.False(t, m.expanded)
	assertInteractionLayout(t, m, "details closed")

	m = updateWithRunes(t, m, "r")
	assert.True(t, m.rejecting)
	assert.Contains(t, m.renderFooter(), "esc cancel")
	assertInteractionLayout(t, m, "request changes input")

	m = updateWithKey(t, m, tea.KeyEnter)
	assert.True(t, m.rejecting)
	assert.Contains(t, m.rejectionError, "short note")
	node, err := m.session.NodeByNumber(1)
	require.NoError(t, err)
	assert.Equal(t, coop.StepReview, node.State)
	assertInteractionLayout(t, m, "empty request changes validation")

	m = updateWithRunes(t, m, "Use the stored price ID before redirecting to Checkout")
	m = updateWithKey(t, m, tea.KeyEnter)
	assert.False(t, m.rejecting)
	node, err = m.session.NodeByNumber(1)
	require.NoError(t, err)
	assert.Equal(t, coop.StepActive, node.State)
	assert.Contains(t, node.RejectionNote, "stored price ID")
	assert.Contains(t, m.statusMessage, "Feedback sent")
	assertInteractionLayout(t, m, "feedback submitted")
}

func TestFollowInteractionJourney(t *testing.T) {
	m := manualNavigationLayoutModel()
	m = prepareInteractiveModel(m, 69, 50)

	assert.True(t, m.userMoved)
	assert.Equal(t, 2, m.cursor)
	assertInteractionLayout(t, m, "manual navigation")

	m = updateWithRunes(t, m, "f")
	assert.False(t, m.userMoved)
	assert.Equal(t, 0, m.cursor)
	assert.Contains(t, m.statusMessage, "Following")
	assertInteractionLayout(t, m, "follow resumed")
}

func TestChapterReviewInteractionJourney(t *testing.T) {
	m := chapterReviewLayoutModel()
	m = attachTestStore(t, m)
	m = prepareInteractiveModel(m, 69, 50)

	target, ok := m.selectedReviewTarget()
	require.True(t, ok)
	assert.Equal(t, "chapter", target.kind)
	assert.Equal(t, []int{1, 2}, target.steps)
	assertInteractionLayout(t, m, "chapter review")

	m = updateWithRunes(t, m, "c")
	node1, err := m.session.NodeByNumber(1)
	require.NoError(t, err)
	node2, err := m.session.NodeByNumber(2)
	require.NoError(t, err)
	assert.Equal(t, coop.StepDone, node1.State)
	assert.Equal(t, coop.StepDone, node2.State)
	assert.NotContains(t, m.renderFooter(), "Review chapter")
	assertInteractionLayout(t, m, "chapter confirmed")
}

func TestCompletionInteractionJourney(t *testing.T) {
	m := completionLayoutModel()
	m = attachTestStore(t, m)
	m = prepareInteractiveModel(m, 56, 18)

	assert.Contains(t, m.View(), "Integration complete")
	assertInteractionLayout(t, m, "completion initial")

	m = updateWithRunes(t, m, "j")
	assert.Equal(t, 1, m.cursor)
	assertInteractionLayout(t, m, "completion moved")

	m = updateWithRunes(t, m, "k")
	assert.Equal(t, 0, m.cursor)
	assertInteractionLayout(t, m, "completion moved back")
}

func TestDetailsToggleKeepsChromePinned(t *testing.T) {
	for _, size := range []layoutSize{
		{name: "narrow_acceptance", width: 56, height: 18},
		{name: "coop_start_split", width: 69, height: 50},
	} {
		t.Run(size.name, func(t *testing.T) {
			m := reviewStepLongPromptLayoutModel()
			m = prepareInteractiveModel(m, size.width, size.height)

			for i := 0; i < 4; i++ {
				m = updateWithKey(t, m, tea.KeyEnter)
				assertLayoutFits(t, m.View(), size)
				assertHeaderIsPinned(t, m.View())
				assertFooterIsPinned(t, m.View(), "q quit")
				assert.NotContains(t, m.View(), "Details opened")
				assert.NotContains(t, m.View(), "Details collapsed")
			}
		})
	}
}

func TestExpandedReviewConfirmationKeepsChromePinned(t *testing.T) {
	m := expandedDetailsLayoutModel()
	m = prepareInteractiveModel(m, 69, 50)

	rendered := m.View()

	assertLayoutFits(t, rendered, layoutSize{name: "expanded_review_confirmation", width: 69, height: 50})
	assertHeaderIsPinned(t, rendered)
	assertFooterIsPinned(t, rendered, "q quit")
	assert.Contains(t, rendered, "Review")
	assert.Contains(t, rendered, "confirm")
	assert.Contains(t, rendered, "Waiting for you")
}

func TestMoveOntoReviewAfterDetailsToggleKeepsChromePinned(t *testing.T) {
	m := testModel()
	m.spinner = staticSpinner()
	m.session.Chapters[0].Nodes[1].State = coop.StepDone
	m.session.Chapters[1].Nodes[0].State = coop.StepReview
	m.session.Chapters[1].Nodes[0].Implementation = &coop.Implementation{
		File:  "server/webhooks/checkout_completed_handler_with_long_name.js",
		Lines: "12-96",
	}
	m.session.Chapters[1].Nodes[0].ReviewPrompt = "Confirm the webhook handler verifies the Stripe signature, handles duplicate events, stores the Checkout Session ID, and does not expose secrets in logs."
	m.session.Chapters[1].Nodes[0].Verifications = []coop.Verification{
		{Check: "Verified webhook signature", Passed: true},
		{Check: "Handled duplicate events", Passed: true},
		{Check: "Ran stripe trigger checkout.session.completed", Passed: true},
	}
	m.cursor = 1
	m = prepareInteractiveModel(m, 69, 50)

	m = updateWithKey(t, m, tea.KeyEnter)
	assert.True(t, m.expanded)
	assertInteractionLayout(t, m, "details opened before review")

	m = updateWithKey(t, m, tea.KeyEsc)
	assert.False(t, m.expanded)
	assertInteractionLayout(t, m, "details closed before review")

	m = updateWithRunes(t, m, "j")
	assert.Equal(t, 2, m.cursor)
	assertInteractionLayout(t, m, "moved onto review card")
	assert.Equal(t, m.height-1, lineIndexContaining(m.View(), "q quit"))
	assert.Contains(t, m.View(), "Review")
	assert.Contains(t, m.View(), "f follow")
}

func attachTestStore(t *testing.T, m Model) Model {
	t.Helper()
	store, err := coop.NewStoreAt(t.TempDir())
	require.NoError(t, err)
	require.NoError(t, store.Write(m.session))
	m.store = store
	return m
}

func prepareInteractiveModel(m Model, width, height int) Model {
	m.ready = true
	m.width = width
	m.height = height
	m.viewport = viewport.New(width, 10)
	m.resizeViewport()
	m.syncViewport()
	return m
}

func updateWithRunes(t *testing.T, m Model, text string) Model {
	t.Helper()
	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(text)})
	return updated.(Model)
}

func updateWithKey(t *testing.T, m Model, key tea.KeyType) Model {
	t.Helper()
	updated, _ := m.Update(tea.KeyMsg{Type: key})
	return updated.(Model)
}

func assertInteractionLayout(t *testing.T, m Model, label string) {
	t.Helper()
	rendered := m.View()
	size := layoutSize{name: strings.ReplaceAll(label, " ", "_"), width: m.width, height: m.height}
	assertLayoutFits(t, rendered, size)
	assertHeaderIsPinned(t, rendered)
	footerToken := "q quit"
	if m.rejecting {
		footerToken = "esc cancel"
	}
	assertFooterIsPinned(t, rendered, footerToken)
}
