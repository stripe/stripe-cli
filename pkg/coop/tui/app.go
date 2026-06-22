// Package tui implements the bubbletea-based terminal UI for co-op mode.
package tui

import (
	tea "charm.land/bubbletea/v2"

	"github.com/stripe/stripe-cli/pkg/coop"
)

type Option func(*Model)

func WithSandboxClaimURL(claimURL string) Option {
	return func(m *Model) {
		m.sandboxClaimURL = claimURL
	}
}

// Run launches the fullscreen co-op TUI for a known session.
func Run(store *coop.Store, sessionID string, opts ...Option) error {
	model := NewModel(store, sessionID, opts...)
	p := tea.NewProgram(model)
	_, err := p.Run()
	return err
}

// RunWaiting launches the TUI in "waiting" mode — it polls for a new session
// to appear (ignoring the provided existing IDs) and transitions once found.
func RunWaiting(store *coop.Store, existingIDs map[string]bool, opts ...Option) error {
	model := NewWaitingModel(store, existingIDs, opts...)
	p := tea.NewProgram(model)
	_, err := p.Run()
	return err
}
