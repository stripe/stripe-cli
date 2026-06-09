package tui

import (
	"fmt"
	"net/url"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/stripe/stripe-cli-docs-plugin/internal/docs"
	"github.com/stripe/stripe-cli-docs-plugin/internal/markdown"
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
	assert.False(t, m.ready)
	// Scroll bindings are disabled on the landing screen.
	assert.False(t, m.keys.Up.Enabled())
	assert.False(t, m.keys.Down.Enabled())
	assert.False(t, m.keys.PageUp.Enabled())
	assert.False(t, m.keys.PageDown.Enabled())
	// Non-scroll bindings are always enabled.
	assert.True(t, m.keys.Quit.Enabled())
	assert.True(t, m.keys.Help.Enabled())
	assert.True(t, m.keys.Palette.Enabled())
	assert.True(t, m.keys.OpenInBrowser.Enabled())
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
	result, _ := m.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	model := result.(Model)

	result, cmd := model.Update(tea.KeyPressMsg{Code: 'q', Text: "q"})
	model = result.(Model)

	require.NotNil(t, cmd)
	assert.True(t, model.quitting)
}

func TestUpdate_QuitCtrlC(t *testing.T) {
	m := New()
	result, _ := m.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	model := result.(Model)

	result, cmd := model.Update(tea.KeyPressMsg{Text: "ctrl+c"})
	model = result.(Model)

	require.NotNil(t, cmd)
	assert.True(t, model.quitting)
}

