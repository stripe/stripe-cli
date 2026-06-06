package tui

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/stripe/stripe-cli/pkg/coop"
)

// Model is the root bubbletea model for the co-op TUI.
type Model struct {
	store       *coop.Store
	sessionID   string
	session     *coop.Session
	lastVersion int

	cursor    int
	expanded  bool
	width     int
	height    int
	userMoved bool

	rejecting      bool
	rejectionInput string
	statusMessage  string

	viewport viewport.Model
	ready    bool

	spinner        spinner.Model
	err            error
	sdkSnippet     string
	sdkSnippetStep int
	sdkLoading     bool
	sdkLoadingStep int

	waiting        bool
	waitingMessage string
	existingIDs    map[string]bool
	lastUpdateTime time.Time
}

// NewModel creates a TUI model for a known session.
func NewModel(store *coop.Store, sessionID string) Model {
	s := spinner.New()
	s.Spinner = spinner.MiniDot
	s.Style = lipgloss.NewStyle().Foreground(HuePurple500)

	return Model{
		store:          store,
		sessionID:      sessionID,
		spinner:        s,
		sdkSnippetStep: -1,
		sdkLoadingStep: -1,
	}
}

// NewWaitingModel creates a TUI model that waits for a new session to appear.
func NewWaitingModel(store *coop.Store, existingIDs map[string]bool) Model {
	s := spinner.New()
	s.Spinner = spinner.MiniDot
	s.Style = lipgloss.NewStyle().Foreground(HuePurple500)

	return Model{
		store:          store,
		spinner:        s,
		sdkSnippetStep: -1,
		sdkLoadingStep: -1,
		waiting:        true,
		existingIDs:    existingIDs,
	}
}

func (m Model) Init() tea.Cmd {
	if m.waiting {
		return tea.Batch(m.spinner.Tick, tickCmd())
	}
	return tea.Batch(m.loadSession(), m.spinner.Tick, tickCmd())
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		return m.handleKey(msg)

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		if !m.ready {
			m.viewport = viewport.New(msg.Width, 10)
			m.ready = true
		}
		m.resizeViewport()
		m.syncViewport()
		return m, nil

	case tickMsg:
		return m, m.checkForUpdates()

	case sessionDiscoveredMsg:
		m.waiting = false
		m.waitingMessage = ""
		m.sessionID = msg.sessionID
		m.cursor = 0
		m.expanded = false
		m.userMoved = false
		m.rejecting = false
		m.rejectionInput = ""
		m.statusMessage = ""
		m.sdkSnippet = ""
		m.sdkSnippetStep = -1
		m.sdkLoading = false
		m.sdkLoadingStep = -1
		return m, m.loadSession()

	case sessionUpdatedMsg:
		wasComplete := m.session != nil && m.session.IsComplete()
		m.session = msg.session
		m.lastVersion = msg.session.Version
		m.lastUpdateTime = time.Now()

		// Child session completed → return to parent with step marked done
		if !wasComplete && m.session.IsComplete() && m.session.ParentSessionID != "" {
			return m, m.returnToParent()
		}

		// Reset cursor when transitioning to completion view
		if !wasComplete && m.session.IsComplete() {
			m.cursor = 0
			m.expanded = false
			m.userMoved = false
		}
		m.resizeViewport()
		if !m.userMoved {
			m.autoScroll()
		}
		m.syncViewport()
		return m, tickCmd()

	case errMsg:
		m.err = msg.err
		return m, tickCmd()

	case sdkSnippetMsg:
		if msg.step == m.sdkLoadingStep {
			m.sdkLoading = false
			m.sdkLoadingStep = -1
		}
		if msg.err == nil && msg.step == m.cursor {
			m.sdkSnippet = msg.snippet
			m.sdkSnippetStep = msg.step
		}
		m.syncViewport()
		return m, nil

	case spinner.TickMsg:
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		m.syncViewport()
		cmds = append(cmds, cmd)
		return m, tea.Batch(cmds...)
	}

	return m, nil
}

