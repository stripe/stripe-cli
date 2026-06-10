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
			name:             "chapter_review_many_changes",
			model:            chapterReviewLayoutModel,
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
			name:         "manual_navigation",
			model:        manualNavigationLayoutModel,
			footerToken:  "f follow",
			expectCursor: true,
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
				if scenario.expectCursor {
					assert.Contains(t, rendered, strings.TrimSpace(cursorMarker), "selected row should remain visible")
				}
				if scenario.expectReviewCard {
					assert.Contains(t, rendered, "Review", "review card should remain visible")
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
	assert.NotContains(t, rendered, "Waiting for agent")
}

func TestSessionUpdateResizesAfterAutoSelectingReview(t *testing.T) {
	m := testModel()
	m.spinner = staticSpinner()
	m.ready = true
	m.width = 56
	m.height = 18
	m.viewport = viewport.New(viewport.WithWidth(m.width), viewport.WithHeight(10))
	m.cursor = 0
	m.session.Chapters[0].Nodes[0].State = coop.StepDone
	m.session.Chapters[0].Nodes[1].State = coop.StepReview
	m.session.Chapters[0].Nodes[1].Title = "Review Checkout Session creation, saved IDs, redirect behavior, and webhook assumptions"
	m.session.Chapters[0].Nodes[1].ReviewPrompt = "Open the local app, start Checkout, inspect the server logs, confirm the saved price ID is reused instead of creating a new Price, confirm the redirect URL is correct, confirm errors are handled without exposing secrets, and confirm the success page reflects the completed payment."
	m.session.Chapters[0].Nodes[1].Implementation = &coop.Implementation{
		File:  "server/src/payments/checkout/session/create_checkout_session_handler_with_long_name.ts",
		Lines: "42-118",
	}
	m.session.Chapters[0].Nodes[1].Verifications = []coop.Verification{
		{Check: "Created product and price", Passed: true},
		{Check: "Confirmed Checkout reuses the saved price ID", Passed: true},
	}

	updatedModel, _ := m.Update(sessionUpdatedMsg{session: m.session})
	updated := updatedModel.(Model)
	rendered := updated.View().Content

	assert.Equal(t, 1, updated.cursor)
	assertLayoutFits(t, rendered, layoutSize{name: "narrow_acceptance", width: 56, height: 18})
	assertHeaderIsPinned(t, rendered)
	assertFooterIsPinned(t, rendered, "enter")
	assert.Contains(t, rendered, "Review")
	assert.Contains(t, rendered, "Confirmation steps")
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
		{name: "manual", model: manualNavigationLayoutModel, token: "f follow"},
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
	m.session.Chapters[0].Nodes[0].State = coop.StepDone
	m.session.Chapters[0].Nodes[1].State = coop.StepActive
	m.session.Chapters[0].Nodes[1].Activity = "Adding the Checkout endpoint and wiring the returned session URL into the app"
	m.cursor = 1
	return m
}

func reviewStepLongPromptLayoutModel() Model {
	m := testModel()
	m.spinner = staticSpinner()
	m.session.Chapters[0].Nodes[0].State = coop.StepReview
	m.session.Chapters[0].Nodes[0].Implementation = &coop.Implementation{
		File:  "server/routes/payments/checkout/session_handler.js",
		Lines: "42-118",
		Note:  "Created product and price setup for Checkout.",
	}
	m.session.Chapters[0].Nodes[0].ReviewPrompt = "Open the local app, complete the Checkout flow, confirm the saved price ID is reused by the Checkout Session, confirm the redirect lands on the success page, and confirm no secret keys are committed."
	m.session.Chapters[0].Nodes[0].Verifications = []coop.Verification{
		{Check: "Created product and price", Passed: true},
		{Check: "Saved price ID for Checkout", Passed: true},
		{Check: "Ran local Checkout flow", Passed: true},
	}
	m.cursor = 0
	return m
}

func chapterReviewLayoutModel() Model {
	m := testModel()
	m.spinner = staticSpinner()
	m.session.Chapters[0].ReviewGranularity = coop.ReviewGranularityChapter
	m.session.Chapters[0].Nodes[0].State = coop.StepReview
	m.session.Chapters[0].Nodes[0].ReviewPrompt = "Confirm the product and price are created once, persisted for later steps, and reused by the Checkout Session."
	m.session.Chapters[0].Nodes[0].Implementation = &coop.Implementation{File: "server/catalog/stripe_products.js", Lines: "10-84"}
	m.session.Chapters[0].Nodes[0].Verifications = []coop.Verification{{Check: "Created product", Passed: true}, {Check: "Created price", Passed: true}}
	m.session.Chapters[0].Nodes[1].State = coop.StepReview
	m.session.Chapters[0].Nodes[1].ReviewPrompt = "Open the app locally, click the Checkout button, complete payment, and confirm the redirect reaches the expected success page."
	m.session.Chapters[0].Nodes[1].Implementation = &coop.Implementation{File: "client/src/components/CheckoutButton.tsx", Lines: "9-88"}
	m.session.Chapters[0].Nodes[1].Verifications = []coop.Verification{{Check: "Rendered Checkout button", Passed: true}, {Check: "Confirmed redirect", Passed: true}}
	m.selectChapter(0)
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
	m.cursor = 2
	m.userMoved = true
	return m
}

func expandedDetailsLayoutModel() Model {
	m := reviewStepLongPromptLayoutModel()
	m.expanded = true
	m.detailTab = 1
	m.session.Chapters[0].Nodes[0].Implementation.Snippet = strings.Repeat("const session = await stripe.checkout.sessions.create({ mode: 'payment' })\n", 8)
	return m
}

func completionLayoutModel() Model {
	m := testModel()
	for i := range m.session.Chapters {
		for j := range m.session.Chapters[i].Nodes {
			m.session.Chapters[i].Nodes[j].State = coop.StepDone
		}
	}
	m.cursor = 0
	return m
}

func staticSpinner() spinner.Model {
	s := spinner.New()
	s.Spinner = spinner.Spinner{Frames: []string{"●"}, FPS: 1}
	return s
}
