package tui

import (
	"strings"
	"testing"

	"charm.land/bubbles/v2/viewport"
	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/stripe/stripe-cli/pkg/coop"
)

func assertContainsPlain(t *testing.T, s, substr string) {
	t.Helper()
	assert.Contains(t, ansi.Strip(s), substr)
}

func assertNotContainsPlain(t *testing.T, s, substr string) {
	t.Helper()
	assert.NotContains(t, ansi.Strip(s), substr)
}

func testModel() Model {
	m := Model{
		width:          80,
		height:         30,
		sdkSnippetStep: -1,
		rejectionInput: newRejectionInput(),
		keys:           newKeyMap(),
		help:           newHelp(),
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
							ReviewPrompt:   "Confirm the saved price ID is reused by Checkout.",
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

	assertContainsPlain(t, header, "Co-op")
	assertContainsPlain(t, header, "one-time-payment")
	assertContainsPlain(t, header, "node")
	assertContainsPlain(t, header, "1/3")
}

func TestRenderHeaderWithClaimURL(t *testing.T) {
	m := testModel()
	m.session.ClaimURL = "https://dashboard.stripe.com/sandbox/claim_abc"
	header := m.renderHeader()

	assertContainsPlain(t, header, "claim_abc")
}

func TestRenderStepList(t *testing.T) {
	m := testModel()
	list := m.renderStepList()

	assertContainsPlain(t, list, "Set up product")
	assertContainsPlain(t, list, "Create product")
	assertContainsPlain(t, list, "Create checkout")
	assertContainsPlain(t, list, "Handle webhooks")
	assertContainsPlain(t, list, "Handle event")
}

func TestRenderStepListAlignsChapterTitleWithRule(t *testing.T) {
	m := testModel()
	lines := strings.Split(ansi.Strip(m.renderStepList()), "\n")

	var titleLine, ruleLine string
	for i, line := range lines {
		if strings.Contains(line, "Set up product") && i+1 < len(lines) {
			titleLine = line
			ruleLine = lines[i+1]
			break
		}
	}

	require.NotEmpty(t, titleLine)
	require.NotEmpty(t, ruleLine)
	titlePrefix := titleLine[:strings.Index(titleLine, "Set up product")]
	rulePrefix := ruleLine[:strings.Index(ruleLine, "─")]
	assert.Equal(t, lipgloss.Width(titlePrefix), lipgloss.Width(rulePrefix))
}

func TestRenderStepListShowsChapterReviewUnit(t *testing.T) {
	m := testModel()
	m.session.Chapters[0].ReviewGranularity = coop.ReviewGranularityChapter
	m.session.Chapters[0].Nodes[0].State = coop.StepReview
	m.session.Chapters[0].Nodes[1].State = coop.StepReview
	m.selectChapter(0)

	list := m.renderStepList()

	assertContainsPlain(t, list, "Needs chapter review (2 steps)")
	assertContainsPlain(t, list, "▸")
	assertContainsPlain(t, list, "Create product  Included")
	assertContainsPlain(t, list, "Create checkout  Included")
	assertNotContainsPlain(t, list, "Create product  Needs review")
}

func TestRenderStepLineAnnotation(t *testing.T) {
	m := testModel()
	node := m.session.Chapters[0].Nodes[0]
	line := m.renderStepLine(node, 0, false, false)

	assertContainsPlain(t, line, "server.js:5-20")
}

func TestRenderStepLineActivity(t *testing.T) {
	m := testModel()
	node := m.session.Chapters[0].Nodes[1]
	line := m.renderStepLine(node, 1, false, false)

	assertContainsPlain(t, line, "Writing endpoint")
}

func TestRenderStepLineCursor(t *testing.T) {
	m := testModel()
	m.cursor = 1
	node := m.session.Chapters[0].Nodes[1]
	line := m.renderStepLine(node, 1, false, true)

	assertContainsPlain(t, line, "▸")
}

func TestRenderStepLineNoCursor(t *testing.T) {
	m := testModel()
	m.cursor = 0
	node := m.session.Chapters[0].Nodes[1]
	line := m.renderStepLine(node, 1, false, false)

	assertNotContainsPlain(t, line, "▸")
}

func TestRenderDetail(t *testing.T) {
	m := testModel()
	m.cursor = 0
	m.expanded = true
	m.detailTab = 1
	detail := m.renderDetail()

	assertContainsPlain(t, detail, "Files")
	assertContainsPlain(t, detail, "Agent wrote")
	assertContainsPlain(t, detail, "server.js:5-20")
	assertContainsPlain(t, detail, "Created product")
}

func TestRenderSummaryDetailDoesNotRepeatLabels(t *testing.T) {
	m := testModel()
	m.cursor = 0
	m.expanded = true
	m.detailTab = 0

	detail := m.renderDetail()

	assertNotContainsPlain(t, detail, "Details:")
	assert.Equal(t, 1, strings.Count(ansi.Strip(detail), "Summary"))
	assertNotContainsPlain(t, detail, "Files  Checks  Reference")
	assertContainsPlain(t, detail, "Confirm the saved price ID is reused")
}

func TestRenderDetailWebhook(t *testing.T) {
	m := testModel()
	m.cursor = 2 // asyncHandler node
	m.expanded = true
	m.detailTab = 2
	detail := m.renderDetail()

	assertContainsPlain(t, detail, "Checks")
	assertContainsPlain(t, detail, "Review command")
	assertContainsPlain(t, detail, "How to verify")
	assertContainsPlain(t, detail, "stripe listen")
	assertContainsPlain(t, detail, "stripe trigger checkout.session.completed")
}

func TestRenderDetailWithSDKSnippet(t *testing.T) {
	m := testModel()
	m.cursor = 0
	m.expanded = true
	m.detailTab = 3
	m.sdkSnippet = "const product = await stripe.products.create({});"
	m.sdkSnippetStep = 0
	detail := m.renderDetail()

	assertContainsPlain(t, detail, "Reference")
	assertContainsPlain(t, detail, "stripe.products.create")
}

func TestRenderDetailFitsPaneWithIndent(t *testing.T) {
	m := testModel()
	m.width = 69
	m.cursor = 0
	m.expanded = true
	m.detailTab = 1
	m.session.Chapters[0].Nodes[0].State = coop.StepReview
	m.session.Chapters[0].Nodes[0].Implementation.Snippet = strings.Repeat("const createdCheckoutSessionWithLongIdentifier = await stripe.checkout.sessions.create({ mode: 'payment' })\n", 5)

	detail := m.renderDetail()

	assertLinesWithinWidth(t, detail, m.width)
	assertContainsPlain(t, detail, "Waiting for you")
}

func TestRenderMarkdownDoesNotIndentSubsequentLines(t *testing.T) {
	m := testModel()
	rendered := ansi.Strip(m.renderMarkdown("first line\n\nsecond line\n\nthird line", 40))

	for _, line := range strings.Split(rendered, "\n") {
		if strings.TrimSpace(line) == "" {
			continue
		}
		assert.NotRegexp(t, `^ {2,}`, line)
	}
}

func TestRenderFooter(t *testing.T) {
	m := testModel()
	m.cursor = 0
	footer := m.renderFooter()

	// Step 0 is done — no review actions
	assertContainsPlain(t, footer, "enter")
	assertContainsPlain(t, footer, "quit")
	assertNotContainsPlain(t, footer, "confirm")
}

func TestRenderFooterReviewStep(t *testing.T) {
	m := testModel()
	m.session.Chapters[0].Nodes[0].State = coop.StepReview
	m.cursor = 0
	footer := m.renderFooter()

	assertContainsPlain(t, footer, "confirm")
	assertContainsPlain(t, footer, "changes")
	assertContainsPlain(t, footer, "Review:")
	assertContainsPlain(t, footer, "Agent changed")
	assertContainsPlain(t, footer, "You check")
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

	assertContainsPlain(t, card, "Review: Create product")
	assertContainsPlain(t, card, "Agent changed:")
	assertContainsPlain(t, card, "server.js:5-20")
	assertContainsPlain(t, card, "Agent verified:")
	assertContainsPlain(t, card, "1/2 check(s) passed")
	assertContainsPlain(t, card, "You check:")
	assertContainsPlain(t, card, "Confirm Checkout uses the saved price ID.")
}

func TestRenderChapterReviewCardNamesCoveredSteps(t *testing.T) {
	m := testModel()
	m.session.Chapters[0].ReviewGranularity = coop.ReviewGranularityChapter
	m.session.Chapters[0].Nodes[0].State = coop.StepReview
	m.session.Chapters[0].Nodes[1].State = coop.StepReview
	m.selectChapter(0)

	card := m.renderReviewCard()
	footer := m.renderFooter()

	assertContainsPlain(t, card, "Review chapter (2 steps): Set up product")
	assertContainsPlain(t, card, "Includes: Create product, Create checkout")
	assertContainsPlain(t, footer, "confirm chapter")
	assertContainsPlain(t, footer, "chapter changes")
}

func TestRenderReviewCardFallbackCheck(t *testing.T) {
	m := testModel()
	m.session.Chapters[1].Nodes[0].State = coop.StepReview
	m.cursor = 2

	card := m.renderReviewCard()

	assertContainsPlain(t, card, "Confirm the completed work matches this step")
}

func TestRenderFooterReviewCommand(t *testing.T) {
	m := testModel()
	m.session.Chapters[1].Nodes[0].State = coop.StepReview
	m.cursor = 2
	footer := m.renderFooter()

	assertContainsPlain(t, footer, "Run:")
	assertContainsPlain(t, footer, "stripe trigger checkout.session.completed")
	assertContainsPlain(t, footer, "y copy")
}

func TestRenderFooterReviewNotice(t *testing.T) {
	m := testModel()
	m.session.Chapters[0].Nodes[0].State = coop.StepReview
	footer := m.renderFooter()

	assertContainsPlain(t, footer, "Waiting for you")
	assertContainsPlain(t, footer, "need review")
}

func TestRenderCompletionView(t *testing.T) {
	m := testModel()
	m.session.Chapters[0].Nodes[0].State = coop.StepDone
	m.session.Chapters[0].Nodes[1].State = coop.StepDone
	m.session.Chapters[1].Nodes[0].State = coop.StepDone

	view := m.renderCompletionView()

	assertContainsPlain(t, view, "Integration complete")
	assertContainsPlain(t, view, "Built")
	assertContainsPlain(t, view, "Set up product")
	assertContainsPlain(t, view, "Handle webhooks")
	assertContainsPlain(t, view, "Important checks")
	assertContainsPlain(t, view, "Confirm the saved price ID is reused by Checkout.")
	assertContainsPlain(t, view, "Next steps")
	assertContainsPlain(t, view, "STRIPE.md")
	assertContainsPlain(t, view, "Deploy")
	assertContainsPlain(t, view, "Finish")
}

func TestCompletionSummaryBoxUsesSinglePaddingSpace(t *testing.T) {
	m := completionLayoutModel()
	body := m.renderCompletionBody()

	assertContainsPlain(t, body, "│ ✓ Integration complete")
	assertNotContainsPlain(t, body, "│  ✓ Integration complete")
}

func TestCompletionBuiltItemsFiltersContextSkippedAndIncomplete(t *testing.T) {
	m := testModel()
	m.session.Chapters = []coop.SessionChapter{
		{
			Key:   "context-chapter",
			Title: "Project context",
			Nodes: []coop.SessionNode{
				{Title: "Understand project", State: coop.StepDone},
			},
		},
		{
			Key:   "built-with-skipped",
			Title: "Built with skipped optional work",
			Nodes: []coop.SessionNode{
				{Title: "Required", State: coop.StepDone},
				{Title: "Optional", State: coop.StepSkipped},
			},
		},
		{
			Key:   "incomplete",
			Title: "Incomplete chapter",
			Nodes: []coop.SessionNode{
				{Title: "Done", State: coop.StepDone},
				{Title: "Still active", State: coop.StepActive},
			},
		},
	}

	assert.Equal(t, []string{"Built with skipped optional work"}, m.completionBuiltItems())
}

func TestCompletionImportantChecksDedupesDoneOnlyAndCaps(t *testing.T) {
	m := testModel()
	m.session.Chapters = []coop.SessionChapter{
		{
			Key:   "checks",
			Title: "Checks",
			Nodes: []coop.SessionNode{
				{Title: "First", State: coop.StepDone, ReviewPrompt: "Check one"},
				{Title: "Duplicate", State: coop.StepDone, ReviewPrompt: "Check one"},
				{Title: "Active", State: coop.StepActive, ReviewPrompt: "Do not include active"},
				{Title: "Second", State: coop.StepDone, ReviewPrompt: "Check two"},
				{Title: "Third", State: coop.StepDone, ReviewPrompt: "Check three"},
				{Title: "Fourth", State: coop.StepDone, ReviewPrompt: "Check four"},
				{Title: "Fifth", State: coop.StepDone, ReviewPrompt: "Do not include after cap"},
			},
		},
	}

	assert.Equal(t, []string{"Check one", "Check two"}, m.completionImportantChecks())
}

func TestCompletionImportantChecksWrapOnWordBoundaries(t *testing.T) {
	m := completionLayoutModel()
	m.session.Chapters[0].Nodes[0].ReviewPrompt = "Open the app and confirm the user-facing flow works as described."

	receipt := m.renderCompletionReceipt(65)

	assertNotContainsPlain(t, receipt, "a\n    s")
	assertNotContainsPlain(t, receipt, "\n    s                                                                described.")
	assertContainsPlain(t, receipt, "as\n    described.")
	assertLinesWithinWidth(t, receipt, 69)
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
	line := m.renderStepLine(node, 0, false, false)

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
	line := m.renderStepLine(node, 0, false, false)

	// Should contain the annotation inline (not wrapped to next line)
	assertContainsPlain(t, line, "Short note")
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

	assertContainsPlain(t, view, "Co-op")
	assertContainsPlain(t, view, "Waiting")
	assertContainsPlain(t, view, "quit")
}

func TestRenderStepLineSkipped(t *testing.T) {
	m := testModel()
	node := coop.SessionNode{
		Key: "skipped", Title: "Skipped step", State: coop.StepSkipped,
		Activity: "Not needed for this project",
	}
	line := m.renderStepLine(node, 0, false, false)
	assertContainsPlain(t, line, "Not needed")
}

func TestRenderDetailSkipped(t *testing.T) {
	m := testModel()
	m.session.Chapters[0].Nodes[0].State = coop.StepSkipped
	m.session.Chapters[0].Nodes[0].Activity = "Already handled"
	m.cursor = 0
	detail := m.renderDetail()
	assertContainsPlain(t, detail, "Skipped")
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
	assertContainsPlain(t, view, "Regenerate")
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

	assertContainsPlain(t, footer, "f follow")
}

func TestRenderFooterRejectionInput(t *testing.T) {
	m := testModel()
	m.session.Chapters[0].Nodes[0].State = coop.StepReview
	m.cursor = 0
	m.rejecting = true
	m.rejectionInput.SetValue("Missing webhook test")

	footer := m.renderFooter()

	assertContainsPlain(t, footer, "enter send")
	assertContainsPlain(t, footer, "esc cancel")
	assertContainsPlain(t, footer, "Missing webhook test")
}

func TestRenderFooterRejectionPlaceholder(t *testing.T) {
	m := testModel()
	m.session.Chapters[1].Nodes[0].State = coop.StepReview
	m.cursor = 2
	m.rejecting = true
	target, _ := m.selectedReviewTarget()
	m.rejectionInput.Placeholder = m.requestChangesPlaceholder(target)
	m.rejectionInput.Focus()

	footer := m.renderFooter()

	assertContainsPlain(t, footer, "signature verification")
	assertContainsPlain(t, footer, "event handling")
}

func TestReviewCardFitsWithinShortViewport(t *testing.T) {
	m := testModel()
	m.ready = true
	m.width = 56
	m.height = 18
	m.viewport = viewport.New(viewport.WithWidth(56), viewport.WithHeight(10))
	m.session.Chapters[0].Nodes[0].State = coop.StepReview
	m.session.Chapters[0].Nodes[0].ReviewPrompt = "Open the local application, complete the Checkout flow, confirm the redirect lands on the success page, confirm the saved price ID is reused, and confirm no secret keys or generated IDs are committed."
	m.session.Chapters[0].Nodes[0].Verifications = []coop.Verification{
		{Check: "Created product and price", Passed: true},
		{Check: "Saved price ID for Checkout", Passed: true},
		{Check: "Ran local Checkout flow", Passed: true},
	}
	m.cursor = 0

	m.resizeViewport()
	m.syncViewport()
	view := m.View().Content

	assert.LessOrEqual(t, lipgloss.Height(view), m.height)
	assertLinesWithinWidth(t, view, m.width)
	assertContainsPlain(t, view, "Stripe Co-op")
	assertContainsPlain(t, view, "q quit")
}

func TestReviewCardShowsDetailsHintWhenClipped(t *testing.T) {
	m := testModel()
	m.ready = true
	m.width = 56
	m.height = 12
	m.viewport = viewport.New(viewport.WithWidth(56), viewport.WithHeight(10))
	m.session.Chapters[0].Nodes[0].State = coop.StepReview
	m.session.Chapters[0].Nodes[0].ReviewPrompt = "Confirm the Checkout flow, success page, saved price ID, webhook event handling, and environment variable setup all match the intended integration."
	m.session.Chapters[0].Nodes[0].Verifications = []coop.Verification{
		{Check: "Created product", Passed: true},
		{Check: "Created price", Passed: true},
		{Check: "Created Checkout Session", Passed: true},
	}
	m.cursor = 0

	footer := m.renderFooter()

	assert.LessOrEqual(t, lipgloss.Height(footer), m.footerHeightBudget())
	assertLinesWithinWidth(t, footer, m.width)
	assertContainsPlain(t, footer, "more checks available")
}

func TestReviewCardFitsCoopStartSplitWidth(t *testing.T) {
	m := testModel()
	m.ready = true
	m.width = 69
	m.height = 50
	m.viewport = viewport.New(viewport.WithWidth(69), viewport.WithHeight(10))
	m.session.Chapters[0].ReviewGranularity = coop.ReviewGranularityChapter
	m.session.Chapters[0].Nodes[0].State = coop.StepReview
	m.session.Chapters[0].Nodes[0].ReviewPrompt = "Confirm the product, price, Checkout Session, redirect URL, success page, saved price ID, webhook event handling, and environment variable setup all match the intended integration."
	m.session.Chapters[0].Nodes[0].Implementation = &coop.Implementation{File: "server/routes/payments/checkout/session/handler/with/a/long/path.js", Lines: "42-118"}
	m.session.Chapters[0].Nodes[0].Verifications = []coop.Verification{
		{Check: "Created product", Passed: true},
		{Check: "Created price", Passed: true},
		{Check: "Created Checkout Session", Passed: true},
	}
	m.session.Chapters[0].Nodes[1].State = coop.StepReview
	m.session.Chapters[0].Nodes[1].ReviewPrompt = "Open the app locally, click the Checkout button, complete payment, and confirm the redirect lands on the expected success page without exposing secret keys."
	m.session.Chapters[0].Nodes[1].Implementation = &coop.Implementation{File: "client/src/components/payments/checkout-button-with-long-name.tsx", Lines: "9-88"}
	m.session.Chapters[0].Nodes[1].Verifications = []coop.Verification{
		{Check: "Rendered Checkout button", Passed: true},
		{Check: "Confirmed redirect", Passed: true},
	}
	m.selectChapter(0)

	m.resizeViewport()
	m.syncViewport()
	view := m.View().Content

	assert.LessOrEqual(t, lipgloss.Height(view), m.height)
	assertLinesWithinWidth(t, view, m.width)
	assertContainsPlain(t, view, "Stripe Co-op")
	assertContainsPlain(t, view, "Review chapter")
	assertContainsPlain(t, view, "q quit")
}

func TestViewportShowsMoreBelowIndicator(t *testing.T) {
	m := testModel()
	m.ready = true
	m.width = 69
	m.height = 12
	m.viewport = viewport.New(viewport.WithWidth(69), viewport.WithHeight(4))
	m.session.Chapters = []coop.SessionChapter{{
		Key:   "long",
		Title: "Long chapter",
		Nodes: []coop.SessionNode{
			{Title: "One", State: coop.StepDone},
			{Title: "Two", State: coop.StepDone},
			{Title: "Three", State: coop.StepDone},
			{Title: "Four", State: coop.StepDone},
			{Title: "Five", State: coop.StepDone},
			{Title: "Six", State: coop.StepDone},
		},
	}}
	m.cursor = 0
	m.resizeViewport()
	m.syncViewport()
	m.viewport.SetHeight(4)
	m.viewport.SetYOffset(0)

	rendered := m.renderViewportRegionWithHeight(4)

	assertContainsPlain(t, rendered, "more below")
	assertLinesWithinWidth(t, rendered, m.width)
}

func assertLinesWithinWidth(t *testing.T, rendered string, width int) {
	t.Helper()
	for _, line := range strings.Split(rendered, "\n") {
		assert.LessOrEqual(t, lipgloss.Width(line), width, "line exceeds width: %q", line)
	}
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
		assert.Contains(t, ansi.Strip(icon), tc.contains, "state %s should contain %s", tc.state, tc.contains)
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
