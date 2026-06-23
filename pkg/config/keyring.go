package config

import (
	"errors"
	"fmt"
	"time"

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

func newSecureStore() SecureStore {
	return newZalandoStore(KeyManagementService)
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
