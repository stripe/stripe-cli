package fsutil

import (
	"fmt"
	"os"

	"github.com/spf13/afero"
)

// RefuseWriteThroughSymlink rejects writes to an existing symlink path.
func RefuseWriteThroughSymlink(fs afero.Fs, filePath, fileDescription string) error {
	lstater, ok := fs.(afero.Lstater)
	if !ok {
		return nil
	}

	fileInfo, _, err := lstater.LstatIfPossible(filePath)
	return refuseWriteThroughSymlink(fileInfo, err, filePath, fileDescription)
}

// RefuseWriteThroughSymlinkOS rejects writes to an existing symlink path on the OS filesystem.
func RefuseWriteThroughSymlinkOS(filePath, fileDescription string) error {
	fileInfo, err := os.Lstat(filePath)
	return refuseWriteThroughSymlink(fileInfo, err, filePath, fileDescription)
}

func refuseWriteThroughSymlink(fileInfo os.FileInfo, err error, filePath, fileDescription string) error {
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("failed to check %s for symlink: %w", fileDescription, err)
	}
	if fileInfo.Mode()&os.ModeSymlink != 0 {
		return fmt.Errorf("refusing to write %s: %s is a symlink", fileDescription, filePath)
	}

	return nil
}
