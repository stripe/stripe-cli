//go:build !darwin

package config

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestZalandoStoreGetNotFound(t *testing.T) {
	store := &zalandoStore{service: "stripe-cli-test-nonexistent"}

	_, err := store.Get("no-such-key")
	assert.Equal(t, ErrKeyNotFound, err)
}

func TestZalandoStoreRemoveNotFound(t *testing.T) {
	store := &zalandoStore{service: "stripe-cli-test-nonexistent"}

	err := store.Remove("no-such-key")
	assert.Equal(t, ErrKeyNotFound, err)
}

func TestZalandoStoreRoundTrip(t *testing.T) {
	store := &zalandoStore{service: "stripe-cli-test"}
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

func TestWSLWinCredStoreTargetName(t *testing.T) {
	store := &wslWinCredStore{service: "StripeCLI"}

	assert.Equal(t, "StripeCLI:default.live_mode_api_key", store.targetName("default.live_mode_api_key"))
	assert.Equal(t, "StripeCLI:session", store.targetName("session"))
}
