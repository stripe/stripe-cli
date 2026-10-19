package agentsetup

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"strings"

	"github.com/stripe/stripe-cli/pkg/errorcategory"
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
	return Scanner{
		LookPath: exec.LookPath,
		ReadFile: os.ReadFile,
		HomeDir:  os.UserHomeDir,
		WorkDir:  os.Getwd,
		ReadDir:  os.ReadDir,
		Stat:     os.Stat,
	}
}

func (s Scanner) withDefaults() Scanner {
	defaults := DefaultScanner()
	if s.LookPath == nil {
		s.LookPath = defaults.LookPath
	}
	if s.ReadFile == nil {
		s.ReadFile = defaults.ReadFile
	}
	if s.HomeDir == nil {
		s.HomeDir = defaults.HomeDir
	}
	if s.WorkDir == nil {
		s.WorkDir = defaults.WorkDir
	}
	if s.ReadDir == nil {
		s.ReadDir = defaults.ReadDir
	}
	if s.Stat == nil {
		s.Stat = defaults.Stat
	}
	return s
}

// RunCommand runs a command silently and returns its standard output. Plugin
// installs run behind a spinner, so streaming subprocess stdio to the terminal
// would interleave with our animation.
func RunCommand(ctx context.Context, name string, args ...string) ([]byte, error) {
	out, err := exec.CommandContext(ctx, name, args...).Output()
	if err == nil {
		return out, nil
	}
	// In cases where the process never started (eg. binary wasn't found), the fallback should be to treat
	// output as the error message.
	stderr := out

	// If a process exits with a non-zero exit code, Output() returns a pointer to a struct that
	// hosts the actual error. Unwrap this pointer in order to get the error message.
	var exitErr *exec.ExitError
	if errors.As(err, &exitErr) {
		stderr = exitErr.Stderr
	}
	if msg := errorFromOutput(stderr); msg != "" {
		return out, errorcategory.New(errorcategory.Internal, msg)
	}

	return out, err
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
