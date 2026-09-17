//go:build !windows

package login

import (
	"errors"
	"os"

	"golang.org/x/sys/unix"
)

func tryOAuthFileLock(f *os.File) (bool, error) {
	err := unix.Flock(int(f.Fd()), unix.LOCK_EX|unix.LOCK_NB)
	if errors.Is(err, unix.EWOULDBLOCK) || errors.Is(err, unix.EAGAIN) {
		return false, nil
	}
	return err == nil, err
}

func unlockOAuthFile(f *os.File) { _ = unix.Flock(int(f.Fd()), unix.LOCK_UN) }
