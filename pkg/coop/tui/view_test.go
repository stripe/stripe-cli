package tui

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/stripe/stripe-cli/pkg/coop"
)

func testModel() Model {
	m := Model{
		width:          80,
		height:         30,
		sdkSnippetStep: -1,
		session: &coop.Session{
			ID:        "test_123",
			Blueprint: "one-time-payment",
			Status:    coop.SessionActive,
			Settings:  map[string]string{"language": "node"},
			Chapters: []coop.SessionChapter{
				{
					Key:   "ch1",
					Title: "Set up product",
					Nodes: []coop.SessionNode{
						{Key: "n1", Title: "Create product", State: coop.StepDone, Type: coop.NodeAPIRequest,
							Implementation: &coop.Implementation{File: "server.js", Lines: "5-20", Note: "Created product"}},
						{Key: "n2", Title: "Create checkout", State: coop.StepActive, Type: coop.NodeAPIRequest,
							Activity: "Writing endpoint"},
					},
				},
				{
					Key:   "ch2",
					Title: "Handle webhooks",
					Nodes: []coop.SessionNode{
						{Key: "n3", Title: "Handle event", State: coop.StepPending, Type: coop.NodeAsyncHandler,
							Events: []string{"checkout.session.completed"}},
					},
				},
			},
		},
	}
	return m
}

func TestRenderHeader(t *testing.T) {
	m := testModel()
	header := m.renderHeader()

	assert.Contains(t, header, "Co-op")
	assert.Contains(t, header, "one-time-payment")
	assert.Contains(t, header, "node")
	assert.Contains(t, header, "1/3")
}

func TestRenderHeaderWithClaimURL(t *testing.T) {
	m := testModel()
	m.session.ClaimURL = "https://dashboard.stripe.com/sandbox/claim_abc"
	header := m.renderHeader()

	assert.Contains(t, header, "claim_abc")
}

func TestRenderStepList(t *testing.T) {
	m := testModel()
	list := m.renderStepList()

	assert.Contains(t, list, "Set up product")
	assert.Contains(t, list, "Create product")
	assert.Contains(t, list, "Create checkout")
	assert.Contains(t, list, "Handle webhooks")
	assert.Contains(t, list, "Handle event")
}

func TestRenderStepLineAnnotation(t *testing.T) {
	m := testModel()
	node := m.session.Chapters[0].Nodes[0]
	line := m.renderStepLine(node, 0)

	assert.Contains(t, line, "server.js:5-20")
}

func TestRenderStepLineActivity(t *testing.T) {
	m := testModel()
	node := m.session.Chapters[0].Nodes[1]
	line := m.renderStepLine(node, 1)

	assert.Contains(t, line, "Writing endpoint")
}

func TestRenderStepLineCursor(t *testing.T) {
	m := testModel()
	m.cursor = 1
	node := m.session.Chapters[0].Nodes[1]
	line := m.renderStepLine(node, 1)

	assert.Contains(t, line, "▸")
}

func TestRenderStepLineNoCursor(t *testing.T) {
	m := testModel()
	m.cursor = 0
	node := m.session.Chapters[0].Nodes[1]
	line := m.renderStepLine(node, 1)

	assert.NotContains(t, line, "▸")
}

func TestRenderDetail(t *testing.T) {
	m := testModel()
	m.cursor = 0
	m.expanded = true
	detail := m.renderDetail()

	// Should show implementation info via glamour
	assert.Contains(t, detail, "Agent wrote")
	assert.Contains(t, detail, "server.js:5-20")
	assert.Contains(t, detail, "Created product")
}

func TestRenderDetailWebhook(t *testing.T) {
	m := testModel()
	m.cursor = 2 // asyncHandler node
	m.expanded = true
	detail := m.renderDetail()

	assert.Contains(t, detail, "How to verify")
	assert.Contains(t, detail, "stripe listen")
	assert.Contains(t, detail, "stripe trigger checkout.session.completed")
}

func TestRenderDetailWithSDKSnippet(t *testing.T) {
	m := testModel()
	m.cursor = 0
	m.expanded = true
	m.sdkSnippet = "const product = await stripe.products.create({});"
	m.sdkSnippetStep = 0
	detail := m.renderDetail()

	assert.Contains(t, detail, "Reference")
	assert.Contains(t, detail, "stripe.products.create")
}

func TestRenderFooter(t *testing.T) {
	m := testModel()
	m.cursor = 0
	footer := m.renderFooter()

	// Step 0 is done — no review actions
	assert.Contains(t, footer, "navigate")
	assert.Contains(t, footer, "details")
	assert.NotContains(t, footer, "confirm")
}

