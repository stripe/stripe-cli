package pager

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
	"syscall"

	"golang.org/x/term"
)

// Pager wraps a writer, optionally piping output through a terminal pager.
type Pager struct {
	w   io.Writer
	pw  io.WriteCloser
	cmd *exec.Cmd
}

// New returns a Pager that pipes through a terminal pager when w is a TTY and
// enabled is true. The caller must call Close when done writing.
func New(w io.Writer, enabled bool) *Pager {
	p := &Pager{w: w}

	if !enabled {
		return p
	}

	f, ok := w.(*os.File)
	if !ok || !term.IsTerminal(int(f.Fd())) {
		return p
	}

	pagerCmd := os.Getenv("PAGER")
	if pagerCmd == "" {
		pagerCmd = defaultPager()
	}

	// F = quit if content fits one screen, R = raw ANSI escapes, X = don't clear screen on exit.
	if os.Getenv("LESS") == "" {
		_ = os.Setenv("LESS", "FRX")
	}

	parts := strings.Fields(pagerCmd)
	bin, err := exec.LookPath(parts[0])
	if err != nil {
		return p
	}
	// G204: We intentionally execute the user's $PAGER. We validate the binary
	// exists via LookPath but gosec's taint analysis still flags it. An allowlist
	// would break users with custom pagers (bat, delta, most, etc.).
	cmd := exec.CommandContext(context.Background(), bin, parts[1:]...) //nolint:gosec
	cmd.Stdout = f
	cmd.Stderr = os.Stderr

	pw, err := cmd.StdinPipe()
	if err != nil {
		return p
	}

	if err := cmd.Start(); err != nil {
		return p
	}

	p.pw = pw
	p.cmd = cmd
	return p
}

func defaultPager() string {
	if _, err := exec.LookPath("less"); err == nil {
		return "less"
	}
	return "more"
}

func (p *Pager) Write(b []byte) (int, error) {
	if p.pw != nil {
		n, err := p.pw.Write(b)
		if isBrokenPipe(err) {
			return n, nil
		}
		if err != nil {
			return n, fmt.Errorf("writing to pager stdin: %w", err)
		}
		return n, nil
	}
	n, err := p.w.Write(b)
	if err != nil {
		return n, fmt.Errorf("writing output: %w", err)
	}
	return n, nil
}

// Close closes the pager's stdin pipe and waits for the pager process to exit.
func (p *Pager) Close() error {
	if p.pw == nil {
		return nil
	}
	_ = p.pw.Close()
	err := p.cmd.Wait()
	if isBrokenPipe(err) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("waiting for pager: %w", err)
	}
	return nil
}

func isBrokenPipe(err error) bool {
	return errors.Is(err, syscall.EPIPE)
}
