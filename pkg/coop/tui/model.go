package tui

import (
	"fmt"
	"math"
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

	selectionCursor int // node index in work view, completion option index in completion view
	selected        navigationItem
	collapsedSteps  map[int]bool
	// stepStateSignatures records what a step looked like when the user last
	// toggled it, so the override expires when the step changes.
	stepStateSignatures map[int]string

	// confirmedStepIndex/confirmedUntil hold the settle frame after a confirm.
	confirmedStepIndex int
	confirmedUntil     time.Time
	detailTab          int
	// cardCollapsed tracks the reader closing the selected step's card. Moving
	// to another step clears it, so arriving on a step opens it.
	cardCollapsed bool
	width         int
	height        int
	userMoved     bool

	rejecting       bool // true while the request-changes input is active
	rejectTarget    reviewTarget
	rejectionInput  textarea.Model
	rejectionError  string
	statusMessage   string
	statusExpiresAt time.Time

	keys  keyMap
	help  help.Model
	theme Theme

	viewport viewport.Model
	ready    bool

	// outlineWidthOverride, when > 0, constrains outline rule and wrap widths to a
	// specific column width instead of the full content width. Used by the split
	// workspace so dividers and wrapped text fit the narrow left column.
	outlineWidthOverride int

	spinner        spinner.Model
	err            error
	sdkSnippet     string
	sdkSnippetNode int
	sdkLoading     bool
	sdkLoadingNode int

	waiting                bool
	waitingMessage         string
	existingSessionIDs     map[string]bool
	lastUpdateTime         time.Time
	agentIsIdle            bool
	reviewDecisionNotifier ReviewDecisionNotifier

	agentHeartbeatMissing bool
	// consecutiveReadErrors tracks failed session polls, so a single transient
	// one does not surface as a fatal error.
	consecutiveReadErrors int

	isDark  bool
	focused bool // true when terminal has focus (default: true, updated via FocusMsg/BlurMsg)
}

func newThemedSpinner(t Theme) spinner.Model {
	return spinner.New(
		spinner.WithSpinner(spinner.MiniDot),
		spinner.WithStyle(lipgloss.NewStyle().Foreground(t.Purple500)),
	)
}

func newThemedRejectionInput(t Theme) textarea.Model {
	ti := textarea.New()
	ti.Prompt = ""
	ti.Placeholder = "Describe what to change..."
	ti.ShowLineNumbers = false
	ti.EndOfBufferCharacter = 0
	ti.DynamicHeight = true
	ti.MinHeight = 1
	ti.MaxHeight = 3
	// Bubbles treats MaxHeight as an input limit unless MaxContentHeight is set.
	// Keep the viewport compact without truncating longer feedback.
	ti.MaxContentHeight = math.MaxInt
	ti.SetVirtualCursor(false)
	ti.SetWidth(60)
	styles := ti.Styles()
	styles.Focused.Placeholder = lipgloss.NewStyle().Foreground(t.Gray500).Italic(true)
	styles.Focused.Text = lipgloss.NewStyle().Foreground(t.Text)
	styles.Focused.CursorLine = lipgloss.NewStyle()
	styles.Focused.Base = lipgloss.NewStyle()
	styles.Blurred = styles.Focused
	ti.SetStyles(styles)
	return ti
}

