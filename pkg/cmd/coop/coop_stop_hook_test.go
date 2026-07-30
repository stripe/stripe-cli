package coopcmd

import (
	"bytes"
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/stripe/stripe-cli/pkg/coop"
)

type stopHookTestStore struct {
	session  *coop.Session
	state    coop.StopHookState
	hasState bool
	removed  int
}

func (s *stopHookTestStore) Read(id string) (*coop.Session, error) {
	if s.session == nil || s.session.ID != id {
		return nil, errors.New("not found")
	}
	return s.session, nil
}

func (s *stopHookTestStore) LatestActiveSession() (*coop.Session, error) {
	if s.session == nil {
		return nil, errors.New("not found")
	}
	return s.session, nil
}

func (s *stopHookTestStore) ReadStopHookState(string) (coop.StopHookState, error) {
	if !s.hasState {
		return coop.StopHookState{}, nil
	}
	return s.state, nil
}

func (s *stopHookTestStore) WriteStopHookState(_ string, state coop.StopHookState) error {
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
	var out bytes.Buffer
	require.NoError(t, runCoopStopHook(&out, store, resumer, sessionID))
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
	assert.Contains(t, decision.SystemMessage, "stripe coop join coop_hook")
}

// Progress means the loop is working, so the budget resets rather than
// accumulating across a long but healthy session.
func TestStopHookResetsCounterOnSessionProgress(t *testing.T) {
	store := &stopHookTestStore{session: stopHookSession()}

	for i := 0; i < maxConsecutiveStopBlocks; i++ {
		require.Equal(t, "block", runStopHook(t, store, actionableResume(), "coop_hook").Decision)
	}
	store.session.Version++

	decision := runStopHook(t, store, actionableResume(), "coop_hook")
	assert.Equal(t, "block", decision.Decision)
	assert.Equal(t, 1, store.state.Blocks)
}

// Discovery mode has no session when the hook is installed, so the launcher
// cannot bake an ID into the command.
func TestStopHookFallsBackToLatestActiveSession(t *testing.T) {
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

	// Verified against codex-cli 0.145.0: this exact shape parses and validates.
	assert.True(t, strings.HasPrefix(config, `hooks.Stop=[{hooks=[{type="command",command="`))
	assert.Contains(t, config, "coop agent stop-hook")
	assert.Contains(t, config, "coop_abc")
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
