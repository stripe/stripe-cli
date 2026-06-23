package config

import (
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
