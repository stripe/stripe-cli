package autoupdate

import (
	"os"

	"github.com/stripe/stripe-cli/pkg/errorcategory"
)

// oldSuffix names the outgoing binary while it is being replaced.
const oldSuffix = ".old"

// removeSupersededBinary deletes the outgoing binary a previous update parked
// next to the new one.
//
// On Windows that delete could not happen at update time, because the process
// being replaced was still running from the file. Any later invocation is a
// process that is not, so this runs on every invocation and almost always finds
// nothing.
func removeSupersededBinary(exe string) {
	_ = os.Remove(exe + oldSuffix)
}

// replaceBinary moves staged into place at dst, which is usually the image of a
// process that is still running.
//
// The live binary is renamed aside first rather than overwritten. On Windows
// that is the only thing that works: the OS locks the image of a running
// executable against writes and deletes, but a rename only touches the directory
// entry, so it is allowed. Unix would tolerate a plain rename over the target,
// but taking the same path on both platforms means the interesting case is the
// one that runs everywhere, including in the unit tests.
func replaceBinary(dst, staged string) error {
	if err := os.Chmod(staged, 0755); err != nil {
		return errorcategory.Errorf(errorcategory.Filesystem, "could not make %s executable: %v", staged, err)
	}

	aside := dst + oldSuffix

	// Left over from an update whose .old file was still in use at the time.
	_ = os.Remove(aside)

	movedAside := false

	if _, err := os.Stat(dst); err == nil {
		if err := os.Rename(dst, aside); err != nil {
			return errorcategory.Errorf(errorcategory.Filesystem, "could not move %s aside: %v", dst, err)
		}

		movedAside = true
	}

	if err := os.Rename(staged, dst); err != nil {
		if movedAside {
			// Put the working binary back. Failing to install an update is
			// recoverable; leaving no stripe on PATH at all is not.
			_ = os.Rename(aside, dst)
		}

		return errorcategory.Errorf(errorcategory.Filesystem, "could not install %s: %v", dst, err)
	}

	// Best effort: on Windows this fails while the calling process is still
	// running from the old image. MaybeRun removes it on the next invocation.
	_ = os.Remove(aside)

	return nil
}
