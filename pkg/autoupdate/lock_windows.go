//go:build windows

package autoupdate

import (
	"errors"

	"golang.org/x/sys/windows"
)

// isTransientLock reports whether a rename failed because another process had
// the file open for a moment, rather than for a reason that waiting will not fix.
//
// Antivirus is the usual holder: Windows Defender opens every newly written
// executable to scan it, without the share mode that would let it be renamed,
// and both files an update renames -- the download and the binary it replaces --
// are exactly that. The scan finishes in well under a second.
func isTransientLock(err error) bool {
	return errors.Is(err, windows.ERROR_SHARING_VIOLATION) ||
		errors.Is(err, windows.ERROR_LOCK_VIOLATION) ||
		errors.Is(err, windows.ERROR_ACCESS_DENIED)
}
