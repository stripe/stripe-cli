package tui

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"charm.land/bubbles/v2/spinner"
	"charm.land/bubbles/v2/viewport"
	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/stripe/stripe-cli/pkg/coop"
)

type layoutSize struct {
	name   string
	width  int
	height int
}

type layoutScenario struct {
	name             string
	model            func() Model
	footerToken      string
	expectCursor     bool
	expectReviewCard bool
	expectCompletion bool
}

var layoutMatrixSizes = []layoutSize{
	{name: "tiny", width: 40, height: 12},
	{name: "narrow_acceptance", width: 56, height: 18},
	{name: "coop_start_split", width: 69, height: 50},
	{name: "normal", width: 80, height: 24},
	{name: "wide", width: 120, height: 40},
}

func TestUILayoutMatrix(t *testing.T) {
	scenarios := []layoutScenario{
		{
			name:        "waiting",
			model:       waitingLayoutModel,
			footerToken: "q quit",
		},
		{
			name:         "active_step",
			model:        activeStepLayoutModel,
			footerToken:  "enter",
			expectCursor: true,
		},
		{
			name:             "review_step_long_prompt",
			model:            reviewStepLongPromptLayoutModel,
			footerToken:      "enter",
			expectCursor:     true,
			expectReviewCard: true,
		},
		{
			name:             "step_review_many_changes",
			model:            stepReviewLayoutModel,
			footerToken:      "enter",
			expectCursor:     true,
			expectReviewCard: true,
		},
		{
			name:             "request_changes_input",
			model:            requestChangesLayoutModel,
			footerToken:      "esc cancel",
			expectCursor:     true,
			expectReviewCard: true,
		},
		{
			name:        "manual_navigation",
			model:       manualNavigationLayoutModel,
			footerToken: "f review",
		},
		{
			name:         "expanded_details",
			model:        expandedDetailsLayoutModel,
			footerToken:  "enter",
			expectCursor: true,
		},
		{
			name:             "completion",
			model:            completionLayoutModel,
			footerToken:      "enter",
			expectCompletion: true,
		},
	}

	for _, scenario := range scenarios {
		for _, size := range layoutMatrixSizes {
			t.Run(scenario.name+"/"+size.name, func(t *testing.T) {
				m := scenario.model()
				rendered := renderLayoutScenario(&m, size)
				writeLayoutCapture(t, scenario.name, size, rendered)

				assertLayoutFits(t, rendered, size)
				assertHeaderIsPinned(t, rendered)
				assertFooterIsPinned(t, rendered, scenario.footerToken)
				// While the feedback editor is open it takes priority over the
				// cursor row: the user is typing into it, and the step it
				// belongs to is still named by the pinned footer note. The card
				// is taller than a short viewport, so both cannot be shown.
				if scenario.expectCursor && !scenario.model().rejecting {
					assertSelectedRowVisible(t, rendered)
				}
				if scenario.expectReviewCard {
					assertReviewAffordanceVisible(t, m, rendered)
					assert.LessOrEqual(t, lipgloss.Height(m.renderFooter()), m.footerHeightBudget(), "review footer should stay within its budget")
				}
				if scenario.expectCompletion {
					assert.Contains(t, rendered, "Integration complete")
					assert.NotContains(t, rendered, "Waiting for agent to continue")
				}
			})
		}
	}
}

func TestCompletionTransitionClearsTransientStatus(t *testing.T) {
	m := activeStepLayoutModel()
	m.ready = true
	m.width = 80
	m.height = 24
	m.viewport = viewport.New(viewport.WithWidth(80), viewport.WithHeight(10))
	m.statusMessage = "Waiting for agent to continue..."
	m.statusExpiresAt = m.lastUpdateTime.Add(10)

	next := completionLayoutModel().session
	next.ID = m.session.ID
	next.Blueprint = m.session.Blueprint
	next.Settings = m.session.Settings

	updatedModel, _ := m.Update(sessionUpdatedMsg{session: next})
	updated := updatedModel.(Model)
	rendered := renderLayoutScenario(&updated, layoutSize{name: "normal", width: 80, height: 24})

	assert.Contains(t, rendered, "Integration complete")
	assert.Contains(t, rendered, "Waiting for agent to publish next steps")
	assert.NotContains(t, rendered, "Waiting for agent to continue")
}

