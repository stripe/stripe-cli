package docs

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/spf13/afero"
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
	fs  afero.Fs
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

// WithFS sets the filesystem implementation. Defaults to the real OS filesystem.
func WithFS(fs afero.Fs) CacheOption {
	return func(c *FSCache) { c.fs = fs }
}

// NewFSCache creates a filesystem-backed cache rooted at dir.
func NewFSCache(dir string, opts ...CacheOption) (*FSCache, error) {
	c := &FSCache{fs: afero.NewOsFs(), dir: dir, ttl: defaultCacheTTL, now: time.Now}
	for _, opt := range opts {
		opt(c)
	}
	if err := c.fs.MkdirAll(dir, 0o750); err != nil {
		return nil, fmt.Errorf("docs: create cache dir: %w", err)
	}
	return c, nil
}

// Get retrieves a cached entry by key, returning the data, cache time, and
// whether the entry was found. Expired entries are removed on access.
func (c *FSCache) Get(key string) ([]byte, time.Time, bool, error) {
	p := filepath.Clean(c.path(key))

	info, err := c.fs.Stat(p)
	if os.IsNotExist(err) {
		return nil, time.Time{}, false, nil
	}
	if err != nil {
		return nil, time.Time{}, false, fmt.Errorf("docs: stat cache entry: %w", err)
	}

	if c.now().Sub(info.ModTime()) > c.ttl {
		_ = c.fs.Remove(p)
		return nil, time.Time{}, false, nil
	}

	data, err := afero.ReadFile(c.fs, p)
	if os.IsNotExist(err) {
		return nil, time.Time{}, false, nil
	}
	if err != nil {
		return nil, time.Time{}, false, fmt.Errorf("docs: read cache entry: %w", err)
	}

	return data, info.ModTime(), true, nil
}

// Set writes data to the cache under key.
func (c *FSCache) Set(key string, data []byte) error {
	p := filepath.Clean(c.path(key))
	if err := afero.WriteFile(c.fs, p, data, 0o600); err != nil {
		return fmt.Errorf("docs: write cache entry: %w", err)
	}
	return nil
}

func (c *FSCache) path(key string) string {
	h := sha256.Sum256([]byte(key))
	name := hex.EncodeToString(h[:]) + ".md"
	return filepath.Join(c.dir, filepath.Base(name))
}
