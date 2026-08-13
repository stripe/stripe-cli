package tui

import (
	"os"
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/stripe/stripe-cli/pkg/coop"
)

// existingSessionAdoptDelay is how long waiting mode holds out for a brand-new
// session before considering an already-active one. Long enough for `coop
// start`'s agent to create its session first; short enough that a re-run
// `coop join --wait` doesn't feel hung.
const existingSessionAdoptDelay = 10 * time.Second

// existingSessionLivenessWindow bounds how stale a pre-existing session may be
// and still be adopted. An agent writes to its session file constantly, so a
// session no one has touched recently belongs to a crashed run — and crashed
// runs stay `active` forever because nothing reaps them. Adopting one would
// silently show the wrong work.
const existingSessionLivenessWindow = 2 * time.Minute

// adoptableExistingSession returns a pre-existing active session that plainly
// belongs to this launch: same working directory, and updated recently enough
// that its agent is still alive. Returns nil when nothing qualifies, which
// keeps waiting mode waiting.
func adoptableExistingSession(store *coop.Store, cwd string, now time.Time) *coop.Session {
	if cwd == "" {
		return nil
	}
	ids, err := store.List()
	if err != nil {
		return nil
	}
	var best *coop.Session
	for _, id := range ids {
		session, err := store.Read(id)
		if err != nil || session.Status != coop.SessionActive {
			continue
		}
		if session.Cwd != cwd {
			continue
		}
		if now.Sub(session.UpdatedAt) > existingSessionLivenessWindow {
			continue
		}
		if best == nil || session.UpdatedAt.After(best.UpdatedAt) {
			best = session
		}
	}
	return best
}

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
	existingSessionIDs := m.existingSessionIDs
	adoptExisting := !m.waitingSince.IsZero() && time.Since(m.waitingSince) >= existingSessionAdoptDelay
	cwd, err := os.Getwd()
	if err != nil {
		cwd = ""
	}
	return func() tea.Msg {
		if existingSessionIDs == nil {
			return noUpdateMsg{}
		}
		ids, err := store.List()
		if err != nil {
			return noUpdateMsg{}
		}
		for _, id := range ids {
			if existingSessionIDs[id] {
				continue
			}
			session, err := store.Read(id)
			if err != nil || session.Status != coop.SessionActive {
				continue
			}
			// Skip sessions that announce a different project. Store.List is
			// ReadDir order over random IDs, so with two concurrent launches
			// this loop would otherwise latch an arbitrary one. Sessions
			// written before Cwd existed carry none; accept those.
			if cwd != "" && session.Cwd != "" && session.Cwd != cwd {
				continue
			}
			return sessionDiscoveredMsg{sessionID: id}
		}
		// No new session appeared. `coop join --wait` is the exact command the
		// fallback prints, so on a re-run after a quit the session it
		// described is already in the baseline and would never be latched.
		// Adopt it — but only when it plainly belongs to this launch, so a
		// stale session from another project (or a crashed earlier run) can
		// never be picked up instead of the one the agent is creating.
		if adoptExisting {
			if session := adoptableExistingSession(store, cwd, time.Now()); session != nil {
				return sessionDiscoveredMsg{sessionID: session.ID}
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
		existingSessionIDs := make(map[string]bool, len(ids))
		for _, id := range ids {
			existingSessionIDs[id] = true
		}
		return waitingBaselineMsg{existingSessionIDs: existingSessionIDs}
	}
}

func (m *Model) fetchSnippetIfNeeded() tea.Cmd {
	nodeIndex, ok := m.selectedNodeIndex()
	if m.session == nil || !ok || m.sdkSnippetNode == nodeIndex {
		return nil
	}
	node, err := m.session.NodeByNumber(nodeIndex + 1)
	if err != nil || node.NodeType != coop.NodeAPIRequest || node.Request() == nil {
		return nil
	}
	lang := m.session.Settings["language"]
	if lang == "" {
		lang = "node"
	}
	request := node.Request()
	path := request.Path
	method := request.Method
	params := request.Params
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
	if m.selectionCursor >= len(suggestions) {
		return nil
	}
	selected := suggestions[m.selectionCursor]
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
	// Follow-up sessions keep immediate parentage. For A -> B -> C, completing C
	// returns to B; B can then surface its own parent relationship if needed.
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
	if m.selectionCursor >= len(suggestions) {
		return false
	}
	id := suggestions[m.selectionCursor].id
	return id == "deploy" || id == "deploy-update" || id == "add-integration"
}