func TestSessionUpdateResizesAfterAutoSelectingReview(t *testing.T) {
	m := testModel()
	m.spinner = staticSpinner()
	m.ready = true
	m.width = 56
	m.height = 18
	m.viewport = viewport.New(viewport.WithWidth(m.width), viewport.WithHeight(10))
	m.selectionCursor = 0
	m.session.Steps[0].Nodes[0].State = coop.NodeDone
	m.session.Steps[0].Nodes[1].State = coop.NodeReview
	m.session.Steps[0].Nodes[1].Title.DefaultMessage = "Review Checkout Session creation, saved IDs, redirect behavior, and webhook assumptions"
	m.session.Steps[0].Nodes[1].ReviewPrompt = "Open the local app, start Checkout, inspect the server logs, confirm the saved price ID is reused instead of creating a new Price, confirm the redirect URL is correct, confirm errors are handled without exposing secrets, and confirm the success page reflects the completed payment."
	m.session.Steps[0].Nodes[1].Implementation = &coop.Implementation{
		File:  "server/src/payments/checkout/session/create_checkout_session_handler_with_long_name.ts",
		Lines: "42-118",
	}
	m.session.Steps[0].Nodes[1].Verifications = []coop.Verification{
		{Check: "Created product and price", Passed: true},
		{Check: "Confirmed Checkout reuses the saved price ID", Passed: true},
	}

	updatedModel, _ := m.Update(sessionUpdatedMsg{session: m.session})
	updated := updatedModel.(Model)
	rendered := updated.View().Content

	assert.Equal(t, navigationStep, updated.selected.kind)
	assert.Equal(t, 0, updated.selected.stepIndex)
	assertLayoutFits(t, rendered, layoutSize{name: "narrow_acceptance", width: 56, height: 18})
	assertHeaderIsPinned(t, rendered)
	assertFooterIsPinned(t, rendered, "enter")
	assert.Contains(t, rendered, "To confirm")
}

func TestFooterActionRowStaysPinnedAcrossFooterModes(t *testing.T) {
	size := layoutSize{name: "coop_start_split", width: 69, height: 50}
	scenarios := []struct {
		name  string
		model func() Model
		token string
	}{
		{name: "active", model: activeStepLayoutModel, token: "enter"},
		{name: "review", model: reviewStepLongPromptLayoutModel, token: "enter"},
		{name: "manual", model: manualNavigationLayoutModel, token: "f review"},
		{name: "request_changes", model: requestChangesLayoutModel, token: "esc cancel"},
	}

	expectedRow := -1
	for _, scenario := range scenarios {
		t.Run(scenario.name, func(t *testing.T) {
			m := scenario.model()
			rendered := renderLayoutScenario(&m, size)
			row := lineIndexContaining(rendered, scenario.token)
			require.NotEqual(t, -1, row, "footer action row should be visible")
			assert.Equal(t, size.height-1, row, "footer action row should sit on the final terminal row")
			if expectedRow == -1 {
				expectedRow = row
			}
			assert.Equal(t, expectedRow, row, "footer action row should not jump when card/status content changes")
		})
	}
}

func TestPinnedViewportCapsStaleViewportHeight(t *testing.T) {
	size := layoutSize{name: "coop_start_split", width: 69, height: 50}
	m := reviewStepLongPromptLayoutModel()
	m = prepareInteractiveModel(m, size.width, size.height)
	m.viewport.SetHeight(size.height)

	rendered := m.View().Content

	assertLayoutFits(t, rendered, size)
	assertHeaderIsPinned(t, rendered)
	assertFooterIsPinned(t, rendered, "enter")
	assert.Equal(t, size.height-1, lineIndexContaining(rendered, "enter"))
}

