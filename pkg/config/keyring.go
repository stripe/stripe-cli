package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"

	log "github.com/sirupsen/logrus"
	zkr "github.com/zalando/go-keyring"
)

// ErrKeyNotFound is returned when a key is not present in the secure store.
var ErrKeyNotFound = errors.New("secure store: key not found")

// SecureStore provides access to OS-level secure credential storage.
type SecureStore interface {
	Get(key string) ([]byte, error)
	Set(key string, data []byte, description string) error
	Remove(key string) error
}

type zalandoStore struct {
	service string
	timeout time.Duration
	get     func(string, string) (string, error)
	set     func(string, string, string) error
	del     func(string, string) error
}

func newZalandoStore(service string) *zalandoStore {
	return &zalandoStore{
		service: service,
		timeout: 3 * time.Second,
		get:     zkr.Get,
		set:     zkr.Set,
		del:     zkr.Delete,
	}
}

func (s *zalandoStore) Get(key string) ([]byte, error) {
	type result struct {
		val string
		err error
	}
	ch := make(chan result, 1)
	go func() {
		val, err := s.get(s.service, key)
		ch <- result{val, err}
	}()
	select {
	case r := <-ch:
		if errors.Is(r.err, zkr.ErrNotFound) {
			return nil, ErrKeyNotFound
		}
		return []byte(r.val), r.err
	case <-time.After(s.timeout):
		return nil, fmt.Errorf("keyring: timed out getting %q", key)
	}
}

func (s *zalandoStore) Set(key string, data []byte, description string) error {
	ch := make(chan error, 1)
	go func() {
		ch <- s.set(s.service, key, string(data))
	}()
	select {
	case err := <-ch:
		return err
	case <-time.After(s.timeout):
		return fmt.Errorf("keyring: timed out setting %q", key)
	}
}

func (s *zalandoStore) Remove(key string) error {
	ch := make(chan error, 1)
	go func() {
		ch <- s.del(s.service, key)
	}()
	select {
	case err := <-ch:
		if errors.Is(err, zkr.ErrNotFound) {
			return ErrKeyNotFound
		}
		return err
	case <-time.After(s.timeout):
		return fmt.Errorf("keyring: timed out removing %q", key)
	}
}

// fallbackStore tries primary for every operation and silently retries against
// fallback whenever primary returns an error other than ErrKeyNotFound.
// This mirrors the GitHub CLI pattern: prefer the OS keyring, but write to a
// plain file when the keyring is unavailable (e.g. headless Linux, containers).
type fallbackStore struct {
	primary         SecureStore
	fallback        SecureStore
	wroteToFallback bool
}

// IsUsingInsecureStorage reports whether credentials have been written to the
// plain-file fallback because the OS keyring was unavailable. Callers (e.g.
// the login flow) can use this to warn the user.
func IsUsingInsecureStorage() bool {
	fb, ok := KeyRing.(*fallbackStore)
	return ok && fb.wroteToFallback
}

// CredentialsFilePath returns the path of the plain-text credentials file used
// by the file fallback store.
func CredentialsFilePath() string {
	return newFileStore().path
}

func (s *fallbackStore) Get(key string) ([]byte, error) {
	logger := log.WithFields(log.Fields{"prefix": "keyring", "key": key})
	logger.Debug("reading from credential store")
	data, err := s.primary.Get(key)
	if err == nil {
		return data, nil
	}
	logger.WithError(err).Debug("credential store unavailable, reading from fallback file")
	return s.fallback.Get(key)
}

func (s *fallbackStore) Set(key string, data []byte, description string) error {
	logger := log.WithFields(log.Fields{"prefix": "keyring", "key": key})
	logger.Debug("writing to credential store")
	if err := s.primary.Set(key, data, description); err != nil {
		logger.WithError(err).Debug("credential store unavailable, writing to fallback file")
		if writeErr := s.fallback.Set(key, data, description); writeErr != nil {
			return writeErr
		}
		s.wroteToFallback = true
		return nil
	}
	return nil
}

func (s *fallbackStore) Remove(key string) error {
	logger := log.WithFields(log.Fields{"prefix": "keyring", "key": key})
	logger.Debug("removing from credential store")
	primaryErr := s.primary.Remove(key)
	fallbackErr := s.fallback.Remove(key)
	if errors.Is(primaryErr, ErrKeyNotFound) && errors.Is(fallbackErr, ErrKeyNotFound) {
		return ErrKeyNotFound
	}
	if primaryErr != nil && !errors.Is(primaryErr, ErrKeyNotFound) {
		logger.WithError(primaryErr).Debug("credential store unavailable during remove")
	}
	return nil
}

// fileStore persists credentials as a JSON object in a 0600-permissioned file
// inside the Stripe config directory.
type fileStore struct {
	path string
}

func newFileStore() *fileStore {
	dir := getConfigFolder(os.Getenv("XDG_CONFIG_HOME"))
	return &fileStore{path: filepath.Join(dir, "credentials.json")}
}

func (s *fileStore) load() (map[string]string, error) {
	data, err := os.ReadFile(s.path)
	if errors.Is(err, os.ErrNotExist) {
		return make(map[string]string), nil
	}
	if err != nil {
		return nil, err
	}
	var m map[string]string
	if err := json.Unmarshal(data, &m); err != nil {
		return nil, err
	}
	return m, nil
}

func (s *fileStore) save(m map[string]string) error {
	if err := os.MkdirAll(filepath.Dir(s.path), 0700); err != nil {
		return err
	}
	data, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(s.path, data, 0600)
}

func (s *fileStore) Get(key string) ([]byte, error) {
	m, err := s.load()
	if err != nil {
		return nil, err
	}
	val, ok := m[key]
	if !ok {
		return nil, ErrKeyNotFound
	}
	return []byte(val), nil
}

func (s *fileStore) Set(key string, data []byte, description string) error {
	m, err := s.load()
	if err != nil {
		return err
	}
	m[key] = string(data)
	return s.save(m)
}

func (s *fileStore) Remove(key string) error {
	m, err := s.load()
	if err != nil {
		return err
	}
	if _, ok := m[key]; !ok {
		return ErrKeyNotFound
	}
	delete(m, key)
	return s.save(m)
}

func newSecureStore() SecureStore {
	return &fallbackStore{
		primary:  newZalandoStore(KeyManagementService),
		fallback: newFileStore(),
	}
}

// MemoryStore is an in-memory SecureStore for use in tests.
type MemoryStore struct {
	items map[string][]byte
}

// NewMemoryStore creates a MemoryStore optionally pre-populated with data.
func NewMemoryStore(initial map[string][]byte) *MemoryStore {
	m := &MemoryStore{items: make(map[string][]byte)}
	for k, v := range initial {
		m.items[k] = v
	}
	return m
}

func (m *MemoryStore) Get(key string) ([]byte, error) {
	data, ok := m.items[key]
	if !ok {
		return nil, ErrKeyNotFound
	}
	return data, nil
}

func (m *MemoryStore) Set(key string, data []byte, description string) error {
	m.items[key] = data
	return nil
}

func (m *MemoryStore) Remove(key string) error {
	if _, ok := m.items[key]; !ok {
		return ErrKeyNotFound
	}
	delete(m.items, key)
	return nil
}
