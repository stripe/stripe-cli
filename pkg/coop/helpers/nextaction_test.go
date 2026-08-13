package helpers

import (
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/stripe/stripe-cli/pkg/coop"
	"github.com/stripe/stripe-cli/pkg/coop/followups"
)

func TestBuildSuggestionsUsesStripeProjectsWhenConfigured(t *testing.T) {
	session := &coop.Session{ID: "sess_123", Blueprint: "one-time-payment"}

	suggestions := BuildSuggestions(session, Environment{HasStripeProjects: true})

	assert.Equal(t, "deploy", suggestions[0].ID)
	assert.Equal(t, "stripe.json found", suggestions[0].Reason)
	assert.Equal(t, "summarize", suggestions[1].ID)
	assert.Equal(t, "add-integration", suggestions[2].ID)
	assert.Equal(t, "done", suggestions[3].ID)
}

func TestBuildSuggestionsUsesExistingDeployTarget(t *testing.T) {
	session := &coop.Session{ID: "sess_123", Blueprint: "one-time-payment"}

	suggestions := BuildSuggestions(session, Environment{HasExistingDeploy: true, HasVercel: true})

	assert.Equal(t, "deploy-update", suggestions[0].ID)
	assert.Equal(t, "Detected: Vercel", suggestions[0].Reason)
}

func TestBuildSuggestionsFiltersAlreadyCompletedActions(t *testing.T) {
	session := &coop.Session{
		ID:        "sess_123",
		Blueprint: "one-time-payment",
		NextSteps: &coop.NextStepsState{
			Completed: []string{"deploy", "summarize"},
		},
	}

	suggestions := BuildSuggestions(session, Environment{HasStripeProjects: true})

	ids := suggestionIDs(suggestions)
	assert.NotContains(t, ids, "deploy")
	assert.NotContains(t, ids, "summarize")
	assert.Contains(t, ids, "add-integration")
	assert.Contains(t, ids, "done")
}

func TestShowSuggestionsFiltersCurrentCompletedAction(t *testing.T) {
	store := &nextActionTestStore{
		session: &coop.Session{
			ID:        "sess_123",
			Blueprint: "one-time-payment",
			NextSteps: &coop.NextStepsState{
				Completed: []string{"summarize"},
			},
		},
	}
	suggestions := BuildSuggestions(store.session, Environment{HasStripeProjects: true})

	err := ShowSuggestions(store, store.session, suggestions, "deploy")

	require.NoError(t, err)
	ids := nextStepSuggestionIDs(store.session.NextSteps.Suggestions)
	assert.NotContains(t, ids, "deploy")
	assert.NotContains(t, ids, "summarize")
	assert.Contains(t, ids, "add-integration")
	assert.Contains(t, ids, "done")
	assert.ElementsMatch(t, []string{"summarize", "deploy"}, store.session.NextSteps.Completed)
}

func TestBuildResponseForDeployStartsGuidedFollowup(t *testing.T) {
	session := &coop.Session{
		ID:        "sess_123",
		Blueprint: "one-time-payment",
	}

	resp := BuildResponse(session, nil, "deploy")

	assert.True(t, resp.OK)
	assert.Equal(t, "sess_123", resp.SessionID)
	assert.Contains(t, resp.AgentPrompt, "guided deploy flow")
	assert.Contains(t, resp.AgentPrompt, `Do not use "stripe coop run"`)
	assert.Contains(t, resp.AgentPrompt, "not co-op blueprints")
	assert.Equal(t, `stripe coop agent start-followup --session="sess_123" --action="deploy"`, resp.Next)
	assert.NotContains(t, resp.Next, "stripe coop run")
}

func TestBuildResponseForSummarizeReturnsExecutableNext(t *testing.T) {
	session := &coop.Session{
		ID:        "sess_123",
		Blueprint: "one-time-payment",
	}

	resp := BuildResponse(session, nil, "summarize")

	assert.Equal(t, "stripe coop agent next-action --session=sess_123 --completed=summarize", resp.Next)
	assert.NotContains(t, resp.Next, "Write STRIPE.md")
}

func TestBuildResponseForDeployUpdateStartsGuidedFollowupWithDetectedTarget(t *testing.T) {
	session := &coop.Session{
		ID:        "sess_123",
		Blueprint: "one-time-payment",
	}
	suggestions := BuildSuggestions(session, Environment{HasExistingDeploy: true, HasVercel: true})

	resp := BuildResponse(session, suggestions, "deploy-update")

	assert.True(t, resp.OK)
	assert.Contains(t, resp.AgentPrompt, "guided deploy-update flow for Vercel")
	assert.Contains(t, resp.AgentPrompt, "existing Vercel deployment configuration")
	assert.Equal(t, `stripe coop agent start-followup --session="sess_123" --action="deploy-update" --target="Vercel"`, resp.Next)
	assert.NotContains(t, resp.Next, "stripe coop run")
}

func TestDeployGuidedActionDoesNotReadKeyMaterialFromWhoami(t *testing.T) {
	session := &coop.Session{
		ID:        "sess_123",
		Blueprint: "one-time-payment",
	}

	resp := BuildResponse(session, nil, "deploy")
	action, err := followups.GuidedActionByID(followups.Deploy, "")
	require.NoError(t, err)

	prompt := resp.AgentPrompt + "\n" + action.AgentContext
	assert.NotContains(t, prompt, "stripe whoami --json")
	assert.NotContains(t, prompt, "using the keys from")
	assert.Contains(t, prompt, "stripe whoami --format json")
	assert.Contains(t, prompt, "does not print key material")
}

