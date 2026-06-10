package tui

import (
	"fmt"
	"strings"
	"testing"

	"charm.land/lipgloss/v2"
	"github.com/stretchr/testify/assert"

	"github.com/stripe/stripe-cli/pkg/coop"
)

type stressScenario struct {
	name             string
	model            func() Model
	expectReviewCard bool
}

func TestUILayoutStressSessions(t *testing.T) {
	scenarios := []stressScenario{
		{name: "long_review_content", model: stressLongReviewModel, expectReviewCard: true},
		{name: "crowded_chapter_review", model: stressCrowdedChapterReviewModel, expectReviewCard: true},
		{name: "long_claim_url_active", model: stressLongClaimURLModel},
		{name: "many_steps_manual_navigation", model: stressManyStepsManualNavigationModel},
		{name: "long_rejection_input", model: stressLongRejectionInputModel, expectReviewCard: true},
	}

	for _, scenario := range scenarios {
		for _, size := range layoutMatrixSizes {
			t.Run(scenario.name+"/"+size.name, func(t *testing.T) {
				m := scenario.model()
				rendered := renderLayoutScenario(&m, size)
				writeLayoutCapture(t, "stress_"+scenario.name, size, rendered)

				assertLayoutFits(t, rendered, size)
				assertHeaderIsPinned(t, rendered)
				if m.rejecting {
					assertFooterIsPinned(t, rendered, "esc cancel")
				} else {
					assertFooterIsPinned(t, rendered, "enter")
				}
				if scenario.expectReviewCard {
					assert.Contains(t, rendered, "Review")
					assert.Contains(t, rendered, "You check")
					assert.LessOrEqual(t, lipgloss.Height(m.renderFooter()), m.footerHeightBudget())
				}
			})
		}
	}
}

func TestUILayoutCopyAudit(t *testing.T) {
	scenarios := []layoutScenario{
		{name: "review", model: reviewStepLongPromptLayoutModel},
		{name: "chapter_review", model: chapterReviewLayoutModel},
		{name: "request_changes", model: requestChangesLayoutModel},
		{name: "completion", model: completionLayoutModel},
	}
	banned := []string{
		"reject",
		"looks good",
		"verify it works",
		"Waiting for agent to continue",
	}

	for _, scenario := range scenarios {
		t.Run(scenario.name, func(t *testing.T) {
			m := scenario.model()
			rendered := renderLayoutScenario(&m, layoutSize{name: "audit", width: 69, height: 50})
			lower := strings.ToLower(rendered)
			for _, phrase := range banned {
				assert.NotContains(t, lower, strings.ToLower(phrase))
			}
			if strings.Contains(rendered, "Review") {
				assert.Contains(t, rendered, "You check")
			}
		})
	}
}

func stressLongReviewModel() Model {
	m := reviewStepLongPromptLayoutModel()
	m.session.Chapters[0].Nodes[0].Title = "Review Checkout Session creation, saved IDs, redirect behavior, and webhook assumptions"
	m.session.Chapters[0].Nodes[0].Implementation = &coop.Implementation{
		File:    "server/src/very/long/path/to/payments/checkout/session/create_checkout_session_handler_with_extremely_specific_name.ts",
		Lines:   "128-276",
		Snippet: strings.Repeat("await stripe.checkout.sessions.create({ mode: 'payment', line_items: [{ price: savedPriceID, quantity: 1 }] })\n", 6),
		Note:    "Created the Checkout Session endpoint, persisted the returned IDs, and reused the saved price ID for later payment confirmation.",
	}
	m.session.Chapters[0].Nodes[0].ReviewPrompt = "Open the app, start Checkout, inspect the server logs, confirm the saved price ID is reused instead of creating a new Price, confirm the redirect URL is correct, confirm errors are handled without exposing secrets, and confirm the success page reflects the completed payment."
	m.session.Chapters[0].Nodes[0].Verifications = []coop.Verification{
		{Check: "Created product", Passed: true},
		{Check: "Created price", Passed: true},
		{Check: "Created Checkout Session", Passed: true},
		{Check: "Ran redirect flow", Passed: true},
		{Check: "Checked secrets are environment-backed", Passed: true},
	}
	return m
}

