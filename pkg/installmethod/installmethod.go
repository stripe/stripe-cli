// Package installmethod detects how the CLI was installed -- which package
// manager, installer, or image put the running binary in place -- and maps that
// to the command that upgrades it, so an out-of-date CLI can tell a user what to
// run rather than only that a newer version exists.
package installmethod

import (
	"os"
	"path/filepath"
	"strings"
)

//
// Public constants
//

// The install methods this package reports. Unknown is a value rather than an
// empty string on purpose: it separates "we looked and could not tell" from a
// CLI old enough that it reported nothing at all.
const (
	APT       = "apt"
	Docker    = "docker"
	Homebrew  = "homebrew"
	NPMGlobal = "npm_global"
	NPMRun    = "npm_run"
	NPX       = "npx"
	Script    = "script"
	Scoop     = "scoop"
	Unknown   = "unknown"
	Winget    = "winget"
	YUM       = "yum"
)

//
// Public types
//

// Env describes the host lookups Detect needs. Injected so that tests can drive
// detection without touching the real environment or filesystem.
type Env struct {
	Getenv       func(string) string
	Executable   func() (string, error)
	EvalSymlinks func(string) (string, error)
	Stat         func(string) error
	ReadFile     func(string) ([]byte, error)
}

// Advice is what to tell a user whose CLI is out of date.
type Advice struct {
	// Command upgrades the CLI, or is empty for an install method we cannot name
	// a command for. Callers should still report that a new version exists in
	// that case: knowing to upgrade is useful even without the command, and a
	// wrong command is worse than no command.
	Command string

	// Suppress is set for a channel that resolves the latest version on every
	// invocation, where an upgrade notice is noise the user cannot act on.
	Suppress bool
}

//
// Public functions
//

// OSEnv returns an Env backed by the real process environment and filesystem.
func OSEnv() Env {
	return Env{
		Getenv:       os.Getenv,
		Executable:   os.Executable,
		EvalSymlinks: filepath.EvalSymlinks,
		Stat:         func(path string) error { _, err := os.Stat(path); return err },
		ReadFile:     os.ReadFile,
	}
}

// Detect reports how the CLI was installed, or Unknown when no signal matches.
//
// Signals are ordered strongest first: a value the installer set itself, then a
// method it stamped next to the binary, then the binary's own location, then
// traces left on the system. Everything after the first two is a heuristic, so
// the order is what keeps a weaker signal from overriding a better one.
func Detect(env Env) string {
	if method := env.Getenv("STRIPE_INSTALL_METHOD"); method != "" {
		// Set by the npm wrapper (npm/wrapper/bin/shim.js), which separates a global
		// install from `npm run` and npx -- a difference only it can see. Reported
		// as-is rather than checked against the constants above, so that a
		// distributor setting its own value has that value reported. UpgradeAdvice
		// names no command for a value it does not know.
		return method
	}

	if method := detectFromExecutable(env); method != "" {
		return method
	}

	// Package managers are checked before the container check below, because our
	// deb and rpm install cleanly inside a container. When one of them did the
	// install, the package manager is the answer and Docker is incidental.
	if err := env.Stat("/var/lib/dpkg/info/stripe.list"); err == nil {
		return APT
	}

	// rpm keeps no per-package file list to stat, so this looks for the repository
	// the install instructions add. It is weaker than the dpkg check above --
	// evidence the user configured Stripe's yum repo, not proof the CLI came from
	// it -- which is why it runs after every stronger signal, including the
	// executable-path checks that identify the other channels outright.
	if err := env.Stat("/etc/yum.repos.d/stripe.repo"); err == nil {
		return YUM
	}

	// The official image (see Dockerfile) copies the bare binary to /bin/stripe on
	// alpine, leaving no packaging trace, so the container itself is the only
	// signal left.
	if err := env.Stat("/.dockerenv"); err == nil {
		return Docker
	}

	return Unknown
}

