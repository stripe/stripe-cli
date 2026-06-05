package tui

import (
	"context"
	"fmt"
	"net/url"
	"strings"
	"time"

	"charm.land/bubbles/v2/help"
	"charm.land/bubbles/v2/key"
	"charm.land/bubbles/v2/viewport"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/joelzwarrington/foam/palette"
	"github.com/stripe/stripe-cli-docs-plugin/internal/browser"
	"github.com/stripe/stripe-cli-docs-plugin/internal/docs"
	"github.com/stripe/stripe-cli-docs-plugin/internal/markdown"
	"github.com/stripe/stripe-cli-docs-plugin/internal/ui"
)

const (
	statusBarHeight      = 1
	statusMessageTimeout = 2 * time.Second
	quitMouseResetDelay  = 50 * time.Millisecond
	maxWordWrap          = 140
)

type openWithQueryMsg string

type statusMsg string

type clearStatusMsg struct{}

type quitAfterMouseResetMsg struct{}

type pageReadyMsg struct {
	page Page
	doc  *markdown.Document
}

type pageLoadedMsg struct {
	page Page
	doc  *markdown.Document
}

// Model is the top-level Bubble Tea model for the docs TUI.
type Model struct {
	// Components
	viewport viewport.Model
	help     help.Model
	keys     KeyMap
	styles   ui.Styles
	palette  Palette

	// Dependencies
	client           *docs.Client
	renderer         markdown.Renderer
	rendererOpts     []markdown.RendererOption
	adaptiveWordWrap bool

	// Content
	page  Page
	doc   *markdown.Document
	title string

	// Initial palette input (set via WithPaletteInput)
	initialQuery string

	// State
	width         int
	height        int
	ready         bool
	loading       bool
	statusMessage string
	quitting      bool

	// Landing animation
	shape      parallelogram
	mouse      tea.Mouse
	mouseReady bool
}

// Option configures a Model.
type Option func(*Model)

// WithClient sets the docs client used to fetch pages.
func WithClient(c *docs.Client) Option {
	return func(m *Model) { m.client = c }
}

// WithRenderer sets the markdown renderer.
func WithRenderer(r markdown.Renderer) Option {
	return func(m *Model) { m.renderer = r }
}

// WithRendererOptions sets the options used to build the markdown renderer.
// The TUI rebuilds the renderer on each window resize, capping word wrap at
// maxWordWrap or the terminal width, whichever is smaller.
func WithRendererOptions(opts ...markdown.RendererOption) Option {
	return func(m *Model) {
		m.rendererOpts = opts
		m.adaptiveWordWrap = true
	}
}

// WithPage sets the page to display. The TUI parses the markdown content
// internally and derives the title from the first h1 heading.
func WithPage(p Page) Option {
	return func(m *Model) { m.page = p }
}

// WithPaletteInput opens the command palette on startup with the given text
// pre-filled. Useful for launching the TUI from a search subcommand so the
// user lands directly in the search palette.
func WithPaletteInput(q string) Option {
	return func(m *Model) { m.initialQuery = q }
}

// WithKeyMap sets a custom keymap.
func WithKeyMap(km KeyMap) Option {
	return func(m *Model) { m.keys = km }
}

// WithStyles sets custom styles.
func WithStyles(s ui.Styles) Option {
	return func(m *Model) { m.styles = s }
}

// WithWindowSize pre-seeds the terminal dimensions so the viewport can be
// initialised in New() without waiting for the first tea.WindowSizeMsg.
func WithWindowSize(width, height int) Option {
	return func(m *Model) {
		m.width = width
		m.height = height
	}
}

// New creates a Model configured with the given options.
func New(opts ...Option) Model {
	h := help.New()
	h.FullSeparator = " • "

	m := Model{
		keys: DefaultKeyMap(),
		help: h,
	}
	m.WithOptions(opts...)

	if m.page.Content != nil {
		if doc, err := markdown.Parse(m.page.Content); err == nil {
			m.doc = doc
			m.title = doc.Title()
		}
	}

	if m.doc == nil {
		m.shape = newParallelogram(paraWidth, paraHeight)
	}

	m.palette = newPalette(m.page, m.doc, m.client)
	m.setScrollEnabled(!m.isLanding())

	if m.width > 0 && m.height > 0 {
		m = m.initViewport(tea.WindowSizeMsg{Width: m.width, Height: m.height})
	}

	return m
}

func (m *Model) setScrollEnabled(enabled bool) {
	m.keys.Up.SetEnabled(enabled)
	m.keys.Down.SetEnabled(enabled)
	m.keys.PageUp.SetEnabled(enabled)
	m.keys.PageDown.SetEnabled(enabled)
}