func (m *Model) applyTheme(isDark bool) {
	m.isDark = isDark
	m.theme = NewTheme(isDark)
	m.spinner.Style = lipgloss.NewStyle().Foreground(m.theme.Purple500)
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
func NewWaitingModel(store *coop.Store, existingSessionIDs map[string]bool, opts ...Option) Model {
	t := NewTheme(true)

	m := Model{
		store:              store,
		spinner:            newThemedSpinner(t),
		rejectionInput:     newThemedRejectionInput(t),
		keys:               newKeyMap(),
		help:               newThemedHelp(t),
		theme:              t,
		isDark:             true,
		focused:            true,
		sdkSnippetNode:     -1,
		sdkLoadingNode:     -1,
		waiting:            true,
		existingSessionIDs: existingSessionIDs,
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

	case mouseActionMsg:
		return m.handleMouseAction(msg)

	case tea.WindowSizeMsg:
		m.handleWindowSize(msg)
		return m, nil

	case tickMsg:
		m.clearExpiredStatus(time.Now())
		if !m.focused {
			return m, tickCmd()
		}
		return m, tea.Batch(m.checkForUpdates(), tickCmd())

	case noUpdateMsg:
		m.clearReadError()
		m.updateAgentIdle(msg.heartbeatAge, msg.heartbeatOK, time.Now())
		return m, nil

	case waitingBaselineMsg:
		if msg.err != nil {
			m.err = fmt.Errorf("failed to snapshot existing sessions: %w", msg.err)
			return m, nil
		}
		m.existingSessionIDs = msg.existingSessionIDs
		return m, nil

	case sessionDiscoveredMsg:
		m.waiting = false
		m.waitingMessage = ""
		m.sessionID = msg.sessionID
		m.resetSessionViewState()
		return m, m.loadSession()

	case sessionUpdatedMsg:
		m.clearReadError()
		wasComplete := m.session != nil && m.session.IsComplete()
		m.resumeFollowingIfReviewAppeared(msg.session)
		m.session = msg.session
		m.lastVersion = msg.session.Version
		m.lastUpdateTime = time.Now()
		m.agentIsIdle = false

		// Child session completed → return to parent with step marked done
		if !wasComplete && m.session.IsComplete() && m.session.ParentSessionID != "" {
			return m, m.returnToParent()
		}

		// Reset selection when transitioning to completion view.
		if !wasComplete && m.session.IsComplete() {
			m.resetSelectionState()
			m.clearStatus()
			m.clearRejectionState()
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
		m.recordReadError(msg.err)
		return m, tickCmd()

	case statusMsg:
		m.setStatus(msg.message, msg.ttl)
		m.resizeViewport()
		m.syncViewport()
		return m, nil

	case sdkSnippetMsg:
		if msg.step == m.sdkLoadingNode {
			m.sdkLoading = false
			m.sdkLoadingNode = -1
		}
		if msg.err == nil && msg.step == m.selectionCursor {
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
	return m.updateRejectionInputIfActive(msg)
}

func (m Model) updateRejectionInputIfActive(msg tea.Msg) (tea.Model, tea.Cmd) {
	if !m.rejecting {
		return m, nil
	}
	return m.updateRejectionInput(msg)
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
	// This surfaces in tmux's status line and the OS taskbar without the pane
	// being focused, which is the only signal that reaches a user watching the
	// agent's pane instead of this one.
	state := tea.ProgressBarDefault
	if m.agentIdle() || m.agentHeartbeatMissing || m.actionableReviewCount() > 0 {
		state = tea.ProgressBarWarning
	}
	return tea.NewProgressBar(state, value)
}

// resumeFollowingIfReviewAppeared clears the manual-navigation latch when a
// step first becomes actionable. userMoved is set by any navigation key and
// otherwise clears only with `f`, so without this a review could appear
// off-screen with nothing bringing the user back to it — which is exactly the
// moment they are most likely to have stepped away.
func (m *Model) resumeFollowingIfReviewAppeared(next *coop.Session) {
	if m.session == nil || next == nil || m.actionableReviewCount() > 0 {
		return
	}
	after := Model{session: next}
	if after.actionableReviewCount() > 0 {
		m.userMoved = false
	}
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
		cursor.Color = m.theme.Purple500
		cursor.Blink = true
		return cursor
	}
	return nil
}

// --- State management ---

func (m *Model) resetSessionViewState() {
	m.resetSelectionState()
	m.clearRejectionState()
	m.clearStatus()
	m.clearSDKSnippetState()
}

func (m *Model) resetSelectionState() {
	m.selectionCursor = 0
	m.selected = navigationItem{}
	m.collapsedSteps = nil
	m.cardCollapsed = false
	m.userMoved = false
}

func (m *Model) clearRejectionState() {
	m.rejecting = false
	m.rejectTarget = reviewTarget{}
	m.rejectionInput.SetValue("")
	m.rejectionInput.Blur()
	m.rejectionError = ""
}

func (m *Model) clearStatus() {
	m.statusMessage = ""
	m.statusExpiresAt = time.Time{}
}

func (m *Model) clearSDKSnippetState() {
	m.sdkSnippet = ""
	m.sdkSnippetNode = -1
	m.sdkLoading = false
	m.sdkLoadingNode = -1
}

func (m *Model) handleWindowSize(msg tea.WindowSizeMsg) {
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
}

func (m *Model) resizeViewport() {
	if !m.ready || m.height == 0 {
		return
	}
	footer := m.renderFooter()
	if m.session != nil && m.session.IsComplete() {
		footer = m.renderCompletionFooter()
	}
	// Same definition the view draws with, so the viewport is never sized to
	// more rows than actually appear on screen.
	vpHeight := m.viewportRegionHeight(m.renderHeader(), footer)
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
		m.ensureCompletionCursorVisible()
		return
	}
	m.ensureValidNavigationSelection()
	content := m.renderStepList()
	m.viewport.SetContent(content)
	// The editor renders at the end of the step's detail, and the viewport
	// offset is driven by the outline's selected row — which knows nothing
	// about the textarea growing as you type, so the pane clipped the text
	// nearest the cursor while the user was writing it. Only takes over when
	// the editor is actually inside the viewport: in the stacked layout it
	// lives in the footer, and the cursor still governs scrolling.
	if m.rejecting && m.ensureEditorVisible(content) {
		return
	}
	if !m.userMoved {
		m.scrollToCursor()
	}
}

// ensureEditorVisible scrolls the feedback editor into view, reporting whether
// it was found in the viewport's content at all.
func (m *Model) ensureEditorVisible(content string) bool {
	for i, line := range strings.Split(content, "\n") {
		if strings.Contains(ansi.Strip(line), "Request changes") {
			// Scroll explicitly rather than through EnsureVisible: the rows the
			// overflow indicator claims are not rows the viewport knows about,
			// so it considered the editor visible while it sat just under the
			// fold. Reserve them here.
			// Two rows for the overflow indicator, and one more for the border
			// the clipped card draws to close itself — that closing line lands
			// on the last visible row, so without it the editor was scrolled to
			// exactly the row the border then took.
			visible := m.viewport.Height() - viewportIndicatorRows - 1
			if visible < 1 {
				visible = 1
			}
			if bottom := i - visible + 1; bottom > m.viewport.YOffset() {
				m.viewport.SetYOffset(bottom)
			}
			return true
		}
	}
	return false
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

func (m *Model) ensureCompletionCursorVisible() {
	line, ok := m.completionLineForCursor()
	if !ok {
		return
	}
	m.viewport.EnsureVisible(line, 0, 0)
}

func (m Model) completionLineForCursor() (int, bool) {
	if m.selectionCursor < 0 {
		return 0, false
	}
	for line, suggestion := range m.completionSuggestionLines() {
		if suggestion == m.selectionCursor {
			return line, true
		}
	}
	return 0, false
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
			m.cardCollapsed = false
			return
		}
	}
	idx := 0
	for i := range m.session.Steps {
		for j := range m.session.Steps[i].Nodes {
			if m.session.Steps[i].Nodes[j].State == coop.NodeReview && m.reviewIsActionable(idx+1) {
				m.selectNode(idx)
				m.cardCollapsed = false
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
	// In the completion view only suggestion navigation, selection, quit, and
	// claim-open are meaningful. Gate everything else so work-view keys (expand,
	// collapse, confirm, follow, etc.) can't leak in and mutate hidden state or
	// fire stray commands against a completion cursor reinterpreted as a node.
	if m.session != nil && m.session.IsComplete() {
		return m.handleCompletionKey(msg)
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

// handleCompletionKey handles the limited key set valid in the completion view.
func (m Model) handleCompletionKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch {
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
	case key.Matches(msg, m.keys.Enter):
		return m.handleEnter()
	case key.Matches(msg, m.keys.OpenClaim):
		if claimURL := m.sandboxClaimLink(); claimURL != "" {
			return m, openBrowserCmd(claimURL)
		}
		return m, nil
	case key.Matches(msg, m.keys.Quit):
		return m, tea.Quit
	}
	return m, nil
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
			m.cardCollapsed = false
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
		if items := m.navigationItems(); len(items) > 0 {
			m.selectNavigationItem(items[0])
		}
		m.resizeViewport()
		m.syncViewport()
		m.viewport.GotoTop()
		return m, nil, true
	case key.Matches(msg, m.keys.Bottom):
		m.userMoved = true
		if items := m.navigationItems(); len(items) > 0 {
			m.selectNavigationItem(items[len(items)-1])
		}
		m.resizeViewport()
		m.syncViewport()
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
		m.cardCollapsed = !m.cardCollapsed
		m.resizeViewport()
		m.syncViewport()
		if !m.cardCollapsed {
			return m, m.fetchSnippetIfNeeded()
		}
		return m, nil
	case key.Matches(msg, m.keys.Enter):
		return m.handleEnter()
	case key.Matches(msg, m.keys.Tab):
		// Gated on the card being explicitly opened, this stopped working the moment it began
		// rendering in place without being opened first. It also cycled modulo
		// a fixed list of three while a step offers only the tabs it has
		// content for, so the count could disagree with the strip on screen.
		if count := m.visibleDetailTabCount(); count > 1 {
			m.detailTab = (m.detailTab + 1) % count
			m.syncViewport()
			return m, m.fetchSnippetIfNeeded()
		}
		return m, nil
	case key.Matches(msg, m.keys.Escape):
		if !m.cardCollapsed {
			m.cardCollapsed = true
			m.resizeViewport()
			m.syncViewport()
		}
		return m, nil
	case key.Matches(msg, m.keys.Follow):
		// Jump straight to whatever is waiting on the user, rather than only
		// resuming auto-follow: the footer advertises this as "go to it".
		m.userMoved = false
		if stepIndex, ok := m.firstActionableReviewStep(); ok {
			m.selectStep(stepIndex)
		} else {
			m.autoScroll()
		}
		m.setStatus("Following the current review step", 3*time.Second)
		m.resizeViewport()
		m.syncViewport()
		return m, nil
	case key.Matches(msg, m.keys.Confirm):
		return m, m.handleConfirm()
	case key.Matches(msg, m.keys.OpenClaim):
		if claimURL := m.sandboxClaimLink(); claimURL != "" {
			return m, openBrowserCmd(claimURL)
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
		return m, m.startReject()
	}
	return m, nil
}

func (m *Model) moveCursorUp() {
	if m.session != nil && m.session.IsComplete() {
		suggestions := m.getCompletionSuggestions()
		if len(suggestions) == 0 {
			return
		}
		if m.selectionCursor > 0 {
			m.selectionCursor--
		} else {
			m.selectionCursor = len(suggestions) - 1
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
		if m.selectionCursor < len(suggestions)-1 {
			m.selectionCursor++
		} else {
			m.selectionCursor = 0
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
		if m.selectionCursor < len(suggestions) {
			selected := suggestions[m.selectionCursor]
			cmd := m.selectCompletionOption()

			switch selected.id {
			case "deploy", "deploy-update":
				waitCmd := m.enterWaitingMode("Waiting for agent to start the guided deploy flow...")
				return m, tea.Batch(cmd, waitCmd)
			case "add-integration":
				waitCmd := m.enterWaitingMode("Waiting for agent to ask which Stripe feature to add...")
				return m, tea.Batch(cmd, waitCmd)
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
	m.cardCollapsed = !m.cardCollapsed
	m.resizeViewport()
	m.syncViewport()
	if !m.cardCollapsed {
		return m, m.fetchSnippetIfNeeded()
	}
	return m, nil
}

func (m *Model) enterWaitingMode(message string) tea.Cmd {
	m.waiting = true
	m.waitingMessage = message
	m.session = nil
	m.resetSessionViewState()
	m.existingSessionIDs = nil
	return m.snapshotWaitingBaseline()
}

func (m *Model) handleConfirm() tea.Cmd {
	if m.session == nil {
		return nil
	}
	target, ok := m.selectedReviewTarget()
	if !ok {
		return nil
	}
	session, err := workflow.NewService(m.store).ConfirmReview(m.session.ID, target.nodeNumbers)
	if err != nil {
		m.err = fmt.Errorf("failed to confirm review: %w", err)
		return nil
	}
	m.session = session
	m.lastVersion = m.session.Version
	m.userMoved = false
	// Hold a brief acknowledgement on the step itself. The status line sits
	// several rows below the card the user was reading, so without this the
	// thing they acted on simply vanished with no confirmation at the point
	// their eyes were.
	m.confirmedStepIndex = target.stepIndex
	m.confirmedUntil = time.Now().Add(confirmSettleDuration)
	m.setStatus("Confirmed. Waiting for agent...", 5*time.Second)
	m.clearRejectionState()
	if m.session.IsComplete() {
		m.resetSelectionState()
		m.clearStatus()
	}
	m.resizeViewport()
	m.syncViewport()
	notify := m.notifyReviewDecision()
	if m.session.IsComplete() && m.session.ParentSessionID != "" {
		return tea.Batch(notify, m.returnToParent())
	}
	return notify
}

func (m *Model) startReject() tea.Cmd {
	if m.session == nil {
		return nil
	}
	if target, ok := m.selectedReviewTarget(); ok {
		m.rejecting = true
		m.rejectTarget = target
		m.rejectionInput.SetValue("")
		m.rejectionInput.Placeholder = m.requestChangesPlaceholder(target)
		m.rejectionError = ""
		m.clearStatus()
		m.resizeViewport()
		m.syncViewport()
		return m.rejectionInput.Focus()
	}
	return nil
}

func (m Model) handleRejectionKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch {
	case key.Matches(msg, m.keys.Escape):
		m.clearRejectionState()
		m.setStatus("Request changes canceled.", 3*time.Second)
		m.resizeViewport()
		m.syncViewport()
		return m, nil
	case key.Matches(msg, m.keys.Submit):
		return m, m.handleReject(strings.TrimSpace(m.rejectionInput.Value()))
	}
	return m.updateRejectionInput(msg)
}

func (m Model) updateRejectionInput(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	m.rejectionInput, cmd = m.rejectionInput.Update(msg)
	m.rejectionError = ""
	m.resizeViewport()
	m.syncViewport()
	return m, cmd
}

func (m *Model) handleReject(note string) tea.Cmd {
	if m.session == nil {
		return nil
	}
	if note == "" {
		m.rejectionError = "Add a short note so the agent knows what to change."
		m.resizeViewport()
		m.syncViewport()
		return nil
	}
	if !m.reviewTargetStillValid(m.rejectTarget) {
		m.clearRejectionState()
		m.setStatus("Review target changed. Request changes canceled.", 4*time.Second)
		m.resizeViewport()
		m.syncViewport()
		return nil
	}
	target := m.rejectTarget
	session, err := workflow.NewService(m.store).RequestChanges(m.session.ID, target.nodeNumbers, note)
	if err != nil {
		m.rejectionError = fmt.Sprintf("failed to request changes: %v", err)
		m.resizeViewport()
		m.syncViewport()
		return nil
	}
	m.session = session
	m.lastVersion = m.session.Version
	m.userMoved = false
	m.clearRejectionState()
	m.setStatus("Feedback sent. Waiting for agent...", 5*time.Second)
	m.resizeViewport()
	m.syncViewport()
	return m.notifyReviewDecision()
}

func (m Model) notifyReviewDecision() tea.Cmd {
	if m.reviewDecisionNotifier == nil || m.session == nil {
		return nil
	}
	sessionID := m.session.ID
	notify := m.reviewDecisionNotifier
	return func() tea.Msg {
		if err := notify(sessionID); err != nil {
			return statusMsg{
				message: fmt.Sprintf("Agent wake-up failed. In the agent pane, run: %s", coop.ResumeCommand(sessionID)),
			}
		}
		return statusMsg{message: "Human decision sent. Waiting for agent...", ttl: 5 * time.Second}
	}
}

type reviewTarget struct {
	title       string
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
		return reviewTarget{title: ch.TitleText(), nodeNumbers: nodeNumbers, stepIndex: stepIndex}, true
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
	step, stepIndex, _, err := m.session.StepByNodeNumber(nodeNumber)
	if err != nil || !m.session.StepReadyForReview(stepIndex) {
		return reviewTarget{}, false
	}
	var nodeNumbers []int
	current := 0
	for i := range m.session.Steps {
		for j := range m.session.Steps[i].Nodes {
			current++
			if i == stepIndex && m.session.Steps[i].Nodes[j].State == coop.NodeReview {
				nodeNumbers = append(nodeNumbers, current)
			}
		}
	}
	if len(nodeNumbers) == 0 {
		return reviewTarget{}, false
	}
	return reviewTarget{title: step.TitleText(), nodeNumbers: nodeNumbers, stepIndex: stepIndex}, true
}

func (m Model) reviewIsActionable(nodeNumber int) bool {
	if m.session == nil {
		return false
	}
	node, err := m.session.NodeByNumber(nodeNumber)
	if err != nil || node.State != coop.NodeReview {
		return false
	}
	_, stepIndex, _, err := m.session.StepByNodeNumber(nodeNumber)
	return err == nil && m.session.StepReadyForReview(stepIndex)
}

func (m Model) reviewTargetStillValid(target reviewTarget) bool {
	if m.session == nil || len(target.nodeNumbers) == 0 {
		return false
	}
	for _, nodeNumber := range target.nodeNumbers {
		node, err := m.session.NodeByNumber(nodeNumber)
		if err != nil || node.State != coop.NodeReview {
			return false
		}
	}
	return target.stepIndex >= 0 && target.stepIndex < len(m.session.Steps) && m.session.StepReadyForReview(target.stepIndex)
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

// stepJustConfirmed reports whether a step is inside its post-confirm settle.
// recordReadError holds a failed poll back from the view until it repeats.
//
// The session file is read every 500ms while the agent writes it, so a torn
// read is routine. Treating one as fatal left the view stuck on an error string
// forever while the poll kept succeeding underneath — worst during exactly the
// step-away-and-return this UI is built for.
func (m *Model) recordReadError(err error) {
	m.consecutiveReadErrors++
	if m.consecutiveReadErrors >= readErrorTolerance {
		m.err = err
	}
}

// clearReadError recovers the view after a poll succeeds again.
func (m *Model) clearReadError() {
	m.consecutiveReadErrors = 0
	m.err = nil
}

func (m Model) stepJustConfirmed(stepIndex int) bool {
	return !m.confirmedUntil.IsZero() && m.confirmedStepIndex == stepIndex && time.Now().Before(m.confirmedUntil)
}

func (m *Model) clearExpiredStatus(now time.Time) {
	if !m.statusExpiresAt.IsZero() && now.After(m.statusExpiresAt) {
		m.statusMessage = ""
		m.statusExpiresAt = time.Time{}
	}
}

func (m *Model) updateAgentIdle(heartbeatAge time.Duration, heartbeatOK bool, now time.Time) {
	if m.session == nil || m.session.IsComplete() {
		m.agentIsIdle = false
		m.agentHeartbeatMissing = false
		return
	}
	if !heartbeatOK {
		// No heartbeat file at all, which is different from an agent that is
		// merely quiet: the process is gone. Treating it as "not idle" left the
		// UI claiming the agent was still working indefinitely. Only trust it
		// once the session has reported at least once, so a session that has
		// not started yet is not mislabeled as crashed.
		m.agentIsIdle = false
		m.agentHeartbeatMissing = !m.lastUpdateTime.IsZero()
		return
	}
	m.agentHeartbeatMissing = false
	if heartbeatAge >= 0 && heartbeatAge < 5*time.Second {
		m.agentIsIdle = false
		return
	}
	if m.lastUpdateTime.IsZero() {
		m.agentIsIdle = false
		return
	}
	m.agentIsIdle = now.Sub(m.lastUpdateTime) > 2*time.Minute
}
