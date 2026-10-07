//go:build !windows

package autoupdate

// isTransientLock reports whether a rename failed because another process had
// the file open for a moment. Unix renames do not care who has a file open, so a
// failure here is never one that waiting will fix.
func isTransientLock(err error) bool {
	return false
}
