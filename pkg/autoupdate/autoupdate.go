// Package autoupdate keeps a self-owned stripe binary current.
//
// The CLI can only replace its own binary when it owns it. A Homebrew, apt,
// Scoop, WinGet, or npm install belongs to that package manager, and rewriting
// the file underneath it leaves the manager's metadata describing a version that
// is no longer on disk. So every entry point here first checks that the running
// binary sits in the directory scripts/install.sh and scripts/install.ps1 write
// to, and does nothing at all otherwise.
package autoupdate

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/mitchellh/go-homedir"
	log "github.com/sirupsen/logrus"

	"github.com/stripe/stripe-cli/pkg/config"
	"github.com/stripe/stripe-cli/pkg/errorcategory"
	"github.com/stripe/stripe-cli/pkg/version"
)

const (
	// EnvDisable turns auto-update off. Any value other than "0" or "false"
	// disables it, so `STRIPE_NO_AUTO_UPDATE=1 stripe listen` works for a single
	// invocation and exporting it works for a whole shell.
	EnvDisable = "STRIPE_NO_AUTO_UPDATE"

	// EnvInstallDir overrides where the install scripts put the binary. It is read
	// here for the same reason they read it: a binary installed somewhere else is
	// still self-owned and still updatable.
	EnvInstallDir = "STRIPE_INSTALL_DIR"

	// envWorker marks the detached child that performs the update so it does not
	// spawn a worker of its own.
	envWorker = "STRIPE_AUTO_UPDATE_WORKER"

	// checkInterval is the minimum time between two checks. The design budget is
	// "at most once a day"; a single stamp file enforces it across invocations.
	checkInterval = 24 * time.Hour

	stampFileName = "autoupdate-check"
	logFileName   = "autoupdate.log"
	lockFileName  = "autoupdate.lock"
)

// now is indirected so tests can move the clock without sleeping.
var now = time.Now

// MaybeRun starts a background update when one is due, then returns.
//
// It runs on every invocation, so it has to stay cheap and silent: in the common
// case it stats two files and returns. When an update is due it hands the work to
// a separate process, because the command the user actually typed must not wait
// on a download — and on Windows a process cannot overwrite its own image anyway.
func MaybeRun(cfg config.IConfig) {
	exe, ok := SelfManaged()
	if !ok {
		return
	}

	// Done before the opt-out check, so that disabling auto-update does not leave a
	// previous update's leftover file behind forever.
	removeSupersededBinary(exe)

	if os.Getenv(envWorker) != "" || !Enabled() || !dueForCheck(cfg) {
		return
	}

	// Stamp before spawning rather than after the worker finishes: two commands
	// started at the same moment would otherwise both read a stale stamp and both
	// start downloading.
	if err := writeStamp(cfg, now()); err != nil {
		debugf("could not record the update check: %v", err)
		return
	}

	if err := spawnWorker(cfg, exe); err != nil {
		debugf("could not start the update worker: %v", err)
	}
}

// Run performs one update attempt in the foreground and reports what it did to
// out. It is what the detached worker executes, and what `stripe auto-update
// --now` calls.
//
// Deliberately not gated on Enabled(): the worker is only ever spawned after
// MaybeRun has checked, and a user who typed --now has asked for this update
// regardless of the standing preference.
func Run(ctx context.Context, cfg config.IConfig, out io.Writer) error {
	exe, ok := SelfManaged()
	if !ok {
		return errorcategory.Errorf(errorcategory.UserInput,
			"auto-update only applies to a stripe binary installed by the install script, which puts it in %s; upgrade this one with whatever installed it",
			InstallDir())
	}

	release, err := acquireLock(cfg)
	if err != nil {
		return err
	}
	defer release()

	latest, err := latestVersion(ctx)
	if err != nil {
		return errorcategory.Errorf(errorcategory.Network, "could not look up the latest version: %v", err)
	}

	if !isNewer(version.Version, latest) {
		fmt.Fprintf(out, "stripe %s is already the latest version\n", version.Version)
		_ = writeStamp(cfg, now())

		return nil
	}

	// Stage inside the install directory, not the OS temp directory: the last step
	// is a rename, and a rename across filesystems fails. On many machines
	// (containers especially) /tmp is a different filesystem than $HOME.
	staging, err := os.MkdirTemp(filepath.Dir(exe), ".stripe-update-")
	if err != nil {
		return errorcategory.Errorf(errorcategory.Filesystem, "could not create a staging directory: %v", err)
	}

	defer func() { _ = os.RemoveAll(staging) }()

	binary, err := download(ctx, latest, staging)
	if err != nil {
		return err
	}

	if err := replaceBinary(exe, binary); err != nil {
		return err
	}

	_ = writeStamp(cfg, now())

	fmt.Fprintf(out, "Updated stripe %s -> %s\n", version.Version, latest)

	return nil
}

// Enabled reports whether the CLI may replace its own binary, ignoring where the
// binary happens to live. Callers that act on the answer should pair it with
// SelfManaged.
func Enabled() bool {
	// A build from source reports version "master" and has no matching release to
	// download; the same check keeps version.CheckLatestVersion quiet.
	if IsDevBuild() {
		return false
	}

	if envDisabled() {
		return false
	}

	return config.AutoUpdateEnabled()
}

