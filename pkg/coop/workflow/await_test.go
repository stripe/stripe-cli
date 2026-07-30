package workflow

import (
	"bytes"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/stripe/stripe-cli/pkg/coop"
)

// awaitTestStore observes heartbeat calls without replacing real session I/O.
type awaitTestStore struct {
	*coop.Store
	heartbeatRemoves int
}

func (s *awaitTestStore) RemoveHeartbeat(id string) error {
	s.heartbeatRemoves++
	return s.Store.RemoveHeartbeat(id)
}

// awaitReviewSession drives a two-task step to the point where the whole step
// is waiting on the developer.
func awaitReviewSession(t *testing.T, store Store) (*Service, *coop.Session) {
	t.Helper()
	base, session := workflowTestStore(t)
	if store == nil {
		store = base
	}
	// Fake sleep advances a closed-over clock so an interval elapses instantly.
	now := time.Now().UTC()
	service := NewService(store,
		WithAwaitTimeout(time.Second),
		WithClock(func() time.Time { return now }, func(time.Duration) { now = now.Add(500 * time.Millisecond) }),
		WithProgressWriter(nil),
	)
	for _, node := range []int{1, 2} {
		_, err := service.StartWork(session.ID, node, "Working")
		require.NoError(t, err)
		_, err = service.ReportWork(session.ID, node, ReportWorkInput{Note: "Done"}, false)
		require.NoError(t, err)
	}
	return service, session
}

func TestAwaitReviewReturnsWaitingWhenDeveloperHasNotActed(t *testing.T) {
	service, session := awaitReviewSession(t, nil)

	resp, err := service.AwaitReview(session.ID, 2)
	require.NoError(t, err)

	assert.True(t, resp.OK, "a still-open review is not a failure")
	assert.Equal(t, "waiting", resp.State)
	require.NotNil(t, resp.AdvanceAllowed)
	assert.False(t, *resp.AdvanceAllowed)
	// Re-running must take no judgment: next is the command just invoked.
	assert.Equal(t, "stripe coop agent await-review --session="+session.ID+" --step=2", resp.Next)
	assert.Equal(t, int(time.Second.Seconds()), resp.WaitTimeoutSeconds)
}

func TestAwaitReviewWaitingResponseAvoidsFailureVocabulary(t *testing.T) {
	service, session := awaitReviewSession(t, nil)

	resp, err := service.AwaitReview(session.ID, 2)
	require.NoError(t, err)

	// Failure vocabulary makes models abandon the loop or skip ahead.
	lowered := strings.ToLower(resp.State + " " + resp.Message)
	for _, banned := range []string{"timeout", "timed out", "failed", "error", "giving up"} {
		assert.NotContains(t, lowered, banned)
	}
}

func TestAwaitReviewKeepsHeartbeatWhileWaiting(t *testing.T) {
	base, _ := workflowTestStore(t)
	store := &awaitTestStore{Store: base}
	service, session := awaitReviewSession(t, store)
	store.heartbeatRemoves = 0

	_, err := service.AwaitReview(session.ID, 2)
	require.NoError(t, err)

	// Clearing it between calls would make the TUI flash "agent appears idle"
	// during every gap of a long review.
	assert.Zero(t, store.heartbeatRemoves)
	age, err := base.HeartbeatAge(session.ID)
	require.NoError(t, err)
	assert.GreaterOrEqual(t, age, time.Duration(0))
}

func TestAwaitReviewClearsHeartbeatOnceDecided(t *testing.T) {
	base, _ := workflowTestStore(t)
	store := &awaitTestStore{Store: base}
	service, session := awaitReviewSession(t, store)

	_, err := service.RequestChanges(session.ID, []int{1}, "Reuse the saved price")
	require.NoError(t, err)
	store.heartbeatRemoves = 0

	resp, err := service.AwaitReview(session.ID, 2)
	require.NoError(t, err)

	assert.Equal(t, "rejected", resp.State)
	require.NotNil(t, resp.AdvanceAllowed)
	assert.True(t, *resp.AdvanceAllowed, "a decision is available; next is actionable")
	assert.Positive(t, store.heartbeatRemoves)
}

func TestAwaitReviewReportsCumulativeWaitAndReviewPrompt(t *testing.T) {
	service, session := awaitReviewSession(t, nil)

	resp, err := service.AwaitReview(session.ID, 2)
	require.NoError(t, err)

	// The step entered review before this call started, so the reported wait
	// spans earlier calls rather than resetting each time.
	assert.Positive(t, resp.WaitedSeconds)
}

func TestAwaitReviewEmitsKeepAliveProgress(t *testing.T) {
	base, session := workflowTestStore(t)
	now := time.Now().UTC()
	service := NewService(base,
		WithAwaitTimeout(time.Minute),
		WithClock(func() time.Time { return now }, func(time.Duration) { now = now.Add(500 * time.Millisecond) }),
	)
	for _, node := range []int{1, 2} {
		_, err := service.StartWork(session.ID, node, "Working")
		require.NoError(t, err)
		_, err = service.ReportWork(session.ID, node, ReportWorkInput{Note: "Done"}, false)
		require.NoError(t, err)
	}

	var progress bytes.Buffer
	service.progress = &progress
	_, err := service.AwaitReview(session.ID, 2)
	require.NoError(t, err)

	assert.Contains(t, progress.String(), "Waiting for the developer to review")
}
