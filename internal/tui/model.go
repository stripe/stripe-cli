package tui

import (
	"fmt"
	"strings"

	"charm.land/bubbles/v2/help"
	"charm.land/bubbles/v2/key"
	"charm.land/bubbles/v2/viewport"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/stripe/stripe-cli-docs-plugin/internal/docs"
	"github.com/stripe/stripe-cli-docs-plugin/markdown"
)

const statusBarHeight = 1

// Model is the top-level Bubble Tea model for the docs TUI.
type Model struct {
	// Components
	viewport viewport.Model
	help     help.Model
	keys     KeyMap
	styles   Styles
	palette  Palette

	// Dependencies
	client   *docs.Client
	renderer markdown.Renderer

	// Content
	page  Page
	doc   *markdown.Document
	title string

	// State
	width  int
	height int
	ready  bool
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
func WithStyles(s Styles) Option {
	return func(m *Model) { m.styles = s }
}

// New creates a Model configured with the given options.
func New(opts ...Option) Model {
	h := help.New()
	h.FullSeparator = " • "
	s := DefaultStyles()

	m := Model{
		keys:   DefaultKeyMap(),
		help:   h,
		styles: s,
	}
	m.WithOptions(opts...)

	if m.page.Content != nil {
		if doc, err := markdown.Parse(m.page.Content); err == nil {
			m.doc = doc
			m.title = doc.Title()
		}
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
	return nil
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

	case tea.KeyPressMsg:
		if m.palette.Visible() {
			if msg.String() == "esc" {
				m.palette.Dismiss()
				return m, nil
			}
			m.palette.Model, cmd = m.palette.Update(msg)
			return m, cmd
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
		}
	}

	if m.palette.Visible() {
		m.palette.Model, cmd = m.palette.Update(msg)
		return m, cmd
	}

	m.viewport, cmd = m.viewport.Update(msg)
	return m, cmd
}

// View renders the current model state to the terminal.
func (m Model) View() tea.View {
	if !m.ready {
		return tea.NewView("loading...")
	}

	content := m.viewport.View() + "\n" + m.status()
	if m.help.ShowAll {
		helpView := lipgloss.NewStyle().PaddingTop(1).PaddingBottom(1).Render(m.help.View(m.keys))
		content += "\n" + helpView
	}

	if m.palette.Visible() {
		content = m.palette.View(content, m.width, m.height)
	}

	view := tea.NewView(content)
	view.AltScreen = true
	view.MouseMode = tea.MouseModeCellMotion
	if m.title != "" {
		view.WindowTitle = m.title
	} else if m.doc != nil {
		view.WindowTitle = m.doc.Title()
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

	title := m.styles.StatusTitle.Render("Stripe")

	name := ""
	if m.title != "" {
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
