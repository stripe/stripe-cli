package docs

import (
	"testing"
	"time"

	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newMemCache(t *testing.T, opts ...CacheOption) *FSCache {
	t.Helper()
	opts = append([]CacheOption{WithFS(afero.NewMemMapFs())}, opts...)
	cache, err := NewFSCache("/cache", opts...)
	require.NoError(t, err)
	return cache
}

func TestFSCache_GetSet(t *testing.T) {
	tests := []struct {
		name string
		key  string
		want []byte
	}{
		{"simple key", "/payments", []byte("page content")},
		{"key with special chars", "/api?version=2024&lang=go", []byte("api content")},
		{"key with slashes and pipes", "/a/b/c|d=e", []byte("nested")},
		{"empty value", "/empty", []byte{}},
		{"binary content", "/bin", []byte{0x00, 0xFF, 0x0A}},
	}

	cache := newMemCache(t)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := cache.Set(tt.key, tt.want)
			require.NoError(t, err)

			got, cachedAt, ok, err := cache.Get(tt.key)
			require.NoError(t, err)
			assert.True(t, ok)
			assert.Equal(t, tt.want, got)
			assert.WithinDuration(t, time.Now(), cachedAt, 2*time.Second)
		})
	}
}

func TestFSCache_Get_Miss(t *testing.T) {
	cache := newMemCache(t)

	got, cachedAt, ok, err := cache.Get("/nonexistent")
	require.NoError(t, err)
	assert.False(t, ok)
	assert.Nil(t, got)
	assert.True(t, cachedAt.IsZero())
}

func TestFSCache_Get_TTLEviction(t *testing.T) {
	now := time.Now()
	clock := func() time.Time { return now }

	cache := newMemCache(t, WithTTL(5*time.Minute), WithClock(clock))

	require.NoError(t, cache.Set("key", []byte("data")))

	got, _, ok, err := cache.Get("key")
	require.NoError(t, err)
	assert.True(t, ok)
	assert.Equal(t, []byte("data"), got)

	now = now.Add(6 * time.Minute)

	got, _, ok, err = cache.Get("key")
	require.NoError(t, err)
	assert.False(t, ok)
	assert.Nil(t, got)

	entries, _ := afero.ReadDir(cache.fs, cache.dir)
	assert.Empty(t, entries)
}

func TestNewFSCache_CreatesDirIfMissing(t *testing.T) {
	fs := afero.NewMemMapFs()
	cache, err := NewFSCache("/nested/cache", WithFS(fs))
	require.NoError(t, err)

	info, err := fs.Stat(cache.dir)
	require.NoError(t, err)
	assert.True(t, info.IsDir())
}

func TestNewFSCache_DefaultTTL(t *testing.T) {
	cache := newMemCache(t)
	assert.Equal(t, defaultCacheTTL, cache.ttl)
}

func TestNewFSCache_CustomTTL(t *testing.T) {
	cache := newMemCache(t, WithTTL(5*time.Minute))
	assert.Equal(t, 5*time.Minute, cache.ttl)
}