func (m Model) View() string {
	if m.err != nil {
		return ErrorStyle.Render(fmt.Sprintf("Error: %s", m.err))
	}
	if !m.ready {
		return m.spinner.View() + " Loading..."
	}
	if m.waiting {
		return m.renderWaitingView()
	}
	if m.session == nil {
		return m.renderWaitingView()
	}
	if m.session.IsComplete() {
		return m.renderCompletionView()
	}

	header := m.renderHeader()
	footer := "\n" + m.renderFooter()
	return header + "\n" + m.viewport.View() + "\n" + footer
}

// --- State management ---

func (m *Model) resizeViewport() {
	if !m.ready || m.height == 0 {
		return
	}
	headerH := lipgloss.Height(m.renderHeader()) + 1
	footerH := lipgloss.Height(m.renderFooter()) + 1
	if m.session != nil && m.session.IsComplete() {
		footerH = lipgloss.Height(m.renderCompletionFooter()) + 1
	}
	vpHeight := m.height - headerH - footerH
	if vpHeight < 3 {
		vpHeight = 3
	}
	m.viewport.Width = m.width
	m.viewport.Height = vpHeight
}

func (m *Model) syncViewport() {
	if !m.ready || m.session == nil {
		return
	}
	content := m.renderStepList()
	if m.session.IsComplete() {
		content = m.renderCompletionBody()
	}
	m.viewport.SetContent(content)
	m.scrollToCursor()
}

func (m *Model) scrollToCursor() {
	allContent := m.renderStepList()
	if m.session != nil && m.session.IsComplete() {
		allContent = m.renderCompletionBody()
	}
	allLines := strings.Split(allContent, "\n")
	targetLine := 0
	for i, line := range allLines {
		if strings.Contains(line, "▸") {
			targetLine = i
			break
		}
	}

	vpTop := m.viewport.YOffset
	vpBottom := vpTop + m.viewport.Height

	if targetLine < vpTop {
		m.viewport.SetYOffset(targetLine)
	} else if targetLine >= vpBottom-2 {
		offset := targetLine - m.viewport.Height/2
		if offset < 0 {
			offset = 0
		}
		m.viewport.SetYOffset(offset)
	}
}

func (m *Model) autoScroll() {
	if m.session == nil {
		return
	}
	idx := 0
	for i := range m.session.Chapters {
		for j := range m.session.Chapters[i].Nodes {
			if m.session.Chapters[i].Nodes[j].State == coop.StepReview {
				m.cursor = idx
				m.expanded = true // auto-expand so developer sees what to confirm
				return
			}
			idx++
		}
	}
	_, activeNum := m.session.ActiveNode()
	if activeNum > 0 {
		m.cursor = activeNum - 1
	}
}

func (m Model) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if m.rejecting {
		return m.handleRejectionKey(msg)
	}

	switch msg.String() {
	case "q", "ctrl+c":
		return m, tea.Quit
	case "up", "k":
		m.moveCursorUp()
		m.syncViewport()
		return m, nil
	case "down", "j":
		m.moveCursorDown()
		m.syncViewport()
		return m, nil
	case "e", "?":
		m.expanded = !m.expanded
		m.syncViewport()
		if m.expanded {
			return m, m.fetchSnippetIfNeeded()
		}
		return m, nil
	case "enter":
		return m.handleEnter()
	case "f":
		m.userMoved = false
		m.autoScroll()
		m.syncViewport()
		return m, nil
	case "c":
		m.handleConfirm()
		return m, nil
	case "o":
		if m.session != nil && m.session.ClaimURL != "" {
			openBrowser(m.session.ClaimURL)
		}
		return m, nil
	case "r":
		m.startReject()
		return m, nil
	}
	return m, nil
}

func (m *Model) moveCursorUp() {
	if m.session != nil && m.session.IsComplete() {
		suggestions := m.getCompletionSuggestions()
		if len(suggestions) == 0 {
			return
		}
		if m.cursor > 0 {
			m.cursor--
		} else {
			m.cursor = len(suggestions) - 1
		}
	} else if m.cursor > 0 {
		m.cursor--
		m.userMoved = true
	}
}

