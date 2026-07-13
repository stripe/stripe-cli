//go:build darwin

package coop

import (
	"time"

	"golang.org/x/sys/unix"
)

// processStartTime returns when the process began, via the kernel proc table.
func processStartTime(pid int) (time.Time, bool) {
	kp, err := unix.SysctlKinfoProc("kern.proc.pid", pid)
	if err != nil || kp == nil {
		return time.Time{}, false
	}
	tv := kp.Proc.P_starttime
	return time.Unix(tv.Sec, int64(tv.Usec)*1000), true
}