func TestRenderFooterReviewStep(t *testing.T) {
	m := testModel()
	m.session.Chapters[0].Nodes[0].State = coop.StepReview
	m.cursor = 0
	footer := m.renderFooter()

	assert.Contains(t, footer, "confirm")
	assert.Contains(t, footer, "request changes")
	assert.Contains(t, footer, "Review:")
}

func TestRenderReviewCardEvidence(t *testing.T) {
	m := testModel()
	m.session.Chapters[0].Nodes[0].State = coop.StepReview
	m.session.Chapters[0].Nodes[0].ReviewPrompt = "Confirm Checkout uses the saved price ID."
	m.session.Chapters[0].Nodes[0].Verifications = []coop.Verification{
		{Check: "unit test", Passed: true},
		{Check: "manual test", Passed: false},
	}
	m.cursor = 0

	card := m.renderReviewCard()

	assert.Contains(t, card, "Review: Create product")
	assert.Contains(t, card, "Changed:")
	assert.Contains(t, card, "server.js:5-20")
	assert.Contains(t, card, "Verified:")
	assert.Contains(t, card, "1/2 check(s) passed")
	assert.Contains(t, card, "Check:")
	assert.Contains(t, card, "Confirm Checkout uses the saved price ID.")
}

func TestRenderReviewCardFallbackCheck(t *testing.T) {
	m := testModel()
	m.session.Chapters[1].Nodes[0].State = coop.StepReview
	m.cursor = 2

	card := m.renderReviewCard()

	assert.Contains(t, card, "Confirm the completed work matches this step")
}

func TestRenderFooterReviewNotice(t *testing.T) {
	m := testModel()
	m.session.Chapters[0].Nodes[0].State = coop.StepReview
	footer := m.renderFooter()

	assert.Contains(t, footer, "Waiting for you")
	assert.Contains(t, footer, "need review")
}

func TestRenderCompletionView(t *testing.T) {
	m := testModel()
	m.session.Chapters[0].Nodes[0].State = coop.StepDone
	m.session.Chapters[0].Nodes[1].State = coop.StepDone
	m.session.Chapters[1].Nodes[0].State = coop.StepDone

	view := m.renderCompletionView()

	assert.Contains(t, view, "Integration complete")
	assert.Contains(t, view, "Next steps")
	assert.Contains(t, view, "STRIPE.md")
	assert.Contains(t, view, "Deploy")
	assert.Contains(t, view, "Finish")
}

func TestGetCompletionSuggestionsDefault(t *testing.T) {
	m := testModel()
	suggestions := m.getCompletionSuggestions()

	assert.Len(t, suggestions, 4)
	assert.Equal(t, "Write a STRIPE.md summary", suggestions[0].title)
	assert.Equal(t, "Finish", suggestions[3].title)
}

func TestGetCompletionSuggestionsFromSession(t *testing.T) {
	m := testModel()
	m.session.NextSteps = &coop.NextStepsState{
		Suggestions: []coop.NextStepSuggestion{
			{ID: "custom", Title: "Custom action", Description: "Do something custom"},
		},
	}
	suggestions := m.getCompletionSuggestions()

	assert.Len(t, suggestions, 1)
	assert.Equal(t, "Custom action", suggestions[0].title)
}

func TestAnnotationWrapsAtNarrowWidth(t *testing.T) {
	m := testModel()
	m.width = 40
	node := coop.SessionNode{
		Key: "test", Title: "Step", State: coop.StepActive,
		Activity: "This is a very long activity note that should wrap",
	}
	line := m.renderStepLine(node, 0)

	// Should have a newline (wrapped)
	assert.True(t, strings.Contains(line, "\n"))
}

func TestAnnotationInlineAtWideWidth(t *testing.T) {
	m := testModel()
	m.width = 120
	node := coop.SessionNode{
		Key: "test", Title: "Step", State: coop.StepActive,
		Activity: "Short note",
	}
	line := m.renderStepLine(node, 0)

	// Should contain the annotation inline (not wrapped to next line)
	assert.Contains(t, line, "Short note")
}

func TestWordWrap(t *testing.T) {
	result := wordWrap("hello world this is a test", 12)
	lines := strings.Split(result, "\n")
	assert.Equal(t, 3, len(lines))
	for _, l := range lines {
		assert.LessOrEqual(t, len(l), 12)
	}
}

func TestWordWrapShort(t *testing.T) {
	result := wordWrap("short", 80)
	assert.Equal(t, "short", result)
}