func stressCrowdedChapterReviewModel() Model {
	m := testModel()
	m.spinner = staticSpinner()
	m.session.Chapters = []coop.SessionChapter{
		{
			Key:               "crowded",
			Title:             "Build a complete Checkout and fulfillment path with intentionally long labels",
			ReviewGranularity: coop.ReviewGranularityChapter,
			Nodes:             make([]coop.SessionNode, 0, 8),
		},
	}
	for i := 0; i < 8; i++ {
		m.session.Chapters[0].Nodes = append(m.session.Chapters[0].Nodes, coop.SessionNode{
			Key:          fmt.Sprintf("node-%d", i),
			Type:         coop.NodeUIComponent,
			Title:        fmt.Sprintf("Step %d with a detailed title that still needs to scan well in the terminal", i+1),
			State:        coop.StepReview,
			ReviewPrompt: fmt.Sprintf("Confirm item %d is observable, documented by verification evidence, and does not require hidden context from previous steps.", i+1),
			Implementation: &coop.Implementation{
				File:  fmt.Sprintf("app/src/features/payments/checkout/step_%d/component_or_handler_with_long_name.tsx", i+1),
				Lines: fmt.Sprintf("%d-%d", 20+i*12, 31+i*12),
			},
			Verifications: []coop.Verification{{Check: fmt.Sprintf("Verification %d passed", i+1), Passed: true}},
		})
	}
	m.selectChapter(0)
	return m
}

func stressLongClaimURLModel() Model {
	m := activeStepLayoutModel()
	m.session.ClaimURL = "https://dashboard.stripe.com/sandbox/claim_test_abcdefghijklmnopqrstuvwxyz0123456789abcdefghijklmnopqrstuvwxyz0123456789"
	m.session.Chapters[0].Nodes[1].Activity = "Running a long local verification flow with several redirects, environment checks, and Stripe object lookups"
	return m
}

func stressManyStepsManualNavigationModel() Model {
	m := testModel()
	m.spinner = staticSpinner()
	m.session.Chapters = nil
	stepCount := 0
	for ch := 0; ch < 6; ch++ {
		chapter := coop.SessionChapter{
			Key:   fmt.Sprintf("chapter-%d", ch),
			Title: fmt.Sprintf("Chapter %d with enough steps to force scrolling", ch+1),
		}
		for node := 0; node < 8; node++ {
			state := coop.StepDone
			if ch == 4 && node == 2 {
				state = coop.StepReview
			}
			chapter.Nodes = append(chapter.Nodes, coop.SessionNode{
				Key:          fmt.Sprintf("node-%d-%d", ch, node),
				Type:         coop.NodeAPIRequest,
				Title:        fmt.Sprintf("Generated step %d.%d with a moderately long label", ch+1, node+1),
				State:        state,
				ReviewPrompt: "Confirm this generated stress step has a visible acceptance check.",
				Implementation: &coop.Implementation{
					File:  fmt.Sprintf("generated/chapter_%d/step_%d/payment_flow_handler.go", ch+1, node+1),
					Lines: "1-20",
				},
			})
			stepCount++
		}
		m.session.Chapters = append(m.session.Chapters, chapter)
	}
	m.cursor = stepCount - 1
	m.userMoved = true
	return m
}

func stressLongRejectionInputModel() Model {
	m := stressLongReviewModel()
	m.rejecting = true
	m.rejectionInput.SetValue("Please rework the endpoint so it validates the stored price ID, handles missing environment variables with a useful error, preserves webhook signature verification, and includes a manual test path I can run locally before confirming.")
	return m
}
