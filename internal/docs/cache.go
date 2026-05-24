package docs

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

const defaultCacheTTL = 1 * time.Hour

// Cache defines the interface for storing and retrieving page content.
type Cache interface {
	Get(key string) (data []byte, cachedAt time.Time, ok bool, err error)
	Set(key string, data []byte) error
}

// FSCache implements Cache using the local filesystem with a TTL-based
// eviction policy. Entries older than the configured TTL are treated as
// misses and removed on access.
type FSCache struct {
	dir string
	ttl time.Duration
	now func() time.Time
}

// CacheOption configures an FSCache.
type CacheOption func(*FSCache)

// WithTTL sets the cache entry time-to-live. Defaults to 1 hour.
func WithTTL(ttl time.Duration) CacheOption {
	return func(c *FSCache) { c.ttl = ttl }
}

// WithClock overrides the time source used for TTL checks.
func WithClock(now func() time.Time) CacheOption {
	return func(c *FSCache) { c.now = now }
}

// NewFSCache creates a filesystem-backed cache rooted at dir.
func NewFSCache(dir string, opts ...CacheOption) (*FSCache, error) {
	if err := os.MkdirAll(dir, 0o750); err != nil {
		return nil, fmt.Errorf("docs: create cache dir: %w", err)
	}
	c := &FSCache{dir: dir, ttl: defaultCacheTTL, now: time.Now}
	for _, opt := range opts {
		opt(c)
	}
	return c, nil
}

// Get retrieves a cached entry by key, returning the data, cache time, and
// whether the entry was found. Expired entries are removed on access.
func (c *FSCache) Get(key string) ([]byte, time.Time, bool, error) {
	path := filepath.Join(c.dir, hash(key)+".md")

	info, err := os.Stat(path)
	if errors.Is(err, os.ErrNotExist) {
		return nil, time.Time{}, false, nil
	}
	if err != nil {
		return nil, time.Time{}, false, fmt.Errorf("docs: stat cache entry: %w", err)
	}

	if c.now().Sub(info.ModTime()) > c.ttl {
		_ = os.Remove(path)
		return nil, time.Time{}, false, nil
	}

	data, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return nil, time.Time{}, false, nil
	}
	if err != nil {
		return nil, time.Time{}, false, fmt.Errorf("docs: read cache entry: %w", err)
	}

	return data, info.ModTime(), true, nil
}

// Set writes data to the cache under key.
func (c *FSCache) Set(key string, data []byte) error {
	path := filepath.Join(c.dir, hash(key)+".md")
	if err := os.WriteFile(path, data, 0o600); err != nil {
		return fmt.Errorf("docs: write cache entry: %w", err)
	}
	return nil
}

func hash(key string) string {
	h := sha256.Sum256([]byte(key))
	return hex.EncodeToString(h[:])
}
