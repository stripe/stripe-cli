package tui

import (
	"fmt"
	"strings"
	"time"

	"charm.land/bubbles/v2/help"
	"charm.land/bubbles/v2/key"
	"charm.land/bubbles/v2/spinner"
	"charm.land/bubbles/v2/textinput"
	"charm.land/bubbles/v2/viewport"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"

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
	detailTab int
	width     int
	height    int
	userMoved bool

	rejecting       bool
	rejectionInput  textinput.Model
	rejectionError  string
	statusMessage   string
	statusExpiresAt time.Time

	keys keyMap
	help help.Model

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

	isDark  bool
	focused bool // true when terminal has focus (default: true, updated via FocusMsg/BlurMsg)
}

func newRejectionInput() textinput.Model {
	ti := textinput.New()
	ti.Prompt = ""
	ti.Placeholder = "Describe what to change..."
	ti.CharLimit = 500
	ti.SetVirtualCursor(false)
	styles := ti.Styles()
	styles.Focused.Placeholder = lipgloss.NewStyle().Foreground(HueGray500).Italic(true)
	styles.Focused.Text = lipgloss.NewStyle().Foreground(lipgloss.Color("#ffffff"))
	ti.SetStyles(styles)
	return ti
}

// NewModel creates a TUI model for a known session.
func NewModel(store *coop.Store, sessionID string) Model {
	s := spinner.New(
		spinner.WithSpinner(spinner.MiniDot),
		spinner.WithStyle(lipgloss.NewStyle().Foreground(HuePurple500)),
	)

	return Model{
		store:          store,
		sessionID:      sessionID,
		spinner:        s,
		rejectionInput: newRejectionInput(),
		keys:           newKeyMap(),
		help:           help.New(),
		isDark:         true,
		focused:        true,
		sdkSnippetStep: -1,
		sdkLoadingStep: -1,
	}
}

// NewWaitingModel creates a TUI model that waits for a new session to appear.
func NewWaitingModel(store *coop.Store, existingIDs map[string]bool) Model {
	s := spinner.New(
		spinner.WithSpinner(spinner.MiniDot),
		spinner.WithStyle(lipgloss.NewStyle().Foreground(HuePurple500)),
	)

	return Model{
		store:          store,
		spinner:        s,
		rejectionInput: newRejectionInput(),
		keys:           newKeyMap(),
		help:           help.New(),
		isDark:         true,
		focused:        true,
		sdkSnippetStep: -1,
		sdkLoadingStep: -1,
		waiting:        true,
		existingIDs:    existingIDs,
	}
}

func (m Model) Init() tea.Cmd {
	if m.waiting {
		return tea.Batch(m.spinner.Tick, tickCmd(), tea.RequestBackgroundColor)
	}
	return tea.Batch(m.loadSession(), m.spinner.Tick, tickCmd(), tea.RequestBackgroundColor)
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		return m.handleKey(msg)

	case tea.MouseWheelMsg:
		m.userMoved = true
		var cmd tea.Cmd
		m.viewport, cmd = m.viewport.Update(msg)
		return m, cmd

	case mouseActionMsg:
		return m.handleMouseAction(msg)

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		if !m.ready {
			m.viewport = viewport.New(viewport.WithWidth(msg.Width), viewport.WithHeight(10))
			m.viewport.MouseWheelEnabled = true
			m.viewport.MouseWheelDelta = 3
			m.viewport.FillHeight = true
			m.viewport.SoftWrap = true
			m.ready = true
		}
		m.resizeViewport()
		m.syncViewport()
		return m, nil

	case tickMsg:
		m.clearExpiredStatus(time.Now())
		if !m.focused {
			return m, tickCmd()
		}
		return m, m.checkForUpdates()

	case sessionDiscoveredMsg:
		m.waiting = false
		m.waitingMessage = ""
		m.sessionID = msg.sessionID
		m.cursor = 0
		m.expanded = false
		m.userMoved = false
		m.rejecting = false
		m.rejectionInput.SetValue("")
		m.rejectionError = ""
		m.statusMessage = ""
		m.statusExpiresAt = time.Time{}
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
			m.statusMessage = ""
			m.statusExpiresAt = time.Time{}
			m.rejecting = false
			m.rejectionInput.SetValue("")
			m.rejectionError = ""
			if m.ready {
				m.viewport.SetYOffset(0)
			}
		}
		if !m.userMoved {
			m.autoScroll()
		}
		m.resizeViewport()
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

	case tea.BackgroundColorMsg:
		m.isDark = msg.IsDark()
		return m, nil

	case tea.FocusMsg:
		m.focused = true
		return m, nil

	case tea.BlurMsg:
		m.focused = false
		return m, nil
	}

	return m, nil
}

