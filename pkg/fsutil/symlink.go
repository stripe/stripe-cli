// Package fsutil provides shared filesystem safety helpers.
package fsutil

import (
	"fmt"
	"os"
	"path/filepath"

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

// RefuseWriteThroughSymlink rejects writes when the destination path or any
// existing parent directory between path and stopPath is a symlink.
func RefuseWriteThroughSymlink(fs afero.Fs, path, stopPath, name string) error {
	return walkPathAndParents(path, stopPath, func(candidate string) error {
		entry, lstated, err := lstatIfPossible(fs, candidate)
		if !lstated {
			return nil
		}

		return refuseWriteThroughSymlink(entry, err, candidate, name)
	})
}

// RefuseWriteThroughSymlinkOS rejects writes when the destination path or any
// existing parent directory between path and stopPath is a symlink on the OS filesystem.
func RefuseWriteThroughSymlinkOS(path, stopPath, name string) error {
	return walkPathAndParents(path, stopPath, func(candidate string) error {
		entry, err := os.Lstat(candidate)
		return refuseWriteThroughSymlink(entry, err, candidate, name)
	})
}

func walkPathAndParents(path, stopPath string, visit func(string) error) error {
	current := filepath.Clean(path)
	stop := filepath.Clean(stopPath)

	for {
		if err := visit(current); err != nil {
			return err
		}
		if current == stop {
			return nil
		}

		parent := filepath.Dir(current)
		if parent == current {
			return nil
		}
		current = parent
	}
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
