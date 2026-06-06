package coop

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// Store manages session persistence with atomic writes.
type Store struct {
	baseDir string
}

// NewStore creates a Store, ensuring the coop directory exists.
func NewStore(configFolder string) (*Store, error) {
	dir := filepath.Join(configFolder, "coop")
	if err := os.MkdirAll(dir, 0700); err != nil {
		return nil, fmt.Errorf("creating coop directory: %w", err)
	}
	return &Store{baseDir: dir}, nil
}

// NewStoreAt creates a Store at a specific path (for testing).
func NewStoreAt(dir string) (*Store, error) {
	if err := os.MkdirAll(dir, 0700); err != nil {
		return nil, fmt.Errorf("creating coop directory: %w", err)
	}
	return &Store{baseDir: dir}, nil
}

func (s *Store) sessionPath(id string) string {
	// Validate ID to prevent path traversal
	base := filepath.Base(id)
	if base != id || id == "" || id == "." || id == ".." {
		return filepath.Join(s.baseDir, "invalid.json")
	}
	return filepath.Join(s.baseDir, id+".json")
}

// Write atomically persists a session (write to .tmp then rename).
// Uses optimistic locking: checks that the file's current version matches
// the session's version before writing. Returns an error on conflict.
func (s *Store) Write(session *Session) error {
	path := s.sessionPath(session.ID)
	unlock, err := s.acquireSessionLock(path)
	if err != nil {
		return err
	}
	defer unlock()

	// Optimistic lock: if file exists, verify version hasn't changed
	if existing, err := os.ReadFile(path); err == nil {
		var current Session
		if json.Unmarshal(existing, &current) == nil {
			if current.Version != session.Version {
				// Re-read to get latest state
				return fmt.Errorf("version conflict: expected %d, file has %d (another writer updated)", session.Version, current.Version)
			}
		}
	}

	session.UpdatedAt = time.Now().UTC()
	session.Version++

	data, err := json.MarshalIndent(session, "", "  ")
	if err != nil {
		return fmt.Errorf("marshaling session: %w", err)
	}

	tmp, err := os.CreateTemp(s.baseDir, filepath.Base(path)+".*.tmp")
	if err != nil {
		return fmt.Errorf("creating temp file: %w", err)
	}
	tmpPath := tmp.Name()
	defer os.Remove(tmpPath)

	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		return fmt.Errorf("writing temp file: %w", err)
	}
	if err := tmp.Close(); err != nil {
		return fmt.Errorf("closing temp file: %w", err)
	}

	if err := os.Rename(tmpPath, path); err != nil {
		return fmt.Errorf("renaming temp file: %w", err)
	}

	return nil
}

func (s *Store) acquireSessionLock(path string) (func(), error) {
	lockPath := path + ".lock"
	deadline := time.Now().Add(5 * time.Second)

	for {
		f, err := os.OpenFile(lockPath, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
		if err == nil {
			f.Close()
			return func() { os.Remove(lockPath) }, nil
		}
		if !os.IsExist(err) {
			return nil, fmt.Errorf("creating lock file: %w", err)
		}

		if info, statErr := os.Stat(lockPath); statErr == nil && time.Since(info.ModTime()) > 30*time.Second {
			os.Remove(lockPath)
			continue
		}

		if time.Now().After(deadline) {
			return nil, fmt.Errorf("timed out waiting for session lock")
		}
		time.Sleep(25 * time.Millisecond)
	}
}

// Read loads a session from disk.
func (s *Store) Read(id string) (*Session, error) {
	data, err := os.ReadFile(s.sessionPath(id))
	if err != nil {
		return nil, fmt.Errorf("reading session %q: %w", id, err)
	}

	var session Session
	if err := json.Unmarshal(data, &session); err != nil {
		return nil, fmt.Errorf("parsing session %q: %w", id, err)
	}

	return &session, nil
}

// ModTime returns the file modification time (for polling).
func (s *Store) ModTime(id string) (time.Time, error) {
	info, err := os.Stat(s.sessionPath(id))
	if err != nil {
		return time.Time{}, err
	}
	return info.ModTime(), nil
}

// List returns all session IDs found on disk.
func (s *Store) List() ([]string, error) {
	entries, err := os.ReadDir(s.baseDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}

	var ids []string
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".json") {
			continue
		}
		if strings.HasSuffix(e.Name(), ".tmp") {
			continue
		}
		ids = append(ids, strings.TrimSuffix(e.Name(), ".json"))
	}
	return ids, nil
}

// LatestSession returns the most recently updated session.
func (s *Store) LatestSession() (*Session, error) {
	ids, err := s.List()
	if err != nil {
		return nil, err
	}
	if len(ids) == 0 {
		return nil, fmt.Errorf("no coop sessions found")
	}

	type entry struct {
		id      string
		modTime time.Time
	}
	var entries []entry
	for _, id := range ids {
		mt, err := s.ModTime(id)
		if err != nil {
			continue
		}
		entries = append(entries, entry{id: id, modTime: mt})
	}

	if len(entries) == 0 {
		return nil, fmt.Errorf("no readable coop sessions found")
	}

	sort.Slice(entries, func(i, j int) bool {
		return entries[i].modTime.After(entries[j].modTime)
	})

	return s.Read(entries[0].id)
}

// LatestActiveSession returns the most recently updated session with status "active".
func (s *Store) LatestActiveSession() (*Session, error) {
	ids, err := s.List()
	if err != nil {
		return nil, err
	}
	if len(ids) == 0 {
		return nil, fmt.Errorf("no coop sessions found")
	}

	type entry struct {
		id      string
		modTime time.Time
	}
	var entries []entry
	for _, id := range ids {
		mt, err := s.ModTime(id)
		if err != nil {
			continue
		}
		entries = append(entries, entry{id: id, modTime: mt})
	}

	sort.Slice(entries, func(i, j int) bool {
		return entries[i].modTime.After(entries[j].modTime)
	})

	for _, e := range entries {
		session, err := s.Read(e.id)
		if err != nil {
			continue
		}
		if session.Status == SessionActive {
			return session, nil
		}
	}

	return nil, fmt.Errorf("no active coop sessions found")
}

// Delete removes a session file.
func (s *Store) Delete(id string) error {
	return os.Remove(s.sessionPath(id))
}

func (s *Store) heartbeatPath(id string) string {
	return s.sessionPath(id) + ".heartbeat"
}

// WriteHeartbeat updates the heartbeat file for a session (signals agent is polling).
func (s *Store) WriteHeartbeat(id string) {
	os.WriteFile(s.heartbeatPath(id), []byte(time.Now().UTC().Format(time.RFC3339)), 0600)
}

// HeartbeatAge returns how long ago the heartbeat was updated.
// Returns -1 if no heartbeat file exists.
func (s *Store) HeartbeatAge(id string) time.Duration {
	info, err := os.Stat(s.heartbeatPath(id))
	if err != nil {
		return -1
	}
	return time.Since(info.ModTime())
}

// RemoveHeartbeat cleans up the heartbeat file.
func (s *Store) RemoveHeartbeat(id string) {
	os.Remove(s.heartbeatPath(id))
}