// WithOptions applies the given options to the Model.
func (m *Model) WithOptions(opts ...Option) {
	for _, opt := range opts {
		opt(m)
	}
}

// Init returns the initial command to run when the TUI starts.
func (m Model) Init() tea.Cmd {
	if m.isLanding() {
		if m.initialQuery != "" {
			return tea.Batch(m.shape.tick, func() tea.Msg {
				return openWithQueryMsg(m.initialQuery)
			})
		}
		return m.shape.tick
	}
	return nil
}

func (m Model) isLanding() bool {
	return m.doc == nil
}

func (m Model) beginQuit() (tea.Model, tea.Cmd) {
	if m.quitting {
		return m, nil
	}
	m.quitting = true
	return m, tea.Tick(quitMouseResetDelay, func(_ time.Time) tea.Msg {
		return quitAfterMouseResetMsg{}
	})
}

// initViewport performs first-frame setup: builds the renderer at the correct
// word-wrap width, creates the viewport, and renders the initial content.
func (m Model) initViewport(msg tea.WindowSizeMsg) Model {
	if m.adaptiveWordWrap {
		renderWidth := min(msg.Width, maxWordWrap)
		opts := append(append([]markdown.RendererOption(nil), m.rendererOpts...), markdown.WithWordWrap(renderWidth))
		if r, err := markdown.NewRenderer(opts...); err == nil {
			m.renderer = r
		}
	}
	m.viewport = viewport.New(
		viewport.WithWidth(msg.Width),
		viewport.WithHeight(m.viewportHeight()),
	)
	m.viewport.MouseWheelEnabled = true
	m.viewport.MouseWheelDelta = 1
	m.viewport.KeyMap = viewport.KeyMap{}
	m.help.SetWidth(msg.Width)
	if m.doc != nil && m.renderer != nil {
		if out, err := m.renderer.Render(m.doc); err == nil {
			m.viewport.SetContent(out)
		}
	}
	m.ready = true
	return m
}

