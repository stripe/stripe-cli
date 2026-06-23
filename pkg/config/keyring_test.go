package config

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	zkr "github.com/zalando/go-keyring"
)

func TestZalandoStoreGetNotFound(t *testing.T) {
	store := newZalandoStore("stripe-cli-test-nonexistent")

	_, err := store.Get("no-such-key")
	assert.Equal(t, ErrKeyNotFound, err)
}

func TestZalandoStoreRemoveNotFound(t *testing.T) {
	store := newZalandoStore("stripe-cli-test-nonexistent")

	err := store.Remove("no-such-key")
	assert.Equal(t, ErrKeyNotFound, err)
}

func TestZalandoStoreRoundTrip(t *testing.T) {
	store := newZalandoStore("stripe-cli-test")
	defer func() {
		_ = store.Remove("test-key")
	}()

	err := store.Set("test-key", []byte("test-value"), "")
	require.NoError(t, err)

	data, err := store.Get("test-key")
	require.NoError(t, err)
	assert.Equal(t, []byte("test-value"), data)

	err = store.Remove("test-key")
	require.NoError(t, err)

	_, err = store.Get("test-key")
	assert.Equal(t, ErrKeyNotFound, err)
}

func TestZalandoStoreGetTimeout(t *testing.T) {
	block := make(chan struct{})
	defer close(block)

	store := newZalandoStore("stripe-cli-test")
	store.timeout = 10 * time.Millisecond
	store.get = func(_, _ string) (string, error) { <-block; return "", nil }

	_, err := store.Get("key")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "timed out")
	assert.Contains(t, err.Error(), "key")
}

func TestZalandoStoreSetTimeout(t *testing.T) {
	block := make(chan struct{})
	defer close(block)

	store := newZalandoStore("stripe-cli-test")
	store.timeout = 10 * time.Millisecond
	store.set = func(_, _, _ string) error { <-block; return nil }

	err := store.Set("key", []byte("val"), "")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "timed out")
	assert.Contains(t, err.Error(), "key")
}

func TestZalandoStoreRemoveTimeout(t *testing.T) {
	block := make(chan struct{})
	defer close(block)

	store := newZalandoStore("stripe-cli-test")
	store.timeout = 10 * time.Millisecond
	store.del = func(_, _ string) error { <-block; return nil }

	err := store.Remove("key")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "timed out")
	assert.Contains(t, err.Error(), "key")
}

func TestZalandoStoreGetReturnsErrNotFoundFromBackend(t *testing.T) {
	store := newZalandoStore("stripe-cli-test")
	store.get = func(_, _ string) (string, error) { return "", zkr.ErrNotFound }

	_, err := store.Get("key")
	assert.Equal(t, ErrKeyNotFound, err)
}

func TestZalandoStoreRemoveReturnsErrNotFoundFromBackend(t *testing.T) {
	store := newZalandoStore("stripe-cli-test")
	store.del = func(_, _ string) error { return zkr.ErrNotFound }

	err := store.Remove("key")
	assert.Equal(t, ErrKeyNotFound, err)
}

// newTestFileStore returns a fileStore backed by a temp directory.
func newTestFileStore(t *testing.T) *fileStore {
	t.Helper()
	return &fileStore{path: filepath.Join(t.TempDir(), "credentials.json")}
}

func TestFileStoreRoundTrip(t *testing.T) {
	store := newTestFileStore(t)

	require.NoError(t, store.Set("k", []byte("v"), ""))

	data, err := store.Get("k")
	require.NoError(t, err)
	assert.Equal(t, []byte("v"), data)

	require.NoError(t, store.Remove("k"))

	_, err = store.Get("k")
	assert.Equal(t, ErrKeyNotFound, err)
}

func TestFileStoreGetNotFound(t *testing.T) {
	_, err := newTestFileStore(t).Get("missing")
	assert.Equal(t, ErrKeyNotFound, err)
}

func TestFileStoreRemoveNotFound(t *testing.T) {
	err := newTestFileStore(t).Remove("missing")
	assert.Equal(t, ErrKeyNotFound, err)
}

