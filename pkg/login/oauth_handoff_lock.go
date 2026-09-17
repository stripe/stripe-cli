package login

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"time"

	"github.com/stripe/stripe-cli/pkg/errorcategory"
)

// A process-local gate supplies Go memory synchronization in addition to the
// kernel lock used for coordination with other CLI processes.
var oauthHandoffGate = make(chan struct{}, 1)

// All continuation operations hold this lock only for bounded machine work, never
// while waiting for a person. The kernel releases it when a process is killed.
func lockOAuthHandoff(ctx context.Context) (func(), error) {
	select {
	case oauthHandoffGate <- struct{}{}:
	case <-ctx.Done():
		return nil, ctx.Err()
	}
	lockedLocally := true
	defer func() {
		if lockedLocally {
			<-oauthHandoffGate
		}
	}()
	path := pendingDeviceAuthPath() + ".lock"
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		return nil, err
	}
	if info, err := os.Lstat(path); err == nil && !info.Mode().IsRegular() {
		return nil, errorcategory.New(errorcategory.Filesystem, "OAuth continuation lock must be a regular file")
	} else if err != nil && !errors.Is(err, os.ErrNotExist) {
		return nil, err
	}
	f, err := os.OpenFile(path, os.O_CREATE|os.O_RDWR, 0600)
	if err != nil {
		return nil, err
	}
	for {
		locked, err := tryOAuthFileLock(f)
		if err != nil {
			f.Close()
			return nil, err
		}
		if locked {
			lockedLocally = false
			return func() { unlockOAuthFile(f); f.Close(); <-oauthHandoffGate }, nil
		}
		select {
		case <-ctx.Done():
			f.Close()
			return nil, ctx.Err()
		case <-time.After(25 * time.Millisecond):
		}
	}
}
