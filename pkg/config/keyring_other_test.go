//go:build !darwin

package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWSLFileStoreRoundTrip(t *testing.T) {
	dir := t.TempDir()
	store, err := newWSLFileStoreWithKey(dir, "test-passphrase")
	require.NoError(t, err)

	err = store.Set("my-key", []byte("my-secret"), "desc")
	require.NoError(t, err)

	data, err := store.Get("my-key")
	require.NoError(t, err)
	assert.Equal(t, []byte("my-secret"), data)
}

func TestWSLFileStoreOverwrite(t *testing.T) {
	dir := t.TempDir()
	store, err := newWSLFileStoreWithKey(dir, "test-passphrase")
	require.NoError(t, err)

	err = store.Set("key", []byte("first"), "")
	require.NoError(t, err)

	err = store.Set("key", []byte("second"), "")
	require.NoError(t, err)

	data, err := store.Get("key")
	require.NoError(t, err)
	assert.Equal(t, []byte("second"), data)
}

func TestWSLFileStoreGetNotFound(t *testing.T) {
	dir := t.TempDir()
	store, err := newWSLFileStoreWithKey(dir, "test-passphrase")
	require.NoError(t, err)

	_, err = store.Get("nonexistent")
	assert.Equal(t, ErrKeyNotFound, err)
}

func TestWSLFileStoreRemove(t *testing.T) {
	dir := t.TempDir()
	store, err := newWSLFileStoreWithKey(dir, "test-passphrase")
	require.NoError(t, err)

	err = store.Set("key", []byte("value"), "")
	require.NoError(t, err)

	err = store.Remove("key")
	require.NoError(t, err)

	_, err = store.Get("key")
	assert.Equal(t, ErrKeyNotFound, err)
}

func TestWSLFileStoreRemoveNotFound(t *testing.T) {
	dir := t.TempDir()
	store, err := newWSLFileStoreWithKey(dir, "test-passphrase")
	require.NoError(t, err)

	err = store.Remove("nonexistent")
	assert.Equal(t, ErrKeyNotFound, err)
}

func TestWSLFileStoreKeys(t *testing.T) {
	dir := t.TempDir()
	store, err := newWSLFileStoreWithKey(dir, "test-passphrase")
	require.NoError(t, err)

	err = store.Set("alpha", []byte("a"), "")
	require.NoError(t, err)
	err = store.Set("beta", []byte("b"), "")
	require.NoError(t, err)

	keys, err := store.Keys()
	require.NoError(t, err)
	assert.ElementsMatch(t, []string{"alpha", "beta"}, keys)
}

func TestWSLFileStoreMultipleKeys(t *testing.T) {
	dir := t.TempDir()
	store, err := newWSLFileStoreWithKey(dir, "test-passphrase")
	require.NoError(t, err)

	err = store.Set("key1", []byte("val1"), "")
	require.NoError(t, err)
	err = store.Set("key2", []byte("val2"), "")
	require.NoError(t, err)
	err = store.Set("key3", []byte("val3"), "")
	require.NoError(t, err)

	data, err := store.Get("key2")
	require.NoError(t, err)
	assert.Equal(t, []byte("val2"), data)

	err = store.Remove("key2")
	require.NoError(t, err)

	// key1 and key3 still accessible
	data, err = store.Get("key1")
	require.NoError(t, err)
	assert.Equal(t, []byte("val1"), data)

	data, err = store.Get("key3")
	require.NoError(t, err)
	assert.Equal(t, []byte("val3"), data)

	// key2 gone
	_, err = store.Get("key2")
	assert.Equal(t, ErrKeyNotFound, err)
}

func TestWSLFileStorePersistence(t *testing.T) {
	dir := t.TempDir()

	// Write with one instance
	store1, err := newWSLFileStoreWithKey(dir, "passphrase")
	require.NoError(t, err)
	err = store1.Set("persist-key", []byte("persist-val"), "")
	require.NoError(t, err)

	// Read with a new instance (simulates process restart)
	store2, err := newWSLFileStoreWithKey(dir, "passphrase")
	require.NoError(t, err)
	data, err := store2.Get("persist-key")
	require.NoError(t, err)
	assert.Equal(t, []byte("persist-val"), data)
}

func TestWSLFileStoreWrongPassphrase(t *testing.T) {
	dir := t.TempDir()

	store1, err := newWSLFileStoreWithKey(dir, "correct-passphrase")
	require.NoError(t, err)
	err = store1.Set("secret", []byte("data"), "")
	require.NoError(t, err)

	// Different passphrase can't decrypt
	store2, err := newWSLFileStoreWithKey(dir, "wrong-passphrase")
	require.NoError(t, err)
	_, err = store2.Get("secret")
	assert.Error(t, err)
}

func TestWSLFileStoreBinaryData(t *testing.T) {
	dir := t.TempDir()
	store, err := newWSLFileStoreWithKey(dir, "pass")
	require.NoError(t, err)

	data := []byte("line1\nline2\x00null\ttab\"quote\\backslash")
	err = store.Set("binary", data, "")
	require.NoError(t, err)

	got, err := store.Get("binary")
	require.NoError(t, err)
	assert.Equal(t, data, got)
}

func TestWSLFileStoreCreatesDirectory(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "nested", "dir")

	store, err := newWSLFileStoreWithKey(dir, "pass")
	require.NoError(t, err)

	err = store.Set("key", []byte("val"), "")
	require.NoError(t, err)

	_, err = os.Stat(filepath.Join(dir, "keys.enc"))
	require.NoError(t, err)
}

// newWSLFileStoreWithKey creates a wslFileStore with a given passphrase for testing.
func newWSLFileStoreWithKey(dir string, passphrase string) (*wslFileStore, error) {
	return newWSLFileStoreFromPassphrase(dir, passphrase)
}