// Update handles incoming messages and updates the model state.
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	if m.quitting {
		if _, ok := msg.(quitAfterMouseResetMsg); ok {
			return m, tea.Quit
		}
		return m, nil
	}

	switch msg := msg.(type) {
	case pageLoadedMsg:
		m.loading = false
		m.page = msg.page
		m.doc = msg.doc
		m.title = msg.doc.Title()
		m.palette = newPalette(m.page, m.doc, m.client)
		m.setScrollEnabled(true)
		if m.ready && m.renderer != nil {
			if out, err := m.renderer.Render(msg.doc); err == nil {
				m.viewport.SetContent(out)
			}
		}
		return m, nil

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		if !m.ready {
			// Initial setup: rebuild renderer and render synchronously so
			// content is ready before the first frame.
			m = m.initViewport(msg)
		} else {
			m.viewport.SetWidth(msg.Width)
			m.viewport.SetHeight(m.viewportHeight())
			m.help.SetWidth(msg.Width)
			// Re-render with the new word wrap out-of-band to avoid blocking
			// the resize. The result is discarded if the width changed again
			// before it arrives.
			if m.adaptiveWordWrap && m.doc != nil {
				return m, m.rerenderCmd()
			}
		}

	case rerenderMsg:
		if msg.forWidth == m.width {
			m.viewport.SetContent(msg.content)
		}
		return m, nil

	case openWithQueryMsg:
		focusCmd := m.palette.Open()
		m.palette.syncKeyMap()
		query := string(msg)
		return m, tea.Batch(focusCmd, func() tea.Msg {
			return tea.PasteMsg{Content: query}
		})

	case closePaletteMsg:
		m.palette.Dismiss()
		return m, nil

	case palette.SelectedMsg:
		if hit, ok := msg.Item.(searchHit); ok {
			m.palette.Dismiss()
			client := m.client
			renderer := m.renderer
			return m, func() tea.Msg {
				u, err := url.Parse(hit.url)
				if err != nil {
					return statusMsg("Invalid URL")
				}
				fetched, err := client.FetchPage(context.Background(), u)
				if err != nil {
					return statusMsg("Failed to load page")
				}
				doc, err := markdown.Parse(fetched.Content)
				if err != nil {
					return statusMsg("Failed to parse page")
				}
				_ = renderer
				return pageReadyMsg{
					page: Page{Content: fetched.Content, URL: fetched.URL},
					doc:  doc,
				}
			}
		}
		if hit, ok := msg.Item.(reference); ok {
			m.palette.Dismiss()
			if hit.external {
				u := hit.url
				return m, func() tea.Msg {
					_ = browser.Open(context.Background(), u)
					return statusMsg("Opened!")
				}
			}
			return m, m.fetchPageCmd(hit.url)
		}

	case pageReadyMsg:
		m.page = msg.page
		m.doc = msg.doc
		m.title = msg.doc.Title()
		m.palette = newPalette(m.page, m.doc, m.client)
		if m.renderer != nil {
			if out, err := m.renderer.Render(msg.doc); err == nil {
				m.viewport.SetContent(out)
			}
		}
		return m, nil

	case statusMsg:
		m.statusMessage = string(msg)
		return m, tea.Tick(statusMessageTimeout, func(_ time.Time) tea.Msg {
			return clearStatusMsg{}
		})

	case clearStatusMsg:
		m.statusMessage = ""
		return m, nil

	case animationFrameMsg:
		if m.isLanding() {
			if cmd, ok := m.shape.update(msg, m.mouse, m.width, m.height); ok {
				return m, cmd
			}
		}
		return m, nil
	case tea.MouseMsg:
		if m.isLanding() {
			prev := m.mouse
			m.mouse = msg.Mouse()
			if m.mouseReady {
				dx := float64(m.mouse.X - prev.X)
				dy := float64(m.mouse.Y - prev.Y)
				m.shape.addMotion(dx, dy)
			} else {
				m.mouseReady = true
				m.shape.addMotion(5, 5) // initial impulse so the effect is visible on first interaction
			}
			return m, nil
		}
	case tea.KeyPressMsg:
		if m.palette.Visible() {
			switch {
			case key.Matches(msg, m.keys.Quit):
				return m.beginQuit()
			case msg.String() == "esc":
				m.palette.Dismiss()
				return m, nil
			default:
				m.palette.Model, cmd = m.palette.Update(msg)
				m.palette.syncKeyMap()
				return m, cmd
			}
		}

		switch {
		case key.Matches(msg, m.keys.Enter):
			if m.isLanding() && !m.loading {
				m.loading = true
				return m, m.fetchPageCmd(&url.URL{Path: "/"})
			}
		case key.Matches(msg, m.keys.Palette):
			focusCmd := m.palette.Open()
			m.palette.Model, cmd = m.palette.Update(msg)
			m.palette.syncKeyMap()
			return m, tea.Batch(focusCmd, cmd)
		case key.Matches(msg, m.keys.Search):
			focusCmd := m.palette.Open()
			m.palette.syncKeyMap()
			return m, focusCmd
		case key.Matches(msg, m.keys.Reference):
			focusCmd := m.palette.Open()
			m.palette.Model, cmd = m.palette.Update(msg)
			m.palette.syncKeyMap()
			return m, tea.Batch(focusCmd, cmd)
		case key.Matches(msg, m.keys.Help):
			m.help.ShowAll = !m.help.ShowAll
			m.viewport.SetHeight(m.viewportHeight())
		case key.Matches(msg, m.keys.Quit):
			return m.beginQuit()
		case key.Matches(msg, m.keys.Up):
			m.viewport.ScrollUp(1)
		case key.Matches(msg, m.keys.Down):
			m.viewport.ScrollDown(1)
		case key.Matches(msg, m.keys.PageUp):
			m.viewport.PageUp()
		case key.Matches(msg, m.keys.PageDown):
			m.viewport.PageDown()
		case key.Matches(msg, m.keys.OpenInBrowser):
			if m.isLanding() {
				_ = browser.Open(context.Background(), &url.URL{Scheme: "https", Host: docsHost, Path: "/"})
			} else {
				_ = m.page.Open(context.Background())
			}
		}
	}

	if m.palette.Visible() {
		m.palette.Model, cmd = m.palette.Update(msg)
		m.palette.syncKeyMap()
		return m, cmd
	}

	if !m.isLanding() {
		m.viewport, cmd = m.viewport.Update(msg)
	}
	return m, cmd
}

// View renders the current model state to the terminal.
func (m Model) View() tea.View {
	if !m.ready {
		return tea.NewView("loading...")
	}

	var content string
	if m.isLanding() {
		content = m.landing()
	} else {
		content = m.viewport.View() + "\n" + m.status()
		if m.help.ShowAll {
			helpView := lipgloss.NewStyle().PaddingTop(1).PaddingBottom(1).Render(m.help.View(m.keys))
			content += "\n" + helpView
		}
	}

	if m.palette.Visible() {
		content = m.palette.View(content, m.width, m.height)
	}

	view := tea.NewView(content)
	view.AltScreen = true
	if m.quitting {
		view.MouseMode = tea.MouseModeNone
	} else if m.isLanding() {
		view.MouseMode = tea.MouseModeAllMotion
	} else {
		view.MouseMode = tea.MouseModeCellMotion
	}
	if m.title != "" {
		view.WindowTitle = m.title
	} else if m.doc != nil {
		view.WindowTitle = m.doc.Title()
	} else {
		view.WindowTitle = "stripe docs"
	}
	return view
}

