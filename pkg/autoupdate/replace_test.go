package autoupdate

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestReplaceBinaryInstallsOverALiveBinary(t *testing.T) {
	dir := t.TempDir()
	dst := filepath.Join(dir, binaryName())
	staged := filepath.Join(dir, "staged")

	require.NoError(t, os.WriteFile(dst, []byte("old"), 0755))
	require.NoError(t, os.WriteFile(staged, []byte("new"), 0644))

	require.NoError(t, replaceBinary(dst, staged))

	installed, err := os.ReadFile(dst)
	require.NoError(t, err)
	assert.Equal(t, "new", string(installed))

	// Unix can unlink the moved-aside file immediately. Windows cannot while a
	// process is running from the image, but nothing is running this one.
	assert.NoFileExists(t, dst+oldSuffix)
	assert.NoFileExists(t, staged)
}

func TestReplaceBinaryInstallsWhenNothingIsThere(t *testing.T) {
	dir := t.TempDir()
	dst := filepath.Join(dir, binaryName())
	staged := filepath.Join(dir, "staged")

	require.NoError(t, os.WriteFile(staged, []byte("new"), 0644))

	require.NoError(t, replaceBinary(dst, staged))

	installed, err := os.ReadFile(dst)
	require.NoError(t, err)
	assert.Equal(t, "new", string(installed))
}

func TestReplaceBinaryClearsAStaleOldFile(t *testing.T) {
	dir := t.TempDir()
	dst := filepath.Join(dir, binaryName())
	staged := filepath.Join(dir, "staged")

	require.NoError(t, os.WriteFile(dst, []byte("old"), 0755))
	require.NoError(t, os.WriteFile(dst+oldSuffix, []byte("older"), 0755))
	require.NoError(t, os.WriteFile(staged, []byte("new"), 0644))

	require.NoError(t, replaceBinary(dst, staged))

	assert.NoFileExists(t, dst+oldSuffix)
}

func TestReplaceBinaryWhenTheOldNameCannotBeCleared(t *testing.T) {
	dir := t.TempDir()
	dst := filepath.Join(dir, binaryName())
	staged := filepath.Join(dir, "staged")

	require.NoError(t, os.WriteFile(dst, []byte("old"), 0755))
	require.NoError(t, os.WriteFile(staged, []byte("new"), 0644))

	// Windows refuses to delete or replace a file while a process is running from
	// it, which is the state of the .old file left behind by a concurrent update. A
	// non-empty directory is refused by os.Remove and os.Rename on every platform,
	// so it stands in for that obstacle in a test that can run anywhere.
	require.NoError(t, os.MkdirAll(filepath.Join(dst+oldSuffix, "occupied"), 0755))

	require.NoError(t, replaceBinary(dst, staged),
		"an unusable .old name must not fail the update")

	installed, err := os.ReadFile(dst)
	require.NoError(t, err)
	assert.Equal(t, "new", string(installed))

	// The outgoing binary went to a suffixed name, and whatever was holding the
	// usual one was left alone rather than clobbered.
	assert.DirExists(t, dst+oldSuffix)
}

func TestReplaceBinaryRestoresTheOldBinaryOnFailure(t *testing.T) {
	dir := t.TempDir()
	dst := filepath.Join(dir, binaryName())
	staged := filepath.Join(dir, "staged")

	require.NoError(t, os.WriteFile(dst, []byte("old"), 0755))
	require.NoError(t, os.WriteFile(staged, []byte("new"), 0644))

	// Fail the move of the new binary into place, after the live one has been moved
	// aside: a full disk, a revoked permission, an antivirus hold on the download.
	original := rename
	rename = func(from, to string) error {
		if from == staged {
			return errors.New("cannot move the new binary into place")
		}

		return original(from, to)
	}

	t.Cleanup(func() { rename = original })

	err := replaceBinary(dst, staged)
	require.Error(t, err)

	restored, readErr := os.ReadFile(dst)
	require.NoError(t, readErr)
	assert.Equal(t, "old", string(restored), "a failed update has to leave a working binary behind")
}

