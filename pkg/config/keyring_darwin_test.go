//go:build darwin

package config

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDarwinKeychainRoundTrip(t *testing.T) {
	kc := &darwinKeychain{service: "stripe-cli-test"}
	defer func() {
		_ = kc.Remove("test-key")
	}()

	err := kc.Set("test-key", []byte("secret-value-123"), "test description")
	require.NoError(t, err)

	data, err := kc.Get("test-key")
	require.NoError(t, err)
	assert.Equal(t, []byte("secret-value-123"), data)

	// Overwrite
	err = kc.Set("test-key", []byte("updated-value"), "")
	require.NoError(t, err)

	data, err = kc.Get("test-key")
	require.NoError(t, err)
	assert.Equal(t, []byte("updated-value"), data)

	// Remove
	err = kc.Remove("test-key")
	require.NoError(t, err)

	_, err = kc.Get("test-key")
	assert.Equal(t, ErrKeyNotFound, err)
}

func TestDarwinKeychainGetNotFound(t *testing.T) {
	kc := &darwinKeychain{service: "stripe-cli-test"}

	_, err := kc.Get("nonexistent-key-xyz")
	assert.Equal(t, ErrKeyNotFound, err)
}

func TestDarwinKeychainRemoveNotFound(t *testing.T) {
	kc := &darwinKeychain{service: "stripe-cli-test"}

	err := kc.Remove("nonexistent-key-xyz")
	assert.Equal(t, ErrKeyNotFound, err)
}

func TestDarwinKeychainBinaryData(t *testing.T) {
	kc := &darwinKeychain{service: "stripe-cli-test"}
	defer func() {
		_ = kc.Remove("binary-key")
	}()

	data := []byte("line1\nline2\x00binary\ttab\"quote")
	err := kc.Set("binary-key", data, "")
	require.NoError(t, err)

	got, err := kc.Get("binary-key")
	require.NoError(t, err)
	assert.Equal(t, data, got)
}

func TestDarwinKeychainNoPersistentPrompts(t *testing.T) {
	kc := &darwinKeychain{service: "stripe-cli-test"}
	defer func() {
		_ = kc.Remove("prompt-test")
	}()

	err := kc.Set("prompt-test", []byte("test-val"), "")
	require.NoError(t, err)

	for i := 0; i < 5; i++ {
		data, err := kc.Get("prompt-test")
		require.NoError(t, err)
		assert.Equal(t, []byte("test-val"), data)
	}
}
