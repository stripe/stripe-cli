package coop

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestStoreWriteRead(t *testing.T) {
	dir := t.TempDir()
	store, err := NewStoreAt(dir)
	require.NoError(t, err)

	session := &Session{
		ID:        "test_001",
		Blueprint: "one-time-payment",
		Status:    SessionActive,
		Steps: []SessionStep{
			{
				StepDefinition: StepDefinition{Key: "ch-1", Title: "Step 1"},
				Nodes: []SessionNode{
					testSessionNode("n-1", "Step 1", NodePending),
				},
			},
		},
		CreatedAt: time.Now().UTC(),
	}

	err = store.Write(session)
	require.NoError(t, err)
	assert.Equal(t, 1, session.Version)

	loaded, err := store.Read("test_001")
	require.NoError(t, err)
	assert.Equal(t, "test_001", loaded.ID)
	assert.Equal(t, "one-time-payment", loaded.Blueprint)
	assert.Equal(t, CurrentSessionSchemaVersion, loaded.SchemaVersion)
	assert.Equal(t, 1, loaded.Version)
	assert.Equal(t, NodePending, loaded.Steps[0].Nodes[0].State)
}

func TestStoreWriteIncrementsVersion(t *testing.T) {
	dir := t.TempDir()
	store, err := NewStoreAt(dir)
	require.NoError(t, err)

	session := &Session{ID: "test_002", Status: SessionActive}
	store.Write(session)
	assert.Equal(t, 1, session.Version)

	store.Write(session)
	assert.Equal(t, 2, session.Version)
}

func TestStoreWriteDetectsVersionConflict(t *testing.T) {
	dir := t.TempDir()
	store, err := NewStoreAt(dir)
	require.NoError(t, err)

	session := &Session{ID: "conflict_test", Status: SessionActive}
	require.NoError(t, store.Write(session))

	stale := *session
	require.NoError(t, store.Write(session))

	err = store.Write(&stale)
	require.ErrorIs(t, err, ErrVersionConflict)
}

func TestStoreUpdateMutatesAndVersionsSession(t *testing.T) {
	dir := t.TempDir()
	store, err := NewStoreAt(dir)
	require.NoError(t, err)

	session := &Session{
		ID:     "update_test",
		Status: SessionActive,
		Steps: []SessionStep{
			{
				StepDefinition: StepDefinition{Key: "step"},
				Nodes:          []SessionNode{testSessionNode("step", "Step", NodePending)},
			},
		},
	}
	require.NoError(t, store.Write(session))

	updated, err := store.Update("update_test", func(session *Session) error {
		node, err := session.NodeByNumber(1)
		require.NoError(t, err)
		node.Activity = "working"
		return nil
	})
	require.NoError(t, err)
	assert.Equal(t, 2, updated.Version)
	assert.Equal(t, CurrentSessionSchemaVersion, updated.SchemaVersion)
	assert.Equal(t, "working", updated.Steps[0].Nodes[0].Activity)

	loaded, err := store.Read("update_test")
	require.NoError(t, err)
	assert.Equal(t, "working", loaded.Steps[0].Nodes[0].Activity)
}

func TestStoreDefaultsSchemaOnReadWriteAndUpdate(t *testing.T) {
	dir := t.TempDir()
	store, err := NewStoreAt(dir)
	require.NoError(t, err)

	rawPath := filepath.Join(dir, "missing_schema.json")
	require.NoError(t, os.WriteFile(rawPath, []byte(`{"id":"missing_schema","status":"active","version":7}`), 0600))

	readSession, err := store.Read("missing_schema")
	require.NoError(t, err)
	assert.Equal(t, CurrentSessionSchemaVersion, readSession.SchemaVersion)

	updated, err := store.Update("missing_schema", func(session *Session) error {
		session.Blueprint = "updated"
		return nil
	})
	require.NoError(t, err)
	assert.Equal(t, CurrentSessionSchemaVersion, updated.SchemaVersion)

	written := &Session{ID: "new_missing_schema", Status: SessionActive}
	require.NoError(t, store.Write(written))
	assert.Equal(t, CurrentSessionSchemaVersion, written.SchemaVersion)
}