func TestFileStorePermissions(t *testing.T) {
	store := newTestFileStore(t)
	require.NoError(t, store.Set("k", []byte("v"), ""))

	info, err := os.Stat(store.path)
	require.NoError(t, err)
	assert.Equal(t, os.FileMode(0600), info.Mode().Perm())
}

// failStore is a SecureStore whose every operation returns a fixed error.
type failStore struct{ err error }

func (f *failStore) Get(_ string) ([]byte, error)           { return nil, f.err }
func (f *failStore) Set(_ string, _ []byte, _ string) error { return f.err }
func (f *failStore) Remove(_ string) error                  { return f.err }

func TestFallbackStoreGetFallsBackOnPrimaryError(t *testing.T) {
	fb := newTestFileStore(t)
	require.NoError(t, fb.Set("k", []byte("from-file"), ""))

	store := &fallbackStore{
		primary:  &failStore{fmt.Errorf("keyring unavailable")},
		fallback: fb,
	}

	data, err := store.Get("k")
	require.NoError(t, err)
	assert.Equal(t, []byte("from-file"), data)
}

func TestFallbackStoreGetReturnsPrimaryValue(t *testing.T) {
	primary := NewMemoryStore(map[string][]byte{"k": []byte("from-primary")})
	store := &fallbackStore{primary: primary, fallback: newTestFileStore(t)}

	data, err := store.Get("k")
	require.NoError(t, err)
	assert.Equal(t, []byte("from-primary"), data)
}

func TestFallbackStoreSetFallsBackOnPrimaryError(t *testing.T) {
	fb := newTestFileStore(t)
	store := &fallbackStore{
		primary:  &failStore{fmt.Errorf("keyring unavailable")},
		fallback: fb,
	}

	require.NoError(t, store.Set("k", []byte("v"), ""))

	data, err := fb.Get("k")
	require.NoError(t, err)
	assert.Equal(t, []byte("v"), data)
}

func TestFallbackStoreRemoveBothNotFound(t *testing.T) {
	store := &fallbackStore{
		primary:  NewMemoryStore(nil),
		fallback: newTestFileStore(t),
	}
	assert.Equal(t, ErrKeyNotFound, store.Remove("missing"))
}

func TestFallbackStoreRemoveFromPrimary(t *testing.T) {
	primary := NewMemoryStore(map[string][]byte{"k": []byte("v")})
	store := &fallbackStore{primary: primary, fallback: newTestFileStore(t)}

	require.NoError(t, store.Remove("k"))
	_, err := primary.Get("k")
	assert.Equal(t, ErrKeyNotFound, err)
}

func TestFallbackStoreRemoveFromFallback(t *testing.T) {
	fb := newTestFileStore(t)
	require.NoError(t, fb.Set("k", []byte("v"), ""))

	store := &fallbackStore{primary: NewMemoryStore(nil), fallback: fb}

	require.NoError(t, store.Remove("k"))
	_, err := fb.Get("k")
	assert.Equal(t, ErrKeyNotFound, err)
}

func TestIsUsingInsecureStorageFalseWhenPrimarySucceeds(t *testing.T) {
	prev := KeyRing
	t.Cleanup(func() { KeyRing = prev })

	KeyRing = &fallbackStore{primary: NewMemoryStore(nil), fallback: newTestFileStore(t)}

	require.NoError(t, KeyRing.Set("k", []byte("v"), ""))
	assert.False(t, IsUsingInsecureStorage())
}

func TestIsUsingInsecureStorageTrueWhenFallbackUsed(t *testing.T) {
	prev := KeyRing
	t.Cleanup(func() { KeyRing = prev })

	KeyRing = &fallbackStore{
		primary:  &failStore{fmt.Errorf("keyring unavailable")},
		fallback: newTestFileStore(t),
	}

	require.NoError(t, KeyRing.Set("k", []byte("v"), ""))
	assert.True(t, IsUsingInsecureStorage())
}

func TestIsUsingInsecureStorageFalseForNonFallbackStore(t *testing.T) {
	prev := KeyRing
	t.Cleanup(func() { KeyRing = prev })

	KeyRing = NewMemoryStore(nil)
	assert.False(t, IsUsingInsecureStorage())
}
