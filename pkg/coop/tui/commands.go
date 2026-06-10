package tui

import (
	"time"

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
		return tickMsg(time.Now())
	}
}

func (m Model) discoverNewSession() tea.Cmd {
	store := m.store
	existingIDs := m.existingIDs
	return func() tea.Msg {
		ids, err := store.List()
		if err != nil {
			return tickMsg(time.Now())
		}
		for _, id := range ids {
			if !existingIDs[id] {
				session, err := store.Read(id)
				if err == nil && session.Status == coop.SessionActive {
					return sessionDiscoveredMsg{sessionID: id}
				}
			}
		}
		return tickMsg(time.Now())
	}
}

func (m *Model) fetchSnippetIfNeeded() tea.Cmd {
	stepIndex, ok := m.selectedStepIndex()
	if m.session == nil || !ok || m.sdkSnippetStep == stepIndex {
		return nil
	}
	node, err := m.session.NodeByNumber(stepIndex + 1)
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
	cursor := stepIndex
	m.sdkLoading = true
	m.sdkLoadingStep = cursor
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
		if m.session.NextSteps == nil {
			m.session.NextSteps = &coop.NextStepsState{}
		}
		m.session.NextSteps.Selected = selected.id
		if err := m.store.Write(m.session); err != nil {
			m.err = err
			return nil
		}
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
		// Mark the step as completed in the parent session
		parent, err := store.Read(parentID)
		if err != nil {
			return sessionDiscoveredMsg{sessionID: parentID}
		}
		if parent.NextSteps == nil {
			parent.NextSteps = &coop.NextStepsState{}
		}
		// Add to completed list (avoid duplicates)
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
		if err := store.Write(parent); err != nil {
			return errMsg{err: err}
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
	return id == "deploy" || id == "deploy-update" || id == "add-integration"
}
