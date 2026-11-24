//go:build unix

package plugins

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"syscall"

	"github.com/spf13/afero"
	"github.com/spf13/afero/mem"
)

// checkPluginPath checks both the named path and symlink targets, including
// ancestors that could replace them. Root-owned ancestors are trusted, but
// store contents must belong to the effective user. POSIX ACLs are out of scope.
func checkPluginPath(fs afero.Fs, path, store string) error {
	for _, component := range strings.Split(path, string(filepath.Separator)) {
		if component == ".." {
			return unsafePluginPath(path, "path contains a parent traversal")
		}
	}
	absolute, err := filepath.Abs(path)
	if err != nil {
		return unsafePluginPath(path, err.Error())
	}
	root, err := filepath.Abs(store)
	if err != nil {
		return unsafePluginPath(store, err.Error())
	}
	links := pluginSymlinkWalk{active: make(map[string]bool)}
	return inspectPluginPath(fs, absolute, root, &links)
}

type pluginSymlinkWalk struct {
	total  int
	active map[string]bool // Links whose targets are still being inspected.
}

func inspectPluginPath(fs afero.Fs, path, store string, links *pluginSymlinkWalk) error {
	parent := filepath.Dir(path)
	if parent != path {
		if err := inspectPluginPath(fs, parent, store, links); err != nil {
			return err
		}
	}

	lstater, ok := fs.(afero.Lstater)
	var info os.FileInfo
	var err error
	var usedLstat bool
	if ok {
		info, usedLstat, err = lstater.LstatIfPossible(path)
	} else {
		info, err = fs.Stat(path)
	}
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return unsafePluginPath(path, fmt.Sprintf("cannot inspect path: %v", err))
	}

	// MemMapFs (including wrappers used for fault injection) has no OS owner
	// metadata or symlinks. Recognize its concrete file info, never a nil Sys
	// value from an arbitrary filesystem, as an explicit in-memory model.
	uid := uint32(os.Geteuid())
	if _, memory := info.(*mem.FileInfo); !memory {
		stat, ok := info.Sys().(*syscall.Stat_t)
		if !ok {
			return unsafePluginPath(path, "cannot determine ownership")
		}
		uid = stat.Uid
		if !usedLstat {
			return unsafePluginPath(path, "cannot inspect symbolic links")
		}
	}

	strict := pluginPathWithin(path, store)
	if uid != uint32(os.Geteuid()) && (strict || uid != 0) {
		return unsafePluginPath(path, "not owned by the current user or a trusted system ancestor")
	}

	if info.Mode()&os.ModeSymlink != 0 {
		if links.active[path] {
			return unsafePluginPath(path, "symbolic link cycle")
		}
		links.total++
		if links.total > 40 {
			return unsafePluginPath(path, "too many symbolic links")
		}
		links.active[path] = true
		defer delete(links.active, path)
		reader, ok := fs.(afero.LinkReader)
		if !ok {
			return unsafePluginPath(path, "cannot read symbolic link")
		}
		target, err := reader.ReadlinkIfPossible(path)
		if err != nil {
			return unsafePluginPath(path, fmt.Sprintf("cannot read symbolic link: %v", err))
		}
		// Cleaning a link target containing '..' before resolving earlier links
		// can inspect a different path than the kernel would follow.
		for _, component := range strings.Split(target, string(filepath.Separator)) {
			if component == ".." {
				return unsafePluginPath(path, "symbolic link target contains a parent traversal")
			}
		}
		if !filepath.IsAbs(target) {
			target = filepath.Join(parent, target)
		}
		targetStore := store
		if strict {
			targetStore = target
		}
		return inspectPluginPath(fs, filepath.Clean(target), targetStore, links)
	}

	if !info.IsDir() && !info.Mode().IsRegular() {
		return unsafePluginPath(path, "not a regular file or directory")
	}
	if info.Mode().Perm()&0022 != 0 {
		// A root-owned sticky temporary ancestor cannot have a user-owned child
		// replaced by another user. This exception never applies to a store itself.
		if strict || !info.IsDir() || uid != 0 || info.Mode()&os.ModeSticky == 0 {
			return unsafePluginPath(path, "writable by group or others")
		}
	}
	return nil
}
