package autoupdate

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// oldSuffix names the outgoing binary while it is being replaced.
const oldSuffix = ".old"

// rename is indirected so tests can fail the second move and check that the
// working binary comes back.
var rename = os.Rename

// replaceBinary moves staged into place at dst, which is the image of the process
// calling it.
//
// The live binary is renamed aside first rather than overwritten. On Windows that
// is the only thing that works: the OS locks the image of a running executable
// against writes and deletes, but a rename only touches the directory entry, so
// it is allowed. Unix would tolerate a plain rename over the target, but taking
// the same path on both platforms means the interesting case is the one that runs
// everywhere, including in the unit tests.
func replaceBinary(dst, staged string) error {
	if err := os.Chmod(staged, 0755); err != nil {
		return fmt.Errorf("chmod failed: %w", err)
	}

	aside, err := asideName(dst)
	if err != nil {
		return fmt.Errorf("cannot make room next to %s: %w", dst, err)
	}

	movedAside := false

	if _, err := os.Stat(dst); err == nil {
		if err := rename(dst, aside); err != nil {
			return fmt.Errorf("cannot move %s aside: %w", dst, err)
		}

		movedAside = true
	}

	if err := rename(staged, dst); err != nil {
		if movedAside {
			// Put the working binary back. Failing to install an update is
			// recoverable; leaving no stripe on PATH at all is not.
			_ = rename(aside, dst)
		}

		return fmt.Errorf("cannot replace binary: %w", err)
	}

	// Best effort: on Windows this fails while the calling process is still running
	// from the old image. removeSupersededBinary clears it on a later invocation.
	_ = os.Remove(aside)

	return nil
}

// asideName returns a free path to move the outgoing binary to.
//
// The usual name is dst+".old", left over from a previous update or not there at
// all. When it is there and cannot be deleted, the move has to go somewhere else:
// Windows refuses to delete or replace a file while any process is running from
// it, and after a concurrent update this process may be running from that very
// file. Reusing the name in that case fails the whole update, so fall back to a
// unique one, which a later invocation cleans up along with the rest.
func asideName(dst string) (string, error) {
	aside := dst + oldSuffix
	if err := os.Remove(aside); err == nil || os.IsNotExist(err) {
		return aside, nil
	}

	// Created rather than just named so that two updates racing here cannot pick
	// the same path. Renaming over the empty placeholder is what claims it.
	placeholder, err := os.CreateTemp(filepath.Dir(dst), filepath.Base(dst)+oldSuffix+".*")
	if err != nil {
		return "", err
	}

	name := placeholder.Name()
	_ = placeholder.Close()

	return name, nil
}

// removeSupersededBinary deletes the outgoing binaries earlier updates parked
// next to the new one.
//
// On Windows those deletes could not happen at update time, because a process was
// still running from each file. Any later invocation is a process that is not, so
// this runs on every invocation and almost always finds nothing. It scans rather
// than deleting one known name because asideName falls back to a suffixed name
// when the usual one is occupied.
func removeSupersededBinary(exe string) {
	dir, base := filepath.Split(exe)

	entries, err := os.ReadDir(dir)
	if err != nil {
		return
	}

	for _, entry := range entries {
		if strings.HasPrefix(entry.Name(), base+oldSuffix) {
			_ = os.Remove(filepath.Join(dir, entry.Name()))
		}
	}
}
