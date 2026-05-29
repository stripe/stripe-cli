package tui

import (
	"fmt"
	"net/url"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/stripe/stripe-cli-docs-plugin/internal/docs"
	"github.com/stripe/stripe-cli-docs-plugin/markdown"
)

func TestNew_Defaults(t *testing.T) {
	m := New()
	assert.Equal(t, DefaultKeyMap(), m.keys)
	assert.False(t, m.ready)
}

func TestNew_WithOptions(t *testing.T) {
	client := docs.NewClient("test")
	r, err := markdown.NewRenderer()
	require.NoError(t, err)

	m := New(
		WithClient(client),
		WithRenderer(r),
		WithPage(Page{
			Content: []byte("# Hello"),
			URL:     &url.URL{Scheme: "https", Host: "docs.stripe.com", Path: "/payments"},
		}),
	)

	assert.Equal(t, client, m.client)
	assert.Equal(t, r, m.renderer)
	assert.NotNil(t, m.doc)
	assert.Equal(t, "Hello", m.title)
}

func TestNew_WithPage_ParsesTitle(t *testing.T) {
	m := New(WithPage(Page{Content: []byte("# My Page\n\nBody")}))
	assert.Equal(t, "My Page", m.title)
}

func TestUpdate_WindowSizeMsg_InitializesViewport(t *testing.T) {
	m := New()
	msg := tea.WindowSizeMsg{Width: 80, Height: 24}

	result, _ := m.Update(msg)
	model := result.(Model)

	assert.True(t, model.ready)
	assert.Equal(t, 80, model.width)
	assert.Equal(t, 24, model.height)
}

func TestUpdate_WindowSizeMsg_RendersDocument(t *testing.T) {
	r, err := markdown.NewRenderer()
	require.NoError(t, err)

	m := New(
		WithRenderer(r),
		WithPage(Page{Content: []byte("# Hello\n\nWorld")}),
	)
	msg := tea.WindowSizeMsg{Width: 80, Height: 24}

	result, _ := m.Update(msg)
	model := result.(Model)

	assert.NotEmpty(t, model.viewport.GetContent())
}

func TestUpdate_QuitKey(t *testing.T) {
	m := New()
	// Initialize viewport first
	m.Update(tea.WindowSizeMsg{Width: 80, Height: 24})

	result, cmd := m.Update(tea.KeyPressMsg{Code: 'q', Text: "q"})
	_ = result

	assert.NotNil(t, cmd)
}

func TestUpdate_ScrollKeys(t *testing.T) {
	r, err := markdown.NewRenderer()
	require.NoError(t, err)

	long := "# Title\n"
	for i := range 50 {
		long += fmt.Sprintf("Line %d\n", i)
	}

	m := New(
		WithRenderer(r),
		WithPage(Page{Content: []byte(long)}),
	)
	result, _ := m.Update(tea.WindowSizeMsg{Width: 80, Height: 10})
	model := result.(Model)

	// Scroll down
	result, _ = model.Update(tea.KeyPressMsg{Code: 'j', Text: "j"})
	model = result.(Model)
	assert.Greater(t, model.viewport.YOffset(), 0)
}

func TestView_BeforeReady(t *testing.T) {
	m := New()
	view := m.View()
	assert.Equal(t, "loading...", view.Content)
}

func TestView_AltScreenEnabled(t *testing.T) {
	m := New()
	result, _ := m.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	model := result.(Model)

	view := model.View()
	assert.True(t, view.AltScreen)
	assert.Equal(t, tea.MouseModeCellMotion, view.MouseMode)
}

func TestView_WindowTitle_FromPage(t *testing.T) {
	r, err := markdown.NewRenderer()
	require.NoError(t, err)

	m := New(
		WithRenderer(r),
		WithPage(Page{Content: []byte("# Custom Title\n\nBody")}),
	)
	result, _ := m.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	model := result.(Model)

	view := model.View()
	assert.Equal(t, "Custom Title", view.WindowTitle)
}
