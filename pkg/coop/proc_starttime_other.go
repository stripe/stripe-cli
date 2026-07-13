//go:build !darwin && !linux

package coop

import "time"

// processStartTime is unavailable on this platform. Callers treat an unknown
// start time as "can't rule out reuse" and leave the lock in place; on Windows
// processAlive is already indeterminate, so the age fallback governs there.
func processStartTime(pid int) (time.Time, bool) {
	_ = pid
	return time.Time{}, false
}
