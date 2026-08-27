//go:build !windows

package autoupdate

import (
	"os/exec"
	"syscall"
)

// detach puts the worker in its own session so that a Ctrl-C in the terminal
// that started the parent, or the parent exiting, does not kill it mid-download.
func detach(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{Setsid: true}
}
