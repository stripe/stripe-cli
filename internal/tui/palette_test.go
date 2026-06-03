package tui

import (
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/stretchr/testify/assert"
)

func TestSearchHit_Item(t *testing.T) {
	h := searchHit{title: "Accept a payment", url: "/payments/accept-a-payment"}
	assert.Equal(t, "Accept a payment", h.FilterValue())
	assert.Equal(t, "Accept a payment", h.Title())
	assert.Equal(t, "/payments/accept-a-payment", h.Description())
}

func TestSyncKeyMap_SearchMode_LabelIsView(t *testing.T) {
	p := newPalette(Page{Content: []byte("# Test")}, nil)
	// Empty input activates the catch-all search mode.
	p.syncKeyMap()
	assert.Equal(t, "view", p.KeyMap.Execute.Help().Desc)
}

func TestSyncKeyMap_CommandsMode_LabelIsExecute(t *testing.T) {
	p := newPalette(Page{Content: []byte("# Test")}, nil)
	p.Open()
	p.Model, _ = p.Update(tea.KeyPressMsg{Code: '>', Text: ">"})
	p.syncKeyMap()
	assert.Equal(t, "execute", p.KeyMap.Execute.Help().Desc)
}

func TestSyncKeyMap_DisabledWhenNoSelection(t *testing.T) {
	p := newPalette(Page{Content: []byte("# Test")}, nil)
	// No items in search mode with empty query → nothing selected.
	p.syncKeyMap()
	assert.False(t, p.KeyMap.Execute.Enabled())
}

func TestSyncKeyMap_EnabledWhenItemSelected(t *testing.T) {
	p := newPalette(Page{Content: []byte("# Test")}, nil)
	p.Open()
	// ">" activates commands mode; empty query returns all commands.
	p.Model, _ = p.Update(tea.KeyPressMsg{Code: '>', Text: ">"})
	p.syncKeyMap()
	assert.True(t, p.KeyMap.Execute.Enabled())
}