// errLocked stands in for Windows refusing a rename while an antivirus scanner
// has the file open, which no other platform can produce on demand.
var errLocked = errors.New("the process cannot access the file because it is being used by another process")

// lockFor fails every rename of from with errLocked, the first times times, and
// counts the attempts.
func lockFor(t *testing.T, from string, times int) *int {
	t.Helper()

	attempts := 0
	original, originalRetryable, originalBackoff := rename, isRetryable, renameBackoff
	rename = func(f, to string) error {
		if f == from {
			attempts++
			if attempts <= times {
				return errLocked
			}
		}

		return original(f, to)
	}
	isRetryable = func(err error) bool { return errors.Is(err, errLocked) }
	renameBackoff = []time.Duration{0, 0, 0}

	t.Cleanup(func() { rename, isRetryable, renameBackoff = original, originalRetryable, originalBackoff })

	return &attempts
}

func TestReplaceBinaryWaitsOutABriefLock(t *testing.T) {
	for _, held := range []string{"live", "staged"} {
		t.Run(held, func(t *testing.T) {
			dir := t.TempDir()
			dst := filepath.Join(dir, binaryName())
			staged := filepath.Join(dir, "staged")

			require.NoError(t, os.WriteFile(dst, []byte("old"), 0755))
			require.NoError(t, os.WriteFile(staged, []byte("new"), 0644))

			locked := dst
			if held == "staged" {
				locked = staged
			}
			attempts := lockFor(t, locked, 2)

			require.NoError(t, replaceBinary(dst, staged))
			assert.Equal(t, 3, *attempts)

			installed, err := os.ReadFile(dst)
			require.NoError(t, err)
			assert.Equal(t, "new", string(installed))
		})
	}
}

func TestReplaceBinaryGivesUpOnALockThatStays(t *testing.T) {
	dir := t.TempDir()
	dst := filepath.Join(dir, binaryName())
	staged := filepath.Join(dir, "staged")

	require.NoError(t, os.WriteFile(dst, []byte("old"), 0755))
	require.NoError(t, os.WriteFile(staged, []byte("new"), 0644))

	attempts := lockFor(t, staged, 100)

	err := replaceBinary(dst, staged)
	require.ErrorIs(t, err, errLocked)
	assert.Equal(t, 1+len(renameBackoff), *attempts)

	restored, readErr := os.ReadFile(dst)
	require.NoError(t, readErr)
	assert.Equal(t, "old", string(restored), "a failed update has to leave a working binary behind")
}

func TestReplaceBinaryDoesNotRetryOtherFailures(t *testing.T) {
	dir := t.TempDir()
	dst := filepath.Join(dir, binaryName())
	staged := filepath.Join(dir, "staged")

	require.NoError(t, os.WriteFile(dst, []byte("old"), 0755))
	require.NoError(t, os.WriteFile(staged, []byte("new"), 0644))

	lockFor(t, "", 0)

	attempts := 0
	locked := rename
	rename = func(from, to string) error {
		if from == staged {
			attempts++
			return errors.New("disk full")
		}

		return locked(from, to)
	}

	require.Error(t, replaceBinary(dst, staged))
	assert.Equal(t, 1, attempts)
}

func TestRemoveSupersededBinary(t *testing.T) {
	dir := t.TempDir()
	exe := filepath.Join(dir, binaryName())

	require.NoError(t, os.WriteFile(exe, []byte("current"), 0755))
	require.NoError(t, os.WriteFile(exe+oldSuffix, []byte("previous"), 0755))
	// What asideName falls back to when .old itself is occupied.
	require.NoError(t, os.WriteFile(exe+oldSuffix+".2971828", []byte("older"), 0755))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "unrelated"), []byte("keep"), 0644))

	removeSupersededBinary(exe)

	assert.NoFileExists(t, exe+oldSuffix)
	assert.NoFileExists(t, exe+oldSuffix+".2971828")
	assert.FileExists(t, exe)
	assert.FileExists(t, filepath.Join(dir, "unrelated"))

	// Nothing to remove is the common case and must not be an error.
	removeSupersededBinary(exe)
}
