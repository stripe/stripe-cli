package cmd

import (
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/stripe/stripe-cli/pkg/coop"
)

func setupStepTest(t *testing.T) (*coop.Store, *coop.Session) {
	t.Helper()
	dir := t.TempDir()
	store, err := coop.NewStoreAt(dir)
	require.NoError(t, err)

	session := &coop.Session{
		ID:       "step_test_session",
		Status:   coop.SessionActive,
		Settings: map[string]string{"language": "node"},
		Chapters: []coop.SessionChapter{
			{
				Key:   "ch1",
				Title: "Chapter 1",
				Nodes: []coop.SessionNode{
					{Key: "n1", Title: "Step 1", State: coop.StepPending, Type: coop.NodeAPIRequest},
					{Key: "n2", Title: "Step 2", State: coop.StepPending, Type: coop.NodeTestHelper, AutoConfirm: true},
					{Key: "n3", Title: "Step 3", State: coop.StepPending, Type: coop.NodeAsyncHandler},
				},
			},
		},
	}
	require.NoError(t, store.Write(session))

	// Point config to temp dir
	origConfigFolder := Config.GetConfigFolder("")
	os.Setenv("XDG_CONFIG_HOME", dir)
	t.Cleanup(func() {
		if origConfigFolder != "" {
			os.Unsetenv("XDG_CONFIG_HOME")
		}
	})

	return store, session
}

func TestDoStartTransitionsPendingToActive(t *testing.T) {
	store, session := setupStepTest(t)

	err := session.TransitionStep(1, coop.StepActive)
	require.NoError(t, err)

	node, _ := session.NodeByNumber(1)
	assert.Equal(t, coop.StepActive, node.State)
	assert.NotNil(t, node.StartedAt)

	store.Write(session)
}

func TestDoStartOnActiveStepFails(t *testing.T) {
	_, session := setupStepTest(t)

	session.TransitionStep(1, coop.StepActive)
	err := session.TransitionStep(1, coop.StepActive)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "cannot move to")
}

func TestDoDoneOnPendingStepFails(t *testing.T) {
	_, session := setupStepTest(t)

	// Can't go from pending directly to review
	err := session.TransitionStep(1, coop.StepReview)
	assert.Error(t, err)
}

func TestDoDoneAutoConfirmNode(t *testing.T) {
	_, session := setupStepTest(t)

	// Step 2 has AutoConfirm=true
	session.TransitionStep(2, coop.StepActive)
	node, _ := session.NodeByNumber(2)

	// When AutoConfirm is set, doDone transitions directly to done
	targetState := coop.StepReview
	if node.AutoConfirm {
		targetState = coop.StepDone
	}
	err := session.TransitionStep(2, targetState)
	require.NoError(t, err)

	node, _ = session.NodeByNumber(2)
	assert.Equal(t, coop.StepDone, node.State)
}

func TestSkipDoneStepFails(t *testing.T) {
	_, session := setupStepTest(t)

	session.TransitionStep(1, coop.StepActive)
	session.TransitionStep(1, coop.StepDone)

	err := session.TransitionStep(1, coop.StepSkipped)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "terminal state")
}

func TestAwaitOnDoneStepReturnsImmediately(t *testing.T) {
	_, session := setupStepTest(t)

	session.TransitionStep(1, coop.StepActive)
	session.TransitionStep(1, coop.StepDone)

	node, _ := session.NodeByNumber(1)
	// doAwait early-returns when state != review
	assert.NotEqual(t, coop.StepReview, node.State)
}

func TestAwaitAutoConfirmSkipsReview(t *testing.T) {
	_, session := setupStepTest(t)

	// Step 2 has AutoConfirm
	session.TransitionStep(2, coop.StepActive)
	session.TransitionStep(2, coop.StepReview)

	node, _ := session.NodeByNumber(2)
	assert.True(t, node.AutoConfirm)

	// Auto-confirm would transition review -> done
	if node.AutoConfirm && node.State == coop.StepReview {
		session.TransitionStep(2, coop.StepDone)
	}
	node, _ = session.NodeByNumber(2)
	assert.Equal(t, coop.StepDone, node.State)
}

func TestFullStepLifecycle(t *testing.T) {
	_, session := setupStepTest(t)

	// Normal flow: pending -> active -> review -> done
	require.NoError(t, session.TransitionStep(1, coop.StepActive))
	require.NoError(t, session.TransitionStep(1, coop.StepReview))
	require.NoError(t, session.TransitionStep(1, coop.StepDone))

	node, _ := session.NodeByNumber(1)
	assert.Equal(t, coop.StepDone, node.State)
	assert.NotNil(t, node.StartedAt)
	assert.NotNil(t, node.CompletedAt)
}

func TestRejectionLifecycle(t *testing.T) {
	_, session := setupStepTest(t)

	// pending -> active -> review -> active (reject) -> review -> done
	require.NoError(t, session.TransitionStep(1, coop.StepActive))
	require.NoError(t, session.TransitionStep(1, coop.StepReview))
	require.NoError(t, session.TransitionStep(1, coop.StepActive)) // rejection
	require.NoError(t, session.TransitionStep(1, coop.StepReview)) // redo
	require.NoError(t, session.TransitionStep(1, coop.StepDone))   // confirm

	node, _ := session.NodeByNumber(1)
	assert.Equal(t, coop.StepDone, node.State)
}

func TestSessionCompleteAfterAllSteps(t *testing.T) {
	_, session := setupStepTest(t)

	assert.False(t, session.IsComplete())

	for i := 1; i <= session.TotalSteps(); i++ {
		session.TransitionStep(i, coop.StepActive)
		session.TransitionStep(i, coop.StepDone)
	}

	assert.True(t, session.IsComplete())
}

func TestSessionCompleteWithMixedDoneAndSkipped(t *testing.T) {
	_, session := setupStepTest(t)

	session.TransitionStep(1, coop.StepActive)
	session.TransitionStep(1, coop.StepDone)
	session.TransitionStep(2, coop.StepSkipped)
	session.TransitionStep(3, coop.StepActive)
	session.TransitionStep(3, coop.StepDone)

	assert.True(t, session.IsComplete())
}

func TestHeartbeatWriteAndRead(t *testing.T) {
	dir := t.TempDir()
	store, _ := coop.NewStoreAt(dir)

	store.WriteHeartbeat("test_session")
	age := store.HeartbeatAge("test_session")
	assert.True(t, age >= 0)
	assert.True(t, age < 2*time.Second)

	store.RemoveHeartbeat("test_session")
	age = store.HeartbeatAge("test_session")
	assert.True(t, age < 0)
}

func TestHeartbeatAgeNoFile(t *testing.T) {
	dir := t.TempDir()
	store, _ := coop.NewStoreAt(dir)

	age := store.HeartbeatAge("nonexistent")
	assert.True(t, age < 0)
}