func TestUpdate_ScrollKeys(t *testing.T) {
	long := "# Title\n"
	for i := range 50 {
		long += fmt.Sprintf("Line %d\n", i)
	}

	m := New(
		WithRendererOptions(),
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

func TestNew_WithWindowSize_ReadyImmediately(t *testing.T) {
	m := New(WithWindowSize(80, 24))
	view := m.View()
	assert.NotEqual(t, "loading...", view.Content)
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

func TestInit_WithPaletteInput_ReturnsCmd(t *testing.T) {
	m := New(WithPaletteInput("payment methods"))
	cmd := m.Init()
	assert.NotNil(t, cmd)
}

func TestUpdate_OpenWithQueryMsg_OpensPaletteWithQuery(t *testing.T) {
	m := New()
	result, _ := m.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	model := result.(Model)

	result, batchCmd := model.Update(openWithQueryMsg("payment methods"))
	model = result.(Model)

	assert.True(t, model.palette.Visible())

	// The batch cmd contains focus + paste; execute it to deliver the paste message.
	if batchCmd != nil {
		if msg := batchCmd(); msg != nil {
			if batchMsg, ok := msg.(tea.BatchMsg); ok {
				for _, c := range batchMsg {
					if c != nil {
						if pasteMsg, ok := c().(tea.PasteMsg); ok {
							result, _ = model.Update(pasteMsg)
							model = result.(Model)
						}
					}
				}
			}
		}
	}

	assert.Equal(t, "payment methods", model.palette.Value())
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

func TestUpdate_PaletteQuitsOnCtrlC(t *testing.T) {
	m := New(WithPage(Page{Content: []byte("# Test\n\nBody")}))
	m.Update(tea.WindowSizeMsg{Width: 80, Height: 24})

	result, _ := m.Update(tea.KeyPressMsg{Code: '>', Text: ">"})
	model := result.(Model)
	assert.True(t, model.palette.Visible())

	_, cmd := model.Update(tea.KeyPressMsg{Code: 'c', Text: "c", Mod: tea.ModCtrl})
	assert.NotNil(t, cmd)
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

func TestUpdate_PaletteOpensOnSlash(t *testing.T) {
	m := New(WithPage(Page{Content: []byte("# Test\n\nBody")}))
	m.Update(tea.WindowSizeMsg{Width: 80, Height: 24})

	result, _ := m.Update(tea.KeyPressMsg{Code: '/', Text: "/"})
	model := result.(Model)

	assert.True(t, model.palette.Visible())
}

func TestUpdate_PaletteOpenSlash_EmptyInput(t *testing.T) {
	m := New(WithPage(Page{Content: []byte("# Test\n\nBody")}))
	m.Update(tea.WindowSizeMsg{Width: 80, Height: 24})

	result, _ := m.Update(tea.KeyPressMsg{Code: '/', Text: "/"})
	model := result.(Model)

	// Slash must not be forwarded to the palette input — search mode has no prefix.
	assert.Empty(t, model.palette.Value())
}

func TestUpdate_PaletteOpenSlash_SyncKeyMap(t *testing.T) {
	m := New(WithPage(Page{Content: []byte("# Test\n\nBody")}))
	m.Update(tea.WindowSizeMsg{Width: 80, Height: 24})

	result, _ := m.Update(tea.KeyPressMsg{Code: '/', Text: "/"})
	model := result.(Model)

	// syncKeyMap must be called immediately on open, not deferred to next keystroke.
	assert.Equal(t, "view", model.palette.KeyMap.Execute.Help().Desc)
}

func TestUpdate_PaletteOpensOnAt(t *testing.T) {
	u := mustParseURL("https://docs.stripe.com/payments")
	doc := docWithReferences(u)
	m := New(WithPage(Page{Content: []byte("# Test\n\nBody"), URL: &url.URL{Scheme: "https", Host: "docs.stripe.com", Path: "/test"}}))
	m.doc = doc
	m.palette = newPalette(m.page, m.doc, m.client)
	m.Update(tea.WindowSizeMsg{Width: 80, Height: 24})

	result, _ := m.Update(tea.KeyPressMsg{Code: '@', Text: "@"})
	model := result.(Model)

	assert.True(t, model.palette.Visible())
}

func TestUpdate_PaletteOpenAt_InputContainsAt(t *testing.T) {
	u := mustParseURL("https://docs.stripe.com/payments")
	doc := docWithReferences(u)
	m := New(WithPage(Page{Content: []byte("# Test\n\nBody"), URL: &url.URL{Scheme: "https", Host: "docs.stripe.com", Path: "/test"}}))
	m.doc = doc
	m.palette = newPalette(m.page, m.doc, m.client)
	m.Update(tea.WindowSizeMsg{Width: 80, Height: 24})

	result, _ := m.Update(tea.KeyPressMsg{Code: '@', Text: "@"})
	model := result.(Model)

	assert.Equal(t, "@", model.palette.Value())
}

func TestUpdate_PaletteOpenAt_SyncKeyMap(t *testing.T) {
	u := mustParseURL("https://docs.stripe.com/payments")
	doc := docWithReferences(u)
	m := New(WithPage(Page{Content: []byte("# Test\n\nBody"), URL: &url.URL{Scheme: "https", Host: "docs.stripe.com", Path: "/test"}}))
	m.doc = doc
	m.palette = newPalette(m.page, m.doc, m.client)
	m.Update(tea.WindowSizeMsg{Width: 80, Height: 24})

	result, _ := m.Update(tea.KeyPressMsg{Code: '@', Text: "@"})
	model := result.(Model)

	assert.Equal(t, "view", model.palette.KeyMap.Execute.Help().Desc)
}

func TestUpdate_PageReadyMsg(t *testing.T) {
	r, err := markdown.NewRenderer()
	require.NoError(t, err)

	m := New(
		WithRenderer(r),
		WithPage(Page{Content: []byte("# Original")}),
	)
	result, _ := m.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	model := result.(Model)

	newDoc, err := markdown.Parse([]byte("# New Page\n\nBody"))
	require.NoError(t, err)

	newPage := Page{Content: []byte("# New Page\n\nBody"), URL: &url.URL{Path: "/new"}}
	result, _ = model.Update(pageReadyMsg{page: newPage, doc: newDoc})
	model = result.(Model)

	assert.Equal(t, "New Page", model.title)
	assert.Equal(t, newDoc, model.doc)
	assert.NotEmpty(t, model.viewport.GetContent())
}

func TestView_ProgressBar_IndeterminateWhenPageLoading(t *testing.T) {
	m := New(WithPage(Page{Content: []byte("# Test\n\nBody")}))
	result, _ := m.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	model := result.(Model)

	model.loading = true
	assert.Equal(t, tea.ProgressBarIndeterminate, model.View().ProgressBar.State)
}

func TestUpdate_PageReadyMsg_ClearsLoading(t *testing.T) {
	r, err := markdown.NewRenderer()
	require.NoError(t, err)

	m := New(WithRenderer(r), WithPage(Page{Content: []byte("# Original")}))
	result, _ := m.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	model := result.(Model)
	model.loading = true

	doc, err := markdown.Parse([]byte("# New\n\nBody"))
	require.NoError(t, err)
	result, _ = model.Update(pageReadyMsg{page: Page{Content: []byte("# New\n\nBody")}, doc: doc})
	model = result.(Model)

	assert.False(t, model.loading)
}

func TestUpdate_StatusMsg_ClearsLoading(t *testing.T) {
	m := New(WithPage(Page{Content: []byte("# Test\n\nBody")}))
	result, _ := m.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	model := result.(Model)
	model.loading = true

	result, _ = model.Update(statusMsg("Failed to load page"))
	model = result.(Model)

	assert.False(t, model.loading)
}

func TestView_ProgressBar_NilWhenPaletteHidden(t *testing.T) {
	m := New(WithPage(Page{Content: []byte("# Test\n\nBody")}))
	result, _ := m.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	model := result.(Model)

	assert.False(t, model.palette.Visible())
	assert.Nil(t, model.View().ProgressBar)
}

func TestView_ProgressBar_NilWhenPaletteVisibleButNotLoading(t *testing.T) {
	m := New(WithPage(Page{Content: []byte("# Test\n\nBody")}))
	result, _ := m.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	model := result.(Model)

	result, _ = model.Update(tea.KeyPressMsg{Code: '/', Text: "/"})
	model = result.(Model)

	assert.True(t, model.palette.Visible())
	assert.False(t, model.palette.Loading())
	assert.Nil(t, model.View().ProgressBar)
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
