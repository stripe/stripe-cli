package agentsetup

import (
	"context"
	"os"
	"strings"
)

// LookPathFunc matches exec.LookPath and exists to make detection testable.
type LookPathFunc func(string) (string, error)

// ReadFileFunc matches os.ReadFile and exists to make status parsing testable.
type ReadFileFunc func(string) ([]byte, error)

// HomeDirFunc matches os.UserHomeDir and exists to make status parsing testable.
type HomeDirFunc func() (string, error)

// WorkDirFunc matches os.Getwd and exists to make local plugin scope testable.
type WorkDirFunc func() (string, error)

// ReadDirFunc matches os.ReadDir and exists to make directory listing testable.
type ReadDirFunc func(string) ([]os.DirEntry, error)

// StatFunc matches os.Stat and exists to make file existence checks testable.
type StatFunc func(string) (os.FileInfo, error)

// RunCommandFunc runs a command and returns its standard output and a concise error
// upon failure
type RunCommandFunc func(context.Context, string, ...string) ([]byte, error)

// Scanner scans local agent installations without mutating them.
type Scanner struct {
	LookPath LookPathFunc
	ReadFile ReadFileFunc
	HomeDir  HomeDirFunc
	WorkDir  WorkDirFunc
	ReadDir  ReadDirFunc
	Stat     StatFunc
}

// DefaultScanner returns a Scanner backed by the real OS.
func DefaultScanner() Scanner {
	return Scanner{}.withDefaults()
}

func (s Scanner) withDefaults() Scanner {
	if s.LookPath == nil {
		s.LookPath, _ = lazyExecutableResolver()
	}
	if s.ReadFile == nil {
		s.ReadFile = os.ReadFile
	}
	if s.HomeDir == nil {
		s.HomeDir = os.UserHomeDir
	}
	if s.WorkDir == nil {
		s.WorkDir = os.Getwd
	}
	if s.ReadDir == nil {
		s.ReadDir = os.ReadDir
	}
	if s.Stat == nil {
		s.Stat = os.Stat
	}
	return s
}

// RunCommand runs a command silently and returns its standard output. Plugin
// installs run behind a spinner, so streaming subprocess stdio to the terminal
// would interleave with our animation.
func RunCommand(ctx context.Context, name string, args ...string) ([]byte, error) {
	return newExecutableResolver().run(ctx, name, args...)
}

func errorFromOutput(out []byte) string {
	text := strings.ReplaceAll(string(out), "\r", "\n")
	var last string
	for _, line := range strings.Split(text, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		last = line
		lower := strings.ToLower(line)
		if strings.Contains(lower, "failed") || strings.Contains(lower, "error") {
			return strings.TrimLeft(line, "✘✗× \t")
		}
	}
	return last
}
