//go:build windows

package autoupdate

import (
	"os/exec"
	"syscall"
)

// detachedProcess is DETACHED_PROCESS from the Windows process creation flags.
// It is not exported by the syscall package.
const detachedProcess = 0x00000008

// detach severs the worker from the console this process is attached to, so that
// the parent exiting does not take the worker with it and the worker never draws
// a console window of its own.
func detach(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{
		HideWindow:    true,
		CreationFlags: detachedProcess | syscall.CREATE_NEW_PROCESS_GROUP,
	}
}