func (m *Model) moveCursorDown() {
	if m.session != nil && m.session.IsComplete() {
		suggestions := m.getCompletionSuggestions()
		if len(suggestions) == 0 {
			return
		}
		if m.cursor < len(suggestions)-1 {
			m.cursor++
		} else {
			m.cursor = 0
		}
	} else if m.session != nil && m.cursor < m.session.TotalSteps()-1 {
		m.cursor++
		m.userMoved = true
	}
}

func (m Model) handleEnter() (tea.Model, tea.Cmd) {
	if m.session != nil && m.session.IsComplete() {
		suggestions := m.getCompletionSuggestions()
		if m.cursor < len(suggestions) {
			selected := suggestions[m.cursor]
			cmd := m.selectCompletionOption()

			switch selected.id {
			case "deploy", "deploy-update":
				m.enterWaitingMode("Waiting for agent to start the deploy session...")
				return m, cmd
			case "add-integration":
				m.enterWaitingMode("Waiting for agent to ask which Stripe feature to add...")
				return m, cmd
			default:
				if selected.id == "summarize" {
					m.statusMessage = "Waiting for agent to write STRIPE.md..."
					m.syncViewport()
				}
				return m, cmd
			}
		}
		return m, nil
	}
	m.expanded = !m.expanded
	m.syncViewport()
	if m.expanded {
		return m, m.fetchSnippetIfNeeded()
	}
	return m, nil
}

func (m *Model) enterWaitingMode(message string) {
	m.waiting = true
	m.waitingMessage = message
	m.session = nil
	m.cursor = 0
	m.expanded = false
	m.userMoved = false
	m.rejecting = false
	m.rejectionInput = ""
	m.existingIDs = make(map[string]bool)
	if ids, err := m.store.List(); err == nil {
		for _, id := range ids {
			m.existingIDs[id] = true
		}
	}
}

func (m *Model) handleConfirm() {
	if m.session == nil {
		return
	}
	node, _ := m.session.NodeByNumber(m.cursor + 1)
	if node != nil && node.State == coop.StepReview {
		m.session.TransitionStep(m.cursor+1, coop.StepDone)
		if err := m.store.Write(m.session); err != nil {
			m.err = fmt.Errorf("failed to save confirmation: %w", err)
		}
		m.lastVersion = m.session.Version
		m.statusMessage = "Waiting for agent to continue..."
		if m.session.IsComplete() {
			m.cursor = 0
			m.expanded = false
		}
		m.resizeViewport()
		m.syncViewport()
	}
}

func (m *Model) startReject() {
	if m.session == nil {
		return
	}
	node, _ := m.session.NodeByNumber(m.cursor + 1)
	if node != nil && node.State == coop.StepReview {
		m.rejecting = true
		m.rejectionInput = ""
		m.expanded = true
		m.statusMessage = ""
		m.syncViewport()
	}
}

func (m Model) handleRejectionKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc", "ctrl+c":
		m.rejecting = false
		m.rejectionInput = ""
		m.statusMessage = "Rejection canceled."
		m.syncViewport()
		return m, nil
	case "enter":
		m.handleReject(strings.TrimSpace(m.rejectionInput))
		return m, nil
	case "backspace", "ctrl+h":
		if len(m.rejectionInput) > 0 {
			runes := []rune(m.rejectionInput)
			m.rejectionInput = string(runes[:len(runes)-1])
		}
		m.syncViewport()
		return m, nil
	}
	if msg.Type == tea.KeyRunes {
		m.rejectionInput += string(msg.Runes)
		m.syncViewport()
	}
	return m, nil
}

func (m *Model) handleReject(note string) {
	if m.session == nil {
		return
	}
	node, _ := m.session.NodeByNumber(m.cursor + 1)
	if node != nil && node.State == coop.StepReview {
		m.session.TransitionStep(m.cursor+1, coop.StepActive)
		if note == "" {
			note = "Rejected by developer. Please redo this step."
		}
		node.RejectionNote = note
		node.Implementation = nil
		node.Verifications = nil
		if err := m.store.Write(m.session); err != nil {
			m.err = fmt.Errorf("failed to save rejection: %w", err)
		}
		m.lastVersion = m.session.Version
		m.rejecting = false
		m.rejectionInput = ""
		m.statusMessage = "Waiting for agent to address your feedback..."
		m.syncViewport()
	}
}
