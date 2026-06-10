package coopcmd

import (
	"context"
	"io"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/stripe/stripe-cli/pkg/coop"
)

func TestCoopStartDebugAgentFlagIsHidden(t *testing.T) {
	rc := newCoopRunCmd()
	flag := rc.cmd.Flags().Lookup("debug-agent")
	require.NotNil(t, flag)
	assert.True(t, flag.Hidden)
}

func TestCoopStartDebugAgentRequiresBlueprint(t *testing.T) {
	rc := &coopRunCmd{debugAgent: true}
	err := rc.runCmd(nil, nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "--debug-agent requires a blueprint ID")
}

func TestDebugAgentPaneCommandUsesStripeBinaryAndSession(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", "/tmp/xdg config")
	rc := &coopRunCmd{}
	cmd, cleanup, err := rc.debugAgentPaneCommandBuilder("/tmp/stripe bin")("coop_123")
	require.NoError(t, err)
	assert.Nil(t, cleanup)
	assert.Equal(t, "XDG_CONFIG_HOME=\"/tmp/xdg config\" \"/tmp/stripe bin\" coop debug-agent --session \"coop_123\"", cmd)
}

func TestCoopDebugAgentRerunsRequestedChanges(t *testing.T) {
	store, session := setupDebugAgentSession(t, []coop.SessionChapter{
		{
			Key:   "chapter",
			Title: "Section",
			Nodes: []coop.SessionNode{
				{Key: "checkout", Title: "Build Checkout", State: coop.StepPending, Type: coop.NodeAPIRequest},
			},
		},
	})

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	done := runDebugAgentForTest(ctx, store, session.ID)

	waitForDebugSession(t, store, session.ID, func(s *coop.Session) bool {
		node, _ := s.NodeByNumber(1)
		return node.State == coop.StepReview
	})
	waitForDebugHeartbeat(t, store, session.ID)

	current, err := store.Read(session.ID)
	require.NoError(t, err)
	require.NoError(t, current.TransitionStep(1, coop.StepActive))
	node, _ := current.NodeByNumber(1)
	node.RejectionNote = "Use the stored price ID"
	node.Implementation = nil
	node.Verifications = nil
	require.NoError(t, store.Write(current))

	waitForDebugSession(t, store, session.ID, func(s *coop.Session) bool {
		node, _ := s.NodeByNumber(1)
		return node.State == coop.StepReview && len(node.Verifications) > 0 && node.Implementation != nil
	})

	current, err = store.Read(session.ID)
	require.NoError(t, err)
	require.NoError(t, current.TransitionStep(1, coop.StepDone))
	require.NoError(t, store.Write(current))

	require.NoError(t, <-done)
	finalSession, err := store.Read(session.ID)
	require.NoError(t, err)
	assert.Equal(t, coop.SessionCompleted, finalSession.Status)
	assert.NotEmpty(t, finalSession.NextSteps.Suggestions)
}

func TestCoopDebugAgentWaitsForChapterReviewAfterChapterIsReady(t *testing.T) {
	store, session := setupDebugAgentSession(t, []coop.SessionChapter{
		{
			Key:   "chapter",
			Title: "Section",
			Nodes: []coop.SessionNode{
				{Key: "product", Title: "Create product", State: coop.StepPending, Type: coop.NodeAPIRequest},
				{Key: "checkout", Title: "Build Checkout", State: coop.StepPending, Type: coop.NodeAPIRequest},
			},
		},
	})

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	done := runDebugAgentForTest(ctx, store, session.ID)

	waitForDebugSession(t, store, session.ID, func(s *coop.Session) bool {
		node1, _ := s.NodeByNumber(1)
		node2, _ := s.NodeByNumber(2)
		return node1.State == coop.StepReview && node2.State == coop.StepReview
	})

	current, err := store.Read(session.ID)
	require.NoError(t, err)
	require.NoError(t, current.TransitionStep(1, coop.StepDone))
	require.NoError(t, current.TransitionStep(2, coop.StepDone))
	require.NoError(t, store.Write(current))

	require.NoError(t, <-done)
	finalSession, err := store.Read(session.ID)
	require.NoError(t, err)
	assert.Equal(t, coop.SessionCompleted, finalSession.Status)
}

func setupDebugAgentSession(t *testing.T, chapters []coop.SessionChapter) (*coop.Store, *coop.Session) {
	t.Helper()
	store, err := coop.NewStoreAt(t.TempDir())
	require.NoError(t, err)
	session := &coop.Session{
		ID:        "debug_agent_session",
		Blueprint: "debug-blueprint",
		Status:    coop.SessionActive,
		Settings:  map[string]string{"language": "node"},
		Chapters:  chapters,
		CreatedAt: time.Now().UTC(),
		UpdatedAt: time.Now().UTC(),
	}
	require.NoError(t, store.Write(session))
	return store, session
}

func runDebugAgentForTest(ctx context.Context, store *coop.Store, sessionID string) <-chan error {
	done := make(chan error, 1)
	agent := &coopDebugAgent{
		store:                    store,
		sessionID:                sessionID,
		delay:                    time.Millisecond,
		pollInterval:             5 * time.Millisecond,
		out:                      io.Discard,
		waitForNextStepSelection: false,
	}
	go func() {
		done <- agent.run(ctx)
	}()
	return done
}

func waitForDebugSession(t *testing.T, store *coop.Store, sessionID string, predicate func(*coop.Session) bool) {
	t.Helper()
	deadline := time.Now().Add(500 * time.Millisecond)
	for time.Now().Before(deadline) {
		session, err := store.Read(sessionID)
		require.NoError(t, err)
		if predicate(session) {
			return
		}
		time.Sleep(5 * time.Millisecond)
	}
	session, err := store.Read(sessionID)
	require.NoError(t, err)
	require.True(t, predicate(session), "session did not reach expected state: %+v", session)
}

func waitForDebugHeartbeat(t *testing.T, store *coop.Store, sessionID string) {
	t.Helper()
	deadline := time.Now().Add(500 * time.Millisecond)
	for time.Now().Before(deadline) {
		if store.HeartbeatAge(sessionID) >= 0 {
			return
		}
		time.Sleep(5 * time.Millisecond)
	}
	require.True(t, store.HeartbeatAge(sessionID) >= 0, "debug agent did not write heartbeat")
}
