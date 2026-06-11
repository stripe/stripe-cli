//go:build !darwin

package config

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	zkr "github.com/zalando/go-keyring"
)

// secretServiceAvailable returns true if the OS keyring backend is usable.
// On Linux CI (no D-Bus), zalando/go-keyring returns errors other than
// ErrNotFound when the secret service is unavailable.
func secretServiceAvailable() bool {
	_, err := zkr.Get("stripe-cli-probe", "probe")
	return err == nil || err == zkr.ErrNotFound
}

func TestZalandoStoreGetNotFound(t *testing.T) {
	if !secretServiceAvailable() {
		t.Skip("secret service not available (no D-Bus)")
	}

	store := &zalandoStore{service: "stripe-cli-test-nonexistent"}

	_, err := store.Get("no-such-key")
	assert.Equal(t, ErrKeyNotFound, err)
}

func TestZalandoStoreRemoveNotFound(t *testing.T) {
	if !secretServiceAvailable() {
		t.Skip("secret service not available (no D-Bus)")
	}

	store := &zalandoStore{service: "stripe-cli-test-nonexistent"}

	err := store.Remove("no-such-key")
	assert.Equal(t, ErrKeyNotFound, err)
}

func TestZalandoStoreRoundTrip(t *testing.T) {
	if !secretServiceAvailable() {
		t.Skip("secret service not available (no D-Bus)")
	}

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
