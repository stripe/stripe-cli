package coopcmd

import (
	"bytes"
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/BurntSushi/toml"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/stripe/stripe-cli/pkg/coop"
)

type stopHookTestStore struct {
	session     *coop.Session
	otherActive []*coop.Session
	state       coop.StopHookState
	hasState    bool
	removed     int
	writeErr    error
}

func (s *stopHookTestStore) Read(id string) (*coop.Session, error) {
	if s.session == nil || s.session.ID != id {
		return nil, errors.New("not found")
	}
	return s.session, nil
}

func (s *stopHookTestStore) ActiveSessions() ([]*coop.Session, error) {
	if s.session == nil {
		return nil, nil
	}
	sessions := []*coop.Session{s.session}
	return append(sessions, s.otherActive...), nil
}

func (s *stopHookTestStore) ReadStopHookState(string) (coop.StopHookState, error) {
	if !s.hasState {
		return coop.StopHookState{}, nil
	}
	return s.state, nil
}

func (s *stopHookTestStore) WriteStopHookState(_ string, state coop.StopHookState) error {
	if s.writeErr != nil {
		return s.writeErr
	}
	s.state = state
	s.hasState = true
	return nil
}

func (s *stopHookTestStore) RemoveStopHookState(string) error {
	s.removed++
	s.hasState = false
	s.state = coop.StopHookState{}
	return nil
}

// stubResumer stands in for workflow.Service's read-only lifecycle query.
type stubResumer struct {
	response coop.CommandResponse
	err      error
}

func (r stubResumer) Resume(string) (coop.CommandResponse, error) {
	return r.response, r.err
}

func runStopHook(t *testing.T, store stopHookStore, resumer stopHookResumer, sessionID string) stopHookDecision {
	t.Helper()
	return runStopHookAt(t, store, resumer, sessionID, time.Now())
}

func runStopHookAt(t *testing.T, store stopHookStore, resumer stopHookResumer, sessionID string, now time.Time) stopHookDecision {
	t.Helper()
	var out bytes.Buffer
	require.NoError(t, runCoopStopHook(&out, store, resumer, sessionID, now))
	var decision stopHookDecision
	require.NoError(t, json.Unmarshal(out.Bytes(), &decision))
	return decision
}

func stopHookSession() *coop.Session {
	return &coop.Session{
		SchemaVersion: coop.CurrentSessionSchemaVersion,
		ID:            "coop_hook",
		Status:        coop.SessionActive,
		Version:       3,
		UpdatedAt:     time.Now().UTC(),
	}
}

func actionableResume() stubResumer {
	return stubResumer{response: coop.CommandResponse{
		OK:           true,
		Message:      `Step "Set up product and pricing" is waiting for human review.`,
		Continuation: coop.Continue("stripe coop agent await-review --session=coop_hook --step=2"),
	}}
}

func TestStopHookBlocksWhileResumeHasAnActionableCommand(t *testing.T) {
	store := &stopHookTestStore{session: stopHookSession()}

	decision := runStopHook(t, store, actionableResume(), "coop_hook")

	assert.Equal(t, "block", decision.Decision)
	assert.Contains(t, decision.Reason, "stripe coop agent await-review --session=coop_hook --step=2")
	assert.Contains(t, decision.Reason, "waiting for human review")
	assert.Equal(t, 1, store.state.Blocks)
}

// Resume returning an empty next is its "nothing to do" signal, so the hook
// does not have to reimplement the lifecycle state machine.
func TestStopHookAllowsWhenResumeHasNothingToDo(t *testing.T) {
	store := &stopHookTestStore{session: stopHookSession()}
	resumer := stubResumer{response: coop.CommandResponse{OK: true}}

	decision := runStopHook(t, store, resumer, "coop_hook")

	assert.Empty(t, decision.Decision)
	assert.Positive(t, store.removed, "bookkeeping resets so a later review starts fresh")
}

// Blocking forever would burn tokens whenever the developer walks away.
func TestStopHookReleasesAfterConsecutiveBlocks(t *testing.T) {
	store := &stopHookTestStore{session: stopHookSession()}

	for i := 0; i < maxConsecutiveStopBlocks; i++ {
		require.Equal(t, "block", runStopHook(t, store, actionableResume(), "coop_hook").Decision)
	}

	decision := runStopHook(t, store, actionableResume(), "coop_hook")
	assert.Empty(t, decision.Decision, "the agent must eventually be allowed to stop")
	assert.Contains(t, decision.SystemMessage, "stripe coop agent resume --session=coop_hook")
}

