//go:build windows

package autoupdate

import (
	"errors"
	"os"
	"os/exec"
	"os/signal"

	log "github.com/sirupsen/logrus"
)

// reexec runs the binary that was just installed as a child process and exits
// with its status.
//
// Windows has no execve. syscall.Exec is a stub there that always fails, so the
// only way to hand the invocation to the new binary is to start it and forward
// what it did. The child inherits this process's standard streams and console, so
// an interactive command such as `stripe login` still works.
//
// Returning instead of exiting means the update landed but this invocation
// carries on running the old image, which is a worse outcome than re-execing and
// a better one than failing the command outright.
func reexec(exe string) {
	cmd := exec.Command(exe, os.Args[1:]...) //nolint:gosec // exe is this process's own path
	cmd.Stdin, cmd.Stdout, cmd.Stderr = os.Stdin, os.Stdout, os.Stderr

	// Ctrl-C is delivered to every process attached to the console, this one
	// included. Ignore it here so that the child, which is the process actually
	// doing the work now, decides what a Ctrl-C means.
	signal.Ignore(os.Interrupt)

	err := cmd.Run()

	var exitErr *exec.ExitError
	if err != nil && !errors.As(err, &exitErr) {
		log.Debug("autoupdate: could not run the updated binary: ", err)

		return
	}

	os.Exit(cmd.ProcessState.ExitCode())
}
