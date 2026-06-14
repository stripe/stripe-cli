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
	theme := NewTheme(true)
	m := Model{
		width:          80,
		height:         30,
		sdkSnippetNode: -1,
		rejectionInput: newThemedRejectionInput(theme),
		keys:           newKeyMap(),
		help:           newThemedHelp(theme),
		theme:          theme,
		isDark:         true,
		session: &coop.Session{
			ID:        "test_123",
			Blueprint: "one-time-payment",
			Status:    coop.SessionActive,
			Settings:  map[string]string{"language": "node"},
			Steps: []coop.SessionStep{
				{
					StepDefinition: coop.StepDefinition{
						Key:   "ch1",
						Title: "Set up product",
					},
					Nodes: []coop.SessionNode{
						{
							NodeDefinition: coop.NodeDefinition{
								Key:          "n1",
								Title:        "Create product",
								Type:         coop.NodeAPIRequest,
								ReviewPrompt: "Confirm the saved price ID is reused by Checkout.",
								Request:      &coop.APIRequest{Path: "/v1/products", Method: "post", Params: map[string]string{"name": "Gold plan"}},
							},
							State:          coop.NodeDone,
							Implementation: &coop.Implementation{File: "server.js", Lines: "5-20", Note: "Created product"}},
						{
							NodeDefinition: coop.NodeDefinition{
								Key:     "n2",
								Title:   "Create checkout",
								Type:    coop.NodeAPIRequest,
								Request: &coop.APIRequest{Path: "/v1/checkout/sessions", Method: "post"},
							},
							State:    coop.NodeActive,
							Activity: "Writing endpoint"},
					},
				},
				{
					StepDefinition: coop.StepDefinition{
						Key:   "ch2",
						Title: "Handle webhooks",
					},
					Nodes: []coop.SessionNode{
						{
							NodeDefinition: coop.NodeDefinition{
								Key:    "n3",
								Title:  "Handle event",
								Type:   coop.NodeAsyncHandler,
								Events: []string{"checkout.session.completed"},
							},
							State: coop.NodePending,
						},
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
	m.session.UsedSandbox = true
	m.sandboxClaimURL = "https://dashboard.stripe.com/sandbox/claim_abc"
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

func TestRenderStepListAlignsStepTitleWithRule(t *testing.T) {
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
	titleDash := strings.Index(titleLine, "-")
	ruleDash := strings.Index(ruleLine, "─")
	require.NotEqual(t, -1, titleDash)
	require.NotEqual(t, -1, ruleDash)
	titlePrefix := titleLine[:titleDash]
	rulePrefix := ruleLine[:ruleDash]
	assert.Equal(t, lipgloss.Width(titlePrefix), lipgloss.Width(rulePrefix))
}

func TestRenderStepListShowsStepReviewUnit(t *testing.T) {
	m := testModel()
	m.session.Steps[0].Nodes[0].State = coop.NodeReview
	m.session.Steps[0].Nodes[1].State = coop.NodeReview
	m.selectStep(0)

	list := m.renderStepList()

	assertContainsPlain(t, list, "Awaiting review")
	assertContainsPlain(t, list, strings.TrimSpace(cursorMarker))
	assertContainsPlain(t, list, "Create product  Included")
	assertContainsPlain(t, list, "Create checkout  Included")
	assertNotContainsPlain(t, list, "Create product  Needs review")
}

func TestRenderStepListShowsSingleStepStepReviewUnit(t *testing.T) {
	m := testModel()
	m.session.Steps[1].Nodes[0].State = coop.NodeReview
	m.selectStep(1)

	list := m.renderStepList()
	footer := m.renderFooter()

	assertContainsPlain(t, list, "Awaiting review")
	assertContainsPlain(t, footer, "confirm all")
}

func TestRenderCollapsedStepShowsStateSummary(t *testing.T) {
	m := testModel()
	m.collapseStep(0)

	list := m.renderStepList()

	assertContainsPlain(t, list, "+ Set up product")
	assertContainsPlain(t, list, "✓1 ●1")
	assertNotContainsPlain(t, list, "Create product")
	assertNotContainsPlain(t, list, "Create checkout")
}

func TestRenderStepLineAnnotation(t *testing.T) {
	m := testModel()
	node := m.session.Steps[0].Nodes[0]
	line := m.renderNodeLine(node, 0, false, false)

	assertContainsPlain(t, line, "server.js:5-20")
}

func TestRenderStepLineActivity(t *testing.T) {
	m := testModel()
	node := m.session.Steps[0].Nodes[1]
	line := m.renderNodeLine(node, 1, false, false)

	assertContainsPlain(t, line, "Writing endpoint")
}

func TestRenderStepLineCursor(t *testing.T) {
	m := testModel()
	m.selectionCursor = 1
	node := m.session.Steps[0].Nodes[1]
	line := m.renderNodeLine(node, 1, false, true)

	assertContainsPlain(t, line, strings.TrimSpace(cursorMarker))
}

func TestRenderStepLineNoCursor(t *testing.T) {
	m := testModel()
	m.selectionCursor = 0
	node := m.session.Steps[0].Nodes[1]
	line := m.renderNodeLine(node, 1, false, false)

	assertNotContainsPlain(t, line, strings.TrimSpace(cursorMarker))
}

func TestRenderDetail(t *testing.T) {
	m := testModel()
	m.selectionCursor = 0
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
	m.selectionCursor = 0
	m.expanded = true
	m.detailTab = 0

	detail := m.renderDetail()

	assertNotContainsPlain(t, detail, "Details:")
	assert.NotContains(t, ansi.Strip(detail), "Summary")
	assertNotContainsPlain(t, detail, "Files  Checks  Reference")
	assertContainsPlain(t, detail, "Confirm the saved price ID is reused")
	assertContainsPlain(t, detail, "Confirmation steps")
	assertNotContainsPlain(t, detail, "POST /v1/products")
	assertNotContainsPlain(t, detail, "You check")
}

func TestRenderSummaryDetailShowsStepSDKSnippet(t *testing.T) {
	m := testModel()
	m.selectionCursor = 0
	m.expanded = true
	m.detailTab = 0
	m.sdkSnippet = "const product = await stripe.products.create({name: 'Gold plan'});"
	m.sdkSnippetNode = 0

	detail := m.renderDetail()

	assertContainsPlain(t, detail, "SDK example")
	assertContainsPlain(t, detail, "stripe.products.create")
}

func TestRenderStepDetailUsesStepOverview(t *testing.T) {
	m := testModel()
	m.selectStep(0)
	m.expanded = true
	m.detailTab = 0

	detail := m.renderDetail()

	assertContainsPlain(t, detail, "✓ Create product")
	assertContainsPlain(t, detail, "● Create checkout")
	assertContainsPlain(t, detail, "Confirmation steps")
	assertContainsPlain(t, detail, "Agent help")
	assertNotContainsPlain(t, detail, "SDK example")
}

func TestRenderDetailWebhook(t *testing.T) {
	m := testModel()
	m.selectionCursor = 2 // asyncHandler node
	m.expanded = true
	m.detailTab = 2
	m.session.Steps[1].Nodes[0].Events = []string{"checkout.session.completed", "invoice.paid"}
	detail := m.renderDetail()

	assertContainsPlain(t, detail, "Checks")
	assertContainsPlain(t, detail, "Review command")
	assertContainsPlain(t, detail, "How to verify")
	assertContainsPlain(t, detail, "stripe listen")
	assertContainsPlain(t, detail, "stripe trigger checkout.session.completed")
	assertContainsPlain(t, detail, "stripe trigger invoice.paid")
}

func TestRenderDetailWebhookReferenceShowsGeneratedExample(t *testing.T) {
	m := testModel()
	m.selectionCursor = 2 // asyncHandler node
	m.expanded = true
	m.detailTab = 3
	m.session.Steps[1].Nodes[0].Events = []string{"checkout.session.completed", "invoice.paid"}

	detail := m.renderDetail()

	assertContainsPlain(t, detail, "Reference")
	assertContainsPlain(t, detail, "Webhook triggers")
	assertContainsPlain(t, detail, "stripe trigger checkout.session.completed")
	assertContainsPlain(t, detail, "stripe trigger invoice.paid")
	assertContainsPlain(t, detail, "Webhook handler example")
	assertContainsPlain(t, detail, "stripe.v2.core.events.retrieve")
	assertContainsPlain(t, detail, "case \"checkout.session.completed\"")
	assertContainsPlain(t, detail, "case \"invoice.paid\"")
}

func TestRenderDetailWithSDKSnippet(t *testing.T) {
	m := testModel()
	m.selectionCursor = 0
	m.expanded = true
	m.detailTab = 3
	m.sdkSnippet = "const product = await stripe.products.create({});"
	m.sdkSnippetNode = 0
	detail := m.renderDetail()

	assertContainsPlain(t, detail, "Reference")
	assertContainsPlain(t, detail, "stripe.products.create")
}

func TestRenderDetailFitsPaneWithIndent(t *testing.T) {
	m := testModel()
	m.width = 69
	m.selectionCursor = 0
	m.expanded = true
	m.detailTab = 1
	m.session.Steps[0].Nodes[0].State = coop.NodeReview
	m.session.Steps[0].Nodes[1].State = coop.NodeDone
	m.session.Steps[0].Nodes[0].Implementation.Snippet = strings.Repeat("const createdCheckoutSessionWithLongIdentifier = await stripe.checkout.sessions.create({ mode: 'payment' })\n", 5)

	detail := m.renderDetail()

	assertLinesWithinWidth(t, detail, m.width)
	assertContainsPlain(t, detail, "Agent wrote")
}

func TestRenderDetailBoxMatchesOutlineWidth(t *testing.T) {
	m := testModel()
	m.width = 69
	m.selectionCursor = 0
	m.expanded = true

	detail := ansi.Strip(m.renderDetail())
	lines := strings.Split(detail, "\n")
	require.NotEmpty(t, lines)

	assert.Equal(t, m.outlineRuleWidth(), lipgloss.Width(strings.TrimPrefix(lines[0], strings.Repeat(" ", detailIndent))))
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
	m.selectionCursor = 0
	footer := m.renderFooter()

	// Step 0 is done — no review actions
	assertContainsPlain(t, footer, "enter")
	assertContainsPlain(t, footer, "quit")
	assertNotContainsPlain(t, footer, "confirm")
}

func TestRenderFooterReviewStep(t *testing.T) {
	m := testModel()
	m.session.Steps[0].Nodes[0].State = coop.NodeReview
	m.session.Steps[0].Nodes[1].State = coop.NodeDone
	m.selectionCursor = 0
	footer := m.renderFooter()

	assertContainsPlain(t, footer, "confirm")
	assertContainsPlain(t, footer, "changes")
	assertContainsPlain(t, footer, "Review")
	assertContainsPlain(t, footer, "Agent changed")
	assertContainsPlain(t, footer, "Confirmation steps")
}

func TestRenderReviewCardEvidence(t *testing.T) {
	m := testModel()
	m.session.Steps[0].Nodes[0].State = coop.NodeReview
	m.session.Steps[0].Nodes[1].State = coop.NodeDone
	m.session.Steps[0].Nodes[0].ReviewPrompt = "Confirm Checkout uses the saved price ID."
	m.session.Steps[0].Nodes[0].Verifications = []coop.Verification{
		{Check: "Visit http://localhost:3000/checkout, click Pay, and confirm Checkout opens with the saved price.", Passed: true},
		{Check: "Confirm the failure banner appears for declined cards.", Passed: false},
	}
	m.selectionCursor = 0

	card := m.renderReviewCard()

	assertContainsPlain(t, card, "Review")
	assertNotContainsPlain(t, card, "Review: Create product")
	assertContainsPlain(t, card, "Agent changed:")
	assertContainsPlain(t, card, "server.js:5-20")
	assertContainsPlain(t, card, "Agent verified:")
	assertContainsPlain(t, card, "1/2 check(s) passed")
	assertContainsPlain(t, card, "Confirmation steps")
	assertContainsPlain(t, card, "Visit http://localhost:3000/checkout")
	assertNotContainsPlain(t, card, "Confirm Checkout uses the saved price ID.")
	assertNotContainsPlain(t, card, "declined cards")
	plain := ansi.Strip(card)
	assert.Less(t, strings.Index(plain, "Confirmation steps"), strings.Index(plain, "Agent changed:"))
}

func TestRenderReviewCardFallsBackToBlueprintConfirmation(t *testing.T) {
	m := testModel()
	m.session.Steps[0].Nodes[0].State = coop.NodeReview
	m.session.Steps[0].Nodes[1].State = coop.NodeDone
	m.session.Steps[0].Nodes[0].ReviewPrompt = "Confirm Checkout uses the saved price ID."
	m.selectionCursor = 0

	card := m.renderReviewCard()

	assertContainsPlain(t, card, "Confirmation steps")
	assertContainsPlain(t, card, "Confirm Checkout uses the saved price ID.")
}

func TestRenderStepReviewCardNamesCoveredSteps(t *testing.T) {
	m := testModel()
	m.session.Steps[0].Nodes[0].State = coop.NodeReview
	m.session.Steps[0].Nodes[1].State = coop.NodeReview
	m.selectStep(0)

	card := m.renderReviewCard()
	footer := m.renderFooter()

	assertContainsPlain(t, card, "Review step")
	assertNotContainsPlain(t, card, "Review step (2 steps): Set up product")
	assertContainsPlain(t, card, "Includes: Create product, Create checkout")
	assertContainsPlain(t, footer, "confirm all")
	assertContainsPlain(t, footer, "changes")
}

func TestRenderReviewCardFallbackCheck(t *testing.T) {
	m := testModel()
	m.session.Steps[1].Nodes[0].State = coop.NodeReview
	m.selectionCursor = 2

	card := m.renderReviewCard()

	assertContainsPlain(t, card, "Confirm the completed work matches this step")
}

func TestRenderFooterReviewCommand(t *testing.T) {
	m := testModel()
	m.session.Steps[1].Nodes[0].State = coop.NodeReview
	m.selectionCursor = 2
	footer := m.renderFooter()

	assertContainsPlain(t, footer, "Run:")
	assertContainsPlain(t, footer, "stripe trigger checkout.session.completed")
	assertContainsPlain(t, footer, "y copy")
}

func TestSelectedReviewCommandForAsyncNodeUsesAllEvents(t *testing.T) {
	m := testModel()
	m.session.Steps[1].Nodes[0].State = coop.NodeReview
	m.session.Steps[1].Nodes[0].Events = []string{"checkout.session.completed", "invoice.paid"}
	m.selectionCursor = 2

	assert.Equal(t, "stripe trigger checkout.session.completed && stripe trigger invoice.paid", m.selectedReviewCommand())
}

func TestRenderFooterReviewNotice(t *testing.T) {
	m := testModel()
	m.session.Steps[0].Nodes[0].State = coop.NodeReview
	m.session.Steps[0].Nodes[1].State = coop.NodeDone
	footer := m.renderFooter()

	assertContainsPlain(t, footer, "Waiting for you")
	assertContainsPlain(t, footer, "review step")
}

func TestRenderCompletionView(t *testing.T) {
	m := withCompletionSuggestions(testModel())
	m.session.Steps[0].Nodes[0].State = coop.NodeDone
	m.session.Steps[0].Nodes[1].State = coop.NodeDone
	m.session.Steps[1].Nodes[0].State = coop.NodeDone

	view := m.renderCompletionView()

	assertContainsPlain(t, view, "Integration complete")
	assertContainsPlain(t, view, "Built")
	assertContainsPlain(t, view, "Set up product")
	assertContainsPlain(t, view, "Handle webhooks")
	assertContainsPlain(t, view, "Important checks")
	assertContainsPlain(t, view, "Confirm the saved price ID is reused by Checkout.")
	assertContainsPlain(t, view, "Next steps")
	assertContainsPlain(t, view, "STRIPE.md")
	assertContainsPlain(t, view, "Add another Stripe feature")
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
	m.session.Steps = []coop.SessionStep{
		{
			StepDefinition: coop.StepDefinition{Key: "context-step", Title: "Project context"},
			Nodes: []coop.SessionNode{
				{NodeDefinition: coop.NodeDefinition{Title: "Understand project"}, State: coop.NodeDone},
			},
		},
		{
			StepDefinition: coop.StepDefinition{Key: "built-with-skipped", Title: "Built with skipped optional work"},
			Nodes: []coop.SessionNode{
				{NodeDefinition: coop.NodeDefinition{Title: "Required"}, State: coop.NodeDone},
				{NodeDefinition: coop.NodeDefinition{Title: "Optional"}, State: coop.NodeSkipped},
			},
		},
		{
			StepDefinition: coop.StepDefinition{Key: "incomplete", Title: "Incomplete step"},
			Nodes: []coop.SessionNode{
				{NodeDefinition: coop.NodeDefinition{Title: "Done"}, State: coop.NodeDone},
				{NodeDefinition: coop.NodeDefinition{Title: "Still active"}, State: coop.NodeActive},
			},
		},
	}

	assert.Equal(t, []string{"Built with skipped optional work"}, m.completionBuiltItems())
}

func TestCompletionImportantChecksDedupesDoneOnlyAndCaps(t *testing.T) {
	m := testModel()
	m.session.Steps = []coop.SessionStep{
		{
			StepDefinition: coop.StepDefinition{Key: "checks", Title: "Checks"},
			Nodes: []coop.SessionNode{
				{NodeDefinition: coop.NodeDefinition{Title: "First", ReviewPrompt: "Check one"}, State: coop.NodeDone},
				{NodeDefinition: coop.NodeDefinition{Title: "Duplicate", ReviewPrompt: "Check one"}, State: coop.NodeDone},
				{NodeDefinition: coop.NodeDefinition{Title: "Active", ReviewPrompt: "Do not include active"}, State: coop.NodeActive},
				{NodeDefinition: coop.NodeDefinition{Title: "Second", ReviewPrompt: "Check two"}, State: coop.NodeDone},
				{NodeDefinition: coop.NodeDefinition{Title: "Third", ReviewPrompt: "Check three"}, State: coop.NodeDone},
				{NodeDefinition: coop.NodeDefinition{Title: "Fourth", ReviewPrompt: "Check four"}, State: coop.NodeDone},
				{NodeDefinition: coop.NodeDefinition{Title: "Fifth", ReviewPrompt: "Do not include after cap"}, State: coop.NodeDone},
			},
		},
	}

	assert.Equal(t, []string{"Check one", "Check two"}, m.completionImportantChecks())
}

func TestCompletionImportantChecksWrapOnWordBoundaries(t *testing.T) {
	m := completionLayoutModel()
	m.session.Steps[0].Nodes[0].ReviewPrompt = "Open the app and confirm the user-facing flow works as described."

	receipt := m.renderCompletionReceipt(65)

	assertNotContainsPlain(t, receipt, "a\n    s")
	assertNotContainsPlain(t, receipt, "\n    s                                                                described.")
	assertContainsPlain(t, receipt, "as\n    described.")
	assertLinesWithinWidth(t, receipt, 69)
}

func TestGetCompletionSuggestionsDefaultEmpty(t *testing.T) {
	m := testModel()
	suggestions := m.getCompletionSuggestions()

	assert.Empty(t, suggestions)
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
		NodeDefinition: coop.NodeDefinition{Key: "test", Title: "Step"},
		State:          coop.NodeActive,
		Activity:       "This is a very long activity note that should wrap",
	}
	line := m.renderNodeLine(node, 0, false, false)

	// Should have a newline (wrapped)
	assert.True(t, strings.Contains(line, "\n"))
}

func TestAnnotationInlineAtWideWidth(t *testing.T) {
	m := testModel()
	m.width = 120
	node := coop.SessionNode{
		NodeDefinition: coop.NodeDefinition{Key: "test", Title: "Step"},
		State:          coop.NodeActive,
		Activity:       "Short note",
	}
	line := m.renderNodeLine(node, 0, false, false)

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
		NodeDefinition: coop.NodeDefinition{Key: "skipped", Title: "Skipped step"},
		State:          coop.NodeSkipped,
		Activity:       "Not needed for this project",
	}
	line := m.renderNodeLine(node, 0, false, false)
	assertContainsPlain(t, line, "Not needed")
}

func TestRenderDetailSkipped(t *testing.T) {
	m := testModel()
	m.session.Steps[0].Nodes[0].State = coop.NodeSkipped
	m.session.Steps[0].Nodes[0].Activity = "Already handled"
	m.selectionCursor = 0
	detail := m.renderDetail()
	assertContainsPlain(t, detail, "Skipped")
}

func TestRenderCompletionViewWithCompleted(t *testing.T) {
	m := withCompletionSuggestions(testModel())
	for i := range m.session.Steps {
		for j := range m.session.Steps[i].Nodes {
			m.session.Steps[i].Nodes[j].State = coop.NodeDone
		}
	}
	m.session.NextSteps.Completed = []string{"summarize"}
	m.width = 80
	m.height = 30

	view := m.renderCompletionView()
	assertContainsPlain(t, view, "✓ Write a STRIPE.md summary")
}

func TestRenderFooterComplete(t *testing.T) {
	m := testModel()
	for i := range m.session.Steps {
		for j := range m.session.Steps[i].Nodes {
			m.session.Steps[i].Nodes[j].State = coop.NodeDone
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
	m.session.Steps[0].Nodes[0].State = coop.NodeReview
	m.session.Steps[0].Nodes[1].State = coop.NodeDone
	m.selectionCursor = 0
	m.rejecting = true
	m.rejectionInput.SetValue("Missing webhook test")

	footer := m.renderFooter()

	assertContainsPlain(t, footer, "enter send")
	assertContainsPlain(t, footer, "esc cancel")
	assertContainsPlain(t, footer, "Missing webhook test")
}

func TestRenderFooterRejectionPlaceholder(t *testing.T) {
	m := testModel()
	m.session.Steps[1].Nodes[0].State = coop.NodeReview
	m.selectionCursor = 2
	m.rejecting = true
	target, _ := m.selectedReviewTarget()
	m.rejectionInput.Placeholder = m.requestChangesPlaceholder(target)
	m.rejectionInput.Focus()

	footer := m.renderFooter()

	assertContainsPlain(t, footer, "Describe what should change in this step")
}

func TestReviewCardFitsWithinShortViewport(t *testing.T) {
	m := testModel()
	m.ready = true
	m.width = 56
	m.height = 18
	m.viewport = viewport.New(viewport.WithWidth(56), viewport.WithHeight(10))
	m.session.Steps[0].Nodes[0].State = coop.NodeReview
	m.session.Steps[0].Nodes[1].State = coop.NodeDone
	m.session.Steps[0].Nodes[0].ReviewPrompt = "Open the local application, complete the Checkout flow, confirm the redirect lands on the success page, confirm the saved price ID is reused, and confirm no secret keys or generated IDs are committed."
	m.session.Steps[0].Nodes[0].Verifications = []coop.Verification{
		{Check: "Created product and price", Passed: true},
		{Check: "Saved price ID for Checkout", Passed: true},
		{Check: "Ran local Checkout flow", Passed: true},
	}
	m.selectionCursor = 0

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
	m.session.Steps[0].Nodes[0].State = coop.NodeReview
	m.session.Steps[0].Nodes[1].State = coop.NodeDone
	m.session.Steps[0].Nodes[0].ReviewPrompt = "Confirm the Checkout flow, success page, saved price ID, webhook event handling, and environment variable setup all match the intended integration."
	m.session.Steps[0].Nodes[0].Verifications = []coop.Verification{
		{Check: "Created product", Passed: true},
		{Check: "Created price", Passed: true},
		{Check: "Created Checkout Session", Passed: true},
	}
	m.selectionCursor = 0

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
	m.session.Steps[0].Nodes[0].State = coop.NodeReview
	m.session.Steps[0].Nodes[0].ReviewPrompt = "Confirm the product, price, Checkout Session, redirect URL, success page, saved price ID, webhook event handling, and environment variable setup all match the intended integration."
	m.session.Steps[0].Nodes[0].Implementation = &coop.Implementation{File: "server/routes/payments/checkout/session/handler/with/a/long/path.js", Lines: "42-118"}
	m.session.Steps[0].Nodes[0].Verifications = []coop.Verification{
		{Check: "Created product", Passed: true},
		{Check: "Created price", Passed: true},
		{Check: "Created Checkout Session", Passed: true},
	}
	m.session.Steps[0].Nodes[1].State = coop.NodeReview
	m.session.Steps[0].Nodes[1].ReviewPrompt = "Open the app locally, click the Checkout button, complete payment, and confirm the redirect lands on the expected success page without exposing secret keys."
	m.session.Steps[0].Nodes[1].Implementation = &coop.Implementation{File: "client/src/components/payments/checkout-button-with-long-name.tsx", Lines: "9-88"}
	m.session.Steps[0].Nodes[1].Verifications = []coop.Verification{
		{Check: "Rendered Checkout button", Passed: true},
		{Check: "Confirmed redirect", Passed: true},
	}
	m.selectStep(0)

	m.resizeViewport()
	m.syncViewport()
	view := m.View().Content

	assert.LessOrEqual(t, lipgloss.Height(view), m.height)
	assertLinesWithinWidth(t, view, m.width)
	assertContainsPlain(t, view, "Stripe Co-op")
	assertContainsPlain(t, view, "Review step")
	assertContainsPlain(t, view, "q quit")
}

func TestViewportShowsMoreBelowIndicator(t *testing.T) {
	m := testModel()
	m.ready = true
	m.width = 69
	m.height = 12
	m.viewport = viewport.New(viewport.WithWidth(69), viewport.WithHeight(4))
	m.session.Steps = []coop.SessionStep{{
		StepDefinition: coop.StepDefinition{Key: "long", Title: "Long step"},
		Nodes: []coop.SessionNode{
			{NodeDefinition: coop.NodeDefinition{Title: "One"}, State: coop.NodeDone},
			{NodeDefinition: coop.NodeDefinition{Title: "Two"}, State: coop.NodeDone},
			{NodeDefinition: coop.NodeDefinition{Title: "Three"}, State: coop.NodeDone},
			{NodeDefinition: coop.NodeDefinition{Title: "Four"}, State: coop.NodeDone},
			{NodeDefinition: coop.NodeDefinition{Title: "Five"}, State: coop.NodeDone},
			{NodeDefinition: coop.NodeDefinition{Title: "Six"}, State: coop.NodeDone},
		},
	}}
	m.selectionCursor = 0
	m.resizeViewport()
	m.syncViewport()
	m.viewport.SetHeight(4)
	m.viewport.SetYOffset(0)

	rendered := m.renderViewportRegionWithHeight(4)

	assertContainsPlain(t, rendered, "more below")
	assertLinesWithinWidth(t, rendered, m.width)
}

func TestViewportClosesClippedDetailBoxBeforeMoreBelowIndicator(t *testing.T) {
	m := testModel()
	m.ready = true
	m.width = 69
	m.height = 12
	m.viewport = viewport.New(viewport.WithWidth(69), viewport.WithHeight(6))
	m.session.Steps[0].Nodes[0].ReviewPrompt = strings.Repeat("Confirm the Checkout flow uses the saved price ID and redirects correctly. ", 5)
	m.selectionCursor = 0
	m.expanded = true
	m.resizeViewport()
	m.syncViewport()
	m.viewport.SetHeight(6)
	m.viewport.SetYOffset(3)

	rendered := ansi.Strip(m.renderViewportRegionWithHeight(6))

	assert.Contains(t, rendered, "╰")
	assert.Contains(t, rendered, "╯")
	assertContainsPlain(t, rendered, "more below")
	assertLinesWithinWidth(t, rendered, m.width)
}

func TestViewportBoundaryDoesNotTurnTopBorderIntoBottomBorder(t *testing.T) {
	rendered := closeOpenBoxAtViewportBoundary("before\n  ╭────────╮")

	assert.Contains(t, rendered, "╭")
	assert.NotContains(t, rendered, "╰")
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
		state    coop.NodeState
		contains string
	}{
		{coop.NodeDone, "✓"},
		{coop.NodeReview, "◆"},
		{coop.NodeSkipped, "–"},
		{coop.NodePending, "○"},
	}

	for _, tc := range cases {
		node := coop.SessionNode{State: tc.state}
		icon := m.nodeIcon(node)
		assert.Contains(t, ansi.Strip(icon), tc.contains, "state %s should contain %s", tc.state, tc.contains)
	}
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
