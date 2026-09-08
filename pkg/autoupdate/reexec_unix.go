//go:build !windows

package autoupdate

import (
	"os"
	"syscall"

	log "github.com/sirupsen/logrus"
)

// reexec hands the invocation to the binary that was just installed, so the
// command the user typed runs on the new version.
func reexec(exe string) {
	err := syscall.Exec(exe, os.Args, os.Environ())
	if err != nil {
		log.Debug("autoupdate: re-exec failed: ", err)
	}
}
