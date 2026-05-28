package pager

import (
	"errors"
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
		os.Setenv("LESS", "FRX")
	}

	parts := strings.Fields(pagerCmd)
	cmd := exec.Command(parts[0], parts[1:]...)
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
		return n, err
	}
	return p.w.Write(b)
}

func (p *Pager) Close() error {
	if p.pw == nil {
		return nil
	}
	p.pw.Close()
	err := p.cmd.Wait()
	if isBrokenPipe(err) {
		return nil
	}
	return err
}

func isBrokenPipe(err error) bool {
	return errors.Is(err, syscall.EPIPE)
}
