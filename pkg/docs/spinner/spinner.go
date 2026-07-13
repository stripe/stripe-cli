// Package spinner provides a simple terminal spinner backed by Bubble Tea.
package spinner

import (
	"fmt"
	"io"
	"os"

	bspinner "charm.land/bubbles/v2/spinner"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"golang.org/x/term"
)

// Spinner runs a spinner animation while a function executes, then prints a
// final message when done. Use New to create one, configure with With* methods,
// and call Run to execute.
type Spinner struct {
	label    string
	style    lipgloss.Style
	finalMsg string
	out      io.Writer
	disabled bool
}

// New returns a Spinner with sensible defaults writing to stderr.
func New() *Spinner {
	return &Spinner{
		style: lipgloss.NewStyle().Foreground(lipgloss.Color("#9289fe")),
		out:   os.Stderr,
	}
}

// WithLabel sets the text shown next to the spinner frame.
func (s *Spinner) WithLabel(label string) *Spinner {
	s.label = label
	return s
}

// WithFinalMsg sets the line printed after fn completes.
func (s *Spinner) WithFinalMsg(msg string) *Spinner {
	s.finalMsg = msg
	return s
}

// WithStyle sets the lipgloss style applied to the spinner frame.
func (s *Spinner) WithStyle(style lipgloss.Style) *Spinner {
	s.style = style
	return s
}

// WithOutput sets the writer the spinner renders to (default: os.Stderr).
func (s *Spinner) WithOutput(w io.Writer) *Spinner {
	s.out = w
	return s
}

// WithDisabled disables the spinner entirely when v is true. fn still runs,
// but no animation or final message is shown.
func (s *Spinner) WithDisabled(v bool) *Spinner {
	s.disabled = v
	return s
}

// Run executes fn, showing the spinner while it runs. The spinner is skipped
// when disabled or when the output is not a TTY. Returns any error from fn.
func (s *Spinner) Run(fn func() error) error {
	f, ok := s.out.(*os.File)
	if s.disabled || !ok || !term.IsTerminal(int(f.Fd())) {
		return fn()
	}

	m := spinnerModel{
		spinner: bspinner.New(
			bspinner.WithSpinner(bspinner.Dot),
			bspinner.WithStyle(s.style),
		),
		label: s.label,
		fn:    fn,
	}
	p := tea.NewProgram(m, tea.WithOutput(s.out), tea.WithInput(os.Stdin))
	final, err := p.Run()
	if err != nil {
		return fmt.Errorf("spinner: %w", err)
	}
	if result, ok := final.(spinnerModel); ok && result.err != nil {
		return result.err
	}
	if s.finalMsg != "" {
		if _, err := fmt.Fprintln(s.out, s.finalMsg); err != nil {
			return fmt.Errorf("spinner: writing final message: %w", err)
		}
	}
	return nil
}

type spinnerModel struct {
	spinner bspinner.Model
	label   string
	fn      func() error
	err     error
	done    bool
}

type doneMsg struct{ err error }

func (m spinnerModel) Init() tea.Cmd {
	return tea.Batch(m.spinner.Tick, func() tea.Msg {
		return doneMsg{err: m.fn()}
	})
}

func (m spinnerModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case doneMsg:
		m.err = msg.err
		m.done = true
		return m, tea.Quit
	case bspinner.TickMsg:
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd
	}
	return m, nil
}

func (m spinnerModel) View() tea.View {
	if m.done {
		return tea.NewView("")
	}
	return tea.NewView(m.spinner.View() + " " + m.label)
}