func renderLayoutScenario(m *Model, size layoutSize) string {
	m.ready = true
	m.width = size.width
	m.height = size.height
	m.viewport = viewport.New(viewport.WithWidth(size.width), viewport.WithHeight(10))
	m.resizeViewport()
	m.syncViewport()
	return m.View().Content
}

func assertLayoutFits(t *testing.T, rendered string, size layoutSize) {
	t.Helper()
	assert.LessOrEqual(t, lipgloss.Height(rendered), size.height, "layout should not exceed terminal height")
	assertLinesWithinWidth(t, rendered, size.width)
}

// assertReviewAffordanceVisible checks the reviewer can still act. What that
// means depends on state: while typing feedback the editor is the thing that
// must survive truncation, otherwise it is the instruction — which may have
// collapsed to a one-line hint on a short terminal.
func assertReviewAffordanceVisible(t *testing.T, m Model, rendered string) {
	t.Helper()
	if m.rejecting {
		assert.Contains(t, ansi.Strip(rendered), "Request changes", "feedback editor should stay visible")
		return
	}
	assert.Contains(t, ansi.Strip(rendered), "To confirm", "review instruction should stay visible, in full or collapsed")
}

func assertHeaderIsPinned(t *testing.T, rendered string) {
	t.Helper()
	lines := strings.Split(rendered, "\n")
	require.NotEmpty(t, lines)
	window := strings.Join(lines[:min(len(lines), 3)], "\n")
	assert.Contains(t, window, "Stripe Co-op", "header should stay in the first three lines")
}

func assertFooterIsPinned(t *testing.T, rendered, token string) {
	t.Helper()
	lines := strings.Split(strings.TrimRight(rendered, "\n"), "\n")
	require.NotEmpty(t, lines)
	window := ansi.Strip(strings.Join(lines[max(len(lines)-4, 0):], "\n"))
	assert.Contains(t, window, token, "primary footer action should stay near the bottom")
}

func lineIndexContaining(rendered, token string) int {
	for i, line := range strings.Split(rendered, "\n") {
		if strings.Contains(ansi.Strip(line), token) {
			return i
		}
	}
	return -1
}

func writeLayoutCapture(t *testing.T, scenario string, size layoutSize, rendered string) {
	t.Helper()
	dir := os.Getenv("COOP_TUI_CAPTURE_DIR")
	if dir == "" {
		return
	}
	require.NoError(t, os.MkdirAll(dir, 0755))
	name := fmt.Sprintf("%s-%s-%dx%d.txt", scenario, size.name, size.width, size.height)
	require.NoError(t, os.WriteFile(filepath.Join(dir, name), []byte(rendered), 0644))
}

func waitingLayoutModel() Model {
	m := NewWaitingModel(nil, nil)
	m.spinner = staticSpinner()
	return m
}

func activeStepLayoutModel() Model {
	m := testModel()
	m.spinner = staticSpinner()
	m.session.Steps[0].Nodes[0].State = coop.NodeDone
	m.session.Steps[0].Nodes[1].State = coop.NodeActive
	m.session.Steps[0].Nodes[1].Activity = "Adding the Checkout endpoint and wiring the returned session URL into the app"
	m.selectionCursor = 1
	return m
}

func reviewStepLongPromptLayoutModel() Model {
	m := testModel()
	m.spinner = staticSpinner()
	m.session.Steps[0].Nodes[0].State = coop.NodeReview
	m.session.Steps[0].Nodes[1].State = coop.NodeDone
	m.session.Steps[0].Nodes[0].Implementation = &coop.Implementation{
		File:  "server/routes/payments/checkout/session_handler.js",
		Lines: "42-118",
		Note:  "Created product and price setup for Checkout.",
	}
	m.session.Steps[0].Nodes[0].ReviewPrompt = "Open the local app, complete the Checkout flow, confirm the saved price ID is reused by the Checkout Session, confirm the redirect lands on the success page, and confirm no secret keys are committed."
	m.session.Steps[0].Nodes[0].Verifications = []coop.Verification{
		{Check: "Created product and price", Passed: true},
		{Check: "Saved price ID for Checkout", Passed: true},
		{Check: "Ran local Checkout flow", Passed: true},
	}
	m.selectionCursor = 0
	return m
}

