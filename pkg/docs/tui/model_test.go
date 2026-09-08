package tui

import (
	"fmt"
	"net/url"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/stripe/stripe-cli/pkg/docs"
	"github.com/stripe/stripe-cli/pkg/docs/markdown"
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
	_ = result.(Model)

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
	assert.False(t, m.keys.Back.Enabled())
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

func TestUpdate_WindowSizeMsg_RerenderPreservesPosition(t *testing.T) {
	src := []byte("# Title\n\n" + strings.Repeat("A paragraph with enough words to wrap across lines. ", 20) + "\n\n## Target\n\n" + strings.Repeat("Body after the target.\n\n", 10))
	m := New(
		WithRendererOptions(markdown.WithStyle("notty")),
		WithPage(Page{Content: src, URL: &url.URL{Path: "/page", Fragment: "target"}}),
	)
	result, _ := m.Update(tea.WindowSizeMsg{Width: 80, Height: 8})
	model := result.(Model)
	initialOffset := model.viewport.YOffset()
	require.Greater(t, initialOffset, 0)

	result, cmd := model.Update(tea.WindowSizeMsg{Width: 35, Height: 8})
	model = result.(Model)
	require.NotNil(t, cmd)
	msg := cmd()
	require.NotNil(t, msg)
	result, _ = model.Update(msg)
	model = result.(Model)

	assert.Equal(t, model.headingOffsets["target"], model.viewport.YOffset())
	assert.Greater(t, model.viewport.YOffset(), initialOffset)

	model.activeFragment = ""
	model.viewport.SetYOffset(5)
	result, cmd = model.Update(tea.WindowSizeMsg{Width: 45, Height: 8})
	model = result.(Model)
	require.NotNil(t, cmd)
	result, _ = model.Update(cmd())
	model = result.(Model)
	assert.Equal(t, 5, model.viewport.YOffset())
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
	_ = result.(Model)
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

func TestUpdate_PaletteQuitKeyDoesNotQuit(t *testing.T) {
	m := New(WithPage(Page{Content: []byte("# Test\n\nBody")}))
	result, _ := m.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	model := result.(Model)

	// Open the palette
	result, _ = model.Update(tea.KeyPressMsg{Code: '>', Text: ">"})
	model = result.(Model)
	assert.True(t, model.palette.Visible())

	// Pressing q should not quit when the palette is open
	result, _ = model.Update(tea.KeyPressMsg{Code: 'q', Text: "q"})
	model = result.(Model)
	assert.False(t, model.quitting)
	assert.True(t, model.palette.Visible())
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

func TestUpdate_ForwardNavigationStartsAtTop(t *testing.T) {
	r, err := markdown.NewRenderer(markdown.WithStyle("notty"))
	require.NoError(t, err)
	m := New(WithRenderer(r), WithPage(Page{Content: longDocument("Original"), URL: &url.URL{Path: "/original"}}))
	result, _ := m.Update(tea.WindowSizeMsg{Width: 80, Height: 8})
	model := result.(Model)
	model.viewport.SetYOffset(10)
	require.Greater(t, model.viewport.YOffset(), 0)

	doc, err := markdown.Parse(longDocument("Next"))
	require.NoError(t, err)
	result, _ = model.Update(pageReadyMsg{page: Page{Content: doc.Source, URL: &url.URL{Path: "/next"}}, doc: doc})
	model = result.(Model)

	assert.Equal(t, 0, model.viewport.YOffset())
	require.Len(t, model.history, 1)
	assert.Greater(t, model.history[0].yOffset, 0)
}

func TestUpdate_FragmentNavigation(t *testing.T) {
	r, err := markdown.NewRenderer(markdown.WithStyle("notty"), markdown.WithWordWrap(40))
	require.NoError(t, err)
	m := New(WithRenderer(r), WithPage(Page{Content: []byte("# Original")}))
	result, _ := m.Update(tea.WindowSizeMsg{Width: 40, Height: 8})
	model := result.(Model)

	src := []byte("# Next\n\n" + strings.Repeat("Paragraph content.\n\n", 10) + "## Target heading\n\n" + strings.Repeat("Body after the target.\n\n", 10))
	doc, err := markdown.Parse(src)
	require.NoError(t, err)
	result, _ = model.Update(pageReadyMsg{page: Page{Content: src, URL: &url.URL{Path: "/next", Fragment: "target-heading"}}, doc: doc})
	model = result.(Model)

	assert.Equal(t, model.headingOffsets["target-heading"], model.viewport.YOffset())
	assert.Greater(t, model.viewport.YOffset(), 0)
	assert.Equal(t, "target-heading", model.activeFragment)

	missingPage := Page{Content: src, URL: &url.URL{Path: "/missing", Fragment: "missing"}}
	result, _ = model.Update(pageReadyMsg{page: missingPage, doc: doc})
	model = result.(Model)
	assert.Equal(t, 0, model.viewport.YOffset())
	assert.Empty(t, model.activeFragment)
}

func TestUpdate_BackWithEmptyHistoryIsNoOp(t *testing.T) {
	page := Page{Content: []byte("# Original"), URL: &url.URL{Path: "/original"}}
	m := New(WithPage(page))

	result, _ := m.Update(tea.KeyPressMsg{Code: tea.KeyBackspace})
	model := result.(Model)

	assert.False(t, model.keys.Back.Enabled())
	assert.Empty(t, model.history)
	assert.Equal(t, page, model.page)
	assert.Equal(t, "Original", model.title)
}

func TestUpdate_BackRestoresPreviousPage(t *testing.T) {
	r, err := markdown.NewRenderer()
	require.NoError(t, err)

	originalPage := Page{Content: []byte("# Original\n\nOriginal body"), URL: &url.URL{Path: "/original"}}
	m := New(WithRenderer(r), WithPage(originalPage))
	result, _ := m.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	model := result.(Model)
	originalDoc := model.doc
	originalContent := model.viewport.GetContent()

	newDoc, err := markdown.Parse([]byte("# New Page\n\nNew body"))
	require.NoError(t, err)
	newPage := Page{Content: []byte("# New Page\n\nNew body"), URL: &url.URL{Path: "/new"}}
	result, _ = model.Update(pageReadyMsg{page: newPage, doc: newDoc})
	model = result.(Model)

	assert.True(t, model.keys.Back.Enabled())
	assert.Len(t, model.history, 1)
	assert.Equal(t, "New Page", model.title)
	assert.NotEqual(t, originalContent, model.viewport.GetContent())

	result, _ = model.Update(tea.KeyPressMsg{Code: tea.KeyBackspace})
	model = result.(Model)

	assert.Equal(t, originalPage, model.page)
	assert.Same(t, originalDoc, model.doc)
	assert.Equal(t, "Original", model.title)
	assert.Equal(t, originalContent, model.viewport.GetContent())
	assert.Empty(t, model.history)
	assert.False(t, model.keys.Back.Enabled())
}

func TestUpdate_BackRestoresScrollOffset(t *testing.T) {
	r, err := markdown.NewRenderer(markdown.WithStyle("notty"))
	require.NoError(t, err)

	pageA := Page{Content: longDocument("Page A"), URL: &url.URL{Path: "/a"}}
	m := New(WithRenderer(r), WithPage(pageA))
	result, _ := m.Update(tea.WindowSizeMsg{Width: 80, Height: 8})
	model := result.(Model)
	model.viewport.SetYOffset(8)
	offsetA := model.viewport.YOffset()

	docB, err := markdown.Parse(longDocument("Page B"))
	require.NoError(t, err)
	pageB := Page{Content: docB.Source, URL: &url.URL{Path: "/b"}}
	result, _ = model.Update(pageReadyMsg{page: pageB, doc: docB})
	model = result.(Model)
	model.viewport.SetYOffset(14)
	offsetB := model.viewport.YOffset()

	docC, err := markdown.Parse(longDocument("Page C"))
	require.NoError(t, err)
	result, _ = model.Update(pageReadyMsg{page: Page{Content: docC.Source, URL: &url.URL{Path: "/c"}}, doc: docC})
	model = result.(Model)

	result, _ = model.Update(tea.KeyPressMsg{Code: tea.KeyBackspace})
	model = result.(Model)
	assert.Equal(t, offsetB, model.viewport.YOffset())

	result, _ = model.Update(tea.KeyPressMsg{Code: tea.KeyBackspace})
	model = result.(Model)
	assert.Equal(t, offsetA, model.viewport.YOffset())
}

func TestUpdate_BackNavigatesMultipleHistoryEntries(t *testing.T) {
	pageA := Page{Content: []byte("# Page A"), URL: &url.URL{Path: "/a"}}
	m := New(WithPage(pageA))
	docA := m.doc

	docB, err := markdown.Parse([]byte("# Page B"))
	require.NoError(t, err)
	pageB := Page{Content: []byte("# Page B"), URL: &url.URL{Path: "/b"}}
	result, _ := m.Update(pageReadyMsg{page: pageB, doc: docB})
	model := result.(Model)

	docC, err := markdown.Parse([]byte("# Page C"))
	require.NoError(t, err)
	pageC := Page{Content: []byte("# Page C"), URL: &url.URL{Path: "/c"}}
	result, _ = model.Update(pageReadyMsg{page: pageC, doc: docC})
	model = result.(Model)

	require.Len(t, model.history, 2)
	assert.True(t, model.keys.Back.Enabled())

	result, _ = model.Update(tea.KeyPressMsg{Code: tea.KeyBackspace})
	model = result.(Model)
	assert.Equal(t, pageB, model.page)
	assert.Same(t, docB, model.doc)
	assert.Equal(t, "Page B", model.title)
	assert.True(t, model.keys.Back.Enabled())

	result, _ = model.Update(tea.KeyPressMsg{Code: tea.KeyBackspace})
	model = result.(Model)
	assert.Equal(t, pageA, model.page)
	assert.Same(t, docA, model.doc)
	assert.Equal(t, "Page A", model.title)
	assert.Empty(t, model.history)
	assert.False(t, model.keys.Back.Enabled())
}

func TestUpdate_NavigationFromLandingDoesNotAddHistory(t *testing.T) {
	m := New()
	result, cmd := m.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	model := result.(Model)
	require.NotNil(t, cmd)

	doc, err := markdown.Parse([]byte("# Documentation"))
	require.NoError(t, err)
	page := Page{Content: []byte("# Documentation"), URL: &url.URL{Path: "/"}}

	result, _ = model.Update(pageReadyMsg{page: page, doc: doc})
	model = result.(Model)

	assert.Equal(t, page, model.page)
	assert.Same(t, doc, model.doc)
	assert.Empty(t, model.history)
	assert.False(t, model.keys.Back.Enabled())
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

func longDocument(title string) []byte {
	return []byte("# " + title + "\n\n" + strings.Repeat("Paragraph content for scrolling.\n\n", 40))
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