func (m Model) View() tea.View {
	var content string
	if m.err != nil {
		content = ErrorStyle.Render(fmt.Sprintf("Error: %s", m.err))
	} else if !m.ready {
		content = m.spinner.View() + " Loading..."
	} else if m.waiting {
		content = m.renderWaitingView()
	} else if m.session == nil {
		content = m.renderWaitingView()
	} else if m.session.IsComplete() {
		content = m.renderCompletionView()
	} else {
		header := m.renderHeader()
		content = m.renderPinnedViewport(header, m.renderFooter())
	}

	v := tea.NewView(content)
	v.AltScreen = true
	v.MouseMode = tea.MouseModeCellMotion
	v.ReportFocus = true
	v.KeyboardEnhancements.ReportEventTypes = true
	v.ProgressBar = m.progressBar()
	v.Cursor = m.rejectionCursor(content)
	v.OnMouse = m.mouseHandler()
	if m.session != nil {
		done := 0
		for _, ch := range m.session.Chapters {
			for _, n := range ch.Nodes {
				if n.State == coop.StepDone || n.State == coop.StepSkipped {
					done++
				}
			}
		}
		v.WindowTitle = fmt.Sprintf("Co-op: %s (%d/%d)", m.session.Blueprint, done, m.session.TotalSteps())
	} else {
		v.WindowTitle = "Stripe Co-op"
	}
	return v
}

func (m Model) progressBar() *tea.ProgressBar {
	if m.err != nil {
		return tea.NewProgressBar(tea.ProgressBarError, 100)
	}
	if m.waiting || m.session == nil {
		return tea.NewProgressBar(tea.ProgressBarIndeterminate, 0)
	}
	total := 0
	done := 0
	for _, ch := range m.session.Chapters {
		for _, n := range ch.Nodes {
			if n.State == coop.StepSkipped {
				continue
			}
			total++
			if n.State == coop.StepDone {
				done++
			}
		}
	}
	if total == 0 {
		return tea.NewProgressBar(tea.ProgressBarNone, 0)
	}
	value := done * 100 / total
	state := tea.ProgressBarDefault
	if m.agentIdle() {
		state = tea.ProgressBarWarning
	}
	return tea.NewProgressBar(state, value)
}

func (m Model) rejectionCursor(content string) *tea.Cursor {
	if !m.rejecting {
		return nil
	}
	lines := strings.Split(content, "\n")
	for y, line := range lines {
		plain := ansi.Strip(line)
		const prefix = "Request changes: "
		idx := strings.Index(plain, prefix)
		if idx < 0 {
			continue
		}
		x := lipgloss.Width(plain[:idx+len(prefix)])
		x += lipgloss.Width(m.rejectionInput.Value())
		cursor := tea.NewCursor(x, y)
		cursor.Shape = tea.CursorBar
		cursor.Color = HuePurple500
		cursor.Blink = true
		return cursor
	}
	return nil
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
	vpHeight := m.height - headerH - footerH - terminalScrollGuard
	if vpHeight < minViewportHeight {
		vpHeight = minViewportHeight
	}
	m.viewport.SetWidth(m.width)
	m.viewport.SetHeight(vpHeight)
}

func (m *Model) syncViewport() {
	if !m.ready || m.session == nil {
		return
	}
	content := m.renderStepList()
	if m.session.IsComplete() {
		content = m.renderCompletionBody()
		m.viewport.SetContent(content)
		m.viewport.SetYOffset(0)
		return
	}
	m.viewport.SetContent(content)
	m.scrollToCursor()
}

func (m *Model) scrollToCursor() {
	targetLine := m.selectedContentLine()

	vpTop := m.viewport.YOffset()
	vpBottom := vpTop + m.viewport.Height()
	scrollThreshold := vpBottom - 2
	if m.session != nil && m.session.IsComplete() {
		scrollThreshold = vpBottom
	}

	if targetLine < vpTop {
		m.viewport.SetYOffset(targetLine)
	} else if targetLine >= scrollThreshold {
		offset := targetLine - m.viewport.Height()/2
		if offset < 0 {
			offset = 0
		}
		m.viewport.SetYOffset(offset)
	}
}

func (m Model) selectedContentLine() int {
	if m.session != nil && !m.session.IsComplete() {
		selectedLine := -1
		for line, step := range m.stepContentLines() {
			if step == m.cursor && (selectedLine == -1 || line < selectedLine) {
				selectedLine = line
			}
		}
		if selectedLine >= 0 {
			return selectedLine
		}
	}
	content := m.renderStepList()
	if m.session != nil && m.session.IsComplete() {
		content = m.renderCompletionBody()
	}
	lines := strings.Split(content, "\n")
	for i, line := range lines {
		if strings.Contains(line, "▸") {
			return i
		}
	}
	return 0
}