// A failing check is the highest-stakes thing the card renders and nothing in
// the matrix produced one, so every frame reviewed so far showed only the
// all-passed shape.
func failedCheckLayoutModel() Model {
	m := reviewStepLongPromptLayoutModel()
	m.session.Steps[0].Nodes[0].Verifications = []coop.Verification{
		{Check: "Created product and price", Passed: true},
		{Check: "Checkout Session creates a new price on every request instead of reusing the persisted price ID", Passed: false},
		{Check: "Ran local Checkout flow", Passed: true},
	}
	return m
}

// Sessions in the fixtures predate step descriptions, so no captured frame had
// a Goal line and its spacing went unreviewed.
func goalLayoutModel() Model {
	m := reviewStepLongPromptLayoutModel()
	m.session.Steps[0].Description = coop.MessageDescriptor{DefaultMessage: "Create a product and a recurring price once, and persist the price ID so later steps reuse it rather than creating a new price per request."}
	return m
}

func stepReviewLayoutModel() Model {
	m := testModel()
	m.spinner = staticSpinner()
	m.session.Steps[0].Nodes[0].State = coop.NodeReview
	m.session.Steps[0].Nodes[0].ReviewPrompt = "Confirm the product and price are created once, persisted for later steps, and reused by the Checkout Session."
	m.session.Steps[0].Nodes[0].Implementation = &coop.Implementation{File: "server/catalog/stripe_products.js", Lines: "10-84"}
	m.session.Steps[0].Nodes[0].Verifications = []coop.Verification{{Check: "Created product", Passed: true}, {Check: "Created price", Passed: true}}
	m.session.Steps[0].Nodes[1].State = coop.NodeReview
	m.session.Steps[0].Nodes[1].ReviewPrompt = "Open the app locally, click the Checkout button, complete payment, and confirm the redirect reaches the expected success page."
	m.session.Steps[0].Nodes[1].Implementation = &coop.Implementation{File: "client/src/components/CheckoutButton.tsx", Lines: "9-88"}
	m.session.Steps[0].Nodes[1].Verifications = []coop.Verification{{Check: "Rendered Checkout button", Passed: true}, {Check: "Confirmed redirect", Passed: true}}
	m.selectStep(0)
	return m
}

func requestChangesLayoutModel() Model {
	m := reviewStepLongPromptLayoutModel()
	m.rejecting = true
	m.rejectionInput.SetValue("The Checkout Session should use the persisted price ID instead of creating a new price for every request.")
	return m
}

func manualNavigationLayoutModel() Model {
	m := reviewStepLongPromptLayoutModel()
	m.selectionCursor = 2
	m.userMoved = true
	return m
}

func expandedDetailsLayoutModel() Model {
	m := reviewStepLongPromptLayoutModel()
	m.detailTab = 1
	m.session.Steps[0].Nodes[0].Implementation.Snippet = strings.Repeat("const session = await stripe.checkout.sessions.create({ mode: 'payment' })\n", 8)
	return m
}

func completionLayoutModel() Model {
	m := testModel()
	for i := range m.session.Steps {
		for j := range m.session.Steps[i].Nodes {
			m.session.Steps[i].Nodes[j].State = coop.NodeDone
		}
	}
	m.selectionCursor = 0
	return m
}

func withCompletionSuggestions(m Model) Model {
	m.session.NextSteps = &coop.NextStepsState{
		Suggestions: []coop.NextStepSuggestion{
			{ID: "summarize", Title: "Write a STRIPE.md summary", Description: "Ask the agent to document what changed."},
			{ID: "add-integration", Title: "Add another Stripe feature", Description: "Ask the agent to start another co-op session."},
			{ID: "done", Title: "Finish", Description: "Close this co-op session."},
		},
	}
	return m
}

func staticSpinner() spinner.Model {
	s := spinner.New()
	s.Spinner = spinner.Spinner{Frames: []string{"●"}, FPS: 1}
	return s
}