// Progress means the loop is working, so the budget resets rather than
// accumulating across a long but healthy session.
func TestStopHookResetsCounterOnSessionProgress(t *testing.T) {
	store := &stopHookTestStore{session: stopHookSession()}

	for i := 0; i < maxConsecutiveStopBlocks; i++ {
		require.Equal(t, "block", runStopHook(t, store, actionableResume(), "coop_hook").Decision)
	}
	// The lifecycle moved on to a different command.
	moved := stubResumer{response: coop.CommandResponse{
		OK:           true,
		Message:      "Task 3 is next.",
		Continuation: coop.Continue("stripe coop agent start-work --session=coop_hook --step=3"),
	}}

	decision := runStopHook(t, store, moved, "coop_hook")
	assert.Equal(t, "block", decision.Decision)
	assert.Equal(t, 1, store.state.Blocks)
}

// The command the hook orders can itself bump the session version — next-action
// republishes suggestions — so a version-keyed budget would reset every round
// and never release.
func TestStopHookBudgetSurvivesSessionVersionChurn(t *testing.T) {
	store := &stopHookTestStore{session: stopHookSession()}

	for i := 0; i < maxConsecutiveStopBlocks; i++ {
		require.Equal(t, "block", runStopHook(t, store, actionableResume(), "coop_hook").Decision)
		store.session.Version++
	}

	decision := runStopHook(t, store, actionableResume(), "coop_hook")
	assert.Empty(t, decision.Decision, "same pending command means no real progress")
}

// The counter is the only thing bounding the loop; if it cannot be persisted,
// blocking would repeat forever with no release.
func TestStopHookFailsOpenWhenCounterCannotBePersisted(t *testing.T) {
	store := &stopHookTestStore{session: stopHookSession(), writeErr: errors.New("read-only fs")}

	decision := runStopHook(t, store, actionableResume(), "coop_hook")

	assert.Empty(t, decision.Decision)
}

// Discovery mode has no session when the hook is installed, so the launcher
// cannot bake an ID into the command.
func TestStopHookFallsBackToTheOnlyActiveSession(t *testing.T) {
	store := &stopHookTestStore{session: stopHookSession()}

	decision := runStopHook(t, store, actionableResume(), "")

	assert.Equal(t, "block", decision.Decision)
}

func TestStopHookAllowsWhenSessionMissing(t *testing.T) {
	decision := runStopHook(t, &stopHookTestStore{}, actionableResume(), "coop_missing")

	assert.Empty(t, decision.Decision, "a missing session must never wedge the agent")
}

func TestStopHookAllowsWhenResumeFails(t *testing.T) {
	store := &stopHookTestStore{session: stopHookSession()}
	resumer := stubResumer{err: errors.New("unreadable")}

	decision := runStopHook(t, store, resumer, "coop_hook")

	assert.Empty(t, decision.Decision)
}

func TestClaudeStopHookSettingsCarryTheHookCommand(t *testing.T) {
	var parsed struct {
		Hooks struct {
			Stop []struct {
				Hooks []struct {
					Type    string `json:"type"`
					Command string `json:"command"`
				} `json:"hooks"`
			} `json:"Stop"`
		} `json:"hooks"`
	}
	require.NoError(t, json.Unmarshal([]byte(claudeStopHookSettings("/usr/local/bin/stripe", "coop_abc")), &parsed))

	require.Len(t, parsed.Hooks.Stop, 1)
	require.Len(t, parsed.Hooks.Stop[0].Hooks, 1)
	assert.Equal(t, "command", parsed.Hooks.Stop[0].Hooks[0].Type)
	assert.Contains(t, parsed.Hooks.Stop[0].Hooks[0].Command, "coop agent stop-hook")
	assert.Contains(t, parsed.Hooks.Stop[0].Hooks[0].Command, "coop_abc")
}

func TestCodexStopHookConfigUsesInlineTOML(t *testing.T) {
	config := codexStopHookConfig("/usr/local/bin/stripe", "coop_abc")

	// Codex parses the -c value as TOML, so prove it actually parses rather
	// than string-matching the shape. Verified against codex-cli 0.145.0.
	key, value, found := strings.Cut(config, "=")
	require.True(t, found)
	assert.Equal(t, "hooks.Stop", key)

	var parsed struct {
		Stop []struct {
			Hooks []struct {
				Type    string `toml:"type"`
				Command string `toml:"command"`
			} `toml:"hooks"`
		} `toml:"Stop"`
	}
	require.NoError(t, toml.Unmarshal([]byte("Stop = "+value), &parsed))
	require.Len(t, parsed.Stop, 1)
	require.Len(t, parsed.Stop[0].Hooks, 1)
	assert.Equal(t, "command", parsed.Stop[0].Hooks[0].Type)
	assert.Contains(t, parsed.Stop[0].Hooks[0].Command, "coop agent stop-hook")
	assert.Contains(t, parsed.Stop[0].Hooks[0].Command, "coop_abc")
}

// The config folder travels as a flag so the command works whether or not the
// harness runs it through a shell.
func TestStopHookCommandPassesConfigDirAsAFlag(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	cmd := stopHookCommand("/opt/my tools/stripe", "coop_abc")

	assert.Contains(t, cmd, "--config-dir")
	assert.NotContains(t, cmd, "XDG_CONFIG_HOME=")
	assert.Contains(t, cmd, `'/opt/my tools/stripe'`)
}