func TestStoreUpdateInvalidSessionID(t *testing.T) {
	dir := t.TempDir()
	store, err := NewStoreAt(dir)
	require.NoError(t, err)

	_, err = store.Update("../bad", func(session *Session) error { return nil })
	require.ErrorIs(t, err, ErrInvalidSessionID)
}

func TestStoreList(t *testing.T) {
	dir := t.TempDir()
	store, err := NewStoreAt(dir)
	require.NoError(t, err)

	store.Write(&Session{ID: "sess_a", Status: SessionActive})
	store.Write(&Session{ID: "sess_b", Status: SessionActive})

	ids, err := store.List()
	require.NoError(t, err)
	assert.Len(t, ids, 2)
	assert.Contains(t, ids, "sess_a")
	assert.Contains(t, ids, "sess_b")
}

func TestStoreLatestSession(t *testing.T) {
	dir := t.TempDir()
	store, err := NewStoreAt(dir)
	require.NoError(t, err)

	store.Write(&Session{ID: "old", Status: SessionActive})
	time.Sleep(10 * time.Millisecond)
	store.Write(&Session{ID: "new", Status: SessionActive})

	latest, err := store.LatestSession()
	require.NoError(t, err)
	assert.Equal(t, "new", latest.ID)
}

func TestStoreModTime(t *testing.T) {
	dir := t.TempDir()
	store, err := NewStoreAt(dir)
	require.NoError(t, err)

	store.Write(&Session{ID: "timed", Status: SessionActive})

	mt, err := store.ModTime("timed")
	require.NoError(t, err)
	assert.WithinDuration(t, time.Now(), mt, 2*time.Second)
}

func TestStoreDelete(t *testing.T) {
	dir := t.TempDir()
	store, err := NewStoreAt(dir)
	require.NoError(t, err)

	store.Write(&Session{ID: "doomed", Status: SessionActive})
	err = store.Delete("doomed")
	require.NoError(t, err)

	_, err = store.Read("doomed")
	assert.Error(t, err)
}

func TestStoreReadNotFound(t *testing.T) {
	dir := t.TempDir()
	store, err := NewStoreAt(dir)
	require.NoError(t, err)

	_, err = store.Read("nonexistent")
	assert.Error(t, err)
}

func TestStoreLatestActiveSession(t *testing.T) {
	dir := t.TempDir()
	store, err := NewStoreAt(dir)
	require.NoError(t, err)

	// Write a completed session
	store.Write(&Session{ID: "completed_one", Status: SessionCompleted})
	time.Sleep(10 * time.Millisecond)

	// Write an active session
	store.Write(&Session{ID: "active_one", Status: SessionActive})
	time.Sleep(10 * time.Millisecond)

	// Write an aborted session (most recent, but not active)
	store.Write(&Session{ID: "aborted_one", Status: SessionAborted})

	// Should find the active one
	session, err := store.LatestActiveSession()
	require.NoError(t, err)
	assert.Equal(t, "active_one", session.ID)
}

func TestStoreLatestActiveSessionNoneActive(t *testing.T) {
	dir := t.TempDir()
	store, err := NewStoreAt(dir)
	require.NoError(t, err)

	store.Write(&Session{ID: "done", Status: SessionCompleted})
	store.Write(&Session{ID: "aborted", Status: SessionAborted})

	_, err = store.LatestActiveSession()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no active")
}

func TestStoreLatestActiveSessionEmpty(t *testing.T) {
	dir := t.TempDir()
	store, err := NewStoreAt(dir)
	require.NoError(t, err)

	_, err = store.LatestActiveSession()
	assert.Error(t, err)
}

func TestStoreAtomicWrite(t *testing.T) {
	dir := t.TempDir()
	store, err := NewStoreAt(dir)
	require.NoError(t, err)

	session := &Session{ID: "atomic_test", Status: SessionActive}
	err = store.Write(session)
	require.NoError(t, err)

	// Verify no .tmp file left behind
	entries, _ := os.ReadDir(dir)
	for _, e := range entries {
		assert.False(t, strings.HasSuffix(e.Name(), ".tmp"), "temp file should be cleaned up")
	}
}

