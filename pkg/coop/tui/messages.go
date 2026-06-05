package tui

import (
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/stripe/stripe-cli/pkg/coop"
)

// tickMsg triggers a file poll check.
type tickMsg time.Time

// sessionUpdatedMsg carries a freshly read session.
type sessionUpdatedMsg struct {
	session *coop.Session
}

// errMsg wraps file read errors.
type errMsg struct {
	err error
}

// sessionDiscoveredMsg is sent when a new session is found in waiting mode.
type sessionDiscoveredMsg struct {
	sessionID string
}

// sdkSnippetMsg carries a fetched SDK snippet.
type sdkSnippetMsg struct {
	step    int
	snippet string
	err     error
}

func tickCmd() tea.Cmd {
	return tea.Tick(500*time.Millisecond, func(t time.Time) tea.Msg {
		return tickMsg(t)
	})
}
