package tui

import (
	tea "charm.land/bubbletea/v2"

	"github.com/stripe/stripe-cli/pkg/coop"
)

func (m Model) loadSession() tea.Cmd {
	return func() tea.Msg {
		session, err := m.store.Read(m.sessionID)
		if err != nil {
			return errMsg{err: err}
		}
		return sessionUpdatedMsg{session: session}
	}
}

func (m Model) checkForUpdates() tea.Cmd {
	if m.waiting {
		return m.discoverNewSession()
	}
	lastVersion := m.lastVersion
	store := m.store
	sessionID := m.sessionID
	return func() tea.Msg {
		session, err := store.Read(sessionID)
		if err != nil {
			return errMsg{err: err}
		}
		if session.Version != lastVersion {
			return sessionUpdatedMsg{session: session}
		}
		age, err := store.HeartbeatAge(sessionID)
		return noUpdateMsg{heartbeatAge: age, heartbeatOK: err == nil}
	}
}

func (m Model) discoverNewSession() tea.Cmd {
	store := m.store
	existingIDs := m.existingIDs
	return func() tea.Msg {
		if existingIDs == nil {
			return noUpdateMsg{}
		}
		ids, err := store.List()
		if err != nil {
			return noUpdateMsg{}
		}
		for _, id := range ids {
			if !existingIDs[id] {
				session, err := store.Read(id)
				if err == nil && session.Status == coop.SessionActive {
					return sessionDiscoveredMsg{sessionID: id}
				}
			}
		}
		return noUpdateMsg{}
	}
}

func (m Model) snapshotWaitingBaseline() tea.Cmd {
	store := m.store
	return func() tea.Msg {
		ids, err := store.List()
		if err != nil {
			return waitingBaselineMsg{err: err}
		}
		existingIDs := make(map[string]bool, len(ids))
		for _, id := range ids {
			existingIDs[id] = true
		}
		return waitingBaselineMsg{existingIDs: existingIDs}
	}
}

func (m *Model) fetchSnippetIfNeeded() tea.Cmd {
	nodeIndex, ok := m.selectedNodeIndex()
	if m.session == nil || !ok || m.sdkSnippetNode == nodeIndex {
		return nil
	}
	node, err := m.session.NodeByNumber(nodeIndex + 1)
	if err != nil || node.Type != coop.NodeAPIRequest || node.Request == nil {
		return nil
	}
	lang := m.session.Settings["language"]
	if lang == "" {
		lang = "node"
	}
	path := node.Request.Path
	method := node.Request.Method
	params := node.Request.Params
	cursor := nodeIndex
	m.sdkLoading = true
	m.sdkLoadingNode = cursor
	return func() tea.Msg {
		snippet, err := coop.FetchSDKSnippet(path, method, params, lang)
		return sdkSnippetMsg{step: cursor, snippet: snippet, err: err}
	}
}

func (m *Model) selectCompletionOption() tea.Cmd {
	suggestions := m.getCompletionSuggestions()
	if m.cursor >= len(suggestions) {
		return nil
	}
	selected := suggestions[m.cursor]
	if m.session != nil {
		session, err := m.store.Update(m.session.ID, func(session *coop.Session) error {
			if session.NextSteps == nil {
				session.NextSteps = &coop.NextStepsState{}
			}
			session.NextSteps.Selected = selected.id
			return nil
		})
		if err != nil {
			m.err = err
			return nil
		}
		m.session = session
		m.lastVersion = m.session.Version
	}

	if selected.id == "done" {
		return tea.Quit
	}

	return nil
}

func (m Model) returnToParent() tea.Cmd {
	parentID := m.session.ParentSessionID
	stepID := m.session.ParentStepID
	store := m.store

	return func() tea.Msg {
		_, err := store.Update(parentID, func(parent *coop.Session) error {
			if parent.NextSteps == nil {
				parent.NextSteps = &coop.NextStepsState{}
			}
			found := false
			for _, id := range parent.NextSteps.Completed {
				if id == stepID {
					found = true
					break
				}
			}
			if !found {
				parent.NextSteps.Completed = append(parent.NextSteps.Completed, stepID)
			}
			return nil
		})
		if err != nil {
			return sessionDiscoveredMsg{sessionID: parentID}
		}
		return sessionDiscoveredMsg{sessionID: parentID}
	}
}

func (m Model) shouldTransitionToNewSession() bool {
	suggestions := m.getCompletionSuggestions()
	if m.cursor >= len(suggestions) {
		return false
	}
	id := suggestions[m.cursor].id
	return id == "add-integration"
}