// IsDevBuild reports whether this binary was built from source rather than cut
// by goreleaser, which stamps the release version into version.Version.
func IsDevBuild() bool {
	return version.Version == "master"
}

func envDisabled() bool {
	value := strings.TrimSpace(os.Getenv(EnvDisable))

	return value != "" && value != "0" && !strings.EqualFold(value, "false")
}

// InstallDir returns the directory the install scripts install stripe into.
func InstallDir() string {
	if dir := os.Getenv(EnvInstallDir); dir != "" {
		return dir
	}

	home, err := homedir.Dir()
	if err != nil {
		return ""
	}

	return filepath.Join(home, ".stripe", "bin")
}

// SelfManaged returns the path of the running binary and whether the install
// scripts own it.
func SelfManaged() (string, bool) {
	exe, err := os.Executable()
	if err != nil {
		return "", false
	}

	return selfManaged(exe, InstallDir())
}

func selfManaged(exe, installDir string) (string, bool) {
	// Resolve every side of the comparison. A user may have symlinked stripe onto
	// their PATH, or symlinked a component of the install path (which macOS does
	// itself for /tmp), and comparing unresolved paths would then read as "not
	// ours" and silently switch updates off.
	exe = resolveSymlinks(exe)

	if installDir == "" {
		return exe, false
	}

	return exe, sameDir(resolveSymlinks(filepath.Dir(exe)), resolveSymlinks(installDir))
}

func resolveSymlinks(path string) string {
	if resolved, err := filepath.EvalSymlinks(path); err == nil {
		return resolved
	}

	return path
}

func sameDir(left, right string) bool {
	left, right = filepath.Clean(left), filepath.Clean(right)

	if runtime.GOOS == "windows" {
		return strings.EqualFold(left, right)
	}

	return left == right
}

func stateDir(cfg config.IConfig) string {
	return cfg.GetConfigFolder(os.Getenv("XDG_CONFIG_HOME"))
}

func dueForCheck(cfg config.IConfig) bool {
	raw, err := os.ReadFile(filepath.Join(stateDir(cfg), stampFileName))
	if err != nil {
		return true
	}

	last, err := time.Parse(time.RFC3339, strings.TrimSpace(string(raw)))
	if err != nil {
		return true
	}

	// A stamp in the future means the clock moved backwards, not that a check is
	// overdue. Waiting is the safe reading: the worst case is a delayed update.
	return now().Sub(last) >= checkInterval
}

func writeStamp(cfg config.IConfig, at time.Time) error {
	dir := stateDir(cfg)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	return os.WriteFile(filepath.Join(dir, stampFileName), []byte(at.Format(time.RFC3339)), 0600)
}

// acquireLock keeps two workers from downloading over each other. The returned
// function releases the lock.
func acquireLock(cfg config.IConfig) (func(), error) {
	dir := stateDir(cfg)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, errorcategory.Errorf(errorcategory.Filesystem, "could not create %s: %v", dir, err)
	}

	path := filepath.Join(dir, lockFileName)

	for attempt := 0; attempt < 2; attempt++ {
		file, err := os.OpenFile(path, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
		if err == nil {
			_ = file.Close()

			return func() { _ = os.Remove(path) }, nil
		}

		if !os.IsExist(err) {
			return nil, errorcategory.Errorf(errorcategory.Filesystem, "could not lock %s: %v", path, err)
		}

		// A worker killed mid-download leaves the lock behind, which would block
		// every future update. Treat one older than the check interval as abandoned.
		info, statErr := os.Stat(path)
		if statErr != nil || now().Sub(info.ModTime()) < checkInterval {
			return nil, errorcategory.Errorf(errorcategory.Filesystem, "another stripe update is already running")
		}

		_ = os.Remove(path)
	}

	return nil, errorcategory.Errorf(errorcategory.Filesystem, "another stripe update is already running")
}

func spawnWorker(cfg config.IConfig, exe string) error {
	dir := stateDir(cfg)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	// The worker has no terminal of its own — the parent owns it, and writing
	// there would interleave with the output of the command the user ran — so its
	// output goes to a file that can be read after the fact.
	logFile, err := os.OpenFile(filepath.Join(dir, logFileName), os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0600)
	if err != nil {
		return err
	}

	cmd := exec.Command(exe, "auto-update", "--now")
	cmd.Env = append(os.Environ(), envWorker+"=1")
	cmd.Stdout = logFile
	cmd.Stderr = logFile
	detach(cmd)

	if err := cmd.Start(); err != nil {
		_ = logFile.Close()

		return err
	}

	// Reap the child if this process outlives it, without blocking on it. If the
	// parent exits first the goroutine dies with it and the kernel reparents the
	// child, which is the whole point of detaching.
	go func() {
		_ = cmd.Wait()
		_ = logFile.Close()
	}()

	return nil
}

func debugf(format string, args ...interface{}) {
	log.WithField("prefix", "autoupdate").Debugf(format, args...)
}
