package cmd

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
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

	t.Setenv("XDG_CONFIG_HOME", dir)

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

func TestCoopConfigFolderUsesXDGConfigHome(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", dir)

	assert.Equal(t, filepath.Join(dir, "stripe"), coopConfigFolder())
}

func TestAgentPromptsUseSandboxCommand(t *testing.T) {
	rc := &coopRunCmd{language: "node"}
	assert.Contains(t, rc.buildAgentPrompt("one-time-payment"), "stripe sandbox create --from-git")
	assert.NotContains(t, rc.buildAgentPrompt("one-time-payment"), "stripe sandboxes create")
	assert.Contains(t, rc.buildAgentPrompt(""), "stripe sandbox create --from-git")

	bp := &coop.Blueprint{Title: "Test integration"}
	session := &coop.Session{}
	assert.Contains(t, agentInstructions(bp, session), "stripe sandbox create --from-git")
	assert.NotContains(t, agentInstructions(bp, session), "stripe sandboxes create")
}

func TestBlueprintMatchScoreRanksRelevantBlueprints(t *testing.T) {
	oneTime, err := coop.LoadBlueprint("one-time-payment")
	require.NoError(t, err)
	deploy, err := coop.LoadBlueprint("deploy-stripe-projects")
	require.NoError(t, err)

	assert.Greater(t, blueprintMatchScore(*oneTime, "accept payments"), 0)
	assert.Equal(t, 0, blueprintMatchScore(*deploy, "accept payments"))
}

func TestDoSkipFinalStepRoutesToNextSteps(t *testing.T) {
	store, session := setupStepTest(t)
	for i := 1; i <= session.TotalSteps(); i++ {
		require.NoError(t, session.TransitionStep(i, coop.StepActive))
		if i < session.TotalSteps() {
			require.NoError(t, session.TransitionStep(i, coop.StepDone))
		}
	}
	require.NoError(t, store.Write(session))

	sc := &coopStepCmd{}
	output := captureStdout(t, func() {
		require.NoError(t, sc.doSkip(store, session, session.TotalSteps()))
	})

	assert.Contains(t, output, `"state": "skipped"`)
	assert.Contains(t, output, `"next": "stripe coop next-steps --session=step_test_session"`)
}

func TestCoopRunStoresParentSessionMetadata(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", dir)

	rc := &coopStartCmd{
		language:      "node",
		parentSession: "parent_session",
		parentStep:    "deploy",
	}
	output := captureStdout(t, func() {
		require.NoError(t, rc.runStartCmd(nil, []string{"one-time-payment"}))
	})

	var resp coopStartResponse
	require.NoError(t, json.Unmarshal([]byte(output), &resp))
	require.Contains(t, resp.Next, "--session="+resp.SessionID)
	require.Contains(t, resp.Next, `--note="Beginning: Understand the project"`)

	store, err := coop.NewStore(coopConfigFolder())
	require.NoError(t, err)
	session, err := store.Read(resp.SessionID)
	require.NoError(t, err)
	assert.Equal(t, "parent_session", session.ParentSessionID)
	assert.Equal(t, "deploy", session.ParentStepID)
}

func TestNextStepsDeployIncludesParentMetadata(t *testing.T) {
	_, session := setupCompletedNextStepsSession(t)
	suggestions := buildSuggestions(session, projectEnvironment{})

	resp := buildNextStepsResponse(session, suggestions, "deploy")

	assert.Equal(t, session.ID, resp.SessionID)
	assert.Equal(t, "stripe coop run deploy-stripe-projects --language=node --parent-session=step_test_session --parent-step=deploy", resp.Next)
}

func TestOutputCoopErrorReturnsRenderedError(t *testing.T) {
	output := captureStderr(t, func() {
		err := outputCoopError("No session found.", "stripe coop run <blueprint>")
		var rendered coopRenderedError
		require.True(t, errors.As(err, &rendered))
	})

	var resp coop.CommandResponse
	require.NoError(t, json.Unmarshal([]byte(output), &resp))
	assert.False(t, resp.OK)
	assert.Equal(t, "No session found.", resp.Error)
	assert.Equal(t, "stripe coop run <blueprint>", resp.Hint)
}

func setupCompletedNextStepsSession(t *testing.T) (*coop.Store, *coop.Session) {
	t.Helper()
	dir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", dir)
	store, err := coop.NewStore(coopConfigFolder())
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
					{Key: "n1", Title: "Step 1", State: coop.StepDone, Type: coop.NodeAPIRequest},
					{Key: "n2", Title: "Step 2", State: coop.StepDone, Type: coop.NodeTestHelper},
				},
			},
		},
	}
	for i := range session.Chapters {
		for j := range session.Chapters[i].Nodes {
			session.Chapters[i].Nodes[j].State = coop.StepDone
		}
	}
	require.NoError(t, store.Write(session))
	return store, session
}

func captureStdout(t *testing.T, fn func()) string {
	t.Helper()

	orig := os.Stdout
	r, w, err := os.Pipe()
	require.NoError(t, err)
	os.Stdout = w

	fn()

	require.NoError(t, w.Close())
	os.Stdout = orig

	var buf bytes.Buffer
	_, err = io.Copy(&buf, r)
	require.NoError(t, err)
	return strings.TrimSpace(buf.String())
}

func captureStderr(t *testing.T, fn func()) string {
	t.Helper()

	orig := os.Stderr
	r, w, err := os.Pipe()
	require.NoError(t, err)
	os.Stderr = w

	fn()

	require.NoError(t, w.Close())
	os.Stderr = orig

	var buf bytes.Buffer
	_, err = io.Copy(&buf, r)
	require.NoError(t, err)
	return strings.TrimSpace(buf.String())
}
