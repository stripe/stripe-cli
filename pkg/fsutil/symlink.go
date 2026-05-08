// Package fsutil provides shared filesystem safety helpers.
package fsutil

import (
	"fmt"
	"os"

	"github.com/spf13/afero"
)

// IsSymlink returns true if path is a symbolic link.
// Returns false if the path does not exist, cannot be lstat'd, or the
// filesystem does not support symlink-aware lstat operations.
func IsSymlink(fs afero.Fs, path string) bool {
	lstater, ok := fs.(afero.Lstater)
	if !ok {
		return false
	}

	entry, lstated, err := lstater.LstatIfPossible(path)
	if err != nil || !lstated {
		return false
	}

	return entry.Mode()&os.ModeSymlink == os.ModeSymlink
}

// RefuseWriteThroughSymlink rejects writes to symlinked paths.
func RefuseWriteThroughSymlink(fs afero.Fs, path, name string) error {
	if IsSymlink(fs, path) {
		return fmt.Errorf("refusing to write %s: %s is a symlink", name, path)
	}

	return nil
}