func (m Model) viewportHeight() int {
	h := m.height - statusBarHeight
	if m.help.ShowAll {
		helpView := lipgloss.NewStyle().PaddingTop(1).PaddingBottom(1).Render(m.help.View(m.keys))
		h -= strings.Count(helpView, "\n") + 1
	}
	return max(1, h)
}

func (m Model) status() string {
	bar := m.styles.Bar
	if m.statusMessage != "" {
		bar = m.styles.Success
	}

	title := m.styles.Brand.Render("Stripe")

	name := ""
	if m.statusMessage != "" {
		name = bar.Padding(0, 1).Render(m.statusMessage)
	} else if m.title != "" {
		name = bar.Padding(0, 1).Render(m.title)
	}

	var rightLabel string
	if m.isLanding() {
		if m.loading {
			rightLabel = "Loading..."
		} else {
			rightLabel = "/ search • ↵ browse"
		}
	} else {
		rightLabel = fmt.Sprintf("%3.f%%", m.viewport.ScrollPercent()*100)
	}
	rightPill := bar.Padding(0, 1).Render(rightLabel)
	helpPill := m.styles.StatusHelp.Render("? help")

	left := title + name
	right := rightPill + helpPill

	gap := max(0, m.width-lipgloss.Width(left)-lipgloss.Width(right))
	fill := lipgloss.PlaceHorizontal(gap, lipgloss.Left, "",
		lipgloss.WithWhitespaceStyle(bar))

	return left + fill + right
}

// rerenderMsg carries the result of an out-of-band word-wrap re-render.
type rerenderMsg struct {
	content  string
	forWidth int
}

// rerenderCmd rebuilds the renderer with an adaptive word-wrap width and
// re-renders the document in a goroutine, returning a rerenderMsg.
func (m Model) rerenderCmd() tea.Cmd {
	width := m.width
	renderWidth := min(width, maxWordWrap)
	opts := append(append([]markdown.RendererOption(nil), m.rendererOpts...), markdown.WithWordWrap(renderWidth))
	doc := m.doc
	return func() tea.Msg {
		r, err := markdown.NewRenderer(opts...)
		if err != nil {
			return nil
		}
		out, err := r.Render(doc)
		if err != nil {
			return nil
		}
		return rerenderMsg{content: out, forWidth: width}
	}
}

func (m Model) fetchPageCmd(dest *url.URL) tea.Cmd {
	client := m.client
	base := m.page.URL
	return func() tea.Msg {
		u := dest
		if !u.IsAbs() {
			if base == nil {
				base = &url.URL{Scheme: "https", Host: docsHost}
			}
			u = base.ResolveReference(u)
		}
		fetched, err := client.FetchPage(context.Background(), u)
		if err != nil {
			return statusMsg("Failed to load page")
		}
		doc, err := markdown.Parse(fetched.Content)
		if err != nil {
			return statusMsg("Failed to parse page")
		}
		return pageReadyMsg{
			page: Page{Content: fetched.Content, URL: fetched.URL},
			doc:  doc,
		}
	}
}

// Landing animation

func (m Model) landing() string {
	logo := m.shape.view(m.styles.LandingDotBright, m.styles.LandingDotMid, m.styles.LandingDotDim)
	title := m.styles.LandingTitle.Render("stripe docs")
	subtitle := m.styles.LandingSubtitle.Render("Search, browse, and read Stripe documentation from the terminal")

	block := lipgloss.JoinVertical(
		lipgloss.Center,
		logo,
		"",
		title,
		subtitle,
	)

	bodyHeight := m.height - statusBarHeight
	if m.help.ShowAll {
		helpView := lipgloss.NewStyle().PaddingTop(1).PaddingBottom(1).Render(m.help.View(m.keys))
		bodyHeight -= strings.Count(helpView, "\n") + 1
	}

	out := lipgloss.Place(m.width, max(1, bodyHeight), lipgloss.Center, lipgloss.Center, block)
	out += "\n" + m.status()
	if m.help.ShowAll {
		helpView := lipgloss.NewStyle().PaddingTop(1).PaddingBottom(1).Render(m.help.View(m.keys))
		out += "\n" + helpView
	}
	return out
}
