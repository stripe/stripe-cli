package tui

import (
	"fmt"
	"strings"
	"time"

	"charm.land/bubbles/v2/help"
	"charm.land/bubbles/v2/key"
	"charm.land/bubbles/v2/spinner"
	"charm.land/bubbles/v2/textarea"
	"charm.land/bubbles/v2/viewport"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"

	"github.com/stripe/stripe-cli/pkg/coop"
	"github.com/stripe/stripe-cli/pkg/coop/workflow"
)

// Model is the root bubbletea model for the co-op TUI.
type Model struct {
	store           *coop.Store
	sessionID       string
	session         *coop.Session
	lastVersion     int
	sandboxClaimURL string

	cursor         int
	selected       navigationItem
	collapsedSteps map[int]bool
	expanded       bool
	detailTab      int
	width          int
	height         int
	userMoved      bool

	rejecting       bool
	rejectionInput  textarea.Model
	rejectionError  string
	statusMessage   string
	statusExpiresAt time.Time

	keys  keyMap
	help  help.Model
	theme Theme

	viewport viewport.Model
	ready    bool

	spinner        spinner.Model
	err            error
	sdkSnippet     string
	sdkSnippetNode int
	sdkLoading     bool
	sdkLoadingNode int

	waiting        bool
	waitingMessage string
	existingIDs    map[string]bool
	lastUpdateTime time.Time

	isDark  bool
	focused bool // true when terminal has focus (default: true, updated via FocusMsg/BlurMsg)
}

func newThemedSpinner(t Theme) spinner.Model {
	return spinner.New(
		spinner.WithSpinner(spinner.MiniDot),
		spinner.WithStyle(lipgloss.NewStyle().Foreground(t.HuePurple500)),
	)
}

func newThemedRejectionInput(t Theme) textarea.Model {
	ti := textarea.New()
	ti.Prompt = ""
	ti.Placeholder = "Describe what to change..."
	ti.ShowLineNumbers = false
	ti.EndOfBufferCharacter = 0
	ti.CharLimit = 500
	ti.DynamicHeight = true
	ti.MinHeight = 1
	ti.MaxHeight = 3
	ti.MaxContentHeight = 6
	ti.SetVirtualCursor(false)
	ti.SetWidth(60)
	styles := ti.Styles()
	styles.Focused.Placeholder = lipgloss.NewStyle().Foreground(t.HueGray500).Italic(true)
	styles.Focused.Text = lipgloss.NewStyle().Foreground(t.HueText)
	styles.Focused.CursorLine = lipgloss.NewStyle()
	styles.Focused.Base = lipgloss.NewStyle()
	styles.Blurred = styles.Focused
	ti.SetStyles(styles)
	return ti
}

func (m *Model) applyTheme(isDark bool) {
	m.isDark = isDark
	m.theme = NewTheme(isDark)
	m.spinner.Style = lipgloss.NewStyle().Foreground(m.theme.HuePurple500)
	m.rejectionInput.SetStyles(newThemedRejectionInput(m.theme).Styles())
	m.help = newThemedHelp(m.theme)
}

// NewModel creates a TUI model for a known session.
func NewModel(store *coop.Store, sessionID string, opts ...Option) Model {
	t := NewTheme(true)

	m := Model{
		store:          store,
		sessionID:      sessionID,
		spinner:        newThemedSpinner(t),
		rejectionInput: newThemedRejectionInput(t),
		keys:           newKeyMap(),
		help:           newThemedHelp(t),
		theme:          t,
		isDark:         true,
		focused:        true,
		sdkSnippetNode: -1,
		sdkLoadingNode: -1,
	}
	for _, opt := range opts {
		opt(&m)
	}
	return m
}

