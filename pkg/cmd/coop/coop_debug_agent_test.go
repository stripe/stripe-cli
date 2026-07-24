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

const debugAgentTestTimeout = 10 * time.Second

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
	cmd, cleanup, err := rc.debugAgentPaneCommandBuilder("/tmp/stripe bin")(&coop.Session{ID: "coop_123"})
	require.NoError(t, err)
	assert.Nil(t, cleanup)
	assert.Equal(t, "XDG_CONFIG_HOME='/tmp/xdg config' '/tmp/stripe bin' coop debug-agent --session 'coop_123'", cmd)
}

func TestCoopDebugAgentRerunsRequestedChanges(t *testing.T) {
	store, session := setupDebugAgentSession(t, []coop.SessionStep{
		commandSessionStep(
			"step",
			"Step",
			commandSessionNode(coop.NodeAPIRequest, "checkout", "Build Checkout", coop.NodePending),
		),
	})

	ctx, cancel := context.WithTimeout(context.Background(), debugAgentTestTimeout)
	defer cancel()
	done := runDebugAgentForTest(ctx, store, session.ID)

	waitForDebugSession(t, store, session.ID, func(s *coop.Session) bool {
		node, _ := s.NodeByNumber(1)
		return node.State == coop.NodeReview
	})
	waitForDebugHeartbeat(t, store, session.ID)

	current, err := store.Read(session.ID)
	require.NoError(t, err)
	require.NoError(t, current.TransitionNode(1, coop.NodeActive))
	node, _ := current.NodeByNumber(1)
	node.RejectionNote = "Use the stored price ID"
	node.Implementation = nil
	node.Verifications = nil
	require.NoError(t, store.Write(current))

	waitForDebugSession(t, store, session.ID, func(s *coop.Session) bool {
		node, _ := s.NodeByNumber(1)
		return node.State == coop.NodeReview && len(node.Verifications) > 0 && node.Implementation != nil
	})

	current, err = store.Read(session.ID)
	require.NoError(t, err)
	require.NoError(t, current.TransitionNode(1, coop.NodeDone))
	require.NoError(t, store.Write(current))

	require.NoError(t, <-done)
	finalSession, err := store.Read(session.ID)
	require.NoError(t, err)
	assert.Equal(t, coop.SessionCompleted, finalSession.Status)
	assert.NotEmpty(t, finalSession.NextSteps.Suggestions)
}

func TestCoopDebugAgentWaitsForStepReviewAfterStepIsReady(t *testing.T) {
	store, session := setupDebugAgentSession(t, []coop.SessionStep{
		commandSessionStep(
			"step",
			"Step",
			commandSessionNode(coop.NodeAPIRequest, "product", "Create product", coop.NodePending),
			commandSessionNode(coop.NodeAPIRequest, "checkout", "Build Checkout", coop.NodePending),
		),
	})

	ctx, cancel := context.WithTimeout(context.Background(), debugAgentTestTimeout)
	defer cancel()
	done := runDebugAgentForTest(ctx, store, session.ID)

	waitForDebugSession(t, store, session.ID, func(s *coop.Session) bool {
		node1, _ := s.NodeByNumber(1)
		node2, _ := s.NodeByNumber(2)
		return node1.State == coop.NodeReview && node2.State == coop.NodeReview
	})

	current, err := store.Read(session.ID)
	require.NoError(t, err)
	require.NoError(t, current.TransitionNode(1, coop.NodeDone))
	require.NoError(t, current.TransitionNode(2, coop.NodeDone))
	require.NoError(t, store.Write(current))

	require.NoError(t, <-done)
	finalSession, err := store.Read(session.ID)
	require.NoError(t, err)
	assert.Equal(t, coop.SessionCompleted, finalSession.Status)
}

func setupDebugAgentSession(t *testing.T, steps []coop.SessionStep) (*coop.Store, *coop.Session) {
	t.Helper()
	store, err := coop.NewStoreAt(t.TempDir())
	require.NoError(t, err)
	session := &coop.Session{
		ID:        "debug_agent_session",
		Blueprint: "debug-blueprint",
		Status:    coop.SessionActive,
		Settings:  map[string]string{"language": "node"},
		Steps:     steps,
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
	deadline := time.Now().Add(debugAgentTestTimeout)
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
	deadline := time.Now().Add(debugAgentTestTimeout)
	for time.Now().Before(deadline) {
		age, err := store.HeartbeatAge(sessionID)
		require.NoError(t, err)
		if age >= 0 {
			return
		}
		time.Sleep(5 * time.Millisecond)
	}
	age, err := store.HeartbeatAge(sessionID)
	require.NoError(t, err)
	require.True(t, age >= 0, "debug agent did not write heartbeat")
}