func TestStoreWriteCleansLockAndTempFiles(t *testing.T) {
	dir := t.TempDir()
	store, err := NewStoreAt(dir)
	require.NoError(t, err)

	session := &Session{ID: "clean_test", Status: SessionActive}
	require.NoError(t, store.Write(session))

	entries, err := os.ReadDir(dir)
	require.NoError(t, err)
	for _, e := range entries {
		assert.False(t, strings.HasSuffix(e.Name(), ".tmp"), "temp file should be cleaned up")
		assert.False(t, strings.HasSuffix(e.Name(), ".lock"), "lock file should be cleaned up")
	}
}

func TestStoreWriteLockTimeoutIncludesRecoveryHint(t *testing.T) {
	dir := t.TempDir()
	store, err := NewStoreAt(dir)
	require.NoError(t, err)

	originalTimeout := sessionLockTimeout
	originalPollInterval := sessionLockPollInterval
	sessionLockTimeout = 10 * time.Millisecond
	sessionLockPollInterval = time.Millisecond
	t.Cleanup(func() {
		sessionLockTimeout = originalTimeout
		sessionLockPollInterval = originalPollInterval
	})

	lockPath := filepath.Join(dir, "locked.json.lock")
	require.NoError(t, os.WriteFile(lockPath, []byte("locked"), 0600))

	err = store.Write(&Session{ID: "locked", Status: SessionActive})
	require.ErrorIs(t, err, ErrLockTimeout)
	assert.Contains(t, err.Error(), lockPath)
	assert.Contains(t, err.Error(), "remove this lock file")
}

func TestStoreListIgnoresTmpFiles(t *testing.T) {
	dir := t.TempDir()
	store, err := NewStoreAt(dir)
	require.NoError(t, err)

	store.Write(&Session{ID: "real", Status: SessionActive})
	// Simulate a leftover tmp file
	os.WriteFile(dir+"/orphan.json.tmp", []byte("{}"), 0600)

	ids, err := store.List()
	require.NoError(t, err)
	assert.Equal(t, []string{"real"}, ids)
}

func TestStoreHeartbeatLifecycle(t *testing.T) {
	dir := t.TempDir()
	store, err := NewStoreAt(dir)
	require.NoError(t, err)

	require.NoError(t, store.Write(&Session{ID: "heartbeat", Status: SessionActive}))

	age, err := store.HeartbeatAge("heartbeat")
	require.NoError(t, err)
	assert.Equal(t, time.Duration(-1), age)

	require.NoError(t, store.WriteHeartbeat("heartbeat"))

	age, err = store.HeartbeatAge("heartbeat")
	require.NoError(t, err)
	assert.GreaterOrEqual(t, age, time.Duration(0))
	assert.Less(t, age, 2*time.Second)

	require.NoError(t, store.RemoveHeartbeat("heartbeat"))

	age, err = store.HeartbeatAge("heartbeat")
	require.NoError(t, err)
	assert.Equal(t, time.Duration(-1), age)
}

func TestStoreHeartbeatInvalidSessionID(t *testing.T) {
	dir := t.TempDir()
	store, err := NewStoreAt(dir)
	require.NoError(t, err)

	require.ErrorIs(t, store.WriteHeartbeat("../bad"), ErrInvalidSessionID)

	_, err = store.HeartbeatAge("../bad")
	require.ErrorIs(t, err, ErrInvalidSessionID)

	require.ErrorIs(t, store.RemoveHeartbeat("../bad"), ErrInvalidSessionID)

	_, err = os.Stat(filepath.Join(dir, "invalid.heartbeat"))
	assert.True(t, errors.Is(err, os.ErrNotExist))
}

func TestLatestSessionSkipsCorruptNewest(t *testing.T) {
	dir := t.TempDir()
	store, err := NewStoreAt(dir)
	require.NoError(t, err)

	require.NoError(t, store.Write(&Session{ID: "coop_valid", Status: SessionActive}))
	corrupt := filepath.Join(dir, "coop_corrupt.json")
	require.NoError(t, os.WriteFile(corrupt, []byte("{ not json"), 0o600))

	// Force the corrupt file to be the most-recently-modified entry.
	now := time.Now()
	require.NoError(t, os.Chtimes(filepath.Join(dir, "coop_valid.json"), now.Add(-time.Hour), now.Add(-time.Hour)))
	require.NoError(t, os.Chtimes(corrupt, now, now))

	got, err := store.LatestSession()
	require.NoError(t, err)
	assert.Equal(t, "coop_valid", got.ID, "LatestSession should skip the corrupt newest file")
}
