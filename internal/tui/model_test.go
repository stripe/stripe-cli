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

func TestUpdate_OpenInBrowser(t *testing.T) {
	calls := stubBrowser(t)
	u := &url.URL{Scheme: "https", Host: "docs.stripe.com", Path: "/payments"}
	m := New(WithPage(Page{Content: []byte("# Payments"), URL: u}))

	result, _ := m.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	model := result.(Model)

	model.Update(tea.KeyPressMsg{Code: 'o', Text: "o"})
	require.Len(t, *calls, 1)
	assert.Contains(t, (*calls)[0].Args, "https://docs.stripe.com/payments")
}

func TestUpdate_OpenInBrowser_NilURL(t *testing.T) {
	calls := stubBrowser(t)
	m := New(WithPage(Page{Content: []byte("# Hello")}))

	result, _ := m.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	model := result.(Model)

	model.Update(tea.KeyPressMsg{Code: 'o', Text: "o"})
	assert.Empty(t, *calls)
}

func TestPalette_ContainsOpenInBrowser(t *testing.T) {
	u := &url.URL{Scheme: "https", Host: "docs.stripe.com", Path: "/payments"}
	m := New(WithPage(Page{Content: []byte("# Payments"), URL: u}))

	result, _ := m.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	model := result.(Model)

	// Open palette
	result, _ = model.Update(tea.KeyPressMsg{Code: '>', Text: ">"})
	model = result.(Model)

	items := model.palette.Items()
	var found bool
	for _, item := range items {
		if item.FilterValue() == "Open in browser" {
			found = true
			break
		}
	}
	assert.True(t, found, "palette should contain 'Open in browser' command")
}

func TestPalette_OpenInBrowser_ExecutesCommand(t *testing.T) {
	calls := stubBrowser(t)
	u := &url.URL{Scheme: "https", Host: "docs.stripe.com", Path: "/payments"}
	m := New(WithPage(Page{Content: []byte("# Payments"), URL: u}))

	result, _ := m.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	model := result.(Model)

	// Open palette and filter to "open"
	result, _ = model.Update(tea.KeyPressMsg{Code: '>', Text: ">"})
	model = result.(Model)
	for _, ch := range "open" {
		result, _ = model.Update(tea.KeyPressMsg{Code: ch, Text: string(ch)})
		model = result.(Model)
	}

	// Execute selected command with Enter
	result, cmd := model.Update(tea.KeyPressMsg{Code: tea.KeyEnter, Text: "enter"})
	model = result.(Model)

	// The command returns a batch; execute returned cmds to trigger the browser open
	if cmd != nil {
		msg := cmd()
		if batchMsg, ok := msg.(tea.BatchMsg); ok {
			for _, c := range batchMsg {
				if c != nil {
					c()
				}
			}
		}
	}

	require.Len(t, *calls, 1)
	assert.Contains(t, (*calls)[0].Args, "https://docs.stripe.com/payments")
}

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
	m := New(WithPage(Page{Content: []byte("# Hello")}))
	result, _ := m.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	model := result.(Model)

	view := model.View()
	assert.True(t, view.AltScreen)
	assert.Equal(t, tea.MouseModeCellMotion, view.MouseMode)
}

func TestView_LandingMouseMode(t *testing.T) {
	m := New()
	result, _ := m.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	model := result.(Model)

	view := model.View()
	assert.True(t, view.AltScreen)
	assert.Equal(t, tea.MouseModeAllMotion, view.MouseMode)
}

func TestInit_LandingReturnsTick(t *testing.T) {
	m := New()
	cmd := m.Init()
	assert.NotNil(t, cmd)
}

func TestInit_WithDocReturnsNil(t *testing.T) {
	m := New(WithPage(Page{Content: []byte("# Hello")}))
	cmd := m.Init()
	assert.Nil(t, cmd)
}

func TestUpdate_TickMsg_Landing(t *testing.T) {
	m := New()
	result, _ := m.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	model := result.(Model)

	result, cmd := model.Update(animationFrameMsg{})
	model = result.(Model)
	assert.NotNil(t, cmd)
}

func TestUpdate_PaletteOpensOnGreaterThan(t *testing.T) {
	m := New(WithPage(Page{Content: []byte("# Test\n\nBody")}))
	m.Update(tea.WindowSizeMsg{Width: 80, Height: 24})

	result, _ := m.Update(tea.KeyPressMsg{Code: '>', Text: ">"})
	model := result.(Model)

	assert.True(t, model.palette.Visible())
}

func TestUpdate_PaletteDismissesOnEsc(t *testing.T) {
	m := New(WithPage(Page{Content: []byte("# Test\n\nBody")}))
	m.Update(tea.WindowSizeMsg{Width: 80, Height: 24})

	result, _ := m.Update(tea.KeyPressMsg{Code: '>', Text: ">"})
	model := result.(Model)
	assert.True(t, model.palette.Visible())

	result, _ = model.Update(tea.KeyPressMsg{Code: tea.KeyEscape, Text: "esc"})
	model = result.(Model)
	assert.False(t, model.palette.Visible())
}

func TestUpdate_PaletteGatesInput(t *testing.T) {
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

	// Open palette
	result, _ = model.Update(tea.KeyPressMsg{Code: '>', Text: ">"})
	model = result.(Model)

	// j should not scroll the viewport while palette is open
	offset := model.viewport.YOffset()
	result, _ = model.Update(tea.KeyPressMsg{Code: 'j', Text: "j"})
	model = result.(Model)
	assert.Equal(t, offset, model.viewport.YOffset())
}

func TestUpdate_StatusMsg(t *testing.T) {
	m := New()
	m.Update(tea.WindowSizeMsg{Width: 80, Height: 24})

	result, cmd := m.Update(statusMsg("Copied!"))
	model := result.(Model)

	assert.Equal(t, "Copied!", model.statusMessage)
	assert.NotNil(t, cmd)
}

func TestUpdate_ClearStatusMsg(t *testing.T) {
	m := New()
	m.statusMessage = "Copied!"

	result, _ := m.Update(clearStatusMsg{})
	model := result.(Model)

	assert.Empty(t, model.statusMessage)
}

func TestStatus_ShowsStatusMessage(t *testing.T) {
	m := New(WithPage(Page{Content: []byte("# Title\n\nBody")}))
	result, _ := m.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	model := result.(Model)

	model.statusMessage = "Copied!"
	status := model.status()
	assert.Contains(t, status, "Copied!")
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
