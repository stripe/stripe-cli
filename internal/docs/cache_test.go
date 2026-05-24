package docs

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

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

	dir := t.TempDir()
	cache, err := NewFSCache(dir)
	require.NoError(t, err)

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
	dir := t.TempDir()
	cache, err := NewFSCache(dir)
	require.NoError(t, err)

	got, cachedAt, ok, err := cache.Get("/nonexistent")
	require.NoError(t, err)
	assert.False(t, ok)
	assert.Nil(t, got)
	assert.True(t, cachedAt.IsZero())
}

func TestFSCache_Get_TTLEviction(t *testing.T) {
	now := time.Now()
	clock := func() time.Time { return now }

	dir := t.TempDir()
	cache, err := NewFSCache(dir, WithTTL(5*time.Minute), WithClock(clock))
	require.NoError(t, err)

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

	entries, _ := os.ReadDir(dir)
	assert.Empty(t, entries)
}

func TestFSCache_Get_UnreadableFile(t *testing.T) {
	dir := t.TempDir()
	cache, err := NewFSCache(dir)
	require.NoError(t, err)

	require.NoError(t, cache.Set("key", []byte("data")))

	path := filepath.Join(dir, hash("key")+".md")
	require.NoError(t, os.Chmod(path, 0o000))
	t.Cleanup(func() { os.Chmod(path, 0o644) })

	_, _, _, err = cache.Get("key")
	assert.Error(t, err)
}

func TestFSCache_Set_UnwritableDir(t *testing.T) {
	dir := t.TempDir()
	cache, err := NewFSCache(dir)
	require.NoError(t, err)

	require.NoError(t, os.Chmod(dir, 0o555))
	t.Cleanup(func() { os.Chmod(dir, 0o755) })

	err = cache.Set("key", []byte("data"))
	assert.Error(t, err)
}

func TestNewFSCache_CreatesDirIfMissing(t *testing.T) {
	dir := t.TempDir() + "/nested/cache"
	_, err := NewFSCache(dir)
	require.NoError(t, err)

	info, err := os.Stat(dir)
	require.NoError(t, err)
	assert.True(t, info.IsDir())
}

func TestNewFSCache_DefaultTTL(t *testing.T) {
	cache, err := NewFSCache(t.TempDir())
	require.NoError(t, err)
	assert.Equal(t, defaultCacheTTL, cache.ttl)
}

func TestNewFSCache_CustomTTL(t *testing.T) {
	cache, err := NewFSCache(t.TempDir(), WithTTL(5*time.Minute))
	require.NoError(t, err)
	assert.Equal(t, 5*time.Minute, cache.ttl)
}