// NewWaitingModel creates a TUI model that waits for a new session to appear.
func NewWaitingModel(store *coop.Store, existingIDs map[string]bool, opts ...Option) Model {
	t := NewTheme(true)

	m := Model{
		store:          store,
		spinner:        newThemedSpinner(t),
		rejectionInput: newThemedRejectionInput(t),
		keys:           newKeyMap(),
		help:           newThemedHelp(t),
		theme:          t,
		isDark:         true,
		focused:        true,
		sdkSnippetNode: -1,
		sdkLoadingNode: -1,
		waiting:        true,
		existingIDs:    existingIDs,
	}
	for _, opt := range opts {
		opt(&m)
	}
	return m
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

	case tea.MouseClickMsg:
		if action, ok := m.mouseActionFor(tea.Mouse(msg)); ok {
			return m.handleMouseAction(action)
		}
		return m, nil

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
		m.selected = navigationItem{}
		m.collapsedSteps = nil
		m.expanded = false
		m.userMoved = false
		m.rejecting = false
		m.rejectionInput.SetValue("")
		m.rejectionError = ""
		m.statusMessage = ""
		m.statusExpiresAt = time.Time{}
		m.sdkSnippet = ""
		m.sdkSnippetNode = -1
		m.sdkLoading = false
		m.sdkLoadingNode = -1
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
			m.selected = navigationItem{}
			m.collapsedSteps = nil
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
		if msg.step == m.sdkLoadingNode {
			m.sdkLoading = false
			m.sdkLoadingNode = -1
		}
		if msg.err == nil && msg.step == m.cursor {
			m.sdkSnippet = msg.snippet
			m.sdkSnippetNode = msg.step
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
		m.applyTheme(msg.IsDark())
		m.resizeViewport()
		m.syncViewport()
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
	switch {
	case m.err != nil:
		content = m.theme.ErrorStyle.Render(fmt.Sprintf("Error: %s", m.err))
	case !m.ready:
		content = m.spinner.View() + " Loading..."
	case m.waiting:
		content = m.renderWaitingView()
	case m.session == nil:
		content = m.renderWaitingView()
	case m.session.IsComplete():
		content = m.renderCompletionView()
	default:
		header := m.renderHeader()
		content = m.renderPinnedViewport(header, m.renderFooter())
	}

	v := tea.NewView(content)
	v.AltScreen = true
	v.MouseMode = tea.MouseModeCellMotion
	v.OnMouse = func(msg tea.MouseMsg) tea.Cmd {
		if action, ok := m.mouseActionFor(msg.Mouse()); ok {
			return func() tea.Msg {
				return action
			}
		}
		return nil
	}
	v.ReportFocus = true
	v.KeyboardEnhancements.ReportEventTypes = true
	v.ProgressBar = m.progressBar()
	v.Cursor = m.rejectionCursor(content)
	if m.session != nil {
		done := 0
		for _, ch := range m.session.Steps {
			for _, n := range ch.Nodes {
				if n.State == coop.NodeDone || n.State == coop.NodeSkipped {
					done++
				}
			}
		}
		v.WindowTitle = fmt.Sprintf("Co-op: %s (%d/%d)", m.session.Blueprint, done, m.session.TotalNodes())
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
	for _, ch := range m.session.Steps {
		for _, n := range ch.Nodes {
			if n.State == coop.NodeSkipped {
				continue
			}
			total++
			if n.State == coop.NodeDone {
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
		cursor := m.rejectionInput.Cursor()
		if cursor == nil {
			cursor = tea.NewCursor(0, 0)
		}
		cursor.X += lipgloss.Width(plain[:idx+len(prefix)])
		cursor.Y += y
		cursor.Shape = tea.CursorBar
		cursor.Color = m.theme.HuePurple500
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
	m.viewport.YPosition = lipgloss.Height(m.renderHeader())
	if m.rejecting {
		m.rejectionInput.SetWidth(m.requestChangesInputWidth())
	}
}

func (m *Model) syncViewport() {
	if !m.ready || m.session == nil {
		return
	}
	if m.session.IsComplete() {
		content := m.renderCompletionBody()
		m.viewport.SetContent(content)
		m.viewport.SetYOffset(0)
		return
	}
	m.ensureValidNavigationSelection()
	content := m.renderStepList()
	m.viewport.SetContent(content)
	if !m.userMoved {
		m.scrollToCursor()
	}
}

func (m *Model) scrollToCursor() {
	targetLine := m.selectedContentLine()
	m.viewport.EnsureVisible(targetLine, 0, 0)

	vpTop := m.viewport.YOffset()
	visibleHeight := m.viewport.Height()
	if m.viewport.TotalLineCount() > visibleHeight && visibleHeight >= 3 {
		visibleHeight -= 2
	}
	vpBottom := vpTop + visibleHeight
	scrollThreshold := vpBottom - 2
	if m.session != nil && m.session.IsComplete() {
		scrollThreshold = vpBottom
	}

	if targetLine < vpTop {
		m.viewport.SetYOffset(targetLine)
	} else if targetLine >= scrollThreshold {
		offset := targetLine - visibleHeight/2
		if offset < 0 {
			offset = 0
		}
		m.viewport.SetYOffset(offset)
	}
}

func (m Model) selectedContentLine() int {
	if m.session != nil && !m.session.IsComplete() {
		selectedLine := -1
		for line, item := range m.navigationContentLines() {
			if m.navigationItemSelected(item) && (selectedLine == -1 || line < selectedLine) {
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
		if strings.Contains(line, cursorMarker) {
			return i
		}
	}
	return 0
}

func (m *Model) autoScroll() {
	if m.session == nil {
		return
	}
	for i := range m.session.Steps {
		if m.stepReviewReady(i) {
			m.selectStep(i)
			m.expanded = false
			return
		}
	}
	idx := 0
	for i := range m.session.Steps {
		for j := range m.session.Steps[i].Nodes {
			if m.session.Steps[i].Nodes[j].State == coop.NodeReview && m.reviewIsActionable(idx+1) {
				m.selectNode(idx)
				m.expanded = false
				return
			}
			idx++
		}
	}
	_, activeNum := m.session.ActiveNode()
	if activeNum > 0 {
		m.selectNode(activeNum - 1)
	}
}

func (m Model) handleKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	if m.rejecting {
		return m.handleRejectionKey(msg)
	}
	if msg.IsRepeat && (key.Matches(msg, m.keys.Confirm) || key.Matches(msg, m.keys.Reject) || key.Matches(msg, m.keys.Copy) || key.Matches(msg, m.keys.OpenClaim)) {
		return m, nil
	}

	if next, cmd, ok := m.handleNavigationKey(msg); ok {
		return next, cmd
	}
	if next, cmd, ok := m.handleViewportKey(msg); ok {
		return next, cmd
	}
	return m.handleActionKey(msg)
}

func (m Model) handleNavigationKey(msg tea.KeyPressMsg) (Model, tea.Cmd, bool) {
	switch {
	case key.Matches(msg, m.keys.Up):
		m.moveCursorUp()
		m.resizeViewport()
		m.syncViewport()
		return m, nil, true
	case key.Matches(msg, m.keys.Down):
		m.moveCursorDown()
		m.resizeViewport()
		m.syncViewport()
		return m, nil, true
	case key.Matches(msg, m.keys.Left):
		if m.collapseSelectedStep() {
			m.userMoved = true
			m.expanded = false
			m.resizeViewport()
			m.syncViewport()
		}
		return m, nil, true
	case key.Matches(msg, m.keys.Right):
		if m.expandSelectedStep() {
			m.userMoved = true
			m.resizeViewport()
			m.syncViewport()
		}
		return m, nil, true
	}
	return m, nil, false
}

func (m Model) handleViewportKey(msg tea.KeyPressMsg) (Model, tea.Cmd, bool) {
	switch {
	case key.Matches(msg, m.keys.PageUp):
		m.userMoved = true
		m.viewport.PageUp()
		return m, nil, true
	case key.Matches(msg, m.keys.PageDown):
		m.userMoved = true
		m.viewport.PageDown()
		return m, nil, true
	case key.Matches(msg, m.keys.Top):
		m.userMoved = true
		m.viewport.GotoTop()
		return m, nil, true
	case key.Matches(msg, m.keys.Bottom):
		m.userMoved = true
		m.viewport.GotoBottom()
		return m, nil, true
	}
	return m, nil, false
}

func (m Model) handleActionKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch {
	case key.Matches(msg, m.keys.Quit):
		return m, tea.Quit
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
		if claimURL := m.sandboxClaimLink(); claimURL != "" {
			openBrowser(claimURL)
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
	} else {
		items := m.navigationItems()
		if len(items) == 0 {
			return
		}
		idx := m.selectedNavigationIndex()
		if idx <= 0 {
			return
		}
		m.selectNavigationItem(items[idx-1])
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
	} else {
		items := m.navigationItems()
		if len(items) == 0 {
			return
		}
		idx := m.selectedNavigationIndex()
		if idx < 0 || idx >= len(items)-1 {
			return
		}
		m.selectNavigationItem(items[idx+1])
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
	m.selected = navigationItem{}
	m.collapsedSteps = nil
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
	session, err := workflow.NewService(m.store).ConfirmReview(m.session.ID, target.nodeNumbers)
	if err != nil {
		m.err = fmt.Errorf("failed to confirm review: %w", err)
		return
	}
	m.session = session
	m.lastVersion = m.session.Version
	if target.kind == "node" && len(target.nodeNumbers) > 0 {
		m.selectNode(target.nodeNumbers[0] - 1)
	}
	m.userMoved = false
	m.setStatus("Confirmed. Waiting for agent...", 5*time.Second)
	m.rejecting = false
	m.rejectionInput.SetValue("")
	m.rejectionError = ""
	if m.session.IsComplete() {
		m.cursor = 0
		m.selected = navigationItem{}
		m.collapsedSteps = nil
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
	session, err := workflow.NewService(m.store).RequestChanges(m.session.ID, target.nodeNumbers, note)
	if err != nil {
		m.err = fmt.Errorf("failed to request changes: %w", err)
		return
	}
	m.session = session
	m.lastVersion = m.session.Version
	if target.kind == "node" && len(target.nodeNumbers) > 0 {
		m.selectNode(target.nodeNumbers[0] - 1)
	}
	m.userMoved = false
	m.rejecting = false
	m.rejectionInput.SetValue("")
	m.rejectionError = ""
	m.setStatus("Feedback sent. Waiting for agent...", 5*time.Second)
	m.resizeViewport()
	m.syncViewport()
}

type reviewTarget struct {
	title       string
	kind        string
	nodeNumbers []int
	stepIndex   int
}

func (m Model) selectedReviewTarget() (reviewTarget, bool) {
	if m.session == nil {
		return reviewTarget{}, false
	}
	if m.selected.kind == navigationStep {
		stepIndex := m.selected.stepIndex
		if !m.stepReviewReady(stepIndex) {
			return reviewTarget{}, false
		}
		ch := m.session.Steps[stepIndex]
		var nodeNumbers []int
		step := 0
		for i := range m.session.Steps {
			for j := range m.session.Steps[i].Nodes {
				step++
				if i == stepIndex && m.session.Steps[i].Nodes[j].State == coop.NodeReview {
					nodeNumbers = append(nodeNumbers, step)
				}
			}
		}
		if len(nodeNumbers) == 0 {
			return reviewTarget{}, false
		}
		return reviewTarget{title: ch.Title, kind: "step", nodeNumbers: nodeNumbers, stepIndex: stepIndex}, true
	}
	nodeIndex, ok := m.selectedNodeIndex()
	if !ok {
		return reviewTarget{}, false
	}
	nodeNumber := nodeIndex + 1
	node, err := m.session.NodeByNumber(nodeNumber)
	if err != nil || node.State != coop.NodeReview {
		return reviewTarget{}, false
	}
	if m.session.ReviewGranularityForNode(nodeNumber) != coop.ReviewGranularityStep {
		return reviewTarget{title: node.Title, kind: "node", nodeNumbers: []int{nodeNumber}, stepIndex: -1}, true
	}
	return reviewTarget{}, false
}

func (m Model) reviewIsActionable(nodeNumber int) bool {
	if m.session == nil {
		return false
	}
	node, err := m.session.NodeByNumber(nodeNumber)
	if err != nil || node.State != coop.NodeReview {
		return false
	}
	if m.session.ReviewGranularityForNode(nodeNumber) != coop.ReviewGranularityStep {
		return true
	}
	_, stepIndex, _, err := m.session.StepByNodeNumber(nodeNumber)
	return err == nil && m.session.StepReadyForReview(stepIndex)
}

func (m Model) selectedReviewCommand() string {
	target, ok := m.selectedReviewTarget()
	if !ok {
		return ""
	}
	return m.reviewCommandLabel(target.nodeNumbers)
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
