package tui

import (
	"time"

	tea "github.com/charmbracelet/bubbletea"

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

func (m Model) fetchSnippetIfNeeded() tea.Cmd {
	if m.session == nil || m.sdkSnippetStep == m.cursor {
		return nil
	}
	node, err := m.session.NodeByNumber(m.cursor + 1)
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
	cursor := m.cursor
	return func() tea.Msg {
		snippet, err := coop.FetchSDKSnippet(path, method, params, lang)
		return sdkSnippetMsg{step: cursor, snippet: snippet, err: err}
	}
}

func (m Model) selectCompletionOption() tea.Cmd {
	suggestions := m.getCompletionSuggestions()
	if m.cursor >= len(suggestions) {
		return nil
	}
	selected := suggestions[m.cursor]
	if selected.id == "done" {
		return tea.Quit
	}
	if m.session != nil {
		if m.session.NextSteps == nil {
			m.session.NextSteps = &coop.NextStepsState{}
		}
		m.session.NextSteps.Selected = selected.id
		m.store.Write(m.session)
		m.lastVersion = m.session.Version
	}

	// Deploy: create the session directly (only one deploy blueprint)
	if selected.id == "deploy" || selected.id == "deploy-update" {
		bp, err := coop.LoadBlueprint("deploy-stripe-projects")
		if err == nil {
			lang := ""
			if m.session != nil {
				lang = m.session.Settings["language"]
			}
			settings := map[string]string{}
			if lang != "" {
				settings["language"] = lang
			}
			newSession := coop.NewSessionFromBlueprint(bp, "coop_deploy", settings)
			newSession.ParentSessionID = m.session.ID
			newSession.ParentStepID = selected.id
			m.store.Write(newSession)
		}
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
		store.Write(parent)

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
