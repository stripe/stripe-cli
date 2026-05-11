// Package fsutil provides shared filesystem safety helpers.
package fsutil

import (
	"fmt"
	"os"

	"github.com/spf13/afero"
)

func lstatIfPossible(fs afero.Fs, path string) (os.FileInfo, bool, error) {
	lstater, ok := fs.(afero.Lstater)
	if !ok {
		return nil, false, nil
	}

	return lstater.LstatIfPossible(path)
}

// IsSymlink returns true if path is a symbolic link.
// Returns false if the path does not exist, cannot be lstat'd, or the
// filesystem does not support symlink-aware lstat operations.
func IsSymlink(fs afero.Fs, path string) bool {
	entry, lstated, err := lstatIfPossible(fs, path)
	if err != nil || !lstated {
		return false
	}

	return entry.Mode()&os.ModeSymlink == os.ModeSymlink
}

// RefuseWriteThroughSymlink rejects writes to symlinked paths.
func RefuseWriteThroughSymlink(fs afero.Fs, path, name string) error {
	entry, lstated, err := lstatIfPossible(fs, path)
	if !lstated {
		return nil
	}

	return refuseWriteThroughSymlink(entry, err, path, name)
}

// RefuseWriteThroughSymlinkOS rejects writes to an existing symlink path on the OS filesystem.
func RefuseWriteThroughSymlinkOS(path, name string) error {
	entry, err := os.Lstat(path)
	return refuseWriteThroughSymlink(entry, err, path, name)
}

func refuseWriteThroughSymlink(entry os.FileInfo, err error, path, name string) error {
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("failed to check %s for symlink: %w", name, err)
	}
	if entry.Mode()&os.ModeSymlink == os.ModeSymlink {
		return fmt.Errorf("refusing to write %s: %s is a symlink", name, path)
	}

	return nil
}