func TestWaitForSelectionReturnsWaitingInsteadOfError(t *testing.T) {
	store := &nextActionTestStore{
		session: &coop.Session{
			ID:        "sess_123",
			NextSteps: &coop.NextStepsState{},
		},
	}
	now := time.Unix(0, 0)

	selected, waited, err := waitForSelection(
		store,
		"sess_123",
		time.Second,
		func() time.Time { return now },
		func(time.Duration) { now = now.Add(500 * time.Millisecond) },
	)

	// An unmade choice is not a failure. Surfacing it as one made the command
	// exit non-zero, which an agent reads as a broken session.
	require.NoError(t, err)
	assert.Empty(t, selected)
	assert.GreaterOrEqual(t, waited, time.Second)
}

func TestWaitForSelectionClearsSelectedAction(t *testing.T) {
	store := &nextActionTestStore{
		session: &coop.Session{
			ID: "sess_123",
			NextSteps: &coop.NextStepsState{
				Selected: "done",
			},
		},
	}
	now := time.Unix(0, 0)

	selected, _, err := waitForSelection(
		store,
		"sess_123",
		time.Second,
		func() time.Time { return now },
		func(time.Duration) { now = now.Add(500 * time.Millisecond) },
	)

	require.NoError(t, err)
	assert.Equal(t, "done", selected)
	assert.Empty(t, store.session.NextSteps.Selected)
}

type nextActionTestStore struct {
	session *coop.Session
	writes  int
}

func (s *nextActionTestStore) Read(id string) (*coop.Session, error) {
	return s.session, nil
}

func (s *nextActionTestStore) LatestSession() (*coop.Session, error) {
	return s.session, nil
}

func (s *nextActionTestStore) Write(session *coop.Session) error {
	s.session = session
	s.writes++
	return nil
}

func suggestionIDs(suggestions []Suggestion) []string {
	ids := make([]string, 0, len(suggestions))
	for _, suggestion := range suggestions {
		ids = append(ids, suggestion.ID)
	}
	return ids
}

func nextStepSuggestionIDs(suggestions []coop.NextStepSuggestion) []string {
	ids := make([]string, 0, len(suggestions))
	for _, suggestion := range suggestions {
		ids = append(ids, suggestion.ID)
	}
	return ids
}

// The developer can pick a next action in the gap between two next-action
// invocations. ShowSuggestions clears Selected, so republishing before
// consuming it would erase the choice and leave the agent waiting forever.
func TestRunDoesNotClearSelectionMadeBetweenInvocations(t *testing.T) {
	store := &nextActionTestStore{
		session: &coop.Session{
			ID:        "sess_123",
			Blueprint: "one-time-payment",
			Status:    coop.SessionCompleted,
			NextSteps: &coop.NextStepsState{
				Suggestions: []coop.NextStepSuggestion{{ID: "done", Title: "Finish"}},
				Selected:    "done",
			},
		},
	}

	resp, err := Run(store, Input{SessionID: "sess_123"})

	require.NoError(t, err)
	assert.Contains(t, resp.Next, "stripe coop stop")
	assert.Empty(t, store.session.NextSteps.Selected)
}

func TestWaitingResponseRepeatsTheCommandAndBlocksAdvance(t *testing.T) {
	session := &coop.Session{ID: "sess_123", Blueprint: "one-time-payment"}

	resp := waitingResponse(session, nil, "", 90*time.Second)

	// Previously this path returned ErrSelectionTimeout after ten minutes,
	// which surfaced as JSON on stderr with a non-zero exit.
	assert.True(t, resp.OK)
	assert.Equal(t, "waiting", resp.State)
	require.NotNil(t, resp.AdvanceAllowed)
	assert.False(t, *resp.AdvanceAllowed)
	assert.Equal(t, "stripe coop agent next-action --session=sess_123", resp.Next)
	assert.Equal(t, 90, resp.WaitedSeconds)
	for _, banned := range []string{"timeout", "timed out", "failed", "error"} {
		assert.NotContains(t, strings.ToLower(resp.Message), banned)
	}
}

// Republishing an identical suggestion set on every call would bump the session
// version each time, churning the TUI and defeating the stop hook's
// progress-based block budget.
func TestShowSuggestionsSkipsNoOpRepublish(t *testing.T) {
	store := &nextActionTestStore{
		session: &coop.Session{
			ID:        "sess_123",
			Status:    coop.SessionActive,
			NextSteps: &coop.NextStepsState{},
		},
	}
	suggestions := BuildSuggestions(store.session, Environment{})

	require.NoError(t, ShowSuggestions(store, store.session, suggestions, ""))
	afterFirst := store.writes

	require.NoError(t, ShowSuggestions(store, store.session, suggestions, ""))

	assert.Equal(t, afterFirst, store.writes)
}

// Finish has no follow-up that reports back with --completed, so consuming the
// selection is the only chance to record it. Without that the session stays
// "completed with a next action owed" and the stop hook never lets the agent go.
func TestRunRecordsTheFinishSelection(t *testing.T) {
	store := &nextActionTestStore{
		session: &coop.Session{
			ID:        "sess_123",
			Blueprint: "one-time-payment",
			Status:    coop.SessionCompleted,
			NextSteps: &coop.NextStepsState{
				Suggestions: []coop.NextStepSuggestion{{ID: coop.FinishActionID, Title: "Finish"}},
				Selected:    coop.FinishActionID,
			},
		},
	}

	resp, err := Run(store, Input{SessionID: "sess_123"})

	require.NoError(t, err)
	assert.Contains(t, resp.Next, "stripe coop stop")
	assert.True(t, store.session.DeveloperFinished(), "Finish must be durable")
}
