//go:build darwin

package config

import (
	"os/exec"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func hasSecurityCLI() bool {
	_, err := exec.LookPath("/usr/bin/security")
	return err == nil
}

func TestDarwinKeychainRoundTrip(t *testing.T) {
	if !hasSecurityCLI() {
		t.Skip("requires /usr/bin/security")
	}

	kc := &darwinKeychain{service: "stripe-cli-test"}
	defer func() {
		_ = kc.Remove("test-key")
	}()

	// Set a value
	err := kc.Set("test-key", []byte("secret-value-123"), "test description")
	require.NoError(t, err)

	// Get it back
	data, err := kc.Get("test-key")
	require.NoError(t, err)
	assert.Equal(t, []byte("secret-value-123"), data)

	// Overwrite it
	err = kc.Set("test-key", []byte("updated-value"), "")
	require.NoError(t, err)

	data, err = kc.Get("test-key")
	require.NoError(t, err)
	assert.Equal(t, []byte("updated-value"), data)

	// Remove it
	err = kc.Remove("test-key")
	require.NoError(t, err)

	// Get after remove should fail
	_, err = kc.Get("test-key")
	assert.Equal(t, ErrKeyNotFound, err)
}

func TestDarwinKeychainGetNotFound(t *testing.T) {
	if !hasSecurityCLI() {
		t.Skip("requires /usr/bin/security")
	}

	kc := &darwinKeychain{service: "stripe-cli-test"}

	_, err := kc.Get("nonexistent-key-xyz")
	assert.Equal(t, ErrKeyNotFound, err)
}

func TestDarwinKeychainRemoveNotFound(t *testing.T) {
	if !hasSecurityCLI() {
		t.Skip("requires /usr/bin/security")
	}

	kc := &darwinKeychain{service: "stripe-cli-test"}

	err := kc.Remove("nonexistent-key-xyz")
	assert.Equal(t, ErrKeyNotFound, err)
}

func TestDarwinKeychainKeys(t *testing.T) {
	if !hasSecurityCLI() {
		t.Skip("requires /usr/bin/security")
	}

	kc := &darwinKeychain{service: "stripe-cli-test"}
	defer func() {
		_ = kc.Remove("key-a")
		_ = kc.Remove("key-b")
	}()

	err := kc.Set("key-a", []byte("val-a"), "")
	require.NoError(t, err)
	err = kc.Set("key-b", []byte("val-b"), "")
	require.NoError(t, err)

	keys, err := kc.Keys()
	require.NoError(t, err)
	assert.Contains(t, keys, "key-a")
	assert.Contains(t, keys, "key-b")
}

func TestDarwinKeychainBinaryData(t *testing.T) {
	if !hasSecurityCLI() {
		t.Skip("requires /usr/bin/security")
	}

	kc := &darwinKeychain{service: "stripe-cli-test"}
	defer func() {
		_ = kc.Remove("binary-key")
	}()

	// Test with data that contains special characters, newlines, and null bytes
	data := []byte("line1\nline2\x00binary\ttab\"quote")
	err := kc.Set("binary-key", data, "")
	require.NoError(t, err)

	got, err := kc.Get("binary-key")
	require.NoError(t, err)
	assert.Equal(t, data, got)
}

func TestDarwinKeychainNoPersistentPrompts(t *testing.T) {
	if !hasSecurityCLI() {
		t.Skip("requires /usr/bin/security")
	}

	kc := &darwinKeychain{service: "stripe-cli-test"}
	defer func() {
		_ = kc.Remove("prompt-test")
	}()

	// Write once
	err := kc.Set("prompt-test", []byte("test-val"), "")
	require.NoError(t, err)

	// Read multiple times — should never prompt
	for i := 0; i < 5; i++ {
		data, err := kc.Get("prompt-test")
		require.NoError(t, err)
		assert.Equal(t, []byte("test-val"), data)
	}
}