// The detail box in the split workspace is indented as a block. Trimming
// whitespace rather than newlines stripped that indent from the first line
// only, leaving the top border two columns left of its own sides.
func TestSplitWorkspaceDetailBoxIsColumnAligned(t *testing.T) {
	m := stepReviewLayoutModel()
	rendered := renderLayoutScenario(&m, layoutSize{name: "wide", width: 120, height: 34})

	// Only the right pane. The nav column now draws a small status card under
	// each step header, so the frame legitimately contains more than one box;
	// this test is about the detail box holding a single left edge.
	const rightPaneStart = 42
	columns := map[int]int{}
	for _, line := range strings.Split(ansi.Strip(rendered), "\n") {
		for col, r := range []rune(line) {
			if col < rightPaneStart {
				continue
			}
			if r == '╭' || r == '│' || r == '╰' {
				columns[col]++
				break
			}
		}
	}

	require.NotEmpty(t, columns, "expected a detail box in the split workspace")
	assert.Len(t, columns, 1, "every row of the box should start in the same column, got %v", columns)
}

// With room the whole blueprint is listed; when the viewport cannot hold it the
// detail view keeps its height and the list narrows to the step in play plus
// one either side, with counts so it never looks complete when it is not.
func TestOutlineWindowsWhenViewportIsShort(t *testing.T) {
	m := stressManyStepsManualNavigationModel()
	m.selectStep(3)

	tall := renderLayoutScenario(&m, layoutSize{name: "tall", width: 92, height: 60})
	assertNotContainsPlain(t, tall, "more above")

	// Assert against the outline itself, not the frame. The markers live inside
	// the scrollable outline, so the one below the selection can sit past the
	// viewport's bottom edge; checking the frame previously only passed because
	// the viewport's own overflow label happened to read "more below" too.
	renderLayoutScenario(&m, layoutSize{name: "short", width: 92, height: 26})
	short := m.renderStepOutline().content
	assertContainsPlain(t, short, "more above")
	assertContainsPlain(t, short, "more below")
}

// The neighbors are orientation, not content: they exist so the user knows
// where they are in the blueprint.
func TestOutlineWindowKeepsOneStepEitherSide(t *testing.T) {
	m := stressManyStepsManualNavigationModel()
	m.selectStep(3)
	m.viewport = viewport.New(viewport.WithWidth(92), viewport.WithHeight(8))

	first, last, above, below := m.outlineWindow()

	assert.Equal(t, 2, first)
	assert.Equal(t, 4, last)
	assert.Equal(t, 2, above)
	assert.Positive(t, below)
}

// A failed check is the one thing the card must never lose to height pressure.
// Before this, an 80x24 terminal clipped the failure out and left a bare
// "enter for details" — nothing on screen suggested anything was wrong.
func TestFailedCheckSurvivesEveryHeight(t *testing.T) {
	for _, size := range captureSizes {
		t.Run(size.name, func(t *testing.T) {
			m := failedCheckLayoutModel()
			rendered := ansi.Strip(renderLayoutScenario(&m, size))

			// The card scrolls now that it renders inline, so the guarantee is
			// that the frame signals the failure somewhere the user is looking,
			// not that the full finding fits on screen unscrolled. The step's
			// status line carries a ✗N badge; at heights too short for any card
			// the footer fallback counts the failures instead.
			named := strings.Contains(rendered, "✗")
			counted := strings.Contains(rendered, "check failed") ||
				strings.Contains(rendered, "checks failed")
			assert.True(t, named || counted,
				"the failure must be signaled somewhere in the frame:\n%s", rendered)
		})
	}
}

// assertSelectedRowVisible checks the timeline's filled node is on screen. The
// cursor used to be a "> " in front of the title; it is the node on the rail
// down the left now, so a row is "selected" when a line opens with it.
func assertSelectedRowVisible(t *testing.T, rendered string) {
	t.Helper()
	for _, line := range strings.Split(ansi.Strip(rendered), "\n") {
		if strings.HasPrefix(line, timelineNodeCurrent) {
			return
		}
	}
	assert.Fail(t, "selected row should remain visible", rendered)
}