func TestFormatDuration(t *testing.T) {
	assert.Equal(t, "5s", formatDuration(5*1e9))
	assert.Equal(t, "59s", formatDuration(59*1e9))
	assert.Equal(t, "1m30s", formatDuration(90*1e9))
}

func TestRenderWaitingView(t *testing.T) {
	m := testModel()
	m.width = 80
	m.height = 20
	m.session = nil
	view := m.renderWaitingView()

	assert.Contains(t, view, "Co-op")
	assert.Contains(t, view, "Waiting")
	assert.Contains(t, view, "quit")
}

func TestRenderStepLineSkipped(t *testing.T) {
	m := testModel()
	node := coop.SessionNode{
		Key: "skipped", Title: "Skipped step", State: coop.StepSkipped,
		Activity: "Not needed for this project",
	}
	line := m.renderStepLine(node, 0)
	assert.Contains(t, line, "Not needed")
}

func TestRenderDetailSkipped(t *testing.T) {
	m := testModel()
	m.session.Chapters[0].Nodes[0].State = coop.StepSkipped
	m.session.Chapters[0].Nodes[0].Activity = "Already handled"
	m.cursor = 0
	detail := m.renderDetail()
	assert.Contains(t, detail, "Skipped")
}

func TestRenderCompletionViewWithCompleted(t *testing.T) {
	m := testModel()
	for i := range m.session.Chapters {
		for j := range m.session.Chapters[i].Nodes {
			m.session.Chapters[i].Nodes[j].State = coop.StepDone
		}
	}
	m.session.NextSteps = &coop.NextStepsState{
		Completed: []string{"summarize"},
	}
	m.width = 80
	m.height = 30

	view := m.renderCompletionView()
	assert.Contains(t, view, "Regenerate")
}

func TestRenderFooterComplete(t *testing.T) {
	m := testModel()
	for i := range m.session.Chapters {
		for j := range m.session.Chapters[i].Nodes {
			m.session.Chapters[i].Nodes[j].State = coop.StepDone
		}
	}
	footer := m.renderFooter()
	// Completion view has its own footer — step footer returns empty
	assert.Equal(t, "", footer)
}

func TestRenderFooterShowsFollowWhenUserMoved(t *testing.T) {
	m := testModel()
	m.userMoved = true

	footer := m.renderFooter()

	assert.Contains(t, footer, "f follow")
}

func TestRenderFooterRejectionInput(t *testing.T) {
	m := testModel()
	m.session.Chapters[0].Nodes[0].State = coop.StepReview
	m.cursor = 0
	m.rejecting = true
	m.rejectionInput = "Missing webhook test"

	footer := m.renderFooter()

	assert.Contains(t, footer, "enter send feedback")
	assert.Contains(t, footer, "esc cancel")
	assert.Contains(t, footer, "Missing webhook test")
}

func TestStepIconAllStates(t *testing.T) {
	m := testModel()

	cases := []struct {
		state    coop.StepState
		contains string
	}{
		{coop.StepDone, "✓"},
		{coop.StepReview, "◆"},
		{coop.StepSkipped, "–"},
		{coop.StepPending, "○"},
	}

	for _, tc := range cases {
		node := coop.SessionNode{State: tc.state}
		icon := m.stepIcon(node)
		assert.Contains(t, icon, tc.contains, "state %s should contain %s", tc.state, tc.contains)
	}
}

func TestGetCompletionSuggestionsWithDeployDone(t *testing.T) {
	m := testModel()
	for i := range m.session.Chapters {
		for j := range m.session.Chapters[i].Nodes {
			m.session.Chapters[i].Nodes[j].State = coop.StepDone
		}
	}
	m.session.NextSteps = &coop.NextStepsState{
		Completed: []string{"deploy"},
	}

	suggestions := m.getCompletionSuggestions()
	// Should show "Redeploy" instead of "Deploy with Stripe Projects"
	found := false
	for _, s := range suggestions {
		if s.id == "deploy" && s.title == "Redeploy" {
			found = true
		}
	}
	assert.True(t, found, "expected 'Redeploy' after deploy completed")
}

func TestClampLines(t *testing.T) {
	long := "this is a line that is way too long for a 20 column terminal"
	result := clampLines(long, 20)
	// Should be truncated
	assert.LessOrEqual(t, len(result), 30) // allow for ANSI codes
}

func TestContentWidthDefault(t *testing.T) {
	m := testModel()
	m.width = 0
	assert.Equal(t, 80, m.contentWidth())

	m.width = 120
	assert.Equal(t, 120, m.contentWidth())
}