func (m *Model) autoScroll() {
	if m.session == nil {
		return
	}
	idx := 0
	for i := range m.session.Chapters {
		for j := range m.session.Chapters[i].Nodes {
			if m.session.Chapters[i].Nodes[j].State == coop.StepReview && m.reviewIsActionable(idx+1) {
				m.cursor = idx
				m.expanded = false
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

func (m Model) handleKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	if m.rejecting {
		return m.handleRejectionKey(msg)
	}
	if msg.IsRepeat && (key.Matches(msg, m.keys.Confirm) || key.Matches(msg, m.keys.Reject) || key.Matches(msg, m.keys.Copy) || key.Matches(msg, m.keys.OpenClaim)) {
		return m, nil
	}

	switch {
	case key.Matches(msg, m.keys.Quit):
		return m, tea.Quit
	case key.Matches(msg, m.keys.Up):
		m.moveCursorUp()
		m.resizeViewport()
		m.syncViewport()
		return m, nil
	case key.Matches(msg, m.keys.Down):
		m.moveCursorDown()
		m.resizeViewport()
		m.syncViewport()
		return m, nil
	case key.Matches(msg, m.keys.PageUp):
		m.userMoved = true
		m.viewport.PageUp()
		return m, nil
	case key.Matches(msg, m.keys.PageDown):
		m.userMoved = true
		m.viewport.PageDown()
		return m, nil
	case key.Matches(msg, m.keys.Top):
		m.userMoved = true
		m.viewport.GotoTop()
		return m, nil
	case key.Matches(msg, m.keys.Bottom):
		m.userMoved = true
		m.viewport.GotoBottom()
		return m, nil
	case key.Matches(msg, m.keys.Expand):
		m.expanded = !m.expanded
		m.resizeViewport()
		m.syncViewport()
		if m.expanded {
			return m, m.fetchSnippetIfNeeded()
		}
		return m, nil
	case key.Matches(msg, m.keys.Enter):
		return m.handleEnter()
	case key.Matches(msg, m.keys.Tab):
		if m.expanded {
			m.detailTab = (m.detailTab + 1) % len(detailSections)
			m.syncViewport()
			return m, m.fetchSnippetIfNeeded()
		}
		return m, nil
	case key.Matches(msg, m.keys.Escape):
		if m.expanded {
			m.expanded = false
			m.resizeViewport()
			m.syncViewport()
		}
		return m, nil
	case key.Matches(msg, m.keys.Follow):
		m.userMoved = false
		m.autoScroll()
		m.setStatus("Following the current review step", 3*time.Second)
		m.resizeViewport()
		m.syncViewport()
		return m, nil
	case key.Matches(msg, m.keys.Confirm):
		m.handleConfirm()
		return m, nil
	case key.Matches(msg, m.keys.OpenClaim):
		if m.session != nil && m.session.ClaimURL != "" {
			openBrowser(m.session.ClaimURL)
		}
		return m, nil
	case key.Matches(msg, m.keys.Copy):
		if command := m.selectedReviewCommand(); command != "" {
			m.setStatus("Copied review command.", 3*time.Second)
			m.resizeViewport()
			m.syncViewport()
			return m, tea.SetClipboard(command)
		}
		return m, nil
	case key.Matches(msg, m.keys.Reject):
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
	m.resizeViewport()
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
	m.rejectionInput.SetValue("")
	m.rejectionError = ""
	m.statusMessage = ""
	m.statusExpiresAt = time.Time{}
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
	target, ok := m.selectedReviewTarget()
	if !ok {
		return
	}
	for _, step := range target.steps {
		if err := m.session.TransitionStep(step, coop.StepDone); err != nil {
			m.err = fmt.Errorf("failed to confirm review: %w", err)
			return
		}
	}
	if err := m.store.Write(m.session); err != nil {
		m.err = fmt.Errorf("failed to save confirmation: %w", err)
	}
	m.lastVersion = m.session.Version
	m.setStatus("Confirmed. Waiting for agent...", 5*time.Second)
	m.rejecting = false
	m.rejectionInput.SetValue("")
	m.rejectionError = ""
	if m.session.IsComplete() {
		m.cursor = 0
		m.expanded = false
		m.statusMessage = ""
		m.statusExpiresAt = time.Time{}
	}
	m.resizeViewport()
	m.syncViewport()
}

func (m *Model) startReject() {
	if m.session == nil {
		return
	}
	if target, ok := m.selectedReviewTarget(); ok {
		m.rejecting = true
		m.rejectionInput.SetValue("")
		m.rejectionInput.Placeholder = m.requestChangesPlaceholder(target)
		m.rejectionInput.Focus()
		m.rejectionError = ""
		m.statusMessage = ""
		m.statusExpiresAt = time.Time{}
		m.resizeViewport()
		m.syncViewport()
	}
}

func (m Model) handleRejectionKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch {
	case key.Matches(msg, m.keys.Escape):
		m.rejecting = false
		m.rejectionInput.SetValue("")
		m.rejectionInput.Blur()
		m.rejectionError = ""
		m.setStatus("Request changes canceled.", 3*time.Second)
		m.resizeViewport()
		m.syncViewport()
		return m, nil
	case key.Matches(msg, m.keys.Enter):
		m.handleReject(strings.TrimSpace(m.rejectionInput.Value()))
		return m, nil
	}
	var cmd tea.Cmd
	m.rejectionInput, cmd = m.rejectionInput.Update(msg)
	m.rejectionError = ""
	m.resizeViewport()
	m.syncViewport()
	return m, cmd
}

func (m *Model) handleReject(note string) {
	if m.session == nil {
		return
	}
	if note == "" {
		m.rejectionError = "Add a short note so the agent knows what to change."
		m.resizeViewport()
		m.syncViewport()
		return
	}
	target, ok := m.selectedReviewTarget()
	if !ok {
		return
	}
	for _, step := range target.steps {
		if err := m.session.TransitionStep(step, coop.StepActive); err != nil {
			m.err = fmt.Errorf("failed to request changes: %w", err)
			return
		}
		node, _ := m.session.NodeByNumber(step)
		node.RejectionNote = note
		node.Implementation = nil
		node.Verifications = nil
	}
	if err := m.store.Write(m.session); err != nil {
		m.err = fmt.Errorf("failed to save request changes: %w", err)
	}
	m.lastVersion = m.session.Version
	m.rejecting = false
	m.rejectionInput.SetValue("")
	m.rejectionError = ""
	m.setStatus("Feedback sent. Waiting for agent...", 5*time.Second)
	m.resizeViewport()
	m.syncViewport()
}

type reviewTarget struct {
	title        string
	kind         string
	steps        []int
	chapterIndex int
}

func (m Model) selectedReviewTarget() (reviewTarget, bool) {
	if m.session == nil {
		return reviewTarget{}, false
	}
	stepNum := m.cursor + 1
	node, err := m.session.NodeByNumber(stepNum)
	if err != nil || node.State != coop.StepReview {
		return reviewTarget{}, false
	}
	if m.session.ReviewGranularityForStep(stepNum) != coop.ReviewGranularityChapter {
		return reviewTarget{title: node.Title, kind: "step", steps: []int{stepNum}, chapterIndex: -1}, true
	}
	ch, chapterIndex, _, err := m.session.ChapterByStepNumber(stepNum)
	if err != nil || !m.session.ChapterReadyForReview(chapterIndex) {
		return reviewTarget{}, false
	}
	var steps []int
	idx := 0
	for i := range m.session.Chapters {
		for j := range m.session.Chapters[i].Nodes {
			idx++
			if i == chapterIndex && m.session.Chapters[i].Nodes[j].State == coop.StepReview {
				steps = append(steps, idx)
			}
		}
	}
	if len(steps) == 0 {
		return reviewTarget{}, false
	}
	return reviewTarget{title: ch.Title, kind: "chapter", steps: steps, chapterIndex: chapterIndex}, true
}

func (m Model) reviewIsActionable(stepNum int) bool {
	if m.session == nil {
		return false
	}
	node, err := m.session.NodeByNumber(stepNum)
	if err != nil || node.State != coop.StepReview {
		return false
	}
	if m.session.ReviewGranularityForStep(stepNum) != coop.ReviewGranularityChapter {
		return true
	}
	_, chapterIndex, _, err := m.session.ChapterByStepNumber(stepNum)
	return err == nil && m.session.ChapterReadyForReview(chapterIndex)
}

func (m Model) selectedReviewCommand() string {
	target, ok := m.selectedReviewTarget()
	if !ok {
		return ""
	}
	return m.reviewCommandLabel(target.steps)
}

func (m *Model) setStatus(message string, ttl time.Duration) {
	m.statusMessage = message
	if ttl <= 0 {
		m.statusExpiresAt = time.Time{}
		return
	}
	m.statusExpiresAt = time.Now().Add(ttl)
}

func (m *Model) clearExpiredStatus(now time.Time) {
	if !m.statusExpiresAt.IsZero() && now.After(m.statusExpiresAt) {
		m.statusMessage = ""
		m.statusExpiresAt = time.Time{}
	}
}
