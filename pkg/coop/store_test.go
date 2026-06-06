package coop

import (
	"os"
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
		Chapters: []SessionChapter{
			{
				Key:   "ch-1",
				Title: "Chapter 1",
				Nodes: []SessionNode{
					{Key: "n-1", Title: "Step 1", State: StepPending},
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
	assert.Equal(t, 1, loaded.Version)
	assert.Equal(t, StepPending, loaded.Chapters[0].Nodes[0].State)
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