// UpgradeAdvice returns what to tell a user running an out-of-date CLI that was
// installed by the given method. goos is the running platform, which the install
// scripts' advice depends on; callers pass runtime.GOOS.
func UpgradeAdvice(method, goos string) Advice {
	switch method {
	case Homebrew:
		// "stripe" is both the formula name in stripe/homebrew-stripe-cli and an
		// alias for homebrew-core's "stripe-cli", so this upgrades either one.
		return Advice{Command: "brew upgrade stripe"}
	case APT:
		return Advice{Command: "sudo apt update && sudo apt upgrade stripe"}
	case YUM:
		return Advice{Command: "sudo yum update stripe"}
	case Scoop:
		return Advice{Command: "scoop update stripe"}
	case Winget:
		return Advice{Command: "winget upgrade Stripe.StripeCLI"}
	case Docker:
		return Advice{Command: "docker pull stripe/stripe-cli"}
	case NPMGlobal:
		return Advice{Command: "npm install -g @stripe/cli"}
	case Script:
		if goos == "windows" {
			return Advice{Command: "irm https://raw.githubusercontent.com/stripe/stripe-cli/master/scripts/install.ps1 | iex"}
		}
		return Advice{Command: "curl -sSL https://raw.githubusercontent.com/stripe/stripe-cli/master/scripts/install.sh | sh"}
	case NPX:
		// npx resolves the latest version on every invocation, so there is nothing
		// to upgrade. Reaching here means a pinned version (`npx @stripe/cli@1.2.3`)
		// or a stale npx cache, neither of which the notice would help with.
		return Advice{Suppress: true}
	case NPMRun:
		// Run through a project's package.json, where the right change depends on
		// that project's dependency range and lockfile. Naming a command risks
		// telling the user to edit the wrong thing, so the notice stands alone.
		return Advice{}
	default:
		return Advice{}
	}
}

//
// Private constants
//

// methodFile is stamped next to the binary by an installer that knows how it
// installed the CLI. scripts/install.sh and scripts/install.ps1 both do, because
// both honor STRIPE_INSTALL_DIR and so install to a location that cannot be
// recognized from here. Any distributor can stamp it to have its channel reported.
const methodFile = ".stripe-install-method"

// maxMethodLength bounds a value read from methodFile before it is reported, so
// an unexpected or hostile file cannot bloat a telemetry request.
const maxMethodLength = 32

//
// Private functions
//

// detectFromExecutable identifies the channels that leave their trace in the
// binary's own location, either as a stamped method or as a recognizable path.
func detectFromExecutable(env Env) string {
	exe, err := env.Executable()
	if err != nil {
		return ""
	}

	// Both the launch path and the resolved path are checked, because each finds
	// installs the other misses.
	//
	// os.Executable does not resolve symlinks on macOS or Windows: it reports the
	// path used to launch the process. Homebrew on an Intel Mac leaves that at
	// /usr/local/bin/stripe and WinGet at ...\WinGet\Links\stripe.exe, both links
	// into the real install directory, so neither is identifiable without
	// resolving. Resolving alone is not enough either -- EvalSymlinks fails on a
	// broken link or an unreadable parent directory, and on Linux os.Executable
	// reads /proc/self/exe, which is already resolved.
	candidates := []string{exe}
	if resolved, err := env.EvalSymlinks(exe); err == nil && resolved != exe {
		candidates = append(candidates, resolved)
	}

	// A stamped method beats every path heuristic, so all candidates are checked
	// for one before any path is inspected.
	for _, path := range candidates {
		if method := methodFromFile(env, filepath.Join(filepath.Dir(path), methodFile)); method != "" {
			return method
		}
	}

	for _, path := range candidates {
		// Lowercased and slash-separated so that one set of checks covers Windows
		// paths and the case-insensitive filesystems macOS and Windows default to.
		//
		// Backslashes are replaced directly rather than with filepath.ToSlash, which
		// is a no-op anywhere but Windows. Doing it unconditionally means the Windows
		// channels below are recognized -- and can be tested -- from any platform.
		normalized := strings.ToLower(strings.ReplaceAll(path, `\`, "/"))

		// "/cellar/" identifies every Homebrew prefix at once (/opt/homebrew,
		// /usr/local, Linuxbrew) once the symlink is resolved. The two prefix
		// checks catch an unresolved link on the prefixes that name themselves.
		if strings.Contains(normalized, "/cellar/") ||
			strings.Contains(normalized, "/homebrew/") ||
			strings.Contains(normalized, "/linuxbrew/") {
			return Homebrew
		}

		// Scoop runs the binary through a shim in scoop/shims, which is not a
		// symlink, so the shim path has to be recognized on its own.
		if strings.Contains(normalized, "/scoop/apps/") ||
			strings.Contains(normalized, "/scoop/shims/") {
			return Scoop
		}

		// Covers both the package directory WinGet extracts to and the Links
		// directory it puts the launcher in.
		if strings.Contains(normalized, "/winget/") {
			return Winget
		}
	}

	return ""
}

// methodFromFile reads a method stamped next to the binary by its installer,
// returning "" when the file is absent or holds anything that is not a plausible
// method name.
//
// Validating the shape rather than membership in the constants above keeps a
// distributor we have never heard of reportable, while keeping arbitrary file
// contents out of telemetry.
func methodFromFile(env Env, path string) string {
	contents, err := env.ReadFile(path)
	if err != nil {
		return ""
	}

	method := strings.TrimSpace(string(contents))
	if method == "" || len(method) > maxMethodLength {
		return ""
	}

	for _, r := range method {
		if (r < 'a' || r > 'z') && (r < '0' || r > '9') && r != '_' && r != '-' {
			return ""
		}
	}

	return method
}
