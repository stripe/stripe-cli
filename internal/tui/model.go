package tui

import (
	"context"
	"fmt"
	"strings"
	"time"

	"charm.land/bubbles/v2/help"
	"charm.land/bubbles/v2/key"
	"charm.land/bubbles/v2/viewport"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/stripe/stripe-cli-docs-plugin/internal/docs"
	"github.com/stripe/stripe-cli-docs-plugin/internal/markdown"
	"github.com/stripe/stripe-cli-docs-plugin/internal/ui"
)

const (
	statusBarHeight      = 1
	statusMessageTimeout = 2 * time.Second
)

type statusMsg string

type clearStatusMsg struct{}

// Model is the top-level Bubble Tea model for the docs TUI.
type Model struct {
	// Components
	viewport viewport.Model
	help     help.Model
	keys     KeyMap
	styles   ui.Styles
	palette  Palette

	// Dependencies
	client   *docs.Client
	renderer markdown.Renderer

	// Content
	page  Page
	doc   *markdown.Document
	title string

	// State
	width         int
	height        int
	ready         bool
	statusMessage string

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

// WithPage sets the page to display. The TUI parses the markdown content
// internally and derives the title from the first h1 heading.
func WithPage(p Page) Option {
	return func(m *Model) { m.page = p }
}

// WithKeyMap sets a custom keymap.
func WithKeyMap(km KeyMap) Option {
	return func(m *Model) { m.keys = km }
}

// WithStyles sets custom styles.
func WithStyles(s ui.Styles) Option {
	return func(m *Model) { m.styles = s }
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

	m.palette = newPalette(m.page)

	return m
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
		return m.shape.tick
	}
	return nil
}

func (m Model) isLanding() bool {
	return m.doc == nil
}

// Update handles incoming messages and updates the model state.
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		if !m.ready {
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
		} else {
			m.viewport.SetWidth(msg.Width)
			m.viewport.SetHeight(m.viewportHeight())
			m.help.SetWidth(msg.Width)
		}

	case closePaletteMsg:
		m.palette.Dismiss()
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
				return m, tea.Quit
			case msg.String() == "esc":
				m.palette.Dismiss()
				return m, nil
			default:
				m.palette.Model, cmd = m.palette.Update(msg)
				return m, cmd
			}
		}

		switch {
		case key.Matches(msg, m.keys.Palette):
			focusCmd := m.palette.Open()
			m.palette.Model, cmd = m.palette.Update(msg)
			return m, tea.Batch(focusCmd, cmd)
		case key.Matches(msg, m.keys.Help):
			m.help.ShowAll = !m.help.ShowAll
			m.viewport.SetHeight(m.viewportHeight())
		case key.Matches(msg, m.keys.Quit):
			return m, tea.Quit
		case key.Matches(msg, m.keys.Up):
			m.viewport.ScrollUp(1)
		case key.Matches(msg, m.keys.Down):
			m.viewport.ScrollDown(1)
		case key.Matches(msg, m.keys.PageUp):
			m.viewport.PageUp()
		case key.Matches(msg, m.keys.PageDown):
			m.viewport.PageDown()
		case key.Matches(msg, m.keys.OpenInBrowser):
			_ = m.page.Open(context.Background())
		}
	}

	if m.palette.Visible() {
		m.palette.Model, cmd = m.palette.Update(msg)
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
	if m.isLanding() {
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
	bar := m.styles.StatusBar
	if m.statusMessage != "" {
		bar = m.styles.StatusMessage
	}

	title := m.styles.StatusTitle.Render("Stripe")

	name := ""
	if m.statusMessage != "" {
		name = bar.Padding(0, 1).Render(m.statusMessage)
	} else if m.title != "" {
		name = bar.Padding(0, 1).Render(m.title)
	}

	percent := bar.Padding(0, 1).Render(fmt.Sprintf("%3.f%%", m.viewport.ScrollPercent()*100))
	helpPill := m.styles.StatusHelp.Render("? help")

	left := title + name
	right := percent + helpPill

	gap := max(0, m.width-lipgloss.Width(left)-lipgloss.Width(right))
	fill := lipgloss.PlaceHorizontal(gap, lipgloss.Left, "",
		lipgloss.WithWhitespaceStyle(bar))

	return left + fill + right
}

// Landing animation

func (m Model) landing() string {
	logo := m.shape.view(m.styles.LandingDotBright, m.styles.LandingDotMid, m.styles.LandingDotDim)
	title := m.styles.LandingTitle.Render("stripe docs")
	subtitle := m.styles.LandingSubtitle.Render("Search, browse, and read Stripe documentation from the terminal")
	hint := m.styles.LandingHint.Render("stripe docs <path>  to get started")

	block := lipgloss.JoinVertical(
		lipgloss.Center,
		logo,
		"",
		title,
		subtitle,
		"",
		hint,
	)

	return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, block)
}
