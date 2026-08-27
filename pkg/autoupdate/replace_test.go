package autoupdate

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestReplaceBinaryInstallsOverALiveBinary(t *testing.T) {
	dir := t.TempDir()
	dst := filepath.Join(dir, "stripe")
	staged := filepath.Join(dir, "staged")

	require.NoError(t, os.WriteFile(dst, []byte("old"), 0755))
	require.NoError(t, os.WriteFile(staged, []byte("new"), 0644))

	require.NoError(t, replaceBinary(dst, staged))

	installed, err := os.ReadFile(dst)
	require.NoError(t, err)
	assert.Equal(t, "new", string(installed))

	// Unix can unlink the moved-aside file immediately. Windows cannot while the
	// image is running, but nothing is running it here either.
	assert.NoFileExists(t, dst+oldSuffix)
	assert.NoFileExists(t, staged)
}

func TestReplaceBinaryInstallsWhenNothingIsThere(t *testing.T) {
	dir := t.TempDir()
	dst := filepath.Join(dir, "stripe")
	staged := filepath.Join(dir, "staged")

	require.NoError(t, os.WriteFile(staged, []byte("new"), 0644))

	require.NoError(t, replaceBinary(dst, staged))

	installed, err := os.ReadFile(dst)
	require.NoError(t, err)
	assert.Equal(t, "new", string(installed))
}

func TestReplaceBinaryClearsAStaleOldFile(t *testing.T) {
	dir := t.TempDir()
	dst := filepath.Join(dir, "stripe")
	staged := filepath.Join(dir, "staged")

	require.NoError(t, os.WriteFile(dst, []byte("old"), 0755))
	require.NoError(t, os.WriteFile(dst+oldSuffix, []byte("older"), 0755))
	require.NoError(t, os.WriteFile(staged, []byte("new"), 0644))

	require.NoError(t, replaceBinary(dst, staged))

	assert.NoFileExists(t, dst+oldSuffix)
}

func TestReplaceBinaryRestoresTheOldBinaryOnFailure(t *testing.T) {
	dir := t.TempDir()
	dst := filepath.Join(dir, "stripe")

	require.NoError(t, os.WriteFile(dst, []byte("old"), 0755))

	// A staged path that does not exist stands in for any failure of the second
	// rename. The point is that the working binary is still on disk afterwards.
	err := replaceBinary(dst, filepath.Join(dir, "missing"))
	require.Error(t, err)

	restored, readErr := os.ReadFile(dst)
	require.NoError(t, readErr)
	assert.Equal(t, "old", string(restored))
}

func TestRemoveSupersededBinary(t *testing.T) {
	dir := t.TempDir()
	exe := filepath.Join(dir, "stripe")

	require.NoError(t, os.WriteFile(exe, []byte("current"), 0755))
	require.NoError(t, os.WriteFile(exe+oldSuffix, []byte("previous"), 0755))

	removeSupersededBinary(exe)

	assert.NoFileExists(t, exe+oldSuffix)
	assert.FileExists(t, exe)

	// Nothing to remove is the common case and must not be an error.
	removeSupersededBinary(exe)
}
