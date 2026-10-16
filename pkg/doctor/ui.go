package doctor

// ui.go — human-facing rendering, following the CLI's house style
// (pkg/cmd/agent_setup_tui.go): ANSI palette colors, Faint for muted text,
// rounded-border containers. In --json mode nothing here is used; JSON goes
// to stdout, logs to stderr.

import (
	"fmt"
	"os"
	"time"

	"charm.land/lipgloss/v2"
	"github.com/briandowns/spinner"
	"golang.org/x/term"
)

var (
	titleStyle  = lipgloss.NewStyle().Bold(true)
	okStyle     = lipgloss.NewStyle().Foreground(lipgloss.Color("2"))
	warnStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("3"))
	failStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("1"))
	accentStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("6"))
	mutedStyle  = lipgloss.NewStyle().Faint(true)
)

func isTTY() bool { return term.IsTerminal(int(os.Stdout.Fd())) }

func okLine(s string) string   { return "  " + okStyle.Render("✓") + " " + s }
func warnLine(s string) string { return "  " + warnStyle.Render("!") + " " + s }
func failLine(s string) string { return "  " + failStyle.Render("✗") + " " + s }
func infoLine(s string) string { return "  " + mutedStyle.Render("·") + " " + s }

// kv renders an aligned key/value detail line.
func kv(k, v string) string {
	return fmt.Sprintf("    %s %s", mutedStyle.Render(fmt.Sprintf("%-30s", k)), v)
}

// withSpinner runs fn behind a spinner when stdout is a TTY; otherwise it
// prints a plain progress line to stderr.
func withSpinner(msg string, fn func() error) error {
	if !isTTY() {
		fmt.Fprintf(os.Stderr, "%s...\n", msg)
		return fn()
	}
	s := spinner.New(spinner.CharSets[14], 100*time.Millisecond, spinner.WithWriter(os.Stderr))
	s.Suffix = " " + msg
	s.Start()
	err := fn()
	s.Stop()
	return err
}
