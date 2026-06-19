package helpers

import (
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/stripe/stripe-cli/pkg/coop"
)

func TestBuildSuggestionsIncludesCoreNextActions(t *testing.T) {
	session := &coop.Session{ID: "sess_123", Blueprint: "one-time-payment"}

	suggestions := BuildSuggestions(session, Environment{})

	assert.Equal(t, "summarize", suggestions[0].ID)
	assert.Equal(t, "add-integration", suggestions[1].ID)
	assert.Equal(t, "done", suggestions[2].ID)
}

func TestWaitForSelectionTimesOut(t *testing.T) {
	store := &nextActionTestStore{
		session: &coop.Session{
			ID:        "sess_123",
			NextSteps: &coop.NextStepsState{},
		},
	}
	now := time.Unix(0, 0)

	selected, err := waitForSelection(
		store,
		"sess_123",
		time.Second,
		func() time.Time { return now },
		func(time.Duration) { now = now.Add(500 * time.Millisecond) },
	)

	assert.Empty(t, selected)
	require.Error(t, err)
	assert.True(t, errors.Is(err, ErrSelectionTimeout))
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

	selected, err := waitForSelection(
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
}

func (s *nextActionTestStore) Read(id string) (*coop.Session, error) {
	return s.session, nil
}

func (s *nextActionTestStore) LatestSession() (*coop.Session, error) {
	return s.session, nil
}

func (s *nextActionTestStore) Write(session *coop.Session) error {
	s.session = session
	return nil
}
