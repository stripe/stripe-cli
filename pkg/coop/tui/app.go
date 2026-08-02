// Package tui implements the bubbletea-based terminal UI for co-op mode.
package tui

import (
	"os"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/colorprofile"

	"github.com/stripe/stripe-cli/pkg/ansi"
	"github.com/stripe/stripe-cli/pkg/coop"
)

type Option func(*Model)

// ReviewDecisionNotifier wakes an agent after a human review decision. The
// notifier must be one-shot and non-blocking from the TUI's perspective; it is
// run as a Bubble Tea command after the session update is durable.
type ReviewDecisionNotifier func(sessionID string) error

func WithSandboxClaimURL(claimURL string) Option {
	return func(m *Model) {
		m.sandboxClaimURL = claimURL
	}
}

func WithReviewDecisionNotifier(notify ReviewDecisionNotifier) Option {
	return func(m *Model) {
		m.reviewDecisionNotifier = notify
	}
}

// ReviewAlertNotifier is told when the set of steps awaiting human review
// becomes non-empty or empty again. focused reports whether the terminal had
// focus at that moment, so callers can ring a bell only when the user is
// looking elsewhere.
type ReviewAlertNotifier func(hasReview, focused bool)

func WithReviewAlertNotifier(notify ReviewAlertNotifier) Option {
	return func(m *Model) {
		m.reviewAlertNotifier = notify
	}
}

// programOptions honors the CLI's own color setting.
//
// Every other command routes color through ansi.ColorsEnabled, which folds in
// --color, CLICOLOR/CLICOLOR_FORCE and TTY detection. The TUI bypassed all of
// it and let lipgloss auto-detect, so `stripe coop join --color off` rendered
// in full color while `stripe customers list --color off` did not.
//
// Ascii drops color while keeping bold and italic, which is what a terminal
// without color support actually shows.
func programOptions() []tea.ProgramOption {
	if ansi.ColorsEnabled(os.Stdout) {
		return nil
	}
	return []tea.ProgramOption{tea.WithColorProfile(colorprofile.Ascii)}
}

// Run launches the fullscreen co-op TUI for a known session.
func Run(store *coop.Store, sessionID string, opts ...Option) error {
	model := NewModel(store, sessionID, opts...)
	p := tea.NewProgram(model, programOptions()...)
	_, err := p.Run()
	return err
}

// RunWaiting launches the TUI in "waiting" mode — it polls for a new session
// to appear (ignoring the provided existing session IDs) and transitions once found.
func RunWaiting(store *coop.Store, existingSessionIDs map[string]bool, opts ...Option) error {
	model := NewWaitingModel(store, existingSessionIDs, opts...)
	p := tea.NewProgram(model, programOptions()...)
	_, err := p.Run()
	return err
}
