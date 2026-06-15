package tui

import (
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/stripe/stripe-cli/pkg/coop"
)

// tickMsg triggers a file poll check.
type tickMsg time.Time

// noUpdateMsg means a scheduled poll completed without finding a newer session.
type noUpdateMsg struct {
	heartbeatAge time.Duration
	heartbeatOK  bool
}

// sessionUpdatedMsg carries a freshly read session.
type sessionUpdatedMsg struct {
	session *coop.Session
}

// errMsg wraps file read errors.
type errMsg struct {
	err error
}

type waitingBaselineMsg struct {
	existingIDs map[string]bool
	err         error
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

type mouseAction int

const (
	mouseActionNone mouseAction = iota
	mouseActionSelectNode
	mouseActionSelectStep
	mouseActionSelectCompletion
	mouseActionOpenClaim
)

type mouseActionMsg struct {
	action mouseAction
	index  int
}

func tickCmd() tea.Cmd {
	return tea.Tick(500*time.Millisecond, func(t time.Time) tea.Msg {
		return tickMsg(t)
	})
}