// Sessions do not record which directory they belong to, so with several active
// at once the discovery-mode fallback cannot tell which agent it is serving.
// Blocking on a guess would hand this agent a command targeting somebody else's
// session and mutate it.
func TestStopHookAllowsWhenMultipleSessionsAreActive(t *testing.T) {
	store := &stopHookTestStore{
		session:     stopHookSession(),
		otherActive: []*coop.Session{{ID: "coop_other", Status: coop.SessionActive}},
	}

	decision := runStopHook(t, store, actionableResume(), "")

	assert.Empty(t, decision.Decision, "ambiguous fallback must fail open")
}

// An explicit --session is unambiguous even when other sessions are active.
func TestStopHookUsesExplicitSessionDespiteOtherActiveSessions(t *testing.T) {
	store := &stopHookTestStore{
		session:     stopHookSession(),
		otherActive: []*coop.Session{{ID: "coop_other", Status: coop.SessionActive}},
	}

	decision := runStopHook(t, store, actionableResume(), "coop_hook")

	assert.Equal(t, "block", decision.Decision)
}

// Sessions only leave "active" through an explicit stop or completion, so an
// abandoned run stays a candidate forever. Adopting one would let an unrelated
// agent mutate it.
func TestStopHookIgnoresAStaleSessionInDiscoveryMode(t *testing.T) {
	session := stopHookSession()
	session.UpdatedAt = time.Now().UTC().Add(-24 * time.Hour)
	store := &stopHookTestStore{session: session}

	decision := runStopHook(t, store, actionableResume(), "")

	assert.Empty(t, decision.Decision)
}

// An explicit --session is trusted regardless of age: the launcher only bakes
// one in for a session it just created.
func TestStopHookUsesExplicitSessionEvenWhenStale(t *testing.T) {
	session := stopHookSession()
	session.UpdatedAt = time.Now().UTC().Add(-24 * time.Hour)
	store := &stopHookTestStore{session: session}

	decision := runStopHook(t, store, actionableResume(), "coop_hook")

	assert.Equal(t, "block", decision.Decision)
}

// Clearing the budget on release would let the next Stop event start from zero
// and block three more times, so the agent could never actually stop.
func TestStopHookReleaseIsDurable(t *testing.T) {
	store := &stopHookTestStore{session: stopHookSession()}

	for i := 0; i < maxConsecutiveStopBlocks; i++ {
		require.Equal(t, "block", runStopHook(t, store, actionableResume(), "coop_hook").Decision)
	}
	require.Empty(t, runStopHook(t, store, actionableResume(), "coop_hook").Decision)

	assert.Empty(t, runStopHook(t, store, actionableResume(), "coop_hook").Decision,
		"the release must hold until the lifecycle moves on")
}

// The real budget is wall-clock: what it detects is a developer who walked
// away, and how many times the agent happened to try to stop is a poor proxy.
func TestStopHookReleasesAfterTheBlockWindowElapses(t *testing.T) {
	store := &stopHookTestStore{session: stopHookSession()}
	start := time.Now()

	require.Equal(t, "block", runStopHookAt(t, store, actionableResume(), "coop_hook", start).Decision)
	require.Equal(t, "block", runStopHookAt(t, store, actionableResume(), "coop_hook", start.Add(time.Minute)).Decision)

	decision := runStopHookAt(t, store, actionableResume(), "coop_hook", start.Add(stopBlockWindow+time.Second))

	assert.Empty(t, decision.Decision)
	assert.Contains(t, decision.SystemMessage, "stripe coop agent resume --session=coop_hook")
}

// A long review with an occasionally drifting agent must not be cut short: the
// window, not the attempt count, is what bounds it.
func TestStopHookKeepsBlockingAcrossALongReview(t *testing.T) {
	store := &stopHookTestStore{session: stopHookSession()}
	start := time.Now()

	for i := 0; i < 20; i++ {
		at := start.Add(time.Duration(i) * time.Minute)
		require.Equal(t, "block", runStopHookAt(t, store, actionableResume(), "coop_hook", at).Decision,
			"still inside the window at %d minutes", i)
	}
}

// Progress restarts the window, so a healthy long session is never cut off.
func TestStopHookWindowRestartsOnProgress(t *testing.T) {
	store := &stopHookTestStore{session: stopHookSession()}
	start := time.Now()

	require.Equal(t, "block", runStopHookAt(t, store, actionableResume(), "coop_hook", start).Decision)

	moved := stubResumer{response: coop.CommandResponse{
		OK:           true,
		Message:      "Task 3 is next.",
		Continuation: coop.Continue("stripe coop agent start-work --session=coop_hook --step=3"),
	}}
	decision := runStopHookAt(t, store, moved, "coop_hook", start.Add(stopBlockWindow+time.Minute))

	assert.Equal(t, "block", decision.Decision, "a new command starts a fresh window")
}
