//go:build !unix

package plugins

import "github.com/spf13/afero"

// POSIX ownership and mode bits do not validate Windows ACLs.
func checkPluginPath(fs afero.Fs, path, store string) error {
	return nil
}
